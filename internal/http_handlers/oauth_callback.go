package http_handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/authorizerdev/authorizer/internal/asyncutil"
	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/codestate"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// AppleUserInfo is the struct for apple user info
type AppleUserInfo struct {
	Email string `json:"email"`
	Name  struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	} `json:"name"`
}

// parseAppleUserField parses the optional Apple `user` callback form field.
// Apple sends this field only on the very first authorization for a given
// app; every subsequent login omits it entirely (documented Apple behavior —
// a one-time grant, not re-sent). An absent/empty field is therefore expected
// steady-state behavior, not an error, and yields a zero-value AppleUserInfo.
// A non-empty but malformed value is a real error (buggy provider or
// tampered request) and is still rejected.
func parseAppleUserField(userRaw string) (*AppleUserInfo, error) {
	appleUser := &AppleUserInfo{}
	if userRaw == "" {
		return appleUser, nil
	}
	if err := json.Unmarshal([]byte(userRaw), appleUser); err != nil {
		return nil, err
	}
	return appleUser, nil
}

// OAuthCallbackHandler handles the OAuth callback for various oauth providers
func (h *httpProvider) OAuthCallbackHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "OAuthCallbackHandler").Logger()
	return func(ctx *gin.Context) {
		provider := ctx.Param("oauth_provider")
		state := ctx.Request.FormValue("state")
		sessionState, err := h.MemoryStoreProvider.GetState(state)
		if sessionState == "" || err != nil {
			log.Debug().Err(err).Msg("Failed to get state from store")
			ctx.JSON(400, gin.H{"error": "invalid oauth state"})
			return
		}
		// The flow's parameters are read from the store, NOT parsed out of the
		// value the provider echoed back. They were validated by /oauth_login
		// and never left this server, so nothing in transit can alter them.
		// Fails closed on anything this server did not write — including an
		// entry from a previous release, whose value was the bare provider name.
		statePayload, err := unmarshalOAuthState(sessionState)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to decode oauth state payload")
			ctx.JSON(400, gin.H{"error": "invalid oauth state"})
			return
		}
		// Ensure the callback route's provider matches what was originally
		// requested, so a code obtained at one provider cannot be redeemed at
		// another.
		if statePayload.Provider != provider {
			log.Debug().
				Str("expected_provider", statePayload.Provider).
				Str("callback_provider", provider).
				Msg("OAuth provider mismatch for state")
			ctx.JSON(400, gin.H{"error": "invalid oauth state"})
			return
		}
		// Prove the browser finishing this flow is the one that started it.
		// The state alone cannot: it is server-generated and stored globally, so
		// its presence only shows SOME flow issued it. Without this an attacker
		// harvests their own valid code+state and delivers it to a victim's
		// browser, logging the victim into the ATTACKER's account (login CSRF,
		// RFC 9700 §4.7). Checked before the state is consumed so a failed
		// attempt cannot burn a legitimate one.
		boundState := cookie.GetOAuthState(ctx)
		if subtle.ConstantTimeCompare([]byte(boundState), []byte(state)) != 1 {
			log.Debug().Bool("cookie_present", boundState != "").Msg("OAuth state is not bound to this browser")
			metrics.RecordSecurityEvent("oauth_state_not_bound", provider)
			cookie.DeleteOAuthState(ctx, h.Config.AppCookieSecure)
			ctx.JSON(400, gin.H{"error": "invalid oauth state"})
			return
		}
		cookie.DeleteOAuthState(ctx, h.Config.AppCookieSecure)

		// remove state from store
		_ = h.MemoryStoreProvider.RemoveState(state)
		stateValue := statePayload.State
		redirectURL := statePayload.RedirectURI
		hostname := parsers.GetHost(ctx)
		if !validators.IsValidRedirectURI(redirectURL, h.Config.AllowedOrigins, hostname) {
			log.Debug().Msg("Invalid redirect URI in OAuth state")
			ctx.JSON(400, gin.H{"error": "invalid redirect uri"})
			return
		}
		inputRoles := strings.Split(statePayload.Roles, ",")
		scopeString := statePayload.Scope
		scopes := parseScopes(scopeString)
		var user *schemas.User
		// providerEmailVerified is the provider's own assertion that the
		// principal controls the email it returned. It gates every path that
		// could attach this login to a pre-existing local account — see the
		// linking branch below.
		var providerEmailVerified bool
		oauthCode := ctx.Request.FormValue("code")
		if oauthCode == "" {
			log.Debug().Err(err).Msg("Invalid oauth code")
			ctx.JSON(400, gin.H{"error": "invalid oauth code"})
			return
		}
		switch provider {
		case constants.AuthRecipeMethodGoogle:
			user, providerEmailVerified, err = h.processGoogleUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodGithub:
			user, providerEmailVerified, err = h.processGithubUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodFacebook:
			user, providerEmailVerified, err = h.processFacebookUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodLinkedIn:
			user, providerEmailVerified, err = h.processLinkedInUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodApple:
			var appleUser *AppleUserInfo
			appleUser, err = parseAppleUserField(ctx.Request.FormValue("user"))
			if err != nil {
				log.Debug().Err(err).Msg("Failed to unmarshal apple user info")
				ctx.JSON(400, gin.H{"error": "invalid apple user info"})
				return
			}
			user, providerEmailVerified, err = h.processAppleUserInfo(ctx, oauthCode, appleUser)
		case constants.AuthRecipeMethodDiscord:
			user, providerEmailVerified, err = h.processDiscordUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodTwitter:
			// Twitter/X uses PKCE: retrieve the verifier stored at login keyed by state.
			verifier, verr := h.MemoryStoreProvider.GetAndRemoveState(pkceVerifierKeyPrefix + state)
			if verr != nil || verifier == "" {
				log.Debug().Err(verr).Msg("Missing PKCE verifier for Twitter callback")
				ctx.JSON(400, gin.H{"error": "invalid oauth state"})
				return
			}
			user, providerEmailVerified, err = h.processTwitterUserInfo(ctx, oauthCode, verifier)
		case constants.AuthRecipeMethodMicrosoft:
			user, providerEmailVerified, err = h.processMicrosoftUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodTwitch:
			user, providerEmailVerified, err = h.processTwitchUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodRoblox:
			user, providerEmailVerified, err = h.processRobloxUserInfo(ctx, oauthCode)
		default:
			log.Debug().Err(err).Msg("Invalid oauth provider")
			err = fmt.Errorf(`invalid oauth provider`)
		}

		if err != nil {
			log.Debug().Err(err).Msg("Failed to process user info")
			metrics.RecordAuthEvent(metrics.EventOAuthCallback, metrics.StatusFailure)
			metrics.RecordSecurityEvent("oauth_callback_failed", provider)
			h.AuditProvider.LogEvent(audit.Event{
				Action:       constants.AuditOAuthCallbackFailedEvent,
				ActorType:    constants.AuditActorTypeUser,
				ResourceType: constants.AuditResourceTypeSession,
				Metadata:     provider,
				IPAddress:    utils.GetIP(ctx.Request),
				UserAgent:    utils.GetUserAgent(ctx.Request),
			})
			ctx.JSON(400, gin.H{
				"error":             "oauth_callback_failed",
				"error_description": "OAuth callback could not be completed. Please try again.",
			})
			return
		}
		if user == nil {
			log.Debug().Err(err).Msg("Failed to get user")
			ctx.JSON(
				500,
				gin.H{"error": "Something Went Wrong. Please Try Again."},
			)
			return
		}
		existingUser, err := h.StorageProvider.GetUserByEmail(ctx, refs.StringValue(user.Email))
		log := log.With().Str("email", refs.StringValue(user.Email)).Logger()
		isSignUp := false

		// An email the identity provider has not attested is attacker-controlled
		// input, and every branch below keys the local account off it — the
		// lookup above decides signup vs. login, and the login branch merges
		// this federated identity into whatever account already holds the
		// address. That is the nOAuth account-takeover class: register a free
		// Entra tenant, set a user's mutable `email` attribute to the victim's
		// address, sign in, and land in the victim's session. The pre-hijack
		// guard further down does not help — it only removes *unverified* local
		// accounts, and verified accounts are exactly what gets stolen.
		//
		// OAuth itself proves nothing about email; that is precisely why OIDC
		// carries a separate `email_verified` claim (Core §5.1), and why Auth0
		// documents checking it before linking accounts.
		if !providerEmailVerified && !h.allowUnverifiedProviderEmail(provider, existingUser, err == nil) {
			log.Debug().Str("provider", provider).Msg("Provider did not attest the email address; refusing to resolve a local account")
			metrics.RecordAuthEvent(metrics.EventOAuthCallback, metrics.StatusFailure)
			metrics.RecordSecurityEvent("oauth_email_unverified", provider)
			h.AuditProvider.LogEvent(audit.Event{
				Action:       constants.AuditOAuthCallbackFailedEvent,
				ActorType:    constants.AuditActorTypeUser,
				ResourceType: constants.AuditResourceTypeSession,
				Metadata:     provider,
				IPAddress:    utils.GetIP(ctx.Request),
				UserAgent:    utils.GetUserAgent(ctx.Request),
			})
			ctx.JSON(400, gin.H{
				"error":             "email_not_verified",
				"error_description": "The identity provider did not confirm that you own this email address.",
			})
			return
		}

		if err != nil {
			isSignupEnabled := h.Config.EnableSignup
			if !isSignupEnabled {
				log.Debug().Err(err).Msg("Signup is disabled")
				ctx.JSON(400, gin.H{"error": "signup is disabled for this instance"})
				return
			}
			// user not registered, register user and generate session token
			user.SignupMethods = provider
			// make sure inputRoles don't include protected roles
			hasProtectedRole := false
			for _, ir := range inputRoles {
				protectedRoles := h.Config.ProtectedRoles
				if utils.StringSliceContains(protectedRoles, ir) {
					hasProtectedRole = true
				}
			}

			if hasProtectedRole {
				log.Debug().Err(err).Msg("Invalid role. User is using protected role")
				ctx.JSON(400, gin.H{"error": "invalid role"})
				return
			}

			user.Roles = strings.Join(inputRoles, ",")
			// Only record the address as verified when the provider actually
			// attested it. This used to be unconditional, which meant a
			// compatibility-mode signup from an unattested address wrote
			// email_verified=true into our own database — a claim we cannot
			// back, and one that downstream consumers trust (SAML IdP issuance
			// refuses to assert an unverified email as the Subject NameID).
			if providerEmailVerified {
				now := time.Now().Unix()
				user.EmailVerifiedAt = &now
			}
			user, err = h.StorageProvider.AddUser(ctx, user)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to add user")
				ctx.JSON(500, gin.H{"error": "failed to process OAuth login"})
				return
			}
			isSignUp = true
		} else {
			if existingUser.RevokedTimestamp != nil {
				log.Debug().Msg("User access has been revoked")
				metrics.RecordAuthEvent(metrics.EventOAuthCallback, metrics.StatusFailure)
				metrics.RecordSecurityEvent("account_revoked", "oauth_callback")
				h.AuditProvider.LogEvent(audit.Event{
					Action:       constants.AuditOAuthCallbackFailedEvent,
					ActorID:      existingUser.ID,
					ActorType:    constants.AuditActorTypeUser,
					ActorEmail:   refs.StringValue(existingUser.Email),
					ResourceType: constants.AuditResourceTypeSession,
					Metadata:     provider,
					IPAddress:    utils.GetIP(ctx.Request),
					UserAgent:    utils.GetUserAgent(ctx.Request),
				})
				ctx.JSON(400, gin.H{"error": "user access has been revoked"})
				return
			}

			// Prevent account pre-hijacking: if the existing account's email
			// was never verified, do not link the OAuth identity to it.
			// Instead, delete the unverified account and treat as a new signup
			// for the OAuth user who actually controls the email address.
			//
			// Scoped to accounts some OTHER credential created. An unverified
			// account this same provider already owns is not a squatter — it is
			// this same principal's own account, created on a previous pass
			// through the signup branch above (which, correctly, no longer marks
			// an unattested address verified). Deleting it would recreate the
			// account on every single login, silently dropping its id, roles and
			// org memberships each time.
			if existingUser.EmailVerifiedAt == nil && !signupMethodsContain(existingUser.SignupMethods, provider) {
				// Deleting is only safe for an account that is actually a
				// squatter — created to intercept this address and never used.
				// The cascade is clean (#749) but total: an account carrying
				// real state would lose its org memberships, enrolled
				// authenticators and federated identities outright, and its FGA
				// grants would be orphaned, since the tuple purge lives in the
				// service layer and this is a direct StorageProvider call. All
				// of that on the say-so of an unauthenticated callback.
				// Refusing is recoverable; deleting is not.
				if hasState, what := h.accountHasState(ctx, existingUser); hasState {
					log.Warn().
						Str("reason", what).
						Str("existing_user_id", existingUser.ID).
						Msg("Refusing OAuth login: an unverified account with this email holds state and must not be replaced")
					metrics.RecordAuthEvent(metrics.EventOAuthCallback, metrics.StatusFailure)
					metrics.RecordSecurityEvent("oauth_email_collision_stateful_account", provider)
					h.AuditProvider.LogEvent(audit.Event{
						Action:       constants.AuditOAuthCallbackFailedEvent,
						ActorID:      existingUser.ID,
						ActorType:    constants.AuditActorTypeUser,
						ActorEmail:   refs.StringValue(existingUser.Email),
						ResourceType: constants.AuditResourceTypeSession,
						Metadata:     provider,
						IPAddress:    utils.GetIP(ctx.Request),
						UserAgent:    utils.GetUserAgent(ctx.Request),
					})
					ctx.JSON(400, gin.H{
						"error":             "email_already_registered",
						"error_description": "An unverified account already exists for this email address. Verify it first — request a new verification email for this address, or sign in with the method that created the account.",
					})
					return
				}
				log.Info().Str("existing_user_id", existingUser.ID).Msg("Removing unverified pre-existing account before OAuth signup")
				// Audited: this destroys an account row, which is
				// security-material even when the account was empty.
				h.AuditProvider.LogEvent(audit.Event{
					Action:       constants.AuditOAuthUnverifiedAccountReplacedEvent,
					ActorID:      existingUser.ID,
					ActorType:    constants.AuditActorTypeUser,
					ActorEmail:   refs.StringValue(existingUser.Email),
					ResourceType: constants.AuditResourceTypeUser,
					ResourceID:   existingUser.ID,
					Metadata:     provider,
					IPAddress:    utils.GetIP(ctx.Request),
					UserAgent:    utils.GetUserAgent(ctx.Request),
				})
				if err := h.StorageProvider.DeleteUser(ctx, existingUser); err != nil {
					log.Debug().Err(err).Msg("Failed to delete unverified user")
					ctx.JSON(500, gin.H{"error": "failed to process OAuth login"})
					return
				}
				// make sure inputRoles don't include protected roles
				hasProtectedRole := false
				for _, ir := range inputRoles {
					if utils.StringSliceContains(h.Config.ProtectedRoles, ir) {
						hasProtectedRole = true
					}
				}
				if hasProtectedRole {
					log.Debug().Msg("Invalid role. User is using protected role")
					ctx.JSON(400, gin.H{"error": "invalid role"})
					return
				}
				user.SignupMethods = provider
				user.Roles = strings.Join(inputRoles, ",")
				now := time.Now().Unix()
				user.EmailVerifiedAt = &now
				user, err = h.StorageProvider.AddUser(ctx, user)
				if err != nil {
					log.Debug().Err(err).Msg("Failed to add user after removing unverified account")
					ctx.JSON(500, gin.H{"error": "failed to process OAuth login"})
					return
				}
				isSignUp = true
			} else {
				user = existingUser

				// user exists in db, check if method was google
				// if not append google to existing signup method and save it
				signupMethod := existingUser.SignupMethods
				if !strings.Contains(signupMethod, provider) {
					signupMethod = signupMethod + "," + provider
				}
				user.SignupMethods = signupMethod

				// There multiple scenarios with roles here in social login
				// 1. user has access to protected roles + roles and trying to login
				// 2. user has not signed up for one of the available role but trying to signup.
				// 		Need to modify roles in this case

				// find the unassigned roles
				existingRoles := strings.Split(existingUser.Roles, ",")
				unasignedRoles := []string{}
				for _, ir := range inputRoles {
					if !utils.StringSliceContains(existingRoles, ir) {
						unasignedRoles = append(unasignedRoles, ir)
					}
				}

				if len(unasignedRoles) > 0 {
					// check if it contains protected unassigned role
					hasProtectedRole := false
					for _, ur := range unasignedRoles {
						protectedRoles := h.Config.ProtectedRoles
						if utils.StringSliceContains(protectedRoles, ur) {
							hasProtectedRole = true
						}
					}

					if hasProtectedRole {
						log.Debug().Err(err).Msg("Invalid role. User is using protected role")
						ctx.JSON(400, gin.H{"error": "invalid role"})
						return
					} else {
						user.Roles = existingUser.Roles + "," + strings.Join(unasignedRoles, ",")
					}
				} else {
					user.Roles = existingUser.Roles
				}

				user, err = h.StorageProvider.UpdateUser(ctx, user)
				if err != nil {
					log.Debug().Err(err).Msg("Failed to update user")
					ctx.JSON(500, gin.H{"error": "failed to process OAuth login"})
					return
				}
			}
		}

		// OIDC `/authorize` bridge:
		// If this social-login callback was initiated from the OpenID Connect authorize flow
		// (`/authorize?...&state=<stateValue>...`), `authorize.go` stores a temporary entry keyed by `stateValue`
		// containing either:
		// - `nonce` (implicit/hybrid-style response), OR
		// - `code@@codeChallenge` (authorization code + PKCE).
		//
		// In the standalone social login flow (`/oauth_login/:provider`), this entry will not exist and we
		// simply generate a nonce and continue.
		code, codeChallenge, nonce, authorizeRedirectURI, authorizeClientID, err := h.consumeAuthorizeState(stateValue)
		if err != nil && !errors.Is(err, goredis.Nil) {
			log.Debug().Err(err).Str("state", stateValue).Msg("Failed to get authorize state from store")
		}
		if nonce == "" {
			nonce = uuid.New().String()
		}
		//  user, inputRoles, scopes, provider, nonce, code
		authToken, err := h.TokenProvider.CreateAuthToken(ctx, &token.AuthTokenConfig{
			User:        user,
			Roles:       inputRoles,
			Scope:       scopes,
			LoginMethod: provider,
			Nonce:       nonce,
			HostName:    hostname,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Failed to create auth token")
			ctx.JSON(500, gin.H{"error": "failed to process OAuth login"})
			return
		}

		// expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
		// if expiresIn <= 0 { expiresIn = 1 }
		// params := "access_token=" + authToken.AccessToken.Token + "&token_type=bearer&expires_in=" + strconv.FormatInt(expiresIn, 10) + "&state=" + stateValue + "&id_token=" + authToken.IDToken.Token + "&nonce=" + nonce
		// Note: If OIDC breaks in the future, use the above params
		params := "state=" + stateValue + "&nonce=" + nonce
		if code != "" {
			params += "&code=" + code
		}

		// MFA gate: matches password/passkey login (resolveMFAGate) before the
		// browser session cookie is established. A withheld-group outcome sets
		// the MFA session cookie (via `side`) instead and redirects with
		// mfa_required=1 rather than the normal state/code params.
		side := &service.ResponseSideEffects{}
		meta := service.RequestMetadata{
			HostURL:   hostname,
			IPAddress: utils.GetIP(ctx.Request),
			UserAgent: utils.GetUserAgent(ctx.Request),
			Request:   ctx.Request,
		}
		withheld, redirectSuffix, gateErr := h.ServiceProvider.EvaluateMFAGateForOAuth(ctx, meta, side, user)
		if gateErr != nil {
			log.Debug().Err(gateErr).Msg("MFA gate rejected OAuth callback")
			ctx.JSON(400, gin.H{"error": gateErr.Error()})
			return
		}
		if withheld {
			service.ApplyToGin(ctx, side)
			if strings.Contains(redirectURL, "?") {
				redirectURL = redirectURL + "&" + redirectSuffix
			} else {
				redirectURL = redirectURL + "?" + strings.TrimPrefix(redirectSuffix, "&")
			}
			ctx.Redirect(http.StatusFound, redirectURL)
			return
		}

		// Code challenge could be optional if PKCE flow is not used. Set only on
		// the normal (non-withheld) path: the `code` is never disclosed to the
		// browser on a withheld redirect (it carries mfa_required=1, not
		// state/code), so setting this state entry before the gate check would
		// leave an orphaned, unreachable entry that just self-expires — not
		// exploitable, but there's no reason to write it before we know the
		// login actually proceeds.
		if code != "" {
			if err := h.MemoryStoreProvider.SetState(code, codestate.EncodeCode(codestate.Code{
				Challenge:   codeChallenge,
				Session:     authToken.FingerPrintHash,
				Nonce:       nonce,
				RedirectURI: authorizeRedirectURI,
				ClientID:    authorizeClientID,
			})); err != nil {
				log.Debug().Err(err).Msg("Failed to set state")
				ctx.JSON(500, gin.H{"error": "failed to process OAuth login"})
				return
			}
		}

		sessionKey := provider + ":" + user.ID
		cookie.SetSession(ctx, authToken.FingerPrintHash, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))
		_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.FingerPrintHash), authToken.SessionTokenExpiresAt)
		_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.AccessToken.Token), authToken.AccessToken.ExpiresAt)

		if authToken.RefreshToken != nil {
			_ = h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, crypto.HashSessionValue(authToken.RefreshToken.Token), authToken.RefreshToken.ExpiresAt)
		}

		bgCtx := context.WithoutCancel(ctx)
		userAgent := utils.GetUserAgent(ctx.Request)
		ip := utils.GetIP(ctx.Request)
		asyncutil.Go(h.Log, func() {
			if isSignUp {
				_ = h.EventsProvider.RegisterEvent(bgCtx, constants.UserSignUpWebhookEvent, provider, user)
				// User is also logged in with signup
				_ = h.EventsProvider.RegisterEvent(bgCtx, constants.UserLoginWebhookEvent, provider, user)
			} else {
				_ = h.EventsProvider.RegisterEvent(bgCtx, constants.UserLoginWebhookEvent, provider, user)
			}
			if err := h.StorageProvider.AddSession(bgCtx, &schemas.Session{
				UserID:    user.ID,
				UserAgent: userAgent,
				IP:        ip,
			}); err != nil {
				log.Debug().Err(err).Msg("Failed to add session")
			}
		})
		if strings.Contains(redirectURL, "?") {
			redirectURL = redirectURL + "&" + params
		} else {
			redirectURL = redirectURL + "?" + strings.TrimPrefix(params, "&")
		}
		// remove state from store
		_ = h.MemoryStoreProvider.RemoveState(state)
		metrics.RecordAuthEvent(metrics.EventOAuthCallback, metrics.StatusSuccess)
		metrics.ActiveSessions.Inc()
		h.AuditProvider.LogEvent(audit.Event{
			Action:       constants.AuditOAuthCallbackSuccessEvent,
			ActorID:      user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   refs.StringValue(user.Email),
			ResourceType: constants.AuditResourceTypeSession,
			ResourceID:   user.ID,
			Metadata:     provider,
			IPAddress:    utils.GetIP(ctx.Request),
			UserAgent:    utils.GetUserAgent(ctx.Request),
		})
		ctx.Redirect(http.StatusFound, redirectURL)
	}
}

