// internal/service/meta_test.go
package service

import (
	"context"
	"testing"

	"github.com/authorizerdev/authorizer/internal/config"
)

// REGRESSION: IsDiscordLoginEnabled was missing entirely from the Meta
// resolver's struct literal - every other social provider (Google, GitHub,
// Facebook, LinkedIn, Apple, Twitter, Microsoft, Twitch, Roblox) derives its
// is_<provider>_login_enabled flag from ClientID/ClientSecret being set, but
// Discord's field was simply never assigned, so it silently defaulted to
// false regardless of configuration. Every frontend consumer that gates a
// Discord login button on this flag (e.g. authorizer-react's
// AuthorizerSocialLogin) could never render one, even with Discord OAuth
// fully configured and working end to end on the backend.
func TestMeta_DiscordLoginEnabled(t *testing.T) {
	cases := []struct {
		name         string
		clientID     string
		clientSecret string
		want         bool
	}{
		{"both set", "client-id", "client-secret", true},
		{"client id missing", "", "client-secret", false},
		{"client secret missing", "client-id", "", false},
		{"neither set", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := &provider{Config: &config.Config{
				DiscordClientID:     c.clientID,
				DiscordClientSecret: c.clientSecret,
			}}
			meta, _, err := p.Meta(context.Background(), RequestMetadata{})
			if err != nil {
				t.Fatalf("Meta() error = %v", err)
			}
			if meta.IsDiscordLoginEnabled != c.want {
				t.Errorf("IsDiscordLoginEnabled = %v, want %v", meta.IsDiscordLoginEnabled, c.want)
			}
		})
	}
}
