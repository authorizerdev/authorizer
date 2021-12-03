package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// openID providers has common claims so single helper can work for facebook + google
func processOpenIDProvider(code string, oauth2Config *oauth2.Config, oidcProvider *oidc.Provider) (db.User, error) {
	user := db.User{}
	ctx := context.Background()
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		return user, fmt.Errorf("invalid exchange code: %s", err.Error())
	}

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return user, fmt.Errorf("error getting id_token")
	}

	verifier := oidcProvider.Verifier(&oidc.Config{ClientID: oauth2Config.ClientID})

	// Parse and verify ID Token payload.
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return user, fmt.Errorf("error verifying OpenId token: %s", err.Error())
	}

	var claims struct {
		Email      string `json:"email"`
		Picture    string `json:"picture"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return user, fmt.Errorf("error parsing OpenId claims: %s", err.Error())
	}

	user = db.User{
		FirstName:       claims.GivenName,
		LastName:        claims.FamilyName,
		Image:           claims.Picture,
		Email:           claims.Email,
		EmailVerifiedAt: time.Now().Unix(),
	}

	return user, nil
}

func processGithubUserInfo(code string) (db.User, error) {
	user := db.User{}
	ctx := context.Background()
	token, err := oauth.OAuthProviders.GithubConfig.Exchange(ctx, code)
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
	user = db.User{
		FirstName:       firstName,
		LastName:        lastName,
		Image:           userRawData["avatar_url"],
		Email:           userRawData["email"],
		EmailVerifiedAt: time.Now().Unix(),
	}

	return user, nil
}

func OAuthCallbackHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := c.Param("oauth_provider")
		state := c.Request.FormValue("state")

		sessionState := session.GetSocailLoginState(state)
		if sessionState == "" {
			c.JSON(400, gin.H{"error": "invalid oauth state"})
		}
		session.RemoveSocialLoginState(state)
		// contains random token, redirect url, role
		sessionSplit := strings.Split(state, "___")

		// TODO validate redirect url
		if len(sessionSplit) < 2 {
			c.JSON(400, gin.H{"error": "invalid redirect url"})
			return
		}

		inputRoles := strings.Split(sessionSplit[2], ",")
		redirectURL := sessionSplit[1]

		var err error
		var oidcProvider *oidc.Provider
		var oauth2Config *oauth2.Config
		user := db.User{}
		code := c.Request.FormValue("code")

		switch provider {
		case enum.Google.String():
			oauth2Config = oauth.OAuthProviders.GoogleConfig
			oidcProvider = oauth.OIDCProviders.GoogleOIDC
			user, err = processOpenIDProvider(code, oauth2Config, oidcProvider)
		case enum.Github.String():
			user, err = processGithubUserInfo(code)
		case enum.Facebook.String():
			oauth2Config = oauth.OAuthProviders.FacebookConfig
			oidcProvider = oauth.OIDCProviders.FacebookOIDC
			user, err = processOpenIDProvider(code, oauth2Config, oidcProvider)
		default:
			err = fmt.Errorf(`invalid oauth provider`)
		}

		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		existingUser, err := db.Mgr.GetUserByEmail(user.Email)

		if err != nil {
			// user not registered, register user and generate session token
			user.SignupMethod = provider
			// make sure inputRoles don't include protected roles
			hasProtectedRole := false
			for _, ir := range inputRoles {
				if utils.StringSliceContains(constants.PROTECTED_ROLES, ir) {
					hasProtectedRole = true
				}
			}

			if hasProtectedRole {
				c.JSON(400, gin.H{"error": "invalid role"})
				return
			}

			user.Roles = strings.Join(inputRoles, ",")
		} else {
			// user exists in db, check if method was google
			// if not append google to existing signup method and save it

			signupMethod := existingUser.SignupMethod
			if !strings.Contains(signupMethod, provider) {
				signupMethod = signupMethod + "," + provider
			}
			user.SignupMethod = signupMethod
			user.Password = existingUser.Password

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
					if utils.StringSliceContains(constants.PROTECTED_ROLES, ur) {
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
		}

		user, _ = db.Mgr.SaveUser(user)
		user, _ = db.Mgr.GetUserByEmail(user.Email)
		userIdStr := fmt.Sprintf("%v", user.ID)
		refreshToken, _, _ := utils.CreateAuthToken(user, enum.RefreshToken, inputRoles)

		accessToken, _, _ := utils.CreateAuthToken(user, enum.AccessToken, inputRoles)
		utils.SetCookie(c, accessToken)
		session.SetToken(userIdStr, accessToken, refreshToken)
		go func() {
			sessionData := db.Session{
				UserID:    user.ID,
				UserAgent: utils.GetUserAgent(c.Request),
				IP:        utils.GetIP(c.Request),
			}

			db.Mgr.SaveSession(sessionData)
		}()

		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}