// allowUnverifiedProviderEmail decides whether a federated login whose provider
// did NOT attest the email address may still resolve a local account.
//
// Default (--oauth-allow-unverified-provider-email=false): never. The address is
// attacker-controlled and it is what selects the account.
//
// Compatibility mode (=true) exists so a deployment upgrading from 2.3.x is not
// locked out the moment it restarts, but it is deliberately NOT a plain "turn
// the check off" switch — that would restore the CVE verbatim. Even in this
// mode, an unattested address may only:
//
//   - create a brand-new account (it selects nobody, so it harms nobody), or
//   - return to an account THIS SAME PROVIDER already owns — a returning user.
//
// It may never merge into an account some other credential owns. That single
// restriction removes the entire cross-credential takeover: an Entra tenant
// cannot reach a password account, a Google account, or any other provider's
// account, which is every practical form of the attack.
//
// The residual risk it does not cover, and the reason this mode is documented
// as temporary: two principals of the SAME unattested provider (two Entra
// tenants both asserting one address) can still collide. Pinning
// --microsoft-tenant-id or setting --microsoft-allowed-tenants closes that, and
// is the actual fix.
func (h *httpProvider) allowUnverifiedProviderEmail(provider string, existingUser *schemas.User, found bool) bool {
	if !h.Config.OAuthAllowUnverifiedProviderEmail {
		return false
	}
	if !found {
		// First-time signup: no account is being selected away from anyone.
		return true
	}
	if existingUser == nil {
		// "Found" with no row is an inconsistent storage result. Fail closed
		// rather than guess which case it was.
		return false
	}
	// Returning user of this same provider, or an attempt to cross into an
	// account another credential owns.
	return signupMethodsContain(existingUser.SignupMethods, provider)
}

