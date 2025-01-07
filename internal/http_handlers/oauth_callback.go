package http_handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/oauth"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
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
		// contains random token, redirect url, role
		sessionSplit := strings.Split(state, "___")

		if len(sessionSplit) < 3 {
			log.Debug().Err(err).Msg("Invalid state")
			ctx.JSON(400, gin.H{"error": "invalid redirect url"})
			return
		}
		// remove state from store
		go h.MemoryStoreProvider.RemoveState(state)
		stateValue := sessionSplit[0]
		redirectURL := sessionSplit[1]
		inputRoles := strings.Split(sessionSplit[2], ",")
		scopeString := sessionSplit[3]
		scopes := []string{}
		if scopeString != "" {
			if strings.Contains(scopeString, ",") {
				scopes = strings.Split(scopeString, ",")
			}
			if strings.Contains(scopeString, " ") {
				scopes = strings.Split(scopeString, " ")
			}
		}
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
			ctx.JSON(400, gin.H{"error": err.Error()})
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
			isSignupDisabled := h.Config.DisableSignup
			if isSignupDisabled {
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
				ctx.JSON(500, gin.H{"error": err.Error()})
				return
			}
			isSignUp = true
		} else {
			user = existingUser
			if user.RevokedTimestamp != nil {
				log.Debug().Msg("User access has been revoked")
				ctx.JSON(400, gin.H{"error": "user access has been revoked"})
				return
			}

			// user exists in db, check if method was google
			// if not append google to existing signup method and save it
			signupMethod := existingUser.SignupMethods
			if !strings.Contains(signupMethod, provider) {
				signupMethod = signupMethod + "," + provider
			}
			user.SignupMethods = signupMethod

			if user.EmailVerifiedAt == nil {
				now := time.Now().Unix()
				user.EmailVerifiedAt = &now
			}

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
				ctx.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}

		// TODO
		// use stateValue to get code / nonce
		// add code / nonce to id_token
		code := ""
		codeChallenge := ""
		nonce := ""
		if stateValue != "" {
			// Get state from store
			authorizeState, _ := h.MemoryStoreProvider.GetState(stateValue)
			if authorizeState != "" {
				authorizeStateSplit := strings.Split(authorizeState, "@@")
				if len(authorizeStateSplit) > 1 {
					code = authorizeStateSplit[0]
					codeChallenge = authorizeStateSplit[1]
				} else {
					nonce = authorizeState
				}
				go h.MemoryStoreProvider.RemoveState(stateValue)
			}
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
		})
		if err != nil {
			log.Debug().Err(err).Msg("Failed to create auth token")
			ctx.JSON(500, gin.H{"error": err.Error()})
		}

		// Code challenge could be optional if PKCE flow is not used
		if code != "" {
			if err := h.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
				log.Debug().Err(err).Msg("Failed to set state")
				ctx.JSON(500, gin.H{"error": err.Error()})
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
		cookie.SetSession(ctx, authToken.FingerPrintHash)
		h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
		h.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

		if authToken.RefreshToken != nil {
			params += `&refresh_token=` + authToken.RefreshToken.Token
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

		ctx.Redirect(http.StatusFound, redirectURL)
	}
}

func (h *httpProvider) processGoogleUserInfo(ctx context.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processGoogleUserInfo").Logger()
	oauth2Token, err := oauth.OAuthProviders.GoogleConfig.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, fmt.Errorf("invalid google exchange code: %s", err.Error())
	}
	verifier := oauth.OIDCProviders.GoogleOIDC.Verifier(&oidc.Config{ClientID: oauth.OAuthProviders.GoogleConfig.ClientID})

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

func (h *httpProvider) processGithubUserInfo(ctx context.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processGithubUserInfo").Logger()
	oauth2Token, err := oauth.OAuthProviders.GithubConfig.Exchange(ctx, code)
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
	json.Unmarshal(body, &userRawData)

	name := strings.Split(userRawData["name"], " ")
	firstName := ""
	lastName := ""
	if len(name) >= 1 && strings.TrimSpace(name[0]) != "" {
		firstName = name[0]
	}
	if len(name) > 1 && strings.TrimSpace(name[1]) != "" {
		lastName = name[0]
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

func (h *httpProvider) processFacebookUserInfo(ctx context.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processFacebookUserInfo").Logger()
	oauth2Token, err := oauth.OAuthProviders.FacebookConfig.Exchange(ctx, code)
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
	json.Unmarshal(body, &userRawData)

	email := fmt.Sprintf("%v", userRawData["email"])

	picObject := userRawData["picture"].(map[string]interface{})["data"]
	picDataObject := picObject.(map[string]interface{})
	firstName := fmt.Sprintf("%v", userRawData["first_name"])
	lastName := fmt.Sprintf("%v", userRawData["last_name"])
	picture := fmt.Sprintf("%v", picDataObject["url"])

	user := &schemas.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &picture,
		Email:      &email,
	}

	return user, nil
}

func (h *httpProvider) processLinkedInUserInfo(ctx context.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processLinkedInUserInfo").Logger()
	oauth2Token, err := oauth.OAuthProviders.LinkedInConfig.Exchange(ctx, code)
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
	json.Unmarshal(body, &userRawData)

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
	json.Unmarshal(body, &emailRawData)

	firstName := userRawData["localizedFirstName"].(string)
	lastName := userRawData["localizedLastName"].(string)
	profilePicture := userRawData["profilePicture"].(map[string]interface{})["displayImage~"].(map[string]interface{})["elements"].([]interface{})[0].(map[string]interface{})["identifiers"].([]interface{})[0].(map[string]interface{})["identifier"].(string)
	emailAddress := emailRawData["elements"].([]interface{})[0].(map[string]interface{})["handle~"].(map[string]interface{})["emailAddress"].(string)

	user := &schemas.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &profilePicture,
		Email:      &emailAddress,
	}

	return user, nil
}

