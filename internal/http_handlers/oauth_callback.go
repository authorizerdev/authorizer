package http_handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
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
		// `sessionState` is the oauth provider saved during `/oauth_login/:oauth_provider`.
		// Ensure the callback route's provider matches what was originally requested.
		if sessionState != provider {
			log.Debug().
				Str("expected_provider", sessionState).
				Str("callback_provider", provider).
				Msg("OAuth provider mismatch for state")
			ctx.JSON(400, gin.H{"error": "invalid oauth state"})
			return
		}
		// contains random token, redirect url, role
		sessionSplit := strings.Split(state, "___")

		if len(sessionSplit) < 4 {
			log.Debug().Msg("Invalid state: expected at least 4 segments")
			ctx.JSON(400, gin.H{"error": "invalid oauth state"})
			return
		}
		// remove state from store
		h.MemoryStoreProvider.RemoveState(state)
		stateValue := sessionSplit[0]
		redirectURL := sessionSplit[1]
		hostname := parsers.GetHost(ctx)
		if !validators.IsValidRedirectURI(redirectURL, h.Config.AllowedOrigins, hostname) {
			log.Debug().Msg("Invalid redirect URI in OAuth state")
			ctx.JSON(400, gin.H{"error": "invalid redirect uri"})
			return
		}
		inputRoles := strings.Split(sessionSplit[2], ",")
		scopeString := sessionSplit[3]
		scopes := parseScopes(scopeString)
		var user *schemas.User
		oauthCode := ctx.Request.FormValue("code")
		if oauthCode == "" {
			log.Debug().Err(err).Msg("Invalid oauth code")
			ctx.JSON(400, gin.H{"error": "invalid oauth code"})
			return
		}
		switch provider {
		case constants.AuthRecipeMethodGoogle:
			user, err = h.processGoogleUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodGithub:
			user, err = h.processGithubUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodFacebook:
			user, err = h.processFacebookUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodLinkedIn:
			user, err = h.processLinkedInUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodApple:
			user_ := AppleUserInfo{}
			userRaw := ctx.Request.FormValue("user")
			err = json.Unmarshal([]byte(userRaw), &user_)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to unmarshal apple user info")
				ctx.JSON(400, gin.H{"error": "invalid apple user info"})
				return
			}
			user, err = h.processAppleUserInfo(ctx, oauthCode, &user_)
		case constants.AuthRecipeMethodDiscord:
			user, err = h.processDiscordUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodTwitter:
			user, err = h.processTwitterUserInfo(ctx, oauthCode, sessionState)
		case constants.AuthRecipeMethodMicrosoft:
			user, err = h.processMicrosoftUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodTwitch:
			user, err = h.processTwitchUserInfo(ctx, oauthCode)
		case constants.AuthRecipeMethodRoblox:
			user, err = h.processRobloxUserInfo(ctx, oauthCode, sessionState)
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
			now := time.Now().Unix()
			user.EmailVerifiedAt = &now
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
			if existingUser.EmailVerifiedAt == nil {
				log.Info().Msg("Removing unverified pre-existing account before OAuth signup")
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
		code, codeChallenge, nonce, authorizeRedirectURI, err := h.consumeAuthorizeState(stateValue)
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

		// Code challenge could be optional if PKCE flow is not used
		if code != "" {
			if err := h.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash+"@@"+nonce+"@@"+url.QueryEscape(authorizeRedirectURI)); err != nil {
				log.Debug().Err(err).Msg("Failed to set state")
				ctx.JSON(500, gin.H{"error": "failed to process OAuth login"})
				return
			}
		}

		expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
		if expiresIn <= 0 {
			expiresIn = 1
		}

		// params := "access_token=" + authToken.AccessToken.Token + "&token_type=bearer&expires_in=" + strconv.FormatInt(expiresIn, 10) + "&state=" + stateValue + "&id_token=" + authToken.IDToken.Token + "&nonce=" + nonce
		// Note: If OIDC breaks in the future, use the above params
		params := "state=" + stateValue + "&nonce=" + nonce
		if code != "" {
			params += "&code=" + code
		}

		sessionKey := provider + ":" + user.ID
		cookie.SetSession(ctx, authToken.FingerPrintHash, h.Config.AppCookieSecure, cookie.ParseSameSite(h.Config.AppCookieSameSite))
		h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
		h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

		if authToken.RefreshToken != nil {
			h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
		}

		go func() {
			if isSignUp {
				h.EventsProvider.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, provider, user)
				// User is also logged in with signup
				h.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, provider, user)
			} else {
				h.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, provider, user)
			}
			if err := h.StorageProvider.AddSession(ctx, &schemas.Session{
				UserID:    user.ID,
				UserAgent: utils.GetUserAgent(ctx.Request),
				IP:        utils.GetIP(ctx.Request),
			}); err != nil {
				log.Debug().Err(err).Msg("Failed to add session")
			}
		}()
		if strings.Contains(redirectURL, "?") {
			redirectURL = redirectURL + "&" + params
		} else {
			redirectURL = redirectURL + "?" + strings.TrimPrefix(params, "&")
		}
		// remove state from store
		h.MemoryStoreProvider.RemoveState(state)
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

