// Package clientmetadata resolves OAuth Client ID Metadata Documents (CIMD):
// client identifiers that are HTTPS URLs pointing at a JSON document describing
// the client, rather than opaque strings registered ahead of time.
//
// It exists because the MCP authorization spec (2025-11-25) makes CIMD the
// recommended registration mechanism — authorization servers SHOULD support it,
// and MAY support RFC 7591 dynamic client registration, which the spec keeps
// only "for backwards compatibility with earlier versions". Without one of the
// two, a client with no prior relationship to this server cannot authenticate at
// all: Claude Code, for instance, refuses an authorization server that offers
// neither rather than prompting for a client id.
//
// CIMD is the better half of that choice for a self-hosted product. DCR is an
// open, unauthenticated write endpoint that mints a row per registration — and
// per Anthropic's own guidance, clients register afresh on every connection, so
// every operator would accumulate client rows without bound. CIMD adds no
// endpoint, no rows and no schema change: the document lives on the client's own
// server and this package reads it.
//
// Specs:
//   - https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-00
//   - https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization
package clientmetadata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/validators"
)

const (
	// maxDocumentBytes caps how much of a client's document is read. The body is
	// attacker-controlled: without a cap, one authorization request could pull an
	// unbounded stream into memory.
	maxDocumentBytes = 64 << 10

	// fetchTimeout bounds a single document fetch. It sits inside the
	// authorization request, so a slow or hanging client server must not hold a
	// request open — the caller gets a clean rejection instead.
	fetchTimeout = 5 * time.Second

	// minCacheTTL / maxCacheTTL clamp whatever the client's Cache-Control says.
	// The spec says to respect cache headers, but the header is written by the
	// party being validated: a hostile max-age of 0 turns every authorization
	// into a fetch, and a hostile max-age of a year pins a document that may
	// later be corrected or revoked.
	minCacheTTL = 1 * time.Minute
	maxCacheTTL = 1 * time.Hour

	// maxCacheEntries bounds the cache so distinct client_id URLs cannot grow it
	// without limit. On overflow the cache is cleared rather than evicted
	// per-entry: this is a small cache of public documents, and a correctness-
	// neutral flush is cheaper than maintaining LRU bookkeeping.
	maxCacheEntries = 512
)

// Document is the subset of a Client ID Metadata Document this server uses.
type Document struct {
	// ClientID MUST equal the URL the document was fetched from.
	ClientID string `json:"client_id"`
	// ClientName is displayed on the consent screen. Self-asserted, so it is
	// shown as a claim and never as an identity — the redirect URI hostname is
	// what the consent screen relies on.
	ClientName string `json:"client_name"`
	// RedirectURIs is the allow-list the presented redirect_uri is checked
	// against.
	RedirectURIs []string `json:"redirect_uris"`
	ClientURI    string   `json:"client_uri"`
	LogoURI      string   `json:"logo_uri"`
	Scope        string   `json:"scope"`
}

// IsLoopbackOnly reports whether every registered redirect URI is a loopback
// address.
//
// Such a client cannot be distinguished from a local impostor: any process on
// the user's machine can bind a port and claim the same metadata document. The
// spec calls this out and says an authorization server SHOULD warn about it,
// because it is not solvable server-side. The consent screen uses this to say so
// out loud.
func (d *Document) IsLoopbackOnly() bool {
	if len(d.RedirectURIs) == 0 {
		return false
	}
	for _, raw := range d.RedirectURIs {
		u, err := url.Parse(raw)
		if err != nil {
			return false
		}
		switch u.Hostname() {
		case "127.0.0.1", "::1", "localhost":
		default:
			return false
		}
	}
	return true
}

// IsMetadataClientID reports whether a client_id is in URL form and should be
// resolved as a metadata document rather than looked up in the registry.
//
// The two forms cannot collide: a registered client_id is an opaque string
// (a UUID in this codebase), and the spec requires a CIMD client_id to be an
// https URL WITH a path component. Requiring the path is not cosmetic — it is
// what keeps a bare origin from being mistaken for a client identifier.
func IsMetadataClientID(clientID string) bool {
	if !strings.HasPrefix(clientID, "https://") {
		return false
	}
	u, err := url.Parse(clientID)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return false
	}
	return u.Path != "" && u.Path != "/"
}

// Provider resolves and caches client metadata documents.
type Provider struct {
	log *zerolog.Logger
	// allowedDomains, when non-empty, restricts which hosts may serve a metadata
	// document (the spec's optional domain trust policy). Empty accepts any
	// HTTPS host, which is what a public MCP server wants.
	allowedDomains map[string]struct{}

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	doc       *Document
	expiresAt time.Time
}

// New builds a Provider. allowedDomains is an optional host allow-list; an empty
// slice accepts any HTTPS host.
func New(log *zerolog.Logger, allowedDomains []string) *Provider {
	allowed := make(map[string]struct{}, len(allowedDomains))
	for _, d := range allowedDomains {
		if d = strings.ToLower(strings.TrimSpace(d)); d != "" {
			allowed[d] = struct{}{}
		}
	}
	return &Provider{log: log, allowedDomains: allowed, cache: map[string]cacheEntry{}}
}

