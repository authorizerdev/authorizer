package handlers

import (
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
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func processGoogleUserInfo(state string, code string, c *gin.Context) error {
	sessionState := session.GetToken(state)
	if sessionState == "" {
		return fmt.Errorf("invalid oauth state")
	}
	session.DeleteToken(sessionState)
	token, err := oauth.OAuthProvider.GoogleConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return fmt.Errorf("invalid google exchange code: %s", err.Error())
	}
	client := oauth.OAuthProvider.GoogleConfig.Client(oauth2.NoContext, token)
	response, err := client.Get(constants.GoogleUserInfoURL)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read google response body: %s", err.Error())
	}

	userRawData := make(map[string]string)
	json.Unmarshal(body, &userRawData)

	existingUser, err := db.Mgr.GetUserByEmail(userRawData["email"])
	user := db.User{
		FirstName:       userRawData["given_name"],
		LastName:        userRawData["family_name"],
		Image:           userRawData["picture"],
		Email:           userRawData["email"],
		EmailVerifiedAt: time.Now().Unix(),
	}
	if err != nil {
		// user not registered, register user and generate session token
		user.SignupMethod = enum.Google.String()
	} else {
		// user exists in db, check if method was google
		// if not append google to existing signup method and save it

		signupMethod := existingUser.SignupMethod
		if !strings.Contains(signupMethod, enum.Google.String()) {
			signupMethod = signupMethod + "," + enum.Google.String()
		}
		user.SignupMethod = signupMethod
		user.Password = existingUser.Password
	}

	user, _ = db.Mgr.SaveUser(user)
	userIdStr := fmt.Sprintf("%d", user.ID)
	refreshToken, _, _ := utils.CreateAuthToken(utils.UserAuthInfo{
		ID:    userIdStr,
		Email: user.Email,
	}, enum.RefreshToken)

	accessToken, _, _ := utils.CreateAuthToken(utils.UserAuthInfo{
		ID:    userIdStr,
		Email: user.Email,
	}, enum.AccessToken)
	utils.SetCookie(c, accessToken)
	session.SetToken(userIdStr, refreshToken)
	return nil
}

func processGithubUserInfo(state string, code string, c *gin.Context) error {
	sessionState := session.GetToken(state)
	if sessionState == "" {
		return fmt.Errorf("invalid oauth state")
	}
	session.DeleteToken(sessionState)
	token, err := oauth.OAuthProvider.GithubConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return fmt.Errorf("invalid google exchange code: %s", err.Error())
	}
	client := http.Client{}
	req, err := http.NewRequest("GET", constants.GithubUserInfoURL, nil)
	if err != nil {
		return fmt.Errorf("error creating github user info request: %s", err.Error())
	}
	req.Header = http.Header{
		"Authorization": []string{fmt.Sprintf("token %s", token.AccessToken)},
	}

	response, err := client.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read google response body: %s", err.Error())
	}

	userRawData := make(map[string]string)
	json.Unmarshal(body, &userRawData)

	existingUser, err := db.Mgr.GetUserByEmail(userRawData["email"])
	name := strings.Split(userRawData["name"], " ")
	firstName := ""
	lastName := ""
	if len(name) >= 1 && strings.TrimSpace(name[0]) != "" {
		firstName = name[0]
	}
	if len(name) > 1 && strings.TrimSpace(name[1]) != "" {
		lastName = name[0]
	}
	user := db.User{
		FirstName:       firstName,
		LastName:        lastName,
		Image:           userRawData["avatar_url"],
		Email:           userRawData["email"],
		EmailVerifiedAt: time.Now().Unix(),
	}
	if err != nil {
		// user not registered, register user and generate session token
		user.SignupMethod = enum.Github.String()
	} else {
		// user exists in db, check if method was google
		// if not append google to existing signup method and save it

		signupMethod := existingUser.SignupMethod
		if !strings.Contains(signupMethod, enum.Github.String()) {
			signupMethod = signupMethod + "," + enum.Github.String()
		}
		user.SignupMethod = signupMethod
		user.Password = existingUser.Password
	}

	user, _ = db.Mgr.SaveUser(user)
	userIdStr := fmt.Sprintf("%d", user.ID)
	refreshToken, _, _ := utils.CreateAuthToken(utils.UserAuthInfo{
		ID:    userIdStr,
		Email: user.Email,
	}, enum.RefreshToken)

	accessToken, _, _ := utils.CreateAuthToken(utils.UserAuthInfo{
		ID:    userIdStr,
		Email: user.Email,
	}, enum.AccessToken)
	utils.SetCookie(c, accessToken)
	session.SetToken(userIdStr, refreshToken)
	return nil
}

func OAuthCallbackHandler(provider enum.OAuthProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		var err error
		if provider == enum.GoogleProvider {
			err = processGoogleUserInfo(c.Request.FormValue("state"), c.Request.FormValue("code"), c)
		}
		if provider == enum.GithubProvider {
			err = processGithubUserInfo(c.Request.FormValue("state"), c.Request.FormValue("code"), c)
		}

		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, constants.FRONTEND_URL+"?error="+err.Error())
		}

		c.Redirect(http.StatusTemporaryRedirect, constants.FRONTEND_URL)
	}
}
