package clientmetadata

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testProvider(t *testing.T, allowed ...string) *Provider {
	t.Helper()
	log := zerolog.Nop()
	return New(&log, allowed, false)
}

func TestIsMetadataClientID(t *testing.T) {
	// The discriminator between "look this up in the client registry" and "go
	// fetch this URL". It has to be exact in both directions: too loose and an
	// ordinary client_id triggers an outbound fetch; too strict and CIMD never
	// engages.
	cases := []struct {
		clientID string
		want     bool
	}{
		{"https://app.example.com/client.json", true},
		{"https://app.example.com/oauth/metadata", true},

		{"", false},
		{"a4b66000-a396-44fe-b8bf-efe9806a911d", false}, // a real registered client_id
		{"local-client", false},
		{"http://app.example.com/client.json", false}, // http is not https
		{"https://app.example.com", false},            // no path component (spec MUST)
		{"https://app.example.com/", false},           // bare root is not a path
		{"ftp://app.example.com/client.json", false},
		{"https:///client.json", false}, // no host
	}
	for _, tc := range cases {
		t.Run(tc.clientID, func(t *testing.T) {
			assert.Equal(t, tc.want, IsMetadataClientID(tc.clientID))
		})
	}
}

// The loopback-only predicate that drives the consent warning moved to
// internal/http_handlers with the consent screen itself, so that "is this
// loopback?" has one implementation shared with the RFC 8252 redirect matcher.
// Its table test moved with it: TestConsentClientIsLoopbackOnly.

// docServer serves a metadata document whose client_id is its own URL, which is
// what a well-behaved client hosts.
func docServer(t *testing.T, body func(selfURL string) string, headers map[string]string) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body(r.Header.Get("X-Self-URL"))))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// TestResolveRejectsUnfetchableAndPrivateHosts is the SSRF guard.
//
// The client_id is an arbitrary attacker-supplied URL that this server fetches,
// which the CIMD spec names as its primary risk: without a check, any caller
// could aim the authorization server at internal addresses it can reach and use
// the error/latency as an oracle. validators.SafeHTTPClient resolves once and
// pins the dial to the validated IP, so a hostile DNS record cannot rebind
// between validation and connection.
func TestResolveRejectsUnfetchableAndPrivateHosts(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()

	// Assert on the SPECIFIC rejection, not merely that an error occurred.
	//
	// An earlier version of this test only required an error, and passed even
	// with the SSRF guard swapped for SafeHTTPClientAllowPrivate — because the
	// request then failed TLS verification against httptest's self-signed cert
	// instead. That is a different failure with different security properties,
	// and against a real attacker-controlled host with a valid certificate there
	// would have been no error at all.
	const wantRejection = "private/internal networks are not allowed"

	// httptest binds loopback, which is exactly what must be refused.
	srv, hits := docServer(t, func(string) string { return `{}` }, nil)
	_, err := p.Resolve(ctx, srv.URL+"/client.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), wantRejection,
		"a loopback client_id URL must be refused by the SSRF guard, not by some later failure")
	assert.Zero(t, *hits, "the refusal must happen before the request is made")

	for _, bad := range []string{
		"https://169.254.169.254/latest/meta-data", // cloud metadata service
		"https://10.0.0.1/client.json",             // RFC 1918
		"https://192.168.1.1/client.json",          // RFC 1918
		"https://[::1]/client.json",                // ipv6 loopback
		"https://127.0.0.1/client.json",            // ipv4 loopback
	} {
		_, err := p.Resolve(ctx, bad)
		require.Error(t, err, "must be refused: %s", bad)
		assert.Contains(t, err.Error(), wantRejection,
			"%s must be refused by the SSRF guard specifically", bad)
	}
}

func TestResolveValidation(t *testing.T) {
	// These run against a document server the provider will refuse to dial
	// (loopback), so they assert the checks that happen BEFORE any network call.
	p := testProvider(t)
	ctx := context.Background()

	t.Run("a non-URL client_id is not a metadata document", func(t *testing.T) {
		_, err := p.Resolve(ctx, "some-registered-client")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a metadata document URL")
	})

	t.Run("a fragment is rejected", func(t *testing.T) {
		// Two spellings of one identifier: the document's own client_id could
		// not match both, so the identity check would be ambiguous.
		_, err := p.Resolve(ctx, "https://app.example.com/client.json#x")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fragment")
	})

	t.Run("userinfo is rejected", func(t *testing.T) {
		_, err := p.Resolve(ctx, "https://evil.com@app.example.com/client.json")
		require.Error(t, err)
	})

	t.Run("a host outside the allow-list is rejected", func(t *testing.T) {
		restricted := testProvider(t, "trusted.example.com")
		_, err := restricted.Resolve(ctx, "https://other.example.com/client.json")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "allowed domain list")
	})
}