func (h *httpProvider) processGoogleUserInfo(ctx *gin.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processGoogleUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodGoogle)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, fmt.Errorf("error getting oauth config: %s", err.Error())
	}
	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, fmt.Errorf("invalid google exchange code: %s", err.Error())
	}

	oidcProvider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	verifier := oidcProvider.Verifier(&oidc.Config{ClientID: h.GoogleClientID})
	if err != nil {
		return nil, fmt.Errorf("failed to create oidc provider: %s", err.Error())
	}
	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Debug().Err(err).Msg("Failed to extract ID Token from OAuth2 token")
		return nil, fmt.Errorf("unable to extract id_token")
	}

	// Parse and verify ID Token payload.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify ID Token")
		return nil, fmt.Errorf("unable to verify id_token: %s", err.Error())
	}
	user := &schemas.User{}
	if err := idToken.Claims(&user); err != nil {
		log.Debug().Err(err).Msg("Failed to parse ID Token claims")
		return nil, fmt.Errorf("unable to extract claims")
	}

	return user, nil
}

func (h *httpProvider) processGithubUserInfo(ctx *gin.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processGithubUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodGithub)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, fmt.Errorf("error getting oauth config: %s", err.Error())
	}

	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, fmt.Errorf("invalid github exchange code: %s", err.Error())
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", constants.GithubUserInfoURL, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create github user info request")
		return nil, fmt.Errorf("error creating github user info request: %s", err.Error())
	}
	req.Header.Set(
		"Authorization", fmt.Sprintf("token %s", oauth2Token.AccessToken),
	)

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to request github user info")
		return nil, err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read github user info response body")
		return nil, fmt.Errorf("failed to read github response body: %s", err.Error())
	}
	if response.StatusCode >= 400 {
		log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request github user info")
		return nil, fmt.Errorf("failed to request github user info: %s", string(body))
	}

	userRawData := make(map[string]string)
	if err := json.Unmarshal(body, &userRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal github user info")
		return nil, fmt.Errorf("failed to parse github user info: %s", err.Error())
	}

	name := strings.Split(userRawData["name"], " ")
	firstName := ""
	lastName := ""
	if len(name) >= 1 && strings.TrimSpace(name[0]) != "" {
		firstName = name[0]
	}
	if len(name) > 1 && strings.TrimSpace(name[1]) != "" {
		lastName = name[1]
	}

	picture := userRawData["avatar_url"]
	email := userRawData["email"]

	if email == "" {
		type GithubUserEmails struct {
			Email   string `json:"email"`
			Primary bool   `json:"primary"`
		}

		// fetch using /users/email endpoint
		req, err := http.NewRequest(http.MethodGet, constants.GithubUserEmails, nil)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to create github emails request")
			return nil, fmt.Errorf("error creating github user info request: %s", err.Error())
		}
		req.Header.Set(
			"Authorization", fmt.Sprintf("token %s", oauth2Token.AccessToken),
		)

		response, err := client.Do(req)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to request github user email")
			return nil, err
		}

		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to read github user email response body")
			return nil, fmt.Errorf("failed to read github response body: %s", err.Error())
		}
		if response.StatusCode >= 400 {
			log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request github user email")
			return nil, fmt.Errorf("failed to request github user info: %s", string(body))
		}

		emailData := []GithubUserEmails{}
		err = json.Unmarshal(body, &emailData)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to parse github user email")
			return nil, fmt.Errorf("failed to parse github user email: %s", err.Error())
		}

		for _, userEmail := range emailData {
			email = userEmail.Email
			if userEmail.Primary {
				break
			}
		}
	}

	user := &schemas.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &picture,
		Email:      &email,
	}

	return user, nil
}