// signupMethodsContain reports whether a stored comma-separated signup-methods
// list contains an exact method. Deliberately not strings.Contains: the
// provider names include the near-miss pair twitch/twitter, and a substring
// match on a security decision would let one provider inherit the other's
// accounts.
func signupMethodsContain(signupMethods, provider string) bool {
	for _, m := range strings.Split(signupMethods, ",") {
		if strings.TrimSpace(m) == provider {
			return true
		}
	}
	return false
}

// flexBool decodes a JSON boolean that some IdPs send quoted. Apple documents
// `email_verified` as "a string or Boolean value", and LinkedIn's userinfo has
// shipped both shapes; decoding either into a plain bool fails the whole claim
// set, which would silently turn a verified email into an unverified one.
type flexBool bool

// UnmarshalJSON accepts true/false, "true"/"false", or anything else (which
// decodes to false — unrecognised is never "verified").
func (b *flexBool) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*b = flexBool(claimTruthy(raw))
	return nil
}

// claimTruthy reads a boolean claim out of an untyped JSON value, tolerating
// the quoted-string form some IdPs emit. Used by the providers whose payloads
// are decoded into a map rather than oidcClaims (Apple, Discord, Roblox).
func claimTruthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(t, "true")
	}
	return false
}

// oidcClaims is the allow-list of OpenID Connect standard claims Authorizer
// maps onto a user. ID tokens are decoded into this and never straight into
// schemas.User, for two reasons:
//
//  1. Type safety. IdPs emit claims whose types don't match the storage
//     schema - Microsoft Entra's `roles` is an "Array of strings" (ID token
//     claims reference) while schemas.User.Roles is a string, and a single
//     mismatch fails the whole decode, so every login for a tenant that
//     assigns app roles broke with "unable to extract claims".
//  2. No claim can land in a storage-only column (_id, roles,
//     signup_methods, is_active, created_at ...) merely by sharing its json
//     tag.
type oidcClaims struct {
	// Subject is the provider-asserted stable identifier for the principal.
	// Unlike `email` it is not user-mutable, so it is the only claim safe to
	// treat as an identity key.
	Subject string `json:"sub"`
	// Issuer and TenantID back the Microsoft tenant checks; see
	// processMicrosoftUserInfo. Ignored for every other provider.
	Issuer   string `json:"iss"`
	TenantID string `json:"tid"`
	// EmailVerified is the provider's assertion that the principal actually
	// controls `Email`. Absent decodes to false — an IdP that does not say
	// "verified" has not verified anything, and this claim is what stops a
	// federated login from linking to somebody else's account.
	EmailVerified flexBool `json:"email_verified"`
	// XmsEdov ("email domain owner verified") is Microsoft Entra's equivalent.
	// Entra v2 ID tokens carry no `email_verified` claim at all, and their
	// `email` is a mutable, unverified profile attribute — the distinction
	// that makes the nOAuth attack work.
	XmsEdov flexBool `json:"xms_edov"`

	Email       string `json:"email"`
	GivenName   string `json:"given_name"`
	FamilyName  string `json:"family_name"`
	MiddleName  string `json:"middle_name"`
	Nickname    string `json:"nickname"`
	Gender      string `json:"gender"`
	Birthdate   string `json:"birthdate"`
	PhoneNumber string `json:"phone_number"`
	Picture     string `json:"picture"`
}