func TestCacheTTLIsClamped(t *testing.T) {
	// Cache-Control is written by the party being validated, so it is a hint.
	// Unclamped, max-age=0 makes every authorization request refetch (a DoS
	// amplifier pointed at this server), and max-age=1yr pins a document that
	// may later be corrected or revoked.
	cases := []struct {
		header string
		want   time.Duration
	}{
		{"", minCacheTTL},
		{"max-age=0", minCacheTTL},
		{"max-age=1", minCacheTTL},
		{"no-store", minCacheTTL},
		{"max-age=31536000", maxCacheTTL},
		{"public, max-age=600", 600 * time.Second},
		{"MAX-AGE=600", 600 * time.Second},
		// fmt.Sscanf stopped at the first non-digit and still reported success,
		// so these parsed as 600 / 60 rather than being rejected. The clamp made
		// every outcome safe, which is why it went unnoticed — a header this
		// malformed should still fall back to the floor rather than be half-read.
		{"max-age=600junk", minCacheTTL},
		{"max-age=600 junk", minCacheTTL},
		{"max-age=", minCacheTTL},
		{"max-age=-5", minCacheTTL},
		{"max-age=abc", minCacheTTL},
	}
	for _, tc := range cases {
		t.Run(tc.header, func(t *testing.T) {
			assert.Equal(t, tc.want, cacheTTL(tc.header))
		})
	}
}

