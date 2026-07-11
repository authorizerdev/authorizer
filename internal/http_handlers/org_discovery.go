package http_handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/service"
)

// OrgDiscoveryHandler serves the public, unauthenticated home-realm-discovery
// endpoint: GET /api/v1/org-discovery?email=<email>. It answers exactly one
// question — "which enterprise SSO connection, if any, should a login for this
// email route to" — so the login UI can redirect to the right tenant IdP.
//
// It is a ROUTING HINT ONLY. It does NOT authenticate, determine membership, or
// restrict signup: a no-match simply means "no SSO routing hint" (the user may
// still be an invited member and log in through whatever their membership
// permits). Privacy (review M5): the response returns only what the UI needs to
// redirect — never organization_id/name/slug (the slug is unavoidably embedded
// in login_url; that is acceptable, but nothing more is disclosed). No-match and
// match-without-connection are indistinguishable by design, so a caller cannot
// enumerate the tenant list by probing domains.
func (h *httpProvider) OrgDiscoveryHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "OrgDiscoveryHandler").Logger()
	return func(c *gin.Context) {
		// Whole endpoint gated behind the operator flag (default on).
		if !h.Config.EnableOrgDiscovery {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "error_description": "organization discovery is disabled"})
			return
		}

		// Per-IP rate limit to blunt domain enumeration. A dedicated bucket keyed
		// on the client IP, separate from the global middleware limiter.
		// ponytail: fail-open on limiter error — a routing hint leaks nothing
		// sensitive, and the global RateLimitMiddleware already ran ahead of us.
		if h.RateLimitProvider != nil {
			allowed, rlErr := h.RateLimitProvider.Allow(c.Request.Context(), "org_discovery:"+c.ClientIP())
			if rlErr != nil {
				log.Debug().Err(rlErr).Msg("org-discovery rate limiter error; allowing")
			} else if !allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited", "error_description": "too many requests"})
				return
			}
		}

		// Malformed email → 400 with a uniform error (never reveal parsing detail).
		domain, err := service.NormalizeEmailDomain(c.Query("email"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "invalid email"})
			return
		}

		// nullConnection is the privacy-preserving default response: no-match,
		// unverified/pending domain, disabled org, and match-without-connection all
		// return exactly this — indistinguishable by design.
		nullConnection := gin.H{"connection": nil}

		ctx := c.Request.Context()

		// Verified domains only. GetOrgDomainByDomain is a primary-key lookup that
		// returns a row solely for a VERIFIED domain (a pending challenge is inert
		// memory-store state, never a row), so this can never route to a
		// pending/unverified domain.
		orgDomain, err := h.StorageProvider.GetOrgDomainByDomain(ctx, domain)
		if err != nil || orgDomain == nil {
			c.JSON(http.StatusOK, nullConnection)
			return
		}

		org, err := h.StorageProvider.GetOrganizationByID(ctx, orgDomain.OrgID)
		if err != nil || org == nil || !org.Enabled {
			c.JSON(http.StatusOK, nullConnection)
			return
		}

		// SAML takes precedence when an org has both an active SAML and an active
		// OIDC connection (spec-locked). login_url is the existing SP-initiated
		// path; the SPA appends the caller's redirect_uri + state, which those
		// endpoints already thread through the IdP round-trip.
		if conn, cErr := h.StorageProvider.GetTrustedIssuerByOrgIDAndKind(ctx, org.ID, constants.TrustKindSSOSAML); cErr == nil && conn != nil && conn.IsActive && conn.EffectiveKind() == constants.TrustKindSSOSAML {
			c.JSON(http.StatusOK, gin.H{"connection": gin.H{
				"type":      "saml",
				"login_url": "/oauth/saml/" + org.Name + "/login",
			}})
			return
		}
		if conn, cErr := h.StorageProvider.GetTrustedIssuerByOrgIDAndKind(ctx, org.ID, constants.TrustKindSSOOIDC); cErr == nil && conn != nil && conn.IsActive && conn.EffectiveKind() == constants.TrustKindSSOOIDC {
			c.JSON(http.StatusOK, gin.H{"connection": gin.H{
				"type":      "oidc",
				"login_url": "/oauth/sso/" + org.Name + "/login",
			}})
			return
		}

		// Verified domain, but the org has no enterprise connection → password /
		// social / magic-link. Indistinguishable from no-match.
		c.JSON(http.StatusOK, nullConnection)
	}
}
