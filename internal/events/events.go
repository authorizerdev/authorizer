package events

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies for events
type Dependencies struct {
	Log             *zerolog.Logger
	StorageProvider storage.Provider
}

// Provider interface for registering events
type Provider interface {
	// RegisterEvent to register event and add webhook logs
	RegisterEvent(ctx context.Context, eventName string, authRecipe string, user *schemas.User) error
}

type provider struct {
	config *config.Config
	deps   *Dependencies
}

// New returns a new events provider
func New(config *config.Config, deps *Dependencies) (Provider, error) {
	return &provider{
		config: config,
		deps:   deps,
	}, nil
}

// RegisterEvent util to register event
func (p *provider) RegisterEvent(ctx context.Context, eventName string, authRecipe string, user *schemas.User) error {
	log := p.deps.Log.With().Str("func", "RegisterEvent").Str("event", eventName).Logger()
	webhooks, err := p.deps.StorageProvider.GetWebhookByEventName(ctx, eventName)
	if err != nil {
		log.Debug().Err(err).Msg("error getting webhook")
		return err
	}
	for _, webhook := range webhooks {
		if !webhook.Enabled {
			continue
		}
		userBytes, err := json.Marshal(user.AsAPIUser())
		if err != nil {
			log.Debug().Err(err).Msg("error marshalling user obj")
			continue
		}
		userMap := map[string]interface{}{}
		err = json.Unmarshal(userBytes, &userMap)
		if err != nil {
			log.Debug().Err(err).Msg("error un-marshalling user obj")
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
			log.Debug().Err(err).Msg("error marshalling requestBody obj")
			continue
		}

		// don't trigger webhook call in case of test
		if p.config.Env == constants.TestEnv {
			_, err := p.deps.StorageProvider.AddWebhookLog(ctx, &schemas.WebhookLog{
				HttpStatus: 200,
				Request:    string(requestBody),
				Response:   string(`{"message": "test"}`),
				WebhookID:  webhook.ID,
			})
			if err != nil {
				log.Debug().Err(err).Msg("error saving webhook log")
			}
			continue
		}

		requestBytesBuffer := bytes.NewBuffer(requestBody)
		req, err := http.NewRequest("POST", webhook.EndPoint, requestBytesBuffer)
		if err != nil {
			log.Debug().Err(err).Msg("error creating request")
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		headersMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(webhook.Headers), &headersMap)
		if err != nil {
			log.Debug().Err(err).Msg("error un-marshalling headers")
		}
		for key, val := range headersMap {
			req.Header.Set(key, val.(string))
		}

		client := &http.Client{Timeout: time.Second * 30}
		resp, err := client.Do(req)
		if err != nil {
			log.Debug().Err(err).Msg("error making request")
			continue
		}
		defer resp.Body.Close()

		responseBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Debug().Err(err).Msg("error reading response")
			continue
		}

		statusCode := int64(resp.StatusCode)
		_, err = p.deps.StorageProvider.AddWebhookLog(ctx, &schemas.WebhookLog{
			HttpStatus: statusCode,
			Request:    string(requestBody),
			Response:   string(responseBytes),
			WebhookID:  webhook.ID,
		})
		if err != nil {
			log.Debug().Err(err).Msg("error saving webhook log")
			continue
		}
	}
	return nil
}
