package http_handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// pkceVerifierKeyPrefix namespaces the PKCE code_verifier entry in the state
// store so it does not collide with the provider state entry keyed by the raw
// oauth state string. Used only by providers that require PKCE (Twitter/X).
const pkceVerifierKeyPrefix = "pkce_verifier:"

// OAuthLoginHandler set host in the oauth state that is useful for redirecting to oauth_callback
func (h *httpProvider) OAuthLoginHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "OAuthLoginHandler").Logger()
	return func(c *gin.Context) {
		// deprecating redirectURL instead use redirect_uri
		redirectURI := strings.TrimSpace(c.Query("redirectURL"))
		if redirectURI == "" {
			redirectURI = strings.TrimSpace(c.Query("redirect_uri"))
		}
		roles := strings.TrimSpace(c.Query("roles"))
		state := strings.TrimSpace(c.Query("state"))
		scopeString := strings.TrimSpace(c.Query("scope"))

		if redirectURI == "" {
			log.Debug().Msg("redirect uri is missing")
			c.JSON(400, gin.H{
				"error": "invalid redirect uri",
			})
			return
		}

		hostname := parsers.GetHost(c)
		if !validators.IsValidRedirectURI(redirectURI, h.Config.AllowedOrigins, hostname) {
			log.Debug().Msg("Invalid redirect URI")
			c.JSON(400, gin.H{
				"error": "invalid redirect uri",
			})
			return
		}

		if state == "" {
			log.Debug().Msg("state is missing, creating new state")
			state = uuid.New().String()
		}

		var scope []string
		if scopeString == "" {
			scope = []string{"openid", "profile", "email"}
		} else {
			scope = strings.Split(scopeString, " ")
		}

		if roles != "" {
			// validate role
			rolesSplit := strings.Split(roles, ",")

			// use protected roles verification for admin login only.
			// though if not associated with user, it will be rejected from oauth_callback
			allowedRoles := h.Config.Roles
			protectedRoles := h.Config.ProtectedRoles
			if !validators.IsValidRoles(rolesSplit, append([]string{}, append(allowedRoles, protectedRoles...)...)) {
				log.Debug().Msg("invalid role")
				c.JSON(400, gin.H{
					"error": "invalid role",
				})
				return
			}
		} else {
			roles = strings.Join(h.Config.DefaultRoles, ",")
		}

		provider := c.Param("oauth_provider")
		log := log.With().Str("provider", provider).Logger()
		cfg, err := h.OAuthProvider.GetOAuthConfig(c, provider)
		if err != nil {
			log.Debug().Err(err).Msg("Error getting oauth config")
			c.JSON(422, gin.H{
				"error": err.Error(),
			})
			return
		}
		// The value sent to the provider is an opaque handle, not the flow's
		// parameters. Nothing the caller supplied travels off this server, so
		// there is no format for a caller's `state` to collide with on the way
		// back — see internal/http_handlers/oauth_state.go.
		oauthStateString, err := newOAuthStateHandle()
		if err != nil {
			log.Debug().Err(err).Msg("Error generating oauth state handle")
			c.JSON(500, gin.H{
				"error": "internal server error",
			})
			return
		}
		statePayload, err := marshalOAuthState(oauthStatePayload{
			Provider:    provider,
			State:       state,
			RedirectURI: redirectURI,
			Roles:       roles,
			Scope:       strings.Join(scope, " "),
		})
		if err != nil {
			log.Debug().Err(err).Msg("Error encoding oauth state")
			c.JSON(500, gin.H{
				"error": "internal server error",
			})
			return
		}
		// Bind this flow to the browser that started it (RFC 9700 §4.7). Set
		// before the state is stored so a store failure cannot leave a usable
		// cookie behind.
		cookie.SetOAuthState(c, oauthStateString, h.Config.AppCookieSecure)
		if err := h.MemoryStoreProvider.SetState(oauthStateString, statePayload); err != nil {
			log.Debug().Err(err).Msg("Error setting state")
			c.JSON(500, gin.H{
				"error": "internal server error",
			})
			return
		}
		var authCodeOpts []oauth2.AuthCodeOption
		if provider == constants.AuthRecipeMethodTwitter {
			// Twitter/X requires PKCE for the OAuth 2.0 user-context auth code flow.
			// Generate a verifier, store it keyed by state for the callback to replay,
			// and send the derived S256 challenge in the authorization request.
			verifier := oauth2.GenerateVerifier()
			if err := h.MemoryStoreProvider.SetState(pkceVerifierKeyPrefix+oauthStateString, verifier); err != nil {
				log.Debug().Err(err).Msg("Error setting pkce verifier")
				c.JSON(500, gin.H{
					"error": "internal server error",
				})
				return
			}
			authCodeOpts = append(authCodeOpts, oauth2.S256ChallengeOption(verifier))
		}
		url := cfg.AuthCodeURL(oauthStateString, authCodeOpts...)
		log.Debug().Str("url", url).Msg("redirecting to oauth provider")
		metrics.RecordAuthEvent(metrics.EventOAuthLogin, metrics.StatusSuccess)
		h.AuditProvider.LogEvent(audit.Event{
			Action:       constants.AuditOAuthLoginInitiatedEvent,
			ActorType:    constants.AuditActorTypeUser,
			ResourceType: constants.AuditResourceTypeSession,
			Metadata:     provider,
			IPAddress:    utils.GetIP(c.Request),
			UserAgent:    utils.GetUserAgent(c.Request),
		})
		c.Redirect(http.StatusTemporaryRedirect, url)
	}
}
