package resolvers

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
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// TestEndpointResolver resolver to test webhook endpoints
func TestEndpointResolver(ctx context.Context, params model.TestEndpointRequest) (*model.TestEndpointResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	if !validators.IsValidWebhookEventName(params.EventName) {
		log.Debug("Invalid event name: ", params.EventName)
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
		log.Debug("error marshalling user obj: ", err)
		return nil, err
	}
	userMap := map[string]interface{}{}
	err = json.Unmarshal(userBytes, &userMap)
	if err != nil {
		log.Debug("error un-marshalling user obj: ", err)
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
		log.Debug("error marshalling requestBody obj: ", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", params.Endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Debug("error creating post request: ", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, val := range params.Headers {
		req.Header.Set(key, val.(string))
	}
	client := &http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		log.Debug("error making request: ", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Debug("error reading response: ", err)
		return nil, err
	}

	statusCode := int64(resp.StatusCode)
	return &model.TestEndpointResponse{
		HTTPStatus: &statusCode,
		Response:   refs.NewStringRef(string(body)),
	}, nil
}