func (h *httpProvider) processAppleUserInfo(ctx context.Context, code string, user_ *AppleUserInfo) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processAppleUserInfo").Logger()
	var user = &schemas.User{}
	oauth2Token, err := oauth.OAuthProviders.AppleConfig.Exchange(ctx, code)
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

	tokenSplit := strings.Split(rawIDToken, ".")
	claimsData := tokenSplit[1]
	decodedClaimsData, err := base64.RawURLEncoding.DecodeString(claimsData)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to decode claims data")
		return user, fmt.Errorf("failed to decrypt claims data: %s", err.Error())
	}

	claims := make(map[string]interface{})
	err = json.Unmarshal(decodedClaimsData, &claims)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to unmarshal claims data")
		return user, fmt.Errorf("failed to unmarshal claims data: %s", err.Error())
	}
	if val, ok := claims["email"]; !ok || val == nil {
		log.Debug().Err(err).Msg("Failed to extract email from claims.")
		return user, fmt.Errorf("unable to extract email, please check the scopes enabled for your app. It needs `email`, `name` scopes")
	} else {
		email := val.(string)
		user.Email = &email
	}

	if val, ok := claims["name"]; ok {
		nameData := val.(map[string]interface{})
		if nameVal, ok := nameData["firstName"]; ok {
			givenName := nameVal.(string)
			user.GivenName = &givenName
		}

		if nameVal, ok := nameData["lastName"]; ok {
			familyName := nameVal.(string)
			user.FamilyName = &familyName
		}
	}
	user.GivenName = &user_.Name.FirstName
	user.FamilyName = &user_.Name.LastName

	return user, err
}

func (h *httpProvider) processDiscordUserInfo(ctx context.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processDiscordUserInfo").Logger()
	oauth2Token, err := oauth.OAuthProviders.DiscordConfig.Exchange(ctx, code)
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
	profilePicture := fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", userRawData["id"].(string), userRawData["avatar"].(string))

	user := &schemas.User{
		GivenName: &firstName,
		Picture:   &profilePicture,
	}

	return user, nil
}

func (h *httpProvider) processTwitterUserInfo(ctx context.Context, code, verifier string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processTwitterUserInfo").Logger()
	oauth2Token, err := oauth.OAuthProviders.TwitterConfig.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
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
	json.Unmarshal(body, &responseRawData)

	userRawData := responseRawData["data"].(map[string]interface{})

	// log.Info(userRawData)
	// Twitter API does not return E-Mail adresses by default. For that case special privileges have
	// to be granted on a per-App basis. See https://developer.twitter.com/en/docs/twitter-api/v1/accounts-and-users/manage-account-settings/api-reference/get-account-verify_credentials

	// Currently Twitter API only provides the full name of a user. To fill givenName and familyName
	// the full name will be split at the first whitespace. This approach will not be valid for all name combinations
	nameArr := strings.SplitAfterN(userRawData["name"].(string), " ", 2)

	firstName := nameArr[0]
	lastName := ""
	if len(nameArr) == 2 {
		lastName = nameArr[1]
	}
	nickname := userRawData["username"].(string)
	profilePicture := userRawData["profile_image_url"].(string)

	user := &schemas.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &profilePicture,
		Nickname:   &nickname,
	}

	return user, nil
}

// process microsoft user information
func (h *httpProvider) processMicrosoftUserInfo(ctx context.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processMicrosoftUserInfo").Logger()
	oauth2Token, err := oauth.OAuthProviders.MicrosoftConfig.Exchange(ctx, code)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to exchange code for token")
		return nil, fmt.Errorf("invalid microsoft exchange code: %s", err.Error())
	}
	// we need to skip issuer check because for common tenant it will return internal issuer which does not match
	verifier := oauth.OIDCProviders.MicrosoftOIDC.Verifier(&oidc.Config{
		ClientID:        oauth.OAuthProviders.MicrosoftConfig.ClientID,
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
func (h *httpProvider) processTwitchUserInfo(ctx context.Context, code string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processTwitchUserInfo").Logger()
	oauth2Token, err := oauth.OAuthProviders.TwitchConfig.Exchange(ctx, code)
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
	verifier := oauth.OIDCProviders.TwitchOIDC.Verifier(&oidc.Config{
		ClientID:        oauth.OAuthProviders.TwitchConfig.ClientID,
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
func (h *httpProvider) processRobloxUserInfo(ctx context.Context, code, verifier string) (*schemas.User, error) {
	log := h.Log.With().Str("func", "processRobloxUserInfo").Logger()
	oauth2Token, err := oauth.OAuthProviders.RobloxConfig.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
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
	json.Unmarshal(body, &userRawData)

	// log.Info(userRawData)
	nameArr := strings.SplitAfterN(userRawData["name"].(string), " ", 2)
	firstName := nameArr[0]
	lastName := ""
	if len(nameArr) == 2 {
		lastName = nameArr[1]
	}
	nickname := userRawData["nickname"].(string)
	profilePicture := userRawData["picture"].(string)
	email := ""
	if val, ok := userRawData["email"]; ok {
		email = val.(string)
	} else {
		email = userRawData["sub"].(string)
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