// toUser maps the claims onto a user, leaving absent claims nil so they don't
// overwrite stored values with empty strings.
func (c *oidcClaims) toUser() *schemas.User {
	user := &schemas.User{}
	for _, f := range []struct {
		value string
		dest  **string
	}{
		{c.Email, &user.Email},
		{c.GivenName, &user.GivenName},
		{c.FamilyName, &user.FamilyName},
		{c.MiddleName, &user.MiddleName},
		{c.Nickname, &user.Nickname},
		{c.Gender, &user.Gender},
		{c.Birthdate, &user.Birthdate},
		{c.PhoneNumber, &user.PhoneNumber},
		{c.Picture, &user.Picture},
	} {
		if f.value != "" {
			*f.dest = refs.NewStringRef(f.value)
		}
	}
	return user
}

func (h *httpProvider) processGoogleUserInfo(ctx *gin.Context, code string) (*schemas.User, bool, error) {
	log := h.Log.With().Str("func", "processGoogleUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodGoogle)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, false, fmt.Errorf("error getting oauth config: %s", err.Error())
	}
	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, false, fmt.Errorf("invalid google exchange code: %s", err.Error())
	}

	issuer := "https://accounts.google.com"
	if mockBase := h.TestOAuthBaseURL(constants.AuthRecipeMethodGoogle); mockBase != "" {
		issuer = mockBase
	}
	oidcProvider, err := getOIDCProvider(ctx, issuer)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create oidc provider: %s", err.Error())
	}
	verifier := oidcProvider.Verifier(&oidc.Config{ClientID: h.GoogleClientID})
	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Debug().Err(err).Msg("Failed to extract ID Token from OAuth2 token")
		return nil, false, fmt.Errorf("unable to extract id_token")
	}

	// Parse and verify ID Token payload.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify ID Token")
		return nil, false, fmt.Errorf("unable to verify id_token: %s", err.Error())
	}
	claims := &oidcClaims{}
	if err := idToken.Claims(claims); err != nil {
		log.Debug().Err(err).Msg("Failed to parse ID Token claims")
		return nil, false, fmt.Errorf("unable to extract claims")
	}

	// Google asserts control of the address via `email_verified`.
	return claims.toUser(), bool(claims.EmailVerified), nil
}