func (h *httpProvider) processFacebookUserInfo(ctx *gin.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processFacebookUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodFacebook)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, fmt.Errorf("error getting oauth config: %s", err.Error())
	}
	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Invalid facebook exchange code")
		return nil, fmt.Errorf("invalid facebook exchange code: %s", err.Error())
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", constants.FacebookUserInfoURL+oauth2Token.AccessToken, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Error creating facebook user info request")
		return nil, fmt.Errorf("error creating facebook user info request: %s", err.Error())
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to process facebook user")
		return nil, err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read facebook response")
		return nil, fmt.Errorf("failed to read facebook response body: %s", err.Error())
	}
	if response.StatusCode >= 400 {
		log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request facebook user info")
		return nil, fmt.Errorf("failed to request facebook user info: %s", string(body))
	}
	userRawData := make(map[string]interface{})
	if err := json.Unmarshal(body, &userRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal facebook user info")
		return nil, fmt.Errorf("failed to parse facebook user info: %s", err.Error())
	}

	email := fmt.Sprintf("%v", userRawData["email"])

	picture := ""
	if picObj, ok := userRawData["picture"].(map[string]interface{}); ok {
		if picData, ok := picObj["data"].(map[string]interface{}); ok {
			picture = fmt.Sprintf("%v", picData["url"])
		}
	}
	firstName := fmt.Sprintf("%v", userRawData["first_name"])
	lastName := fmt.Sprintf("%v", userRawData["last_name"])

	user := &schemas.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &picture,
		Email:      &email,
	}

	return user, nil
}

func (h *httpProvider) processLinkedInUserInfo(ctx *gin.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processLinkedInUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodLinkedIn)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, fmt.Errorf("error getting oauth config: %s", err.Error())
	}

	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, fmt.Errorf("invalid linkedin exchange code: %s", err.Error())
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", constants.LinkedInUserInfoURL, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create linkedin user info request")
		return nil, fmt.Errorf("error creating linkedin user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", oauth2Token.AccessToken)},
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to request linkedin user info")
		return nil, err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read linkedin user info response body")
		return nil, fmt.Errorf("failed to read linkedin response body: %s", err.Error())
	}

	if response.StatusCode >= 400 {
		log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request linkedin user info")
		return nil, fmt.Errorf("failed to request linkedin user info: %s", string(body))
	}

	userRawData := make(map[string]interface{})
	if err := json.Unmarshal(body, &userRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal linkedin user info")
		return nil, fmt.Errorf("failed to parse linkedin user info: %s", err.Error())
	}

	req, err = http.NewRequest("GET", constants.LinkedInEmailURL, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create linkedin email info request")
		return nil, fmt.Errorf("error creating linkedin user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", oauth2Token.AccessToken)},
	}

	response, err = client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to request linkedin email info")
		return nil, err
	}

	defer response.Body.Close()
	body, err = io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read linkedin email info response body")
		return nil, fmt.Errorf("failed to read linkedin email response body: %s", err.Error())
	}
	if response.StatusCode >= 400 {
		log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request linkedin user info")
		return nil, fmt.Errorf("failed to request linkedin user info: %s", string(body))
	}
	emailRawData := make(map[string]interface{})
	if err := json.Unmarshal(body, &emailRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal linkedin email info")
		return nil, fmt.Errorf("failed to parse linkedin email info: %s", err.Error())
	}

	firstName, _ := userRawData["localizedFirstName"].(string)
	lastName, _ := userRawData["localizedLastName"].(string)

	// Safely extract profile picture from nested LinkedIn structure
	profilePicture := ""
	if pp, ok := userRawData["profilePicture"].(map[string]interface{}); ok {
		if di, ok := pp["displayImage~"].(map[string]interface{}); ok {
			if elems, ok := di["elements"].([]interface{}); ok && len(elems) > 0 {
				if elem, ok := elems[0].(map[string]interface{}); ok {
					if ids, ok := elem["identifiers"].([]interface{}); ok && len(ids) > 0 {
						if id, ok := ids[0].(map[string]interface{}); ok {
							profilePicture, _ = id["identifier"].(string)
						}
					}
				}
			}
		}
	}

	// Safely extract email from nested LinkedIn structure
	emailAddress := ""
	if elems, ok := emailRawData["elements"].([]interface{}); ok && len(elems) > 0 {
		if elem, ok := elems[0].(map[string]interface{}); ok {
			if handle, ok := elem["handle~"].(map[string]interface{}); ok {
				emailAddress, _ = handle["emailAddress"].(string)
			}
		}
	}
	if emailAddress == "" {
		return nil, fmt.Errorf("failed to extract email from linkedin response")
	}

	user := &schemas.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &profilePicture,
		Email:      &emailAddress,
	}

	return user, nil
}

