package events

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/validators"
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
	// RegisterScimGroupEvent fires a SCIM group-lifecycle webhook. Unlike
	// RegisterEvent the payload carries a `group` object (not a `user`), so it
	// can dispatch group.created/updated/deleted for the SCIM provisioning flow.
	RegisterScimGroupEvent(ctx context.Context, eventName string, group *schemas.ScimGroup) error
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
	userBytes, err := json.Marshal(user.AsAPIUser())
	if err != nil {
		log.Debug().Err(err).Msg("error marshalling user obj")
		return err
	}
	userMap := map[string]interface{}{}
	if err := json.Unmarshal(userBytes, &userMap); err != nil {
		log.Debug().Err(err).Msg("error un-marshalling user obj")
		return err
	}
	for _, webhook := range webhooks {
		if !webhook.Enabled {
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
		p.deliver(ctx, &log, webhook, reqBody)
	}
	return nil
}

// RegisterScimGroupEvent fires a SCIM group-lifecycle webhook. The payload
// carries a `group` object (id, org, displayName, de-namespaced externalId)
// instead of a `user`, sharing the same delivery path (SSRF pin, HMAC, log).
func (p *provider) RegisterScimGroupEvent(ctx context.Context, eventName string, group *schemas.ScimGroup) error {
	log := p.deps.Log.With().Str("func", "RegisterScimGroupEvent").Str("event", eventName).Logger()
	webhooks, err := p.deps.StorageProvider.GetWebhookByEventName(ctx, eventName)
	if err != nil {
		log.Debug().Err(err).Msg("error getting webhook")
		return err
	}
	groupMap := map[string]interface{}{
		"id":           group.ID,
		"org_id":       group.OrgID,
		"display_name": group.DisplayName,
	}
	if group.ExternalID != nil {
		// De-namespace back to the raw IdP value ("<orgID>:<raw>") for the consumer.
		groupMap["external_id"] = strings.TrimPrefix(*group.ExternalID, group.OrgID+":")
	}
	for _, webhook := range webhooks {
		if !webhook.Enabled {
			continue
		}
		reqBody := map[string]interface{}{
			"webhook_id":        webhook.ID,
			"event_name":        eventName,
			"event_description": webhook.EventDescription,
			"group":             groupMap,
		}
		p.deliver(ctx, &log, webhook, reqBody)
	}
	return nil
}

// deliver marshals reqBody and POSTs it to one webhook endpoint, logging the
// outcome. It centralises the security-sensitive delivery mechanics (SSRF pin,
// HMAC signature, TestEnv short-circuit, response log) shared by every event
// type. Errors are logged and swallowed per-webhook — one bad endpoint never
// aborts delivery to the others.
func (p *provider) deliver(ctx context.Context, log *zerolog.Logger, webhook *schemas.Webhook, reqBody map[string]interface{}) {
	requestBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Debug().Err(err).Msg("error marshalling requestBody obj")
		return
	}

	// don't trigger webhook call in case of test
	if p.config.Env == constants.TestEnv {
		if _, err := p.deps.StorageProvider.AddWebhookLog(ctx, &schemas.WebhookLog{
			HttpStatus: 200,
			Request:    string(requestBody),
			Response:   string(`{"message": "test"}`),
			WebhookID:  webhook.ID,
		}); err != nil {
			log.Debug().Err(err).Msg("error saving webhook log")
		}
		return
	}

	// SSRF protection: resolve the host once and pin the dialer to the
	// validated IP so http.Client cannot be tricked into re-resolving
	// (DNS rebinding TOCTOU).
	client, err := webhookHTTPClient(ctx, webhook.EndPoint, time.Second*30, p.config.TestAllowPrivateWebhookHosts)
	if err != nil {
		log.Debug().Err(err).Str("endpoint", webhook.EndPoint).Msg("webhook endpoint rejected by SSRF filter")
		_, _ = p.deps.StorageProvider.AddWebhookLog(ctx, &schemas.WebhookLog{
			HttpStatus: 0,
			Request:    string(requestBody),
			Response:   fmt.Sprintf(`{"error": "SSRF validation failed: %s"}`, err.Error()),
			WebhookID:  webhook.ID,
		})
		return
	}

	// Compute HMAC-SHA256 signature for payload authenticity
	mac := hmac.New(sha256.New, []byte(p.config.ClientSecret))
	mac.Write(requestBody)
	signature := hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest("POST", webhook.EndPoint, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Debug().Err(err).Msg("error creating request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Authorizer-Signature", signature)
	headersMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(webhook.Headers), &headersMap); err != nil {
		log.Debug().Err(err).Msg("error un-marshalling headers")
	}
	for key, val := range headersMap {
		// Header values come from admin-configured JSON (a Map scalar), so a
		// value may be a number/bool/object; coerce instead of asserting to
		// avoid panicking this bare goroutine (which would crash the process).
		req.Header.Set(key, headerValueString(val))
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("error making request")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	responseBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		log.Debug().Err(err).Msg("error reading response")
		return
	}

	if _, err := p.deps.StorageProvider.AddWebhookLog(ctx, &schemas.WebhookLog{
		HttpStatus: int64(resp.StatusCode),
		Request:    string(requestBody),
		Response:   string(responseBytes),
		WebhookID:  webhook.ID,
	}); err != nil {
		log.Debug().Err(err).Msg("error saving webhook log")
	}
}

// webhookHTTPClient builds the SSRF-hardened *http.Client for a webhook delivery.
// allowPrivate (Config.TestAllowPrivateWebhookHosts) is the ONLY thing that
// switches this to validators.SafeHTTPClientAllowPrivate — see that function's
// doc comment for why it exists (e2e-playground's webhook-sink mock only, reachable
// solely at a docker-compose-private address). Every other invariant (scheme
// allow-list, DNS-rebinding host pinning, TLS SNI) is unchanged either way. Mirrors
// internal/http_handlers/oauth_sso.go's ssoHTTPClient, kept independent so the two
// escape hatches can never relax each other.
func webhookHTTPClient(ctx context.Context, rawURL string, timeout time.Duration, allowPrivate bool) (*http.Client, error) {
	if allowPrivate {
		return validators.SafeHTTPClientAllowPrivate(ctx, rawURL, timeout)
	}
	return validators.SafeHTTPClient(ctx, rawURL, timeout)
}

// headerValueString coerces a free-form JSON header value to a string without
// panicking. Mirrors the identically named helper in internal/service used by
// the webhook TestEndpoint path (kept local to avoid an import cycle).
func headerValueString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