// setGithubHeaders applies the headers GitHub's REST API docs ask every
// request to carry: a Bearer credential plus the explicit media type and API
// version, so a future default-version bump can't silently change the payload
// shape under us.
func setGithubHeaders(req *http.Request, accessToken string) {
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func (h *httpProvider) processGithubUserInfo(ctx *gin.Context, code string) (*schemas.User, bool, error) {
	log := h.Log.With().Str("func", "processGithubUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodGithub)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, false, fmt.Errorf("error getting oauth config: %s", err.Error())
	}

	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, false, fmt.Errorf("invalid github exchange code: %s", err.Error())
	}
	userInfoURL := constants.GithubUserInfoURL
	emailsURL := constants.GithubUserEmails
	if mockBase := h.TestOAuthBaseURL(constants.AuthRecipeMethodGithub); mockBase != "" {
		userInfoURL = mockBase + "/userinfo"
		emailsURL = mockBase + "/user/emails"
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create github user info request")
		return nil, false, fmt.Errorf("error creating github user info request: %s", err.Error())
	}
	setGithubHeaders(req, oauth2Token.AccessToken)

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to request github user info")
		return nil, false, err
	}

	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read github user info response body")
		return nil, false, fmt.Errorf("failed to read github response body: %s", err.Error())
	}
	if response.StatusCode >= 400 {
		log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request github user info")
		return nil, false, fmt.Errorf("failed to request github user info: %s", string(body))
	}

	// Only the three fields below are used. A typed struct (rather than a
	// map[string]string) is required: GitHub's /user payload also carries
	// numbers (id, public_repos), booleans (site_admin) and nulls, any of
	// which fails a whole-map string decode.
	var userRawData struct {
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	}
	if err := json.Unmarshal(body, &userRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal github user info")
		return nil, false, fmt.Errorf("failed to parse github user info: %s", err.Error())
	}

	name := strings.Split(userRawData.Name, " ")
	firstName := ""
	lastName := ""
	if len(name) >= 1 && strings.TrimSpace(name[0]) != "" {
		firstName = name[0]
	}
	if len(name) > 1 && strings.TrimSpace(name[1]) != "" {
		lastName = name[1]
	}

	picture := userRawData.AvatarURL
	email := userRawData.Email

	if email == "" {
		type GithubUserEmails struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}

		// fetch using /users/email endpoint
		req, err := http.NewRequest(http.MethodGet, emailsURL, nil)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to create github emails request")
			return nil, false, fmt.Errorf("error creating github user info request: %s", err.Error())
		}
		setGithubHeaders(req, oauth2Token.AccessToken)

		response, err := client.Do(req)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to request github user email")
			return nil, false, err
		}

		defer func() { _ = response.Body.Close() }()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to read github user email response body")
			return nil, false, fmt.Errorf("failed to read github response body: %s", err.Error())
		}
		if response.StatusCode >= 400 {
			log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request github user email")
			return nil, false, fmt.Errorf("failed to request github user info: %s", string(body))
		}

		emailData := []GithubUserEmails{}
		err = json.Unmarshal(body, &emailData)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to parse github user email")
			return nil, false, fmt.Errorf("failed to parse github user email: %s", err.Error())
		}

		// GET /user/emails lists every address on the account, verified or
		// not. An unverified address proves nothing about who controls it, and
		// the caller looks the user up by email - accepting one would let a
		// GitHub account that merely *typed* someone else's address log into
		// that person's existing Authorizer account. Only verified addresses
		// are eligible; the primary one wins.
		for _, userEmail := range emailData {
			if !userEmail.Verified {
				continue
			}
			email = userEmail.Email
			if userEmail.Primary {
				break
			}
		}
		if email == "" {
			log.Debug().Msg("No verified email on github account")
			return nil, false, fmt.Errorf("failed to get a verified email address from github")
		}
	}

	user := &schemas.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &picture,
		Email:      &email,
	}

	return user, true, nil
}

func (h *httpProvider) processFacebookUserInfo(ctx *gin.Context, code string) (*schemas.User, bool, error) {
	log := h.Log.With().Str("func", "processFacebookUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodFacebook)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, false, fmt.Errorf("error getting oauth config: %s", err.Error())
	}
	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Invalid facebook exchange code")
		return nil, false, fmt.Errorf("invalid facebook exchange code: %s", err.Error())
	}
	userInfoURL := constants.FacebookUserInfoURL
	if mockBase := h.TestOAuthBaseURL(constants.AuthRecipeMethodFacebook); mockBase != "" {
		userInfoURL = mockBase + "/userinfo?access_token="
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", userInfoURL+oauth2Token.AccessToken, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Error creating facebook user info request")
		return nil, false, fmt.Errorf("error creating facebook user info request: %s", err.Error())
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to process facebook user")
		return nil, false, err
	}

	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read facebook response")
		return nil, false, fmt.Errorf("failed to read facebook response body: %s", err.Error())
	}
	if response.StatusCode >= 400 {
		log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request facebook user info")
		return nil, false, fmt.Errorf("failed to request facebook user info: %s", string(body))
	}
	// Typed decode, not fmt.Sprintf over a map: Graph API omits `email`
	// entirely when "no valid email address is available" (user/reference/user),
	// and formatting a missing key stored the literal string "<nil>" as the
	// user's email. Same for first_name/last_name.
	var userRawData struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Picture   struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}
	if err := json.Unmarshal(body, &userRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal facebook user info")
		return nil, false, fmt.Errorf("failed to parse facebook user info: %s", err.Error())
	}

	email := userRawData.Email
	if email == "" {
		log.Debug().Msg("Facebook user info has no email")
		return nil, false, fmt.Errorf("failed to get email from facebook user info: the account has no available email address")
	}

	picture := userRawData.Picture.Data.URL
	firstName := userRawData.FirstName
	lastName := userRawData.LastName

	user := &schemas.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &picture,
		Email:      &email,
	}

	return user, true, nil
}

func (h *httpProvider) processLinkedInUserInfo(ctx *gin.Context, code string) (*schemas.User, bool, error) {
	log := h.Log.With().Str("func", "processLinkedInUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodLinkedIn)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, false, fmt.Errorf("error getting oauth config: %s", err.Error())
	}

	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, false, fmt.Errorf("invalid linkedin exchange code: %s", err.Error())
	}

	userInfoURL := constants.LinkedInUserInfoURL
	if mockBase := h.TestOAuthBaseURL(constants.AuthRecipeMethodLinkedIn); mockBase != "" {
		userInfoURL = mockBase + "/userinfo"
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create linkedin user info request")
		return nil, false, fmt.Errorf("error creating linkedin user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", oauth2Token.AccessToken)},
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to request linkedin user info")
		return nil, false, err
	}

	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read linkedin user info response body")
		return nil, false, fmt.Errorf("failed to read linkedin response body: %s", err.Error())
	}

	if response.StatusCode >= 400 {
		log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request linkedin user info")
		return nil, false, fmt.Errorf("failed to request linkedin user info: %s", string(body))
	}

	// OIDC userinfo shape (sub/name/given_name/family_name/picture/locale/
	// email/email_verified) - one call, no separate /v2/emailAddress hop.
	var userRawData struct {
		GivenName     string   `json:"given_name"`
		FamilyName    string   `json:"family_name"`
		Picture       string   `json:"picture"`
		Email         string   `json:"email"`
		EmailVerified flexBool `json:"email_verified"`
	}
	if err := json.Unmarshal(body, &userRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal linkedin user info")
		return nil, false, fmt.Errorf("failed to parse linkedin user info: %s", err.Error())
	}

	// `email` is documented as optional - it is only present when the member
	// granted the `email` scope. Without it there is no identity key at all
	// (LinkedIn's `sub` is pairwise per-app), so this is a hard error rather
	// than a synthetic-email fallback.
	if userRawData.Email == "" {
		log.Debug().Msg("LinkedIn user info has no email")
		return nil, false, fmt.Errorf("failed to extract email from linkedin response")
	}

	user := &schemas.User{
		GivenName:  &userRawData.GivenName,
		FamilyName: &userRawData.FamilyName,
		Picture:    &userRawData.Picture,
		Email:      &userRawData.Email,
	}

	return user, bool(userRawData.EmailVerified), nil
}

