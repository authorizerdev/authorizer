package http_handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// maxRegisteredClients is a hard ceiling on the number of rows in the client
// registry, checked before a self-registration is accepted.
//
// RFC 7591 §5 allows an unauthenticated registration endpoint but expects it to
// be "rate-limited or otherwise limited to prevent a denial-of-service attack on
// the client registration endpoint". The per-IP limiter bounds the RATE; this
// bounds the STOCK, which is the part that actually persists — Anthropic
// documents DCR clients re-registering on every connection, so a busy
// deployment accumulates rows without one. Keycloak's initial-access-token count
// limit exists for the same reason.
//
// It counts EVERY client, not just self-registered ones: there is no filtered
// count on the storage interface and adding one would mean a method across all
// six backends for a guard rail. A deployment with more clients than this that
// also wants anonymous registration should pre-register instead.
const maxRegisteredClients = 1000

// maxRedirectURIs bounds the list an anonymous caller can store in one row.
const maxRedirectURIs = 10

// maxClientNameLength bounds the self-asserted name that the consent screen
// renders. Templates escape it, so this is about layout and log volume, not
// injection.
const maxClientNameLength = 200

// clientRegistrationRequest is the RFC 7591 §2 client metadata this server
// accepts. Unknown members are ignored, which §3.1 requires ("the authorization
// server MUST ignore any client metadata sent by the client that it does not
// understand").
// Fields the client may send that are deliberately absent here — client_uri,
// logo_uri, scope, contacts — are not merely unstored but unrendered: the
// consent screen shows a name and a redirect host and nothing else, because
// every one of those values is self-asserted and a logo is the most effective
// way to impersonate a known product. Adding a field here means deciding where
// it is displayed and what it would let a rogue client claim.
type clientRegistrationRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// RegisterClientHandler implements OAuth 2.0 Dynamic Client Registration
// (RFC 7591) for MCP clients that cannot use Client ID Metadata Documents.
//
// Deliberately narrow. This endpoint registers PUBLIC, interactive clients and
// nothing else:
//
//   - token_endpoint_auth_method MUST be "none". RFC 7591 §2 defaults the field
//     to client_secret_basic when omitted, but issuing a secret to an anonymous
//     caller would create a confidential client nobody vouched for, and a
//     confidential client is exactly what must not be self-service. Omitted is
//     therefore read as "none" rather than as the RFC default, and any other
//     value is refused.
//   - grant_types are limited to authorization_code (+ refresh_token). In
//     particular client_credentials is refused: it issues a token with no user
//     and no consent, and the Kind assigned here (dynamic, not service_account)
//     means the token endpoint would reject it anyway — this makes the refusal
//     explicit at registration time instead of at first use.
//
// RFC 7592 client management (registration_access_token, registration_client_uri)
// is NOT implemented, which RFC 7591 §3.2.1 permits — neither field is required
// in the response. Nothing here can be read back, modified or deleted through
// this endpoint; a self-registered client is disposable by design.
func (h *httpProvider) RegisterClientHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		log := h.Log.With().Str("func", "RegisterClientHandler").Logger()

		// Defence in depth: the route is only mounted when the flag is on, so
		// this should be unreachable. It is here because an unauthenticated
		// write endpoint is the wrong place to rely on registration order.
		if !h.Config.EnableDynamicClientRegistration {
			gc.JSON(http.StatusNotFound, gin.H{
				"error":             "invalid_request",
				"error_description": "dynamic client registration is not enabled on this server",
			})
			return
		}

		var req clientRegistrationRequest
		if err := gc.ShouldBindJSON(&req); err != nil {
			log.Debug().Err(err).Msg("malformed registration request")
			registrationError(gc, "invalid_client_metadata", "the request body must be a JSON object of client metadata")
			return
		}

		// RFC 7591 §2: "If unspecified or omitted, the default is
		// client_secret_basic". That default is refused rather than honoured —
		// see the doc comment. An explicit "none" is the only accepted value.
		if m := strings.TrimSpace(req.TokenEndpointAuthMethod); m != "" && m != constants.TokenEndpointAuthMethodNone {
			log.Debug().Str("token_endpoint_auth_method", m).Msg("registration refused: confidential client requested")
			registrationError(gc, "invalid_client_metadata",
				"only public clients may register dynamically; token_endpoint_auth_method must be \"none\"")
			return
		}

		// RFC 7591 §2 defaults grant_types to authorization_code when omitted.
		grantTypes := req.GrantTypes
		if len(grantTypes) == 0 {
			grantTypes = []string{constants.GrantTypeAuthorizationCode}
		}
		// Normalised in place: the value that is VALIDATED must be the value that
		// is STORED, or a padded " authorization_code" passes here and is
		// persisted with the space, ready to fail a later exact comparison.
		for i, gt := range grantTypes {
			grantTypes[i] = strings.TrimSpace(gt)
			switch grantTypes[i] {
			case constants.GrantTypeAuthorizationCode, constants.GrantTypeRefreshToken:
			default:
				log.Debug().Str("grant_type", gt).Msg("registration refused: unsupported grant type")
				registrationError(gc, "invalid_client_metadata",
					"only the authorization_code and refresh_token grant types may be registered dynamically")
				return
			}
		}

		// RFC 7591 §2 defaults response_types to ["code"] when omitted, which is
		// the only value supported here. Refused rather than silently narrowed:
		// §3.2.1 does let the server return metadata different from what was
		// requested, but a client that asked for a token in the fragment and is
		// handed back "code" would discover the difference only when its callback
		// receives something it cannot parse. /authorize enforces the same rule.
		for _, rt := range req.ResponseTypes {
			if strings.TrimSpace(rt) != "code" {
				log.Debug().Str("response_type", rt).Msg("registration refused: unsupported response type")
				registrationError(gc, "invalid_client_metadata",
					"only response_type \"code\" may be registered dynamically")
				return
			}
		}

		// redirect_uris is required: every grant accepted here is redirect-based,
		// and RFC 7591 §2 makes the field required for such clients.
		if len(req.RedirectURIs) == 0 {
			registrationError(gc, "invalid_redirect_uri", "redirect_uris is required")
			return
		}
		if len(req.RedirectURIs) > maxRedirectURIs {
			registrationError(gc, "invalid_redirect_uri", "too many redirect_uris")
			return
		}
		normalized := make([]string, 0, len(req.RedirectURIs))
		for _, raw := range req.RedirectURIs {
			u := strings.TrimSpace(raw)
			if err := validateRegistrationRedirectURI(u); err != nil {
				log.Debug().Str("redirect_uri", u).Err(err).Msg("registration refused: invalid redirect_uri")
				registrationError(gc, "invalid_redirect_uri", err.Error())
				return
			}
			normalized = append(normalized, u)
		}

		// Stock ceiling. Checked before the write and deliberately not atomic
		// with it: two racing registrations can both pass at the boundary, which
		// overshoots by the number of concurrent requests and not by more. A
		// transaction here would need a storage-interface change across six
		// backends to prevent an overshoot the per-IP limiter already bounds.
		if _, page, err := h.StorageProvider.ListClients(gc.Request.Context(), &model.Pagination{Limit: 1}); err != nil {
			log.Warn().Err(err).Msg("could not count clients before registration")
			gc.JSON(http.StatusServiceUnavailable, gin.H{
				"error":             "temporarily_unavailable",
				"error_description": "could not process the registration; please retry",
			})
			return
		} else if page != nil && page.Total >= maxRegisteredClients {
			metrics.RecordSecurityEvent("dcr_registration_limit_reached", "register")
			log.Warn().Int64("total", page.Total).Msg("registration refused: client registry is full")
			gc.JSON(http.StatusForbidden, gin.H{
				"error":             "access_denied",
				"error_description": "this server is not accepting new client registrations",
			})
			return
		}

		clientName := strings.TrimSpace(req.ClientName)
		if clientName == "" {
			clientName = "Unnamed client"
		}
		// Truncated on a RUNE boundary, not a byte one. Slicing bytes can split a
		// multi-byte character and leave invalid UTF-8, which SQLite stores
		// happily and Postgres rejects outright ("invalid byte sequence for
		// encoding UTF8") — a registration that works on one backend and 500s on
		// another, from a name an anonymous caller chose.
		if utf8.RuneCountInString(clientName) > maxClientNameLength {
			clientName = string([]rune(clientName)[:maxClientNameLength])
		}

		// The client_id is a server-generated opaque UUID, never anything the
		// caller supplied: RFC 7591 §2 has no client_id request field, and
		// honouring one would let a caller claim an existing client's identifier.
		clientID := uuid.NewString()
		created, err := h.StorageProvider.AddClient(gc.Request.Context(), &schemas.Client{
			ClientID: clientID,
			Kind:     constants.ClientKindDynamic,
			Name:     clientName,
			// ClientSecret stays empty. This is a public client; the token
			// endpoint skips secret verification when none is presented and PKCE
			// is what binds the code (see clientauth.ResolveClient).
			RedirectURIs:            strings.Join(normalized, ","),
			GrantTypes:              strings.Join(grantTypes, ","),
			TokenEndpointAuthMethod: constants.TokenEndpointAuthMethodNone,
			IsActive:                true,
		})
		if err != nil {
			log.Warn().Err(err).Msg("could not persist dynamically registered client")
			gc.JSON(http.StatusServiceUnavailable, gin.H{
				"error":             "temporarily_unavailable",
				"error_description": "could not process the registration; please retry",
			})
			return
		}

		metrics.RecordSecurityEvent("dcr_client_registered", "register")
		log.Info().Str("client_id", created.ClientID).Str("client_name", clientName).
			Msg("dynamically registered a new public client")

		// RFC 7591 §3.2.1: 201 Created, no-store, and the registered metadata
		// echoed back. client_secret and client_secret_expires_at are absent
		// because no secret was issued.
		gc.Header("Cache-Control", "no-store")
		gc.Header("Pragma", "no-cache")
		gc.JSON(http.StatusCreated, gin.H{
			"client_id":                  created.ClientID,
			"client_id_issued_at":        created.CreatedAt,
			"client_name":                clientName,
			"redirect_uris":              normalized,
			"grant_types":                grantTypes,
			"response_types":             []string{"code"},
			"token_endpoint_auth_method": constants.TokenEndpointAuthMethodNone,
		})
	}
}

