package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// OAuthCallbackHandler handles the OAuth callback for various oauth providers
func OAuthCallbackHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("oauth_provider")
		state := c.Request.FormValue("state")

		sessionState := memorystore.Provider.GetState(state)
		if sessionState == "" {
			log.Debug("Invalid oauth state: ", state)
			c.JSON(400, gin.H{"error": "invalid oauth state"})
		}
		memorystore.Provider.GetState(state)
		// contains random token, redirect url, role
		sessionSplit := strings.Split(state, "___")

		if len(sessionSplit) < 3 {
			log.Debug("Unable to get redirect url from state: ", state)
			c.JSON(400, gin.H{"error": "invalid redirect url"})
			return
		}

		stateValue := sessionSplit[0]
		redirectURL := sessionSplit[1]
		inputRoles := strings.Split(sessionSplit[2], ",")
		scopes := strings.Split(sessionSplit[3], ",")

		var err error
		user := models.User{}
		code := c.Request.FormValue("code")
		switch provider {
		case constants.SignupMethodGoogle:
			user, err = processGoogleUserInfo(code)
		case constants.SignupMethodGithub:
			user, err = processGithubUserInfo(code)
		case constants.SignupMethodFacebook:
			user, err = processFacebookUserInfo(code)
		default:
			log.Info("Invalid oauth provider")
			err = fmt.Errorf(`invalid oauth provider`)
		}

		if err != nil {
			log.Debug("Failed to process user info: ", err)
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		existingUser, err := db.Provider.GetUserByEmail(user.Email)
		log := log.WithField("user", user.Email)

		if err != nil {
			if envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableSignUp) {
				log.Debug("Failed to signup as disabled")
				c.JSON(400, gin.H{"error": "signup is disabled for this instance"})
				return
			}
			// user not registered, register user and generate session token
			user.SignupMethods = provider
			// make sure inputRoles don't include protected roles
			hasProtectedRole := false
			for _, ir := range inputRoles {
				if utils.StringSliceContains(envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyProtectedRoles), ir) {
					hasProtectedRole = true
				}
			}

			if hasProtectedRole {
				log.Debug("Signup is not allowed with protected roles:", inputRoles)
				c.JSON(400, gin.H{"error": "invalid role"})
				return
			}

			user.Roles = strings.Join(inputRoles, ",")
			now := time.Now().Unix()
			user.EmailVerifiedAt = &now
			user, _ = db.Provider.AddUser(user)
		} else {
			if user.RevokedTimestamp != nil {
				log.Debug("User access revoked at: ", user.RevokedTimestamp)
				c.JSON(400, gin.H{"error": "user access has been revoked"})
			}

			// user exists in db, check if method was google
			// if not append google to existing signup method and save it
			signupMethod := existingUser.SignupMethods
			if !strings.Contains(signupMethod, provider) {
				signupMethod = signupMethod + "," + provider
			}
			user = existingUser
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
					if utils.StringSliceContains(envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyProtectedRoles), ur) {
						hasProtectedRole = true
					}
				}

				if hasProtectedRole {
					log.Debug("Invalid role. User is using protected unassigned role")
					c.JSON(400, gin.H{"error": "invalid role"})
					return
				} else {
					user.Roles = existingUser.Roles + "," + strings.Join(unasignedRoles, ",")
				}
			} else {
				user.Roles = existingUser.Roles
			}

			user, err = db.Provider.UpdateUser(user)
			if err != nil {
				log.Debug("Failed to update user: ", err)
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}

		authToken, err := token.CreateAuthToken(c, user, inputRoles, scopes)
		if err != nil {
			log.Debug("Failed to create auth token: ", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}

		expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
		if expiresIn <= 0 {
			expiresIn = 1
		}

		params := "access_token=" + authToken.AccessToken.Token + "&token_type=bearer&expires_in=" + strconv.FormatInt(expiresIn, 10) + "&state=" + stateValue + "&id_token=" + authToken.IDToken.Token

		cookie.SetSession(c, authToken.FingerPrintHash)
		memorystore.Provider.SetState(authToken.FingerPrintHash, authToken.FingerPrint+"@"+user.ID)
		memorystore.Provider.SetState(authToken.AccessToken.Token, authToken.FingerPrint+"@"+user.ID)

		if authToken.RefreshToken != nil {
			params = params + `&refresh_token=` + authToken.RefreshToken.Token
			memorystore.Provider.SetState(authToken.RefreshToken.Token, authToken.FingerPrint+"@"+user.ID)
		}

		go db.Provider.AddSession(models.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(c.Request),
			IP:        utils.GetIP(c.Request),
		})
		if strings.Contains(redirectURL, "?") {
			redirectURL = redirectURL + "&" + params
		} else {
			redirectURL = redirectURL + "?" + params
		}

		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}