func (h *httpProvider) processAppleUserInfo(ctx *gin.Context, code string, appleUser *AppleUserInfo) (*schemas.User, bool, error) {
	log := h.Log.With().Str("func", "processAppleUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodApple)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, false, fmt.Errorf("error getting oauth config: %s", err.Error())
	}

	var user = &schemas.User{}
	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return user, false, fmt.Errorf("invalid apple exchange code: %s", err.Error())
	}

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Debug().Err(err).Msg("Failed to extract ID Token from OAuth2 token")
		return user, false, fmt.Errorf("unable to extract id_token")
	}

	// Verify the Apple ID token signature, issuer, and audience using OIDC discovery
	issuer := "https://appleid.apple.com"
	if mockBase := h.TestOAuthBaseURL(constants.AuthRecipeMethodApple); mockBase != "" {
		issuer = mockBase
	}
	oidcProvider, err := getOIDCProvider(ctx, issuer)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create Apple OIDC provider")
		return user, false, fmt.Errorf("failed to create oidc provider: %s", err.Error())
	}
	verifier := oidcProvider.Verifier(&oidc.Config{ClientID: h.AppleClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify Apple ID Token")
		return user, false, fmt.Errorf("unable to verify id_token: %s", err.Error())
	}

	claims := make(map[string]interface{})
	if err := idToken.Claims(&claims); err != nil {
		log.Debug().Err(err).Msg("Failed to parse Apple ID Token claims")
		return user, false, fmt.Errorf("failed to parse claims: %s", err.Error())
	}

	if val, ok := claims["email"]; !ok || val == nil {
		log.Debug().Msg("Failed to extract email from claims.")
		return user, false, fmt.Errorf("unable to extract email, please check the scopes enabled for your app. It needs `email`, `name` scopes")
	} else {
		email, _ := val.(string)
		user.Email = &email
	}

	// Apple documents `email_verified` as "a string or Boolean value", so it
	// arrives as either true or "true" — claimTruthy accepts both. Absent means
	// unverified.
	emailVerified := claimTruthy(claims["email_verified"])

	user.GivenName = &appleUser.Name.FirstName
	user.FamilyName = &appleUser.Name.LastName

	return user, emailVerified, nil
}

// processDiscordUserInfo exchanges the Discord OAuth code for the user's
// profile via GET /users/@me (constants.DiscordUserInfoURL), which returns a
// flat id/username/avatar/email object. defaultDiscordScopes (cmd/root.go)
// already requests `identify email` by default, and Discord returns a real,
// deliverable email whenever the user grants it - no special app-dashboard
// permission needed, unlike X's confirmed_email. Discord's consent screen
// does allow granular per-scope denial though, so a user can authorize
// `identify` while declining `email`; for that case, fall back to a stable
// synthetic address keyed on Discord's permanent id (never username, which
// can change) so the signup-vs-login lookup (GetUserByEmail in
// OAuthCallbackHandler) still recognizes a returning user instead of
// creating a duplicate account - the same fallback discipline
// processTwitterUserInfo uses above for X, which never returns a real email
// at all.
func (h *httpProvider) processDiscordUserInfo(ctx *gin.Context, code string) (*schemas.User, bool, error) {
	log := h.Log.With().Str("func", "processDiscordUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodDiscord)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, false, fmt.Errorf("error getting oauth config: %s", err.Error())
	}
	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, false, fmt.Errorf("invalid discord exchange code: %s", err.Error())
	}

	userInfoURL := constants.DiscordUserInfoURL
	if mockBase := h.TestOAuthBaseURL(constants.AuthRecipeMethodDiscord); mockBase != "" {
		userInfoURL = mockBase + "/userinfo"
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create Discord user info request")
		return nil, false, fmt.Errorf("error creating Discord user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", oauth2Token.AccessToken)},
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to request Discord user info")
		return nil, false, err
	}

	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read Discord user info response body")
		return nil, false, fmt.Errorf("failed to read Discord response body: %s", err.Error())
	}

	if response.StatusCode >= 400 {
		log.Debug().Err(err).Msg("Failed to request Discord user info")
		return nil, false, fmt.Errorf("failed to request Discord user info: %s", string(body))
	}

	// Unmarshal the response body into a map. GET /users/@me returns a flat
	// object - no "user" sub-key to unwrap (unlike /oauth2/@me, which this
	// used to call).
	userRawData := make(map[string]interface{})
	if err := json.Unmarshal(body, &userRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal Discord response")
		return nil, false, fmt.Errorf("failed to unmarshal Discord response: %s", err.Error())
	}

	// Extract the username
	firstName, ok := userRawData["username"].(string)
	if !ok {
		log.Debug().Err(err).Msg("Username is not in expected format or missing in user data")
		return nil, false, fmt.Errorf("username is not in expected format or missing in user data")
	}
	discordID, ok := userRawData["id"].(string)
	if !ok || discordID == "" {
		log.Debug().Msg("Discord user info missing id")
		return nil, false, fmt.Errorf("discord response missing id field")
	}
	// `avatar` is nullable (?string in Discord's user object) for accounts on
	// the default avatar - building the CDN URL from an empty hash yields a
	// dead ".../avatars/<id>/.png" link, so leave the picture unset instead.
	profilePicture := ""
	if avatar, ok := userRawData["avatar"].(string); ok && avatar != "" {
		profilePicture = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", discordID, avatar)
	}

	email := resolveDiscordEmail(discordID, userRawData)
	// GET /users/@me carries a `verified` flag for the account's email. The
	// synthetic fallback is trusted by construction: it lives on a reserved
	// non-routable domain keyed by Discord's permanent id, so it can never
	// collide with an address a real person could prove they own.
	emailVerified := claimTruthy(userRawData["verified"])
	if email == discordSyntheticEmail(discordID) {
		emailVerified = true
	}

	user := &schemas.User{
		GivenName: &firstName,
		Picture:   &profilePicture,
		Email:     &email,
	}

	return user, emailVerified, nil
}

// resolveDiscordEmail prefers the real email Discord returns; falls back to
// a synthetic one keyed on the permanent id if the user denied the `email`
// scope at Discord's consent screen (see processDiscordUserInfo doc
// comment). Mirrors resolveTwitterEmail below.
func resolveDiscordEmail(discordID string, userRawData map[string]interface{}) string {
	if realEmail, ok := userRawData["email"].(string); ok && realEmail != "" {
		return realEmail
	}
	return discordSyntheticEmail(discordID)
}

// discordSyntheticEmail derives a stable, non-routable synthetic email from
// Discord's permanent user id, used when a user grants `identify` but denies
// the `email` scope (see processDiscordUserInfo doc comment). Mirrors
// twitterSyntheticEmail below.
func discordSyntheticEmail(discordID string) string {
	return fmt.Sprintf("discord-%s@discord.oauth.internal", discordID)
}