func (h *httpProvider) processAppleUserInfo(ctx *gin.Context, code string, user_ *AppleUserInfo) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processAppleUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodApple)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, fmt.Errorf("error getting oauth config: %s", err.Error())
	}

	var user = &schemas.User{}
	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return user, fmt.Errorf("invalid apple exchange code: %s", err.Error())
	}

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Debug().Err(err).Msg("Failed to extract ID Token from OAuth2 token")
		return user, fmt.Errorf("unable to extract id_token")
	}

	// Verify the Apple ID token signature, issuer, and audience using OIDC discovery
	oidcProvider, err := oidc.NewProvider(ctx, "https://appleid.apple.com")
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create Apple OIDC provider")
		return user, fmt.Errorf("failed to create oidc provider: %s", err.Error())
	}
	verifier := oidcProvider.Verifier(&oidc.Config{ClientID: h.AppleClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify Apple ID Token")
		return user, fmt.Errorf("unable to verify id_token: %s", err.Error())
	}

	claims := make(map[string]interface{})
	if err := idToken.Claims(&claims); err != nil {
		log.Debug().Err(err).Msg("Failed to parse Apple ID Token claims")
		return user, fmt.Errorf("failed to parse claims: %s", err.Error())
	}

	if val, ok := claims["email"]; !ok || val == nil {
		log.Debug().Msg("Failed to extract email from claims.")
		return user, fmt.Errorf("unable to extract email, please check the scopes enabled for your app. It needs `email`, `name` scopes")
	} else {
		email, _ := val.(string)
		user.Email = &email
	}

	user.GivenName = &user_.Name.FirstName
	user.FamilyName = &user_.Name.LastName

	return user, nil
}

func (h *httpProvider) processDiscordUserInfo(ctx *gin.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processDiscordUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodDiscord)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, fmt.Errorf("error getting oauth config: %s", err.Error())
	}
	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, fmt.Errorf("invalid discord exchange code: %s", err.Error())
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", constants.DiscordUserInfoURL, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create Discord user info request")
		return nil, fmt.Errorf("error creating Discord user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", oauth2Token.AccessToken)},
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to request Discord user info")
		return nil, err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read Discord user info response body")
		return nil, fmt.Errorf("failed to read Discord response body: %s", err.Error())
	}

	if response.StatusCode >= 400 {
		log.Debug().Err(err).Msg("Failed to request Discord user info")
		return nil, fmt.Errorf("failed to request Discord user info: %s", string(body))
	}

	// Unmarshal the response body into a map
	responseRawData := make(map[string]interface{})
	if err := json.Unmarshal(body, &responseRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal Discord response")
		return nil, fmt.Errorf("failed to unmarshal Discord response: %s", err.Error())
	}

	// Safely extract the user data
	userRawData, ok := responseRawData["user"].(map[string]interface{})
	if !ok {
		log.Debug().Err(err).Msg("User data is not in expected format or missing in response")
		return nil, fmt.Errorf("user data is not in expected format or missing in response")
	}

	// Extract the username
	firstName, ok := userRawData["username"].(string)
	if !ok {
		log.Debug().Err(err).Msg("Username is not in expected format or missing in user data")
		return nil, fmt.Errorf("username is not in expected format or missing in user data")
	}
	discordID, _ := userRawData["id"].(string)
	avatar, _ := userRawData["avatar"].(string)
	profilePicture := fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", discordID, avatar)

	user := &schemas.User{
		GivenName: &firstName,
		Picture:   &profilePicture,
	}

	return user, nil
}

func (h *httpProvider) processTwitterUserInfo(ctx *gin.Context, code, verifier string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processTwitterUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodTwitter)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, fmt.Errorf("error getting oauth config: %s", err.Error())
	}

	oauth2Token, err := cfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, fmt.Errorf("invalid twitter exchange code: %s", err.Error())
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", constants.TwitterUserInfoURL, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create Twitter user info request")
		return nil, fmt.Errorf("error creating Twitter user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", oauth2Token.AccessToken)},
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to request Twitter user info")
		return nil, err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read Twitter user info response body")
		return nil, fmt.Errorf("failed to read Twitter response body: %s", err.Error())
	}

	if response.StatusCode >= 400 {
		log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request Twitter user info")
		return nil, fmt.Errorf("failed to request Twitter user info: %s", string(body))
	}

	responseRawData := make(map[string]interface{})
	if err := json.Unmarshal(body, &responseRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal twitter user info")
		return nil, fmt.Errorf("failed to parse twitter user info: %s", err.Error())
	}

	userRawData, ok := responseRawData["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("twitter response missing data field")
	}

	// Twitter API does not return E-Mail adresses by default. For that case special privileges have
	// to be granted on a per-App basis. See https://developer.twitter.com/en/docs/twitter-api/v1/accounts-and-users/manage-account-settings/api-reference/get-account-verify_credentials

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

	user := &schemas.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &profilePicture,
		Nickname:   &nickname,
	}

	return user, nil
}