func TestCacheIsBounded(t *testing.T) {
	// Distinct client_id URLs are attacker-controlled, so an unbounded cache is
	// a memory-growth primitive.
	p := testProvider(t)
	for i := 0; i < maxCacheEntries+10; i++ {
		p.store(fmt.Sprintf("https://app.example.com/%d.json", i), &Document{}, time.Hour)
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	assert.LessOrEqual(t, len(p.cache), maxCacheEntries)
}

func TestCacheIsUsed(t *testing.T) {
	p := testProvider(t)
	const id = "https://app.example.com/client.json"
	p.store(id, &Document{ClientID: id, ClientName: "cached"}, time.Hour)

	// Resolving must not attempt a fetch — this host does not exist, so a cache
	// miss would surface as a DNS error rather than a document.
	doc, err := p.Resolve(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, "cached", doc.ClientName)

	t.Run("an expired entry is not served", func(t *testing.T) {
		p.store(id, &Document{ClientID: id, ClientName: "stale"}, -time.Second)
		assert.Nil(t, p.cached(id))
	})
}

// TestFetchValidatesDocumentContent covers the checks applied to a document that
// was successfully retrieved. It drives fetch() through a real TLS server via a
// provider whose SSRF guard is bypassed for the test only — the guard itself is
// covered by TestResolveRejectsUnfetchableAndPrivateHosts.
func TestFetchValidatesDocumentContent(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{
			name:    "client_id must match the URL it was fetched from",
			body:    `{"client_id":"https://someone-else.example.com/c.json","client_name":"n","redirect_uris":["https://a/cb"]}`,
			wantErr: "does not match its URL",
		},
		{
			name:    "client_name is required",
			body:    `{"client_id":"%[1]s","redirect_uris":["https://a/cb"]}`,
			wantErr: "missing client_name",
		},
		{
			name:    "redirect_uris is required",
			body:    `{"client_id":"%[1]s","client_name":"n"}`,
			wantErr: "missing redirect_uris",
		},
		{
			name:    "a plaintext non-loopback redirect_uri is rejected",
			body:    `{"client_id":"%[1]s","client_name":"n","redirect_uris":["http://app.example.com/cb"]}`,
			wantErr: "https or loopback",
		},
		{
			name:    "malformed JSON is rejected",
			body:    `not json`,
			wantErr: "not valid JSON",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var selfURL string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				body := tc.body
				if strings.Contains(body, "%[1]s") {
					body = fmt.Sprintf(body, selfURL)
				}
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()
			selfURL = srv.URL + "/client.json"

			p := testProvider(t)
			// fetchViaClient exercises the validation independently of the
			// SSRF-hardened dialer, which refuses httptest's loopback address.
			_, _, err := p.fetchViaClient(context.Background(), selfURL, srv.Client())
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}

	t.Run("a valid document is accepted", func(t *testing.T) {
		var selfURL string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprintf(w, `{"client_id":%q,"client_name":"Example Client","redirect_uris":["http://127.0.0.1/callback"]}`, selfURL)
		}))
		defer srv.Close()
		selfURL = srv.URL + "/client.json"

		doc, ttl, err := testProvider(t).fetchViaClient(context.Background(), selfURL, srv.Client())
		require.NoError(t, err)
		assert.Equal(t, "Example Client", doc.ClientName)
		// The redirect list is what the consent screen's loopback warning is
		// computed from (see TestConsentClientIsLoopbackOnly); assert it survives
		// the fetch intact rather than re-testing the predicate here.
		assert.Equal(t, []string{"http://127.0.0.1/callback"}, doc.RedirectURIs)
		assert.Equal(t, minCacheTTL, ttl)
	})

	t.Run("an oversized document is rejected rather than truncated", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"padding":"` + strings.Repeat("a", maxDocumentBytes+100) + `"}`))
		}))
		defer srv.Close()

		_, _, err := testProvider(t).fetchViaClient(context.Background(), srv.URL+"/c.json", srv.Client())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too large")
	})

	t.Run("a non-200 response is rejected", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		_, _, err := testProvider(t).fetchViaClient(context.Background(), srv.URL+"/c.json", srv.Client())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 404")
	})
}

// TestFetchRefusesRedirects pins the redirect policy.
//
// The document must be AT the client_id URL — that identity binding is the whole
// mechanism CIMD rests on — so a hop away from it serves a document for a
// different identifier. Following one is also useless in production: the
// SSRF-hardened client pins the dial to the IP validated for the ORIGINAL host,
// so a redirect elsewhere re-issues the request against that same address
// carrying a foreign Host header instead of reaching the named host.
//
// The redirect target here serves a perfectly valid document for its own URL, so
// nothing but the redirect policy itself can make this fail.
func TestFetchRefusesRedirects(t *testing.T) {
	var targetURL string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `{"client_id":%q,"client_name":"Moved Client","redirect_uris":["https://app.example.com/cb"]}`, targetURL)
	}))
	defer target.Close()
	targetURL = target.URL + "/moved.json"

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetURL, http.StatusFound)
	}))
	defer redirector.Close()

	_, _, err := testProvider(t).fetchViaClient(
		context.Background(), redirector.URL+"/client.json", redirector.Client())
	require.Error(t, err, "a redirected document MUST NOT resolve")
	// The 302 is surfaced as the final response rather than followed.
	assert.Contains(t, err.Error(), "status 302")
}

// TestFetchViaClientDoesNotMutateTheCallersClient guards the mechanism behind
// TestFetchRefusesRedirects. The policy is applied to a copy, because the
// injected-client seam (SetHTTPClientForTest) hands in a client the caller owns.
// Setting the field in place would work and still be a side effect on someone
// else's value.
func TestFetchViaClientDoesNotMutateTheCallersClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := srv.Client()
	require.Nil(t, client.CheckRedirect, "precondition: the caller's client has no redirect policy")

	_, _, _ = testProvider(t).fetchViaClient(context.Background(), srv.URL+"/c.json", client)

	assert.Nil(t, client.CheckRedirect, "fetchViaClient MUST NOT mutate the client it was handed")
}

// TestIsMetadataClientIDForExcludesTheReservedClient guards against a
// configuration-triggered change of identity source.
//
// --client-id is free-form. If an operator set it to something that parses as a
// document URL, the reserved client would stop being resolved from the registry
// and start being resolved by FETCHING that URL — silently, and with whoever
// controls that URL then describing the deployment's own primary client.
//
// No attacker path exists (the admin API does not accept a client_id; it is
// server-generated), which is exactly why this is worth a cheap guard rather
// than an argument about likelihood.
func TestIsMetadataClientIDForExcludesTheReservedClient(t *testing.T) {
	const url = "https://app.example.com/client.json"

	assert.True(t, IsMetadataClientIDFor(url, "some-normal-client-id"),
		"an ordinary deployment must still resolve URL client_ids as documents")

	assert.False(t, IsMetadataClientIDFor(url, url),
		"the deployment's own reserved client_id must never be resolved by fetching it")
	assert.False(t, IsMetadataClientIDFor(url, "  "+url+"  "),
		"whitespace in --client-id must not defeat the exclusion")

	assert.False(t, IsMetadataClientIDFor("normal-client", "normal-client"),
		"a non-URL reserved client_id was never a document anyway")
}