// Resolve fetches and validates the metadata document named by clientID.
//
// Every failure returns an error rather than a partial document: a client whose
// identity cannot be established must not reach the consent screen, where a
// half-validated `client_name` would be shown to a user as if it meant something.
func (p *Provider) Resolve(ctx context.Context, clientID string) (*Document, error) {
	if !IsMetadataClientID(clientID) {
		return nil, fmt.Errorf("client_id is not a metadata document URL")
	}
	u, err := url.Parse(clientID)
	if err != nil {
		return nil, fmt.Errorf("client_id is not a valid URL")
	}
	// A fragment would make two spellings of one identifier, and the document's
	// own client_id could then never match both.
	if u.Fragment != "" || u.User != nil {
		return nil, fmt.Errorf("client_id must not contain a fragment or userinfo")
	}
	if len(p.allowedDomains) > 0 {
		if _, ok := p.allowedDomains[strings.ToLower(u.Hostname())]; !ok {
			return nil, fmt.Errorf("client_id host is not in the allowed domain list")
		}
	}

	if doc := p.cached(clientID); doc != nil {
		return doc, nil
	}

	doc, ttl, err := p.fetch(ctx, clientID)
	if err != nil {
		return nil, err
	}
	p.store(clientID, doc, ttl)
	return doc, nil
}

func (p *Provider) cached(clientID string) *Document {
	p.mu.RLock()
	defer p.mu.RUnlock()
	e, ok := p.cache[clientID]
	if !ok || time.Now().After(e.expiresAt) {
		return nil
	}
	return e.doc
}

func (p *Provider) store(clientID string, doc *Document, ttl time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.cache) >= maxCacheEntries {
		p.cache = map[string]cacheEntry{}
	}
	p.cache[clientID] = cacheEntry{doc: doc, expiresAt: time.Now().Add(ttl)}
}

// fetch retrieves and validates the document. The URL is attacker-supplied, so
// the request goes through validators.SafeHTTPClient: it resolves the host once
// and pins the dial to the validated IP, so a DNS-rebinding TOCTOU cannot make
// this server reach a private address after the check passed. Without that, CIMD
// would hand any caller an SSRF primitive against everything the server can
// reach — the risk the spec's security considerations lead with.
func (p *Provider) fetch(ctx context.Context, clientID string) (*Document, time.Duration, error) {
	client, err := validators.SafeHTTPClient(ctx, clientID, fetchTimeout)
	if err != nil {
		return nil, 0, fmt.Errorf("client_id URL is not fetchable: %w", err)
	}
	return p.fetchViaClient(ctx, clientID, client)
}

// fetchViaClient performs the request and validates the document.
//
// Split from fetch so the document-validation rules can be tested against a real
// server without the SSRF-hardened dialer, which refuses httptest's loopback
// address by design. Production has exactly one caller — fetch — so the safe
// client is never bypassed outside tests. Do not add another.
func (p *Provider) fetchViaClient(ctx context.Context, clientID string, client *http.Client) (*Document, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("client_id URL is not fetchable")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("could not fetch client metadata document")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("client metadata document returned status %d", resp.StatusCode)
	}

	// LimitReader with one extra byte so an oversized body is detected rather
	// than silently truncated into something that might still parse.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDocumentBytes+1))
	if err != nil {
		return nil, 0, fmt.Errorf("could not read client metadata document")
	}
	if len(body) > maxDocumentBytes {
		return nil, 0, fmt.Errorf("client metadata document is too large")
	}

	var doc Document
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, 0, fmt.Errorf("client metadata document is not valid JSON")
	}

	// The spec's central requirement: the document must claim the identity it
	// was found at. Without this any host could serve a document claiming to be
	// any other client, and the URL would stop meaning anything.
	if doc.ClientID != clientID {
		return nil, 0, fmt.Errorf("client metadata document client_id does not match its URL")
	}
	if strings.TrimSpace(doc.ClientName) == "" {
		return nil, 0, fmt.Errorf("client metadata document is missing client_name")
	}
	if len(doc.RedirectURIs) == 0 {
		return nil, 0, fmt.Errorf("client metadata document is missing redirect_uris")
	}
	for _, r := range doc.RedirectURIs {
		ru, err := url.Parse(r)
		if err != nil || !ru.IsAbs() {
			return nil, 0, fmt.Errorf("client metadata document has an invalid redirect_uri")
		}
		// OAuth 2.1 §1.5: redirect URIs are HTTPS or loopback. An http:// URI on
		// a public host would send an authorization code over cleartext.
		if ru.Scheme != "https" {
			switch ru.Hostname() {
			case "127.0.0.1", "::1", "localhost":
			default:
				return nil, 0, fmt.Errorf("client metadata document redirect_uri must be https or loopback")
			}
		}
	}

	return &doc, cacheTTL(resp.Header.Get("Cache-Control")), nil
}

// cacheTTL derives a cache lifetime from Cache-Control, clamped. The header is
// written by the party being validated, so it is a hint, not an instruction: an
// unclamped max-age of 0 makes every authorization request refetch, and a
// max-age of a year pins a document that may later be corrected or revoked.
func cacheTTL(header string) time.Duration {
	ttl := minCacheTTL
	for _, part := range strings.Split(header, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if v, ok := strings.CutPrefix(part, "max-age="); ok {
			var secs int
			if _, err := fmt.Sscanf(v, "%d", &secs); err == nil && secs > 0 {
				ttl = time.Duration(secs) * time.Second
			}
		}
	}
	if ttl < minCacheTTL {
		ttl = minCacheTTL
	}
	if ttl > maxCacheTTL {
		ttl = maxCacheTTL
	}
	return ttl
}
