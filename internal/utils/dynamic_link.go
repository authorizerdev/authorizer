package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/authorizerdev/authorizer/internal/config"
)

const airbridgeAPIURL = "https://api.airbridge.io/v1/tracking-links"

type airbridgeRequest struct {
	Channel      string                 `json:"channel"`
	DeeplinkURL  string                 `json:"deeplinkUrl,omitempty"`
	FallbackPaths map[string]string     `json:"fallbackPaths,omitempty"`
}

type airbridgeResponse struct {
	Data struct {
		TrackingLink struct {
			ShortURL string `json:"shortUrl"`
		} `json:"trackingLink"`
	} `json:"data"`
}

// GenerateAirbridgeLink calls the Airbridge API to create a tracking link.
func GenerateAirbridgeLink(apiToken, channel, deeplinkURL, fallbackURL string) (string, error) {
	reqBody := airbridgeRequest{
		Channel:     channel,
		DeeplinkURL: deeplinkURL,
		FallbackPaths: map[string]string{
			"desktop": fallbackURL,
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal airbridge request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, airbridgeAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create airbridge request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("airbridge API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("airbridge API returned status %d", resp.StatusCode)
	}

	var result airbridgeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode airbridge response: %w", err)
	}

	if result.Data.TrackingLink.ShortURL == "" {
		return "", fmt.Errorf("airbridge response missing shortUrl")
	}

	return result.Data.TrackingLink.ShortURL, nil
}

// MaybeWrapDynamicLink wraps a URL with an Airbridge tracking link if configured.
// Falls back to the original URL if Airbridge is not configured or the API call fails.
func MaybeWrapDynamicLink(cfg *config.Config, originalURL string) string {
	if cfg.AirbridgeAPIToken == "" {
		return originalURL
	}

	channel := cfg.AirbridgeChannel
	if channel == "" {
		channel = "email"
	}

	deeplinkURL := originalURL
	if cfg.AirbridgeDeeplinkURL != "" {
		deeplinkURL = cfg.AirbridgeDeeplinkURL + "?token=" + extractToken(originalURL)
	}

	shortURL, err := GenerateAirbridgeLink(cfg.AirbridgeAPIToken, channel, deeplinkURL, originalURL)
	if err != nil {
		return originalURL
	}

	return shortURL
}

// extractToken extracts the token query parameter from a URL string.
func extractToken(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("token")
}