// process microsoft user information
func (h *httpProvider) processMicrosoftUserInfo(ctx *gin.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processMicrosoftUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodMicrosoft)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, fmt.Errorf("error getting oauth config: %s", err.Error())
	}
	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, fmt.Errorf("invalid microsoft exchange code: %s", err.Error())
	}
	oidcProvider, err := oidc.NewProvider(ctx, fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", h.MicrosoftTenantID))
	// we need to skip issuer check because for common tenant it will return internal issuer which does not match
	verifier := oidcProvider.Verifier(&oidc.Config{
		ClientID:        h.MicrosoftClientID,
		SkipIssuerCheck: true,
	})
	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Debug().Err(err).Msg("Failed to extract ID Token from OAuth2 token")
		return nil, fmt.Errorf("unable to extract id_token")
	}
	// Parse and verify ID Token payload.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify ID Token")
		return nil, fmt.Errorf("unable to verify id_token: %s", err.Error())
	}
	user := &schemas.User{}
	if err := idToken.Claims(&user); err != nil {
		log.Debug().Err(err).Msg("Failed to parse ID Token claims")
		return nil, fmt.Errorf("unable to extract claims")
	}

	return user, nil
}

// process twitch user information
func (h *httpProvider) processTwitchUserInfo(ctx *gin.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processTwitchUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodTwitch)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, fmt.Errorf("error getting oauth config: %s", err.Error())
	}

	oauth2Token, err := cfg.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, fmt.Errorf("invalid twitch exchange code: %s", err.Error())
	}

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Debug().Err(err).Msg("Failed to extract ID Token from OAuth2 token")
		return nil, fmt.Errorf("unable to extract id_token")
	}
	oidcProvider, err := oidc.NewProvider(ctx, "https://id.twitch.tv/oauth2")
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create OIDC provider")
		return nil, fmt.Errorf("failed to create oidc provider: %s", err.Error())
	}
	verifier := oidcProvider.Verifier(&oidc.Config{
		ClientID:        h.TwitchClientID,
		SkipIssuerCheck: true,
	})

	// Parse and verify ID Token payload.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify ID Token")
		return nil, fmt.Errorf("unable to verify id_token: %s", err.Error())
	}

	user := &schemas.User{}
	if err := idToken.Claims(&user); err != nil {
		log.Debug().Err(err).Msg("Failed to parse ID Token claims")
		return nil, fmt.Errorf("unable to extract claims")
	}

	return user, nil
}

// process roblox user information
func (h *httpProvider) processRobloxUserInfo(ctx *gin.Context, code, verifier string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processRobloxUserInfo").Logger()
	cfg, err := h.OAuthProvider.GetOAuthConfig(ctx, constants.AuthRecipeMethodRoblox)
	if err != nil {
		log.Debug().Err(err).Msg("Error getting oauth config")
		return nil, fmt.Errorf("error getting oauth config: %s", err.Error())
	}
	oauth2Token, err := cfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, fmt.Errorf("invalid roblox exchange code: %s", err.Error())
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", constants.RobloxUserInfoURL, nil)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create roblox user info request")
		return nil, fmt.Errorf("error creating roblox user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer %s", oauth2Token.AccessToken)},
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to request roblox user info")
		return nil, err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read roblox user info response body")
		return nil, fmt.Errorf("failed to read roblox response body: %s", err.Error())
	}

	if response.StatusCode >= 400 {
		log.Debug().Err(err).Str("body", string(body)).Msg("Failed to request roblox user info")
		return nil, fmt.Errorf("failed to request roblox user info: %s", string(body))
	}

	userRawData := make(map[string]interface{})
	if err := json.Unmarshal(body, &userRawData); err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal roblox user info")
		return nil, fmt.Errorf("failed to parse roblox user info: %s", err.Error())
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
	email := ""
	if val, ok := userRawData["email"].(string); ok && val != "" {
		email = val
	} else if sub, ok := userRawData["sub"].(string); ok {
		email = sub
	}
	user := &schemas.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &profilePicture,
		Nickname:   &nickname,
		Email:      &email,
	}

	return user, nil
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
