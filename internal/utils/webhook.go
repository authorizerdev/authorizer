package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	log "github.com/sirupsen/logrus"
)

// RegisterEvent util to register event
// TODO change user to user ref
func RegisterEvent(ctx context.Context, eventName string, authRecipe string, user *models.User) error {
	webhooks, err := db.Provider.GetWebhookByEventName(ctx, eventName)
	if err != nil {
		log.Debug("error getting webhook: %v", err)
		return err
	}
	for _, webhook := range webhooks {
		if !refs.BoolValue(webhook.Enabled) {
			continue
		}
		userBytes, err := json.Marshal(user.AsAPIUser())
		if err != nil {
			log.Debug("error marshalling user obj: ", err)
			continue
		}
		userMap := map[string]interface{}{}
		err = json.Unmarshal(userBytes, &userMap)
		if err != nil {
			log.Debug("error un-marshalling user obj: ", err)
			continue
		}

		reqBody := map[string]interface{}{
			"webhook_id":        webhook.ID,
			"event_name":        eventName,
			"event_description": webhook.EventDescription,
			"user":              userMap,
		}

		if eventName == constants.UserLoginWebhookEvent || eventName == constants.UserSignUpWebhookEvent {
			reqBody["auth_recipe"] = authRecipe
		}

		requestBody, err := json.Marshal(reqBody)
		if err != nil {
			log.Debug("error marshalling requestBody obj: ", err)
			continue
		}

		// dont trigger webhook call in case of test
		envKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyEnv)
		if err != nil {
			continue
		}
		if envKey == constants.TestEnv {
			_, err := db.Provider.AddWebhookLog(ctx, &models.WebhookLog{
				HttpStatus: 200,
				Request:    string(requestBody),
				Response:   string(`{"message": "test"}`),
				WebhookID:  webhook.ID,
			})
			if err != nil {
				log.Debug("error saving webhook log:", err)
			}
			continue
		}

		requestBytesBuffer := bytes.NewBuffer(requestBody)
		req, err := http.NewRequest("POST", refs.StringValue(webhook.Endpoint), requestBytesBuffer)
		if err != nil {
			log.Debug("error creating webhook post request: ", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		for key, val := range webhook.Headers {
			req.Header.Set(key, val.(string))
		}

		client := &http.Client{Timeout: time.Second * 30}
		resp, err := client.Do(req)
		if err != nil {
			log.Debug("error making request: ", err)
			continue
		}
		defer resp.Body.Close()

		responseBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Debug("error reading response: ", err)
			continue
		}

		statusCode := int64(resp.StatusCode)
		_, err = db.Provider.AddWebhookLog(ctx, &models.WebhookLog{
			HttpStatus: statusCode,
			Request:    string(requestBody),
			Response:   string(responseBytes),
			WebhookID:  webhook.ID,
		})
		if err != nil {
			log.Debug("failed to add webhook log: ", err)
			continue
		}
	}
	return nil
}
