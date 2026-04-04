package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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
func (g *graphqlProvider) TestEndpoint(ctx context.Context, params *model.TestEndpointRequest) (*model.TestEndpointResponse, error) {
	log := g.Log.With().Str("func", "TestEndpoint").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
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

	// SSRF protection: validate endpoint URL and resolved IPs
	if err := validateEndpointURL(params.Endpoint); err != nil {
		log.Debug().Err(err).Str("endpoint", params.Endpoint).Msg("endpoint URL rejected by SSRF filter")
		return nil, fmt.Errorf("invalid endpoint: %s", err.Error())
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

// validateEndpointURL checks the webhook endpoint URL for SSRF.
// Rejects private/loopback/link-local IPs and non-http(s) schemes.
func validateEndpointURL(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("malformed URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http and https schemes are allowed")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("missing host")
	}

	// Resolve the hostname to IP addresses
	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("failed to resolve host: %s", err.Error())
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return fmt.Errorf("invalid IP address resolved")
		}
		if isPrivateIP(ip) {
			return fmt.Errorf("requests to private/internal networks are not allowed")
		}
	}
	return nil
}

// isPrivateIP returns true if the IP is in a private, loopback, link-local,
// or otherwise non-routable range.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{parseCIDR("10.0.0.0/8")},
		{parseCIDR("172.16.0.0/12")},
		{parseCIDR("192.168.0.0/16")},
		{parseCIDR("127.0.0.0/8")},
		{parseCIDR("169.254.0.0/16")},  // link-local
		{parseCIDR("100.64.0.0/10")},   // CGN
		{parseCIDR("::1/128")},         // IPv6 loopback
		{parseCIDR("fc00::/7")},        // IPv6 ULA
		{parseCIDR("fe80::/10")},       // IPv6 link-local
		{parseCIDR("0.0.0.0/8")},       // "this" network
		{parseCIDR("192.0.0.0/24")},    // IETF protocol assignments
		{parseCIDR("192.0.2.0/24")},    // TEST-NET-1
		{parseCIDR("198.51.100.0/24")}, // TEST-NET-2
		{parseCIDR("203.0.113.0/24")},  // TEST-NET-3
		{parseCIDR("224.0.0.0/4")},     // multicast
		{parseCIDR("240.0.0.0/4")},     // reserved
	}
	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseCIDR(cidr string) *net.IPNet {
	_, network, _ := net.ParseCIDR(cidr)
	return network
}