// processTwitterUserInfo exchanges the Twitter/X OAuth code for the user's
// profile. By default Twitter/X does not return a real email address (see
// comment below), so the returned user's Email is a synthetic address
// derived from Twitter's numeric id, not a real, contactable address - this
// lets the caller's signup-vs-login lookup (GetUserByEmail) recognize a
// returning Twitter user instead of creating a duplicate account on every
// login. Operators who opt into X's `users.email` scope + app permission get
// a real confirmed_email instead (see TwitterUserInfoURL's doc comment).
func (h *httpProvider) processTwitterUserInfo(ctx *gin.Context, code, verifier string) (*schemas.User, bool, error) {
	log := h.Log.With().Str("func", "processTwitterUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodTwitter)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, false, fmt.Errorf("error getting oauth config: %s", err.Error())
	}

	oauth2Token, err := cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, false, fmt.Errorf("invalid twitter exchange code: %s", err.Error())
	}

	userInfoURL := constants.TwitterUserInfoURL
	if mockBase := h.TestOAuthBaseURL(constants.AuthRecipeMethodTwitter); mockBase != "" {
		userInfoURL = mockBase + "/userinfo"
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create Twitter user info request")
		return nil, false, fmt.Errorf("error creating Twitter user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", oauth2Token.AccessToken)},
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to request Twitter user info")
		return nil, false, err
	}

	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read Twitter user info response body")
		return nil, false, fmt.Errorf("failed to read Twitter response body: %s", err.Error())
	}

	if response.StatusCode >= 400 {
		log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request Twitter user info")
		return nil, false, fmt.Errorf("failed to request Twitter user info: %s", string(body))
	}

	responseRawData := make(map[string]interface{})
	if err := json.Unmarshal(body, &responseRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal twitter user info")
		return nil, false, fmt.Errorf("failed to parse twitter user info: %s", err.Error())
	}

	userRawData, ok := responseRawData["data"].(map[string]interface{})
	if !ok {
		return nil, false, fmt.Errorf("twitter response missing data field")
	}

	// Twitter API does not return E-Mail adresses by default. For that case special privileges have
	// to be granted on a per-App basis. See https://developer.twitter.com/en/docs/twitter-api/v1/accounts-and-users/manage-account-settings/api-reference/get-account-verify_credentials
	//
	// Without an email, the signup-vs-login check in OAuthCallbackHandler
	// (h.StorageProvider.GetUserByEmail(ctx, refs.StringValue(user.Email)))
	// would always be called with "" and never match, so every Twitter login
	// would be treated as a brand-new signup - duplicate accounts on every
	// login for the same person. To keep that lookup working, synthesize a
	// stable, non-routable email from Twitter's numeric `id` (permanent,
	// unlike `username`, which can change). Unlike GitHub's noreply address
	// (a real, GitHub-issued, deliverable email fetched from GitHub's own
	// API further down in this file), this one is constructed client-side
	// and never delivers anywhere - it exists purely as a stable lookup key.
	twitterID, ok := userRawData["id"].(string)
	if !ok || twitterID == "" {
		log.Debug().Msg("Twitter user info missing id")
		return nil, false, fmt.Errorf("twitter response missing id field")
	}

	// Currently Twitter API only provides the full name of a user. To fill givenName and familyName
	// the full name will be split at the first whitespace. This approach will not be valid for all name combinations
	firstName := ""
	lastName := ""
	if name, ok := userRawData["name"].(string); ok {
		nameArr := strings.SplitAfterN(name, " ", 2)
		firstName = nameArr[0]
		if len(nameArr) == 2 {
			lastName = nameArr[1]
		}
	}
	nickname, _ := userRawData["username"].(string)
	profilePicture, _ := userRawData["profile_image_url"].(string)

	email := resolveTwitterEmail(twitterID, userRawData)
	// X only returns `confirmed_email` to apps granted the `users.email` scope
	// plus the app-dashboard permission, and the name says it: X has confirmed
	// it. The synthetic fallback is trusted by construction — a reserved
	// non-routable domain keyed by X's permanent numeric id, which no real
	// mailbox can occupy.
	emailVerified := true

	user := &schemas.User{
		Email:      &email,
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &profilePicture,
		Nickname:   &nickname,
	}

	return user, emailVerified, nil
}

// twitterSyntheticEmail derives a stable, non-routable synthetic email from
// Twitter's numeric user id, so the same Twitter account always maps to the
// same Authorizer email (see processTwitterUserInfo doc comment).
func twitterSyntheticEmail(twitterID string) string {
	return fmt.Sprintf("twitter-%s@twitter.oauth.internal", twitterID)
}

// resolveTwitterEmail picks the email to store for a Twitter-authenticated
// user. X's `users.email` scope + "Request email from users" app permission
// (see the TwitterUserInfoURL doc comment in internal/constants/oauth_info_urls.go
// for how an operator opts in) makes the API return a real, deliverable
// confirmed_email; this is preferred when present. Otherwise falls back to
// the synthetic per-id email, which still correctly prevents duplicate
// accounts (see processTwitterUserInfo doc comment) without a real address.
func resolveTwitterEmail(twitterID string, userRawData map[string]interface{}) string {
	if confirmedEmail, ok := userRawData["confirmed_email"].(string); ok && confirmedEmail != "" {
		return confirmedEmail
	}
	return twitterSyntheticEmail(twitterID)
}

// process microsoft user information
func (h *httpProvider) processMicrosoftUserInfo(ctx *gin.Context, code string) (*schemas.User, bool, error) {
	log := h.Log.With().Str("func", "processMicrosoftUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodMicrosoft)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, false, fmt.Errorf("error getting oauth config: %s", err.Error())
	}
	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, false, fmt.Errorf("invalid microsoft exchange code: %s", err.Error())
	}
	issuer := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", h.MicrosoftTenantID)
	if mockBase := h.TestOAuthBaseURL(constants.AuthRecipeMethodMicrosoft); mockBase != "" {
		issuer = mockBase
	}
	oidcProvider, err := getOIDCProvider(ctx, issuer)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create oidc provider: %s", err.Error())
	}
	// The multi-tenant discovery documents ("common"/"organizations"/
	// "consumers") advertise a templated issuer containing {tenantid}, which
	// never literally equals the `iss` of a real token, so go-oidc's built-in
	// comparison cannot be used. Skipping it is not the same as not checking:
	// validateMicrosoftTenant below reconstructs the expected issuer from the
	// token's own `tid` and enforces it, plus the operator's tenant policy.
	verifier := oidcProvider.Verifier(&oidc.Config{
		ClientID:        h.MicrosoftClientID,
		SkipIssuerCheck: true,
	})
	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Debug().Err(err).Msg("Failed to extract ID Token from OAuth2 token")
		return nil, false, fmt.Errorf("unable to extract id_token")
	}
	// Parse and verify ID Token payload.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify ID Token")
		return nil, false, fmt.Errorf("unable to verify id_token: %s", err.Error())
	}
	claims := &oidcClaims{}
	if err := idToken.Claims(claims); err != nil {
		log.Debug().Err(err).Msg("Failed to parse ID Token claims")
		return nil, false, fmt.Errorf("unable to extract claims")
	}

	// The test double issues tokens from a stand-in issuer with no tenant
	// model at all; tenant policy is meaningless there.
	if mockBase := h.TestOAuthBaseURL(constants.AuthRecipeMethodMicrosoft); mockBase != "" {
		return claims.toUser(), bool(claims.EmailVerified), nil
	}

	tenantPinned, err := validateMicrosoftTenant(claims, h.MicrosoftTenantID, h.Config.MicrosoftAllowedTenants)
	if err != nil {
		log.Debug().Err(err).Str("tid", claims.TenantID).Msg("Microsoft tenant validation failed")
		return nil, false, err
	}

	// Entra v2 ID tokens have no `email_verified` claim, and `email` is a
	// mutable, unverified profile attribute any tenant admin can set to any
	// string — including somebody else's address. Two signals make it
	// trustworthy, and nothing else does:
	//
	//   - xms_edov ("email domain owner verified"), Microsoft's own attestation
	//     that the token's tenant owns the email's domain. It is an optional
	//     claim; operators enable it in the app registration.
	//   - the tenant being pinned or allowlisted, which means the address can
	//     only have come from a directory the operator already trusts.
	//
	// Without either, this is the nOAuth setup: an attacker registers a free
	// Entra tenant, sets a user's `email` to the victim's address, and signs in.
	return claims.toUser(), tenantPinned || bool(claims.XmsEdov), nil
}

// microsoftMultiTenantAliases are the Entra endpoint aliases that accept tokens
// from tenants the operator has never heard of. Any other configured value is a
// specific tenant (a GUID or a verified domain name) and is therefore pinned.
var microsoftMultiTenantAliases = map[string]bool{
	"common":        true,
	"organizations": true,
	"consumers":     true,
}