func processGoogleUserInfo(code string) (models.User, error) {
	user := models.User{}
	ctx := context.Background()
	oauth2Token, err := oauth.OAuthProviders.GoogleConfig.Exchange(ctx, code)
	if err != nil {
		log.Debug("Failed to exchange code for token: ", err)
		return user, fmt.Errorf("invalid google exchange code: %s", err.Error())
	}

	verifier := oauth.OIDCProviders.GoogleOIDC.Verifier(&oidc.Config{ClientID: oauth.OAuthProviders.GoogleConfig.ClientID})

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Debug("Failed to extract ID Token from OAuth2 token")
		return user, fmt.Errorf("unable to extract id_token")
	}

	// Parse and verify ID Token payload.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Debug("Failed to verify ID Token: ", err)
		return user, fmt.Errorf("unable to verify id_token: %s", err.Error())
	}

	if err := idToken.Claims(&user); err != nil {
		log.Debug("Failed to parse ID Token claims: ", err)
		return user, fmt.Errorf("unable to extract claims")
	}

	return user, nil
}

func processGithubUserInfo(code string) (models.User, error) {
	user := models.User{}
	token, err := oauth.OAuthProviders.GithubConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Debug("Failed to exchange code for token: ", err)
		return user, fmt.Errorf("invalid github exchange code: %s", err.Error())
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", constants.GithubUserInfoURL, nil)
	if err != nil {
		log.Debug("Failed to create github user info request: ", err)
		return user, fmt.Errorf("error creating github user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("token %s", token.AccessToken)},
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug("Failed to request github user info: ", err)
		return user, err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Debug("Failed to read github user info response body: ", err)
		return user, fmt.Errorf("failed to read github response body: %s", err.Error())
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

	user = models.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &picture,
		Email:      userRawData["email"],
	}

	return user, nil
}

func processFacebookUserInfo(code string) (models.User, error) {
	user := models.User{}
	token, err := oauth.OAuthProviders.FacebookConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Debug("Invalid facebook exchange code: ", err)
		return user, fmt.Errorf("invalid facebook exchange code: %s", err.Error())
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", constants.FacebookUserInfoURL+token.AccessToken, nil)
	if err != nil {
		log.Debug("Error creating facebook user info request: ", err)
		return user, fmt.Errorf("error creating facebook user info request: %s", err.Error())
	}

	response, err := client.Do(req)
	if err != nil {
		log.Debug("Failed to process facebook user: ", err)
		return user, err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Debug("Failed to read facebook response: ", err)
		return user, fmt.Errorf("failed to read facebook response body: %s", err.Error())
	}

	userRawData := make(map[string]interface{})
	json.Unmarshal(body, &userRawData)

	email := fmt.Sprintf("%v", userRawData["sub"])

	picObject := userRawData["picture"].(map[string]interface{})["data"]
	picDataObject := picObject.(map[string]interface{})
	firstName := fmt.Sprintf("%v", userRawData["first_name"])
	lastName := fmt.Sprintf("%v", userRawData["last_name"])
	picture := fmt.Sprintf("%v", picDataObject["url"])

	user = models.User{
		GivenName:  &firstName,
		FamilyName: &lastName,
		Picture:    &picture,
		Email:      email,
	}

	return user, nil
}
