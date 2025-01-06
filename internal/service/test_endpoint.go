package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
	"github.com/google/uuid"
)

// TestEndpoint is a service to test a webhook endpoint
// Permission: authorizer:admin
func (s *service) TestEndpoint(ctx context.Context, params *model.TestEndpointRequest) (*model.TestEndpointResponse, error) {
	log := s.Log.With().Str("func", "TestEndpoint").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !s.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	if !validators.IsValidWebhookEventName(params.EventName) {
		log.Debug().Str("event_name", params.EventName).Msg("Invalid event name")
		return nil, fmt.Errorf("invalid event_name %s", params.EventName)
	}

	user := model.User{
		ID:            uuid.NewString(),
		Email:         refs.NewStringRef("test_endpoint@authorizer.dev"),
		EmailVerified: true,
		SignupMethods: constants.AuthRecipeMethodMagicLinkLogin,
		GivenName:     refs.NewStringRef("Foo"),
		FamilyName:    refs.NewStringRef("Bar"),
	}

	userBytes, err := json.Marshal(user)
	if err != nil {
		log.Debug().Err(err).Msg("error marshalling user obj")
		return nil, err
	}
	userMap := map[string]interface{}{}
	err = json.Unmarshal(userBytes, &userMap)
	if err != nil {
		log.Debug().Err(err).Msg("error un-marshalling user obj")
		return nil, err
	}

	reqBody := map[string]interface{}{
		"event_name": constants.UserLoginWebhookEvent,
		"user":       userMap,
	}

	if params.EventName == constants.UserLoginWebhookEvent {
		reqBody["auth_recipe"] = constants.AuthRecipeMethodMagicLinkLogin
	}

	requestBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Debug().Err(err).Msg("error marshalling requestBody obj")
		return nil, err
	}

	req, err := http.NewRequest("POST", params.Endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Debug().Err(err).Msg("error creating post request")
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, val := range params.Headers {
		req.Header.Set(key, val.(string))
	}
	client := &http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("error making request")
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Debug().Err(err).Msg("error reading response")
		return nil, err
	}

	statusCode := int64(resp.StatusCode)
	return &model.TestEndpointResponse{
		HTTPStatus: &statusCode,
		Response:   refs.NewStringRef(string(body)),
	}, nil
}
