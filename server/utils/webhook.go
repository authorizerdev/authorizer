package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	log "github.com/sirupsen/logrus"
)

func RegisterEvent(ctx context.Context, eventName string, authRecipe string, user models.User) error {
	webhook, err := db.Provider.GetWebhookByEventName(ctx, eventName)
	if err != nil {
		return err
	}

	userBytes, err := json.Marshal(user)
	if err != nil {
		log.Debug("error marshalling user obj: ", err)
		return err
	}
	userMap := map[string]interface{}{}
	err = json.Unmarshal(userBytes, &userMap)
	if err != nil {
		log.Debug("error un-marshalling user obj: ", err)
		return err
	}

	reqBody := map[string]interface{}{
		"event_name": eventName,
		"user":       userMap,
	}

	if eventName == constants.UserLoginWebhookEvent || eventName == constants.UserSignUpWebhookEvent {
		reqBody["auth_recipe"] = authRecipe
	}

	requestBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Debug("error marshalling requestBody obj: ", err)
		return err
	}

	requestBytesBuffer := bytes.NewBuffer(requestBody)
	req, err := http.NewRequest("POST", StringValue(webhook.Endpoint), requestBytesBuffer)
	if err != nil {
		log.Debug("error creating webhook post request: ", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if webhook.Headers != nil {
		for key, val := range webhook.Headers {
			req.Header.Set(key, val.(string))
		}
	}

	client := &http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		log.Debug("error making request: ", err)
		return err
	}
	defer resp.Body.Close()

	responseBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Debug("error reading response: ", err)
		return err
	}

	statusCode := int64(resp.StatusCode)
	_, err = db.Provider.AddWebhookLog(ctx, models.WebhookLog{
		HttpStatus: statusCode,
		Request:    string(requestBytesBuffer.Bytes()),
		Response:   string(responseBytes),
		WebhookID:  webhook.ID,
	})

	if err != nil {
		log.Debug("failed to add webhook log: ", err)
		return err
	}

	return nil
}
