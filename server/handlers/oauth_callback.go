package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// OAuthCallbackHandler handles the OAuth callback for various oauth providers
func OAuthCallbackHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("oauth_provider")
		state := c.Request.FormValue("state")

		sessionState := sessionstore.GetState(state)
		if sessionState == "" {
			c.JSON(400, gin.H{"error": "invalid oauth state"})
		}
		sessionstore.GetState(state)
		// contains random token, redirect url, role
		sessionSplit := strings.Split(state, "___")

		if len(sessionSplit) < 3 {
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
			err = fmt.Errorf(`invalid oauth provider`)
		}

		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		existingUser, err := db.Provider.GetUserByEmail(user.Email)

		if err != nil {
			if envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableSignUp) {
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
				c.JSON(400, gin.H{"error": "invalid role"})
				return
			}

			user.Roles = strings.Join(inputRoles, ",")
			now := time.Now().Unix()
			user.EmailVerifiedAt = &now
			user, _ = db.Provider.AddUser(user)
		} else {
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
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}

		authToken, err := token.CreateAuthToken(c, user, inputRoles, scopes)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		expiresIn := int64(1800)
		params := "access_token=" + authToken.AccessToken.Token + "&token_type=bearer&expires_in=" + strconv.FormatInt(expiresIn, 10) + "&state=" + stateValue + "&id_token=" + authToken.IDToken.Token

		cookie.SetSession(c, authToken.FingerPrintHash)
		sessionstore.SetState(authToken.FingerPrintHash, authToken.FingerPrint+"@"+user.ID)
		sessionstore.SetState(authToken.AccessToken.Token, authToken.FingerPrint+"@"+user.ID)

		if authToken.RefreshToken != nil {
			params = params + `&refresh_token=` + authToken.RefreshToken.Token
			sessionstore.SetState(authToken.RefreshToken.Token, authToken.FingerPrint+"@"+user.ID)
		}

		go utils.SaveSessionInDB(c, user.ID)
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
		return user, fmt.Errorf("invalid google exchange code: %s", err.Error())
	}

	verifier := oauth.OIDCProviders.GoogleOIDC.Verifier(&oidc.Config{ClientID: oauth.OAuthProviders.GoogleConfig.ClientID})

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return user, fmt.Errorf("unable to extract id_token")
	}

	// Parse and verify ID Token payload.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return user, fmt.Errorf("unable to verify id_token: %s", err.Error())
	}

	if err := idToken.Claims(&user); err != nil {
		return user, fmt.Errorf("unable to extract claims")
	}

	return user, nil
}

func processGithubUserInfo(code string) (models.User, error) {
	user := models.User{}
	token, err := oauth.OAuthProviders.GithubConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return user, fmt.Errorf("invalid github exchange code: %s", err.Error())
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", constants.GithubUserInfoURL, nil)
	if err != nil {
		return user, fmt.Errorf("error creating github user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("token %s", token.AccessToken)},
	}

	response, err := client.Do(req)
	if err != nil {
		return user, err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
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
		Email:      userRawData["sub"],
	}

	return user, nil
}

func processFacebookUserInfo(code string) (models.User, error) {
	user := models.User{}
	token, err := oauth.OAuthProviders.FacebookConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return user, fmt.Errorf("invalid facebook exchange code: %s", err.Error())
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", constants.FacebookUserInfoURL+token.AccessToken, nil)
	if err != nil {
		return user, fmt.Errorf("error creating facebook user info request: %s", err.Error())
	}

	response, err := client.Do(req)
	if err != nil {
		log.Println("error processing facebook user info:", err)
		return user, err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
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
