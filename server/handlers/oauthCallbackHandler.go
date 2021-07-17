package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yauthdev/yauth/server/constants"
	"github.com/yauthdev/yauth/server/db"
	"github.com/yauthdev/yauth/server/enum"
	"github.com/yauthdev/yauth/server/oauth"
	"github.com/yauthdev/yauth/server/session"
	"github.com/yauthdev/yauth/server/utils"
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
	user := db.User{}
	if err != nil {
		// user not registered, register user and generate session token
		user = db.User{
			FirstName:       userRawData["given_name"],
			LastName:        userRawData["family_name"],
			Image:           userRawData["picture"],
			Email:           userRawData["email"],
			EmailVerifiedAt: time.Now().Unix(),
			SignupMethod:    enum.Google.String(),
		}

		user, _ = db.Mgr.SaveUser(user)
	} else {
		// user exists in db, check if method was google
		// if not append google to existing signup method and save it

		signupMethod := existingUser.SignupMethod
		if !strings.Contains(signupMethod, enum.Google.String()) {
			signupMethod += signupMethod + "," + enum.Google.String()
		}
		user = db.User{
			FirstName:       userRawData["given_name"],
			LastName:        userRawData["family_name"],
			Image:           userRawData["picture"],
			Email:           userRawData["email"],
			EmailVerifiedAt: time.Now().Unix(),
			SignupMethod:    signupMethod,
			Password:        existingUser.Password,
		}

		user, _ = db.Mgr.SaveUser(user)

	}

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

func HandleOAuthCallback(provider enum.OAuthProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		if provider == enum.GoogleProvider {
			err := processGoogleUserInfo(c.Request.FormValue("state"), c.Request.FormValue("code"), c)
			if err != nil {
				c.Redirect(http.StatusTemporaryRedirect, constants.FRONTEND_URL+"?error="+err.Error())
			}

			c.Redirect(http.StatusTemporaryRedirect, constants.FRONTEND_URL)
		}
	}
}