// registrationError writes an RFC 7591 §3.2.2 registration error response.
func registrationError(gc *gin.Context, code, description string) {
	gc.Header("Cache-Control", "no-store")
	gc.JSON(http.StatusBadRequest, gin.H{
		"error":             code,
		"error_description": description,
	})
}

// validateRegistrationRedirectURI enforces the MCP authorization spec's rule
// that "all redirect URIs MUST be either localhost or use HTTPS", which is also
// what RFC 8252 §7.3 and RFC 9700 require of native and public clients.
//
// The check is on the value being STORED. /authorize separately matches the
// presented redirect_uri against this list with redirectURIMatches, so a URI
// that gets past here still cannot be widened later.
func validateRegistrationRedirectURI(raw string) error {
	if raw == "" {
		return errors.New("redirect_uri must not be empty")
	}
	// Comma is the storage encoding for the list (Client.ParsedRedirectURIs), so
	// a URI containing one would be silently split into two registered redirect
	// targets — a redirect-target injection. Commas are legal in a URI query, so
	// this is a real input rather than a theoretical one. Checked before parsing
	// because it is a property of the stored string, not of the URL.
	if strings.Contains(raw, ",") {
		return errors.New("redirect_uri must not contain a comma")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("redirect_uri is not a valid URI")
	}
	if !u.IsAbs() {
		return errors.New("redirect_uri must be absolute")
	}
	// RFC 6749 §3.1.2: the endpoint URI "MUST NOT include a fragment component".
	if u.Fragment != "" || strings.Contains(raw, "#") {
		return errors.New("redirect_uri must not contain a fragment")
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		if u.Hostname() == "" {
			return errors.New("redirect_uri must have a host")
		}
		return nil
	case "http":
		// Loopback only. http://evil.example.com would otherwise be registrable
		// and every code issued to it would cross the network in the clear.
		if !isLoopbackHost(u.Hostname()) {
			return errors.New("an http redirect_uri is only allowed for loopback addresses; use https")
		}
		return nil
	default:
		// Custom schemes (myapp://) are refused: this server cannot tell which
		// application the OS will hand the code to, and an MCP client has no
		// need for one.
		return errors.New("redirect_uri must use https, or http on a loopback address")
	}
}