// validateMicrosoftTenant enforces the operator's tenant policy on a verified
// Entra ID token and reports whether the originating tenant is one the operator
// explicitly trusts.
//
// go-oidc has already checked the signature and that `aud` equals our client id
// — neither of which constrains WHICH tenant minted the token, because the
// multi-tenant endpoints sign with Microsoft's global keys. The tenant is the
// only thing that does, so it is checked here:
//
//   - `iss` must be the issuer the token's own `tid` implies, so a token cannot
//     claim one tenant in `iss` and another in `tid`;
//   - a pinned `--microsoft-tenant-id` must match `tid` exactly;
//   - a non-empty `--microsoft-allowed-tenants` must contain `tid`.
//
// Returns true when the tenant was pinned or allowlisted.
func validateMicrosoftTenant(claims *oidcClaims, configuredTenant string, allowedTenants []string) (bool, error) {
	tid := strings.TrimSpace(claims.TenantID)
	if tid == "" {
		return false, fmt.Errorf("microsoft id_token is missing the tid claim")
	}
	if expected := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tid); claims.Issuer != expected {
		return false, fmt.Errorf("microsoft id_token issuer does not match its tenant")
	}

	if len(allowedTenants) > 0 {
		if !utils.StringSliceContains(allowedTenants, tid) {
			return false, fmt.Errorf("microsoft tenant is not allowed")
		}
		return true, nil
	}

	configuredTenant = strings.TrimSpace(configuredTenant)
	if !microsoftMultiTenantAliases[strings.ToLower(configuredTenant)] {
		// A specific tenant was configured: only that directory may sign in.
		if !strings.EqualFold(configuredTenant, tid) {
			return false, fmt.Errorf("microsoft id_token was issued by an unexpected tenant")
		}
		return true, nil
	}

	// Multi-tenant with no allowlist. The login is still permitted — this is a
	// documented deployment mode — but the tenant is not trusted, so the caller
	// must not treat the email as proof of anything.
	return false, nil
}

// process twitch user information
func (h *httpProvider) processTwitchUserInfo(ctx *gin.Context, code string) (*schemas.User, bool, error) {
	log := h.Log.With().Str("func", "processTwitchUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodTwitch)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, false, fmt.Errorf("error getting oauth config: %s", err.Error())
	}

	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, false, fmt.Errorf("invalid twitch exchange code: %s", err.Error())
	}

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Debug().Err(err).Msg("Failed to extract ID Token from OAuth2 token")
		return nil, false, fmt.Errorf("unable to extract id_token")
	}
	issuer := "https://id.twitch.tv/oauth2"
	if mockBase := h.TestOAuthBaseURL(constants.AuthRecipeMethodTwitch); mockBase != "" {
		issuer = mockBase
	}
	oidcProvider, err := getOIDCProvider(ctx, issuer)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create OIDC provider")
		return nil, false, fmt.Errorf("failed to create oidc provider: %s", err.Error())
	}
	verifier := oidcProvider.Verifier(&oidc.Config{
		ClientID:        h.TwitchClientID,
		SkipIssuerCheck: true,
	})

	// Parse and verify ID Token payload.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify ID Token")
		return nil, false, fmt.Errorf("unable to verify id_token: %s", err.Error())
	}

	claims := &oidcClaims{}
	if err := idToken.Claims(claims); err != nil {
		log.Debug().Err(err).Msg("Failed to parse ID Token claims")
		return nil, false, fmt.Errorf("unable to extract claims")
	}

	// Twitch is single-issuer (SkipIssuerCheck above is harmless — signature
	// and `aud` already pin the token to Twitch), and its ID token carries the
	// standard `email_verified`.
	return claims.toUser(), bool(claims.EmailVerified), nil
}

// process roblox user information
func (h *httpProvider) processRobloxUserInfo(ctx *gin.Context, code string) (*schemas.User, bool, error) {
	log := h.Log.With().Str("func", "processRobloxUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodRoblox)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, false, fmt.Errorf("error getting oauth config: %s", err.Error())
	}
	// Roblox is a confidential client (client_secret set); PKCE is optional and
	// no code_challenge is sent at login, so no code_verifier is replayed here.
	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, false, fmt.Errorf("invalid roblox exchange code: %s", err.Error())
	}

	userInfoURL := constants.RobloxUserInfoURL
	if mockBase := h.TestOAuthBaseURL(constants.AuthRecipeMethodRoblox); mockBase != "" {
		userInfoURL = mockBase + "/userinfo"
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create roblox user info request")
		return nil, false, fmt.Errorf("error creating roblox user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", oauth2Token.AccessToken)},
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to request roblox user info")
		return nil, false, err
	}

	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read roblox user info response body")
		return nil, false, fmt.Errorf("failed to read roblox response body: %s", err.Error())
	}

	if response.StatusCode >= 400 {
		log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request roblox user info")
		return nil, false, fmt.Errorf("failed to request roblox user info: %s", string(body))
	}

	userRawData := make(map[string]interface{})
	if err := json.Unmarshal(body, &userRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal roblox user info")
		return nil, false, fmt.Errorf("failed to parse roblox user info: %s", err.Error())
	}

	firstName := ""
	lastName := ""
	if name, ok := userRawData["name"].(string); ok {
		nameArr := strings.SplitAfterN(name, " ", 2)
		firstName = nameArr[0]
		if len(nameArr) == 2 {
			lastName = nameArr[1]
		}
	}
	nickname, _ := userRawData["nickname"].(string)
	profilePicture, _ := userRawData["picture"].(string)
	sub, _ := userRawData["sub"].(string)
	email := resolveRobloxEmail(sub, userRawData)
	// Roblox's userinfo is OIDC-standard, so a real address comes with
	// `email_verified`. The synthetic fallback (the default config, which does
	// not request the `email` scope) is trusted by construction: reserved
	// non-routable domain keyed by the permanent `sub`.
	emailVerified := claimTruthy(userRawData["email_verified"])
	if sub != "" && email == robloxSyntheticEmail(sub) {
		emailVerified = true
	}
	user := &schemas.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &profilePicture,
		Nickname:   &nickname,
		Email:      &email,
	}

	return user, emailVerified, nil
}

// resolveRobloxEmail prefers the real email Roblox returns; falls back to a
// synthetic one keyed on the permanent sub if it's absent. defaultRobloxScopes
// (cmd/root.go) is ["openid", "profile"] - no `email` scope - so real Roblox
// userinfo (an OIDC-standard endpoint, see constants.RobloxUserInfoURL)
// returns the mandatory `sub` claim but omits `email` under the default
// config. Falling back to the bare `sub` would store a non-email-shaped
// numeric id in user.Email, so this synthesizes a stable, non-routable
// address instead - mirrors resolveTwitterEmail/resolveDiscordEmail above.
func resolveRobloxEmail(sub string, userRawData map[string]interface{}) string {
	if val, ok := userRawData["email"].(string); ok && val != "" {
		return val
	}
	if sub != "" {
		return robloxSyntheticEmail(sub)
	}
	return ""
}

// robloxSyntheticEmail derives a stable, non-routable synthetic email from
// Roblox's numeric user id, used when a user's granted scopes don't include
// `email` (see processRobloxUserInfo doc comment). Mirrors
// twitterSyntheticEmail/discordSyntheticEmail above.
func robloxSyntheticEmail(sub string) string {
	return fmt.Sprintf("roblox-%s@roblox.oauth.internal", sub)
}

// parseScopes parses a scope string into a slice of individual scope values.
// Commas take precedence over spaces as delimiter. If neither delimiter is
// present, the entire string is returned as a single-element slice.
// RFC 6749 §3.3 defines space as the standard delimiter; commas are accepted
// as a convenience.
func parseScopes(scopeString string) []string {
	if scopeString == "" {
		return []string{}
	}
	if strings.Contains(scopeString, ",") {
		return strings.Split(scopeString, ",")
	}
	if strings.Contains(scopeString, " ") {
		return strings.Split(scopeString, " ")
	}
	return []string{scopeString}
}
