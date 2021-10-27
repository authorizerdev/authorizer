package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func processGoogleUserInfo(code string) (db.User, error) {
	user := db.User{}
	token, err := oauth.OAuthProvider.GoogleConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return user, fmt.Errorf("invalid google exchange code: %s", err.Error())
	}
	client := oauth.OAuthProvider.GoogleConfig.Client(oauth2.NoContext, token)
	response, err := client.Get(constants.GoogleUserInfoURL)
	if err != nil {
		return user, err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return user, fmt.Errorf("failed to read google response body: %s", err.Error())
	}

	userRawData := make(map[string]string)
	json.Unmarshal(body, &userRawData)

	user = db.User{
		FirstName:       userRawData["given_name"],
		LastName:        userRawData["family_name"],
		Image:           userRawData["picture"],
		Email:           userRawData["email"],
		EmailVerifiedAt: time.Now().Unix(),
	}

	return user, nil
}

func processGithubUserInfo(code string) (db.User, error) {
	user := db.User{}
	token, err := oauth.OAuthProvider.GithubConfig.Exchange(oauth2.NoContext, code)
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

func processFacebookUserInfo(code string) (db.User, error) {
	user := db.User{}
	token, err := oauth.OAuthProvider.FacebookConfig.Exchange(oauth2.NoContext, code)
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
		log.Println("err:", err)
		return user, err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return user, fmt.Errorf("failed to read facebook response body: %s", err.Error())
	}

	userRawData := make(map[string]interface{})
	json.Unmarshal(body, &userRawData)

	email := fmt.Sprintf("%v", userRawData["email"])

	picObject := userRawData["picture"].(map[string]interface{})["data"]
	picDataObject := picObject.(map[string]interface{})
	user = db.User{
		FirstName:       fmt.Sprintf("%v", userRawData["first_name"]),
		LastName:        fmt.Sprintf("%v", userRawData["last_name"]),
		Image:           fmt.Sprintf("%v", picDataObject["url"]),
		Email:           email,
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
		user := db.User{}
		code := c.Request.FormValue("code")
		switch provider {
		case enum.Google.String():
			user, err = processGoogleUserInfo(code)
		case enum.Github.String():
			user, err = processGithubUserInfo(code)
		case enum.Facebook.String():
			user, err = processFacebookUserInfo(code)
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
				signupMethod = signupMethod + "," + enum.Github.String()
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
