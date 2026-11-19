package gateway

import (
	"context"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// TestIncomingHeaderMatcherRefusesForwardingHeaders is the boundary half of the
// regression for the REST bypass of GHSA-93hc-xq3w-xw87.
//
// runtime.DefaultHeaderMatcher strips a "Grpc-Metadata-" prefix and forwards
// whatever follows, so `Grpc-Metadata-X-Forwarded-For: <attacker>` reached the
// gRPC layer as a genuine `x-forwarded-for` metadata entry ahead of the
// authoritative chain AnnotateContext appends. The admin-secret lockout then
// bucketed on an attacker-chosen value again, unauthenticated, over REST.
//
// The client IP must come from the connection, never from a header a client
// can spell.
func TestIncomingHeaderMatcherRefusesForwardingHeaders(t *testing.T) {
	for _, smuggled := range []string{
		"Grpc-Metadata-X-Forwarded-For",
		"Grpc-Metadata-X-Real-Ip",
		"Grpc-Metadata-X-Forwarded-Host",
		"grpc-metadata-x-forwarded-for",
		"GRPC-METADATA-X-FORWARDED-FOR",
	} {
		t.Run(smuggled, func(t *testing.T) {
			md := annotate(t, map[string]string{smuggled: "203.0.113.9"})
			for _, key := range []string{"x-forwarded-for", "x-real-ip", "x-forwarded-host"} {
				for _, v := range md.Get(key) {
					assert.NotEqual(t, "203.0.113.9", v,
						"%s smuggled an attacker value into %s", smuggled, key)
				}
			}
		})
	}

	// X-Authorizer-Url is the same class, one key over. It decides the host of
	// emailed password-reset / magic-link URLs and the `iss` claim of the tokens
	// minted from them, so a caller who can set it can have a victim emailed a
	// genuine reset link pointing at a host the caller controls. The gin path
	// pins it to --url; the gRPC path did not, so the prefixed spelling
	// reinstated host poisoning over REST.
	t.Run("the authorizer-url header cannot be smuggled", func(t *testing.T) {
		for _, smuggled := range []string{
			"Grpc-Metadata-X-Authorizer-Url",
			"grpc-metadata-x-authorizer-url",
			"GRPC-METADATA-X-AUTHORIZER-URL",
			"X-Authorizer-Url",
		} {
			md := annotate(t, map[string]string{smuggled: "https://evil.example.net"})
			for _, key := range []string{"x-authorizer-url", "grpcgateway-x-authorizer-url"} {
				for _, v := range md.Get(key) {
					assert.NotEqual(t, "https://evil.example.net", v,
						"%s smuggled an attacker-chosen host into %s", smuggled, key)
				}
			}
		}
	})

	t.Run("the admin-secret header is still forwarded", func(t *testing.T) {
		md := annotate(t, map[string]string{"x-authorizer-admin-secret": "s3cret"})
		assert.Equal(t, []string{"s3cret"}, md.Get("x-authorizer-admin-secret"),
			"the filter must not break admin header auth over REST")
	})

	t.Run("the gateway's own chain still reaches the server", func(t *testing.T) {
		md := annotate(t, nil)
		require.Len(t, md.Get("x-forwarded-for"), 1)
		assert.Equal(t, "198.51.100.10", md.Get("x-forwarded-for")[0],
			"the real HTTP peer must still be forwarded")
	})
}

// TestIncomingHeaderMatcherRefusesEveryListedHeader drives the whole
// refusedForwardingHeaders set rather than a hand-written copy of it, in both
// the bare and the Grpc-Metadata- prefixed spelling.
//
// The point is that adding a key to the map is enough: a new entry is covered
// the moment it lands, so the list cannot drift away from the policy. The
// earlier test enumerated four keys by hand, which is exactly how a fifth gets
// forgotten.
func TestIncomingHeaderMatcherRefusesEveryListedHeader(t *testing.T) {
	require.NotEmpty(t, refusedForwardingHeaders, "the policy cannot be empty")

	const spoof = "attacker-controlled-value"

	// Both fallbacks. "default" is what ships; "legacy" is the one that
	// forwards Grpc-Metadata-, so it is the case that actually exercises the
	// refusal list. Dropping "legacy" makes this whole test vacuous.
	for _, fb := range []struct {
		name     string
		annotate func(*testing.T, map[string]string) metadata.MD
	}{
		{"default", annotate},
		{"legacy", annotateLegacy},
	} {
		for _, header := range mustRefuse {
			spellings := []string{
				"Grpc-Metadata-" + header,
				strings.ToLower("grpc-metadata-" + header),
				strings.ToUpper("GRPC-METADATA-" + header),
			}
			// The bare spelling of the two keys runtime.AnnotateContext
			// populates itself is grpc-gateway's own output, not something this
			// matcher can refuse — see headersGrpcGatewayInjects.
			if _, injected := headersGrpcGatewayInjects[header]; !injected {
				spellings = append(spellings, header)
			}
			for _, spelling := range spellings {
				t.Run(fb.name+"/"+spelling, func(t *testing.T) {
					md := fb.annotate(t, map[string]string{spelling: spoof})
					for key, values := range md {
						for _, v := range values {
							assert.NotContains(t, v, spoof,
								"%s reached metadata as %s=%q; a client must not be able to set it",
								spelling, key, v)
						}
					}
				})
			}
		}
	}
}

// TestLegacyFallbackWouldSmuggleWithoutTheList is the falsification half: it
// pins that runtime.LegacyHeaderMatcher really does forward an unlisted
// Grpc-Metadata- header, so the "legacy" cases above are meaningful rather than
// another matcher that refuses everything anyway.
//
// If this starts failing, upstream has removed LegacyHeaderMatcher's forwarding
// too, and the test above has quietly become vacuous again — find another
// fallback that forwards, do not delete the case.
func TestLegacyFallbackWouldSmuggleWithoutTheList(t *testing.T) {
	key := "X-Not-In-The-Refusal-List"
	require.NotContains(t, refusedForwardingHeaders, key,
		"precondition: this key must be absent from the policy")

	md := annotateLegacy(t, map[string]string{"Grpc-Metadata-" + key: "smuggled"})
	assert.Equal(t, []string{"smuggled"}, md.Get("x-not-in-the-refusal-list"),
		"LegacyHeaderMatcher no longer forwards Grpc-Metadata-; the legacy cases above prove nothing now")
}

// mustRefuse is the test's OWN copy of the header list, and the duplication is
// deliberate.
//
// Driving the loop from refusedForwardingHeaders made this test vacuous:
// emptying the policy emptied the test along with it, so the suite stayed green
// with the whole list deleted. A policy cannot be its own oracle.
//
// The drift a second copy risks is covered from the other direction, by
// TestRefusalPolicyMatchesTheTestsList: the policy may hold no key this list
// omits, and this list may name no key the policy omits. Adding a header means
// editing both, and forgetting either one fails.
var mustRefuse = []string{
	"X-Forwarded-For",
	"X-Real-Ip",
	"X-Client-Ip",
	"X-Cluster-Client-Ip",
	"X-Originating-Ip",
	"X-Original-Forwarded-For",
	"X-Remote-Ip",
	"X-Remote-Addr",
	"True-Client-Ip",
	"Cf-Connecting-Ip",
	"Cf-Connecting-Ipv6",
	"Fastly-Client-Ip",
	"Fly-Client-Ip",
	"X-Azure-Clientip",
	"X-Azure-Socketip",
	"X-Appengine-User-Ip",
	"X-Envoy-External-Address",
	"X-Forwarded-Host",
	"X-Authorizer-Url",
	"X-Forwarded-Proto",
	"X-Forwarded-Scheme",
	"X-Forwarded-Port",
	"X-Forwarded-Server",
	"X-Original-Host",
	"X-Original-Url",
	"X-Rewrite-Url",
	"X-Host",
	"Forwarded",
	"X-Forwarded",
}

// TestRefusalPolicyMatchesTheTestsList keeps mustRefuse and
// refusedForwardingHeaders in step, in both directions.
func TestRefusalPolicyMatchesTheTestsList(t *testing.T) {
	expected := make(map[string]struct{}, len(mustRefuse))
	for _, h := range mustRefuse {
		expected[h] = struct{}{}
	}
	assert.Len(t, mustRefuse, len(expected), "mustRefuse has a duplicate entry")

	for h := range refusedForwardingHeaders {
		assert.Contains(t, expected, h,
			"the policy refuses %s but no test exercises it; add it to mustRefuse", h)
	}
	for _, h := range mustRefuse {
		assert.Contains(t, refusedForwardingHeaders, h,
			"mustRefuse expects %s to be refused but the policy does not list it", h)
	}
}

// headersGrpcGatewayInjects are the two keys runtime.AnnotateContext fills in
// by itself, from r.RemoteAddr and from r.Host or the incoming X-Forwarded-Host
// header. Their bare spelling therefore DOES reach metadata, by grpc-gateway's
// design and without passing through this matcher at all, so no header policy
// here can refuse it. What this matcher governs is the Grpc-Metadata- spelling,
// which a client can otherwise use to place a value of its own AHEAD of
// grpc-gateway's authoritative one.
//
// Their consumers carry the second half of the defence:
// transport.clientIPFromGRPC takes the RIGHTMOST x-forwarded-for value, which
// is the real peer, and transport.hostURLFromGRPC prefers --url and reads
// x-authorizer-url — a value the gateway injects from the raw request via
// runtime.WithMetadata — never x-forwarded-host.
var headersGrpcGatewayInjects = map[string]struct{}{
	"X-Forwarded-For":  {},
	"X-Forwarded-Host": {},
}

// TestRefusedForwardingHeadersAreCanonical guards the lookup's precondition.
//
// incomingHeaderMatcherOption looks the map up by
// textproto.CanonicalMIMEHeaderKey, so a key written in any other casing is
// dead weight that silently refuses nothing. gofmt will not catch it and the
// test above would still pass, because a key that matches nothing also smuggles
// nothing.
func TestRefusedForwardingHeadersAreCanonical(t *testing.T) {
	for header := range refusedForwardingHeaders {
		assert.Equal(t, textproto.CanonicalMIMEHeaderKey(header), header,
			"key must be written in canonical form or the lookup never matches it")
	}
}

// TestIncomingHeaderMatcherStillForwardsOrdinaryHeaders keeps the refusal from
// turning into a blanket deny: the matcher must still pass through the headers
// the REST surface depends on.
func TestIncomingHeaderMatcherStillForwardsOrdinaryHeaders(t *testing.T) {
	md := annotate(t, map[string]string{
		"Authorization": "Bearer token-value",
		"Accept":        "application/json",
	})
	assert.Equal(t, []string{"Bearer token-value"}, md.Get("authorization"),
		"bearer auth must survive the filter")
	// DefaultHeaderMatcher passes a permanent header through under the
	// "grpcgateway-" prefix; only Authorization keeps its own name.
	assert.Equal(t, []string{"application/json"}, md.Get("grpcgateway-accept"),
		"an ordinary permanent header must still be forwarded")
}

// TestGrpcMetadataPrefixStaysClosedUpstream records WHY the refusal list is
// currently a regression guard rather than the only lock.
//
// grpc-gateway v2.33.0's DefaultHeaderMatcher no longer forwards an arbitrary
// Grpc-Metadata-<key> header, so the smuggling vector behind
// GHSA-93hc-xq3w-xw87's REST bypass is closed upstream as well. If this test
// starts failing, the upstream behaviour has come back and
// refusedForwardingHeaders is load-bearing again — do not relax it.
func TestGrpcMetadataPrefixStaysClosedUpstream(t *testing.T) {
	md := annotate(t, map[string]string{"Grpc-Metadata-X-Not-Refused": "value"})
	assert.Empty(t, md.Get("x-not-refused"),
		"grpc-gateway forwarded a Grpc-Metadata- header again; the refusal list is now the only lock")
}

// annotate drives the REAL mux built by Handler's options.
func annotate(t *testing.T, headers map[string]string) metadata.MD {
	t.Helper()
	return annotateWith(t, runtime.WithIncomingHeaderMatcher(
		incomingHeaderMatcher(runtime.DefaultHeaderMatcher)), headers)
}

// annotateLegacy drives the same policy over runtime.LegacyHeaderMatcher, the
// fallback that DOES forward a Grpc-Metadata- header. That is the behaviour
// refusedForwardingHeaders exists to survive, and the only configuration in
// which emptying the list makes these tests fail.
func annotateLegacy(t *testing.T, headers map[string]string) metadata.MD {
	t.Helper()
	return annotateWith(t, runtime.WithIncomingHeaderMatcher(
		incomingHeaderMatcher(runtime.LegacyHeaderMatcher)), headers)
}

func annotateWith(t *testing.T, opt runtime.ServeMuxOption, headers map[string]string) metadata.MD {
	t.Helper()
	mux := runtime.NewServeMux(opt)
	r, err := http.NewRequest(http.MethodPost, "http://localhost:8080/v1/admin/login", nil)
	require.NoError(t, err)
	r.RemoteAddr = "198.51.100.10:40000"
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	ctx, err := runtime.AnnotateContext(context.Background(), mux, r,
		"/authorizer.v1.AuthorizerAdminService/AdminLogin")
	require.NoError(t, err)
	out, _ := metadata.FromOutgoingContext(ctx)
	return out
}
