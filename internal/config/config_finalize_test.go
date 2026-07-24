package config

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// fullSMTP / fullTwilio set the minimum credentials Finalize() checks so the
// corresponding service is derived as available.
func withSMTP(c *Config) {
	c.SMTPHost = "smtp.example.com"
	c.SMTPPort = 587
	c.SMTPSenderEmail = "no-reply@example.com"
}

func withTwilio(c *Config) {
	c.TwilioAPIKey = "key"
	c.TwilioAPISecret = "secret"
	c.TwilioAccountSID = "sid"
	c.TwilioSender = "+10000000000"
}

// TestFinalizeMFADerivation verifies that MFA methods default on, that email/SMS
// OTP only count toward availability when their provider is configured, and that
// EnableMFA is the OR of the usable methods.
func TestFinalizeMFADerivation(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*Config)
		wantTOTP     bool
		wantWebauthn bool
		wantEmail    bool
		wantSMS      bool
		wantMFA      bool
	}{
		{
			name:         "defaults: TOTP+WebAuthn on, no providers -> MFA available",
			setup:        func(c *Config) {},
			wantTOTP:     true,
			wantWebauthn: true,
			wantEmail:    true, // flag on, but email service unavailable
			wantSMS:      true, // flag on, but SMS service unavailable
			wantMFA:      true, // TOTP/WebAuthn alone make MFA available
		},
		{
			name:         "TOTP+WebAuthn disabled, no providers -> no MFA available",
			setup:        func(c *Config) { c.DisableTOTPLogin = true; c.DisableWebauthnMFA = true },
			wantTOTP:     false,
			wantWebauthn: false,
			wantEmail:    true,
			wantSMS:      true,
			wantMFA:      false, // email/SMS on but neither provider configured
		},
		{
			name:         "TOTP disabled but WebAuthn on -> MFA still available",
			setup:        func(c *Config) { c.DisableTOTPLogin = true },
			wantTOTP:     false,
			wantWebauthn: true,
			wantEmail:    true,
			wantSMS:      true,
			wantMFA:      true,
		},
		{
			name:         "TOTP+WebAuthn disabled, email service configured -> MFA via email OTP",
			setup:        func(c *Config) { c.DisableTOTPLogin = true; c.DisableWebauthnMFA = true; withSMTP(c) },
			wantTOTP:     false,
			wantWebauthn: false,
			wantEmail:    true,
			wantSMS:      true,
			wantMFA:      true,
		},
		{
			name: "TOTP+WebAuthn+email disabled, SMS service configured -> MFA via SMS OTP",
			setup: func(c *Config) {
				c.DisableTOTPLogin = true
				c.DisableWebauthnMFA = true
				c.DisableEmailOTP = true
				withTwilio(c)
			},
			wantTOTP:     false,
			wantWebauthn: false,
			wantEmail:    false,
			wantSMS:      true,
			wantMFA:      true,
		},
		{
			name: "all methods disabled -> no MFA even with providers configured",
			setup: func(c *Config) {
				c.DisableTOTPLogin = true
				c.DisableWebauthnMFA = true
				c.DisableEmailOTP = true
				c.DisableSMSOTP = true
				withSMTP(c)
				withTwilio(c)
			},
			wantTOTP:     false,
			wantWebauthn: false,
			wantEmail:    false,
			wantSMS:      false,
			wantMFA:      false,
		},
		{
			name: "DisableMFA kill switch forces MFA off despite usable methods",
			setup: func(c *Config) {
				c.DisableMFA = true
				withSMTP(c)
				withTwilio(c)
			},
			wantTOTP:     true, // per-method flags still derive true...
			wantWebauthn: true,
			wantEmail:    true,
			wantSMS:      true,
			wantMFA:      false, // ...but the kill switch forces overall MFA off
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			tt.setup(c)
			c.Finalize()

			if c.EnableTOTPLogin != tt.wantTOTP {
				t.Errorf("EnableTOTPLogin = %v, want %v", c.EnableTOTPLogin, tt.wantTOTP)
			}
			if c.EnableWebauthnMFA != tt.wantWebauthn {
				t.Errorf("EnableWebauthnMFA = %v, want %v", c.EnableWebauthnMFA, tt.wantWebauthn)
			}
			if c.EnableEmailOTP != tt.wantEmail {
				t.Errorf("EnableEmailOTP = %v, want %v", c.EnableEmailOTP, tt.wantEmail)
			}
			if c.EnableSMSOTP != tt.wantSMS {
				t.Errorf("EnableSMSOTP = %v, want %v", c.EnableSMSOTP, tt.wantSMS)
			}
			if c.EnableMFA != tt.wantMFA {
				t.Errorf("EnableMFA = %v, want %v", c.EnableMFA, tt.wantMFA)
			}
		})
	}
}

// TestFinalizeIsSMSServiceEnabled verifies IsSMSServiceEnabled is true when
// either Twilio is fully configured or Env == E2EEnv, and false when
// neither is set.
func TestFinalizeIsSMSServiceEnabled(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Config)
		want  bool
	}{
		{
			name:  "neither Twilio nor e2e env configured -> false",
			setup: func(c *Config) {},
			want:  false,
		},
		{
			name:  "Twilio configured, production env -> true",
			setup: withTwilio,
			want:  true,
		},
		{
			name:  "e2e env, no Twilio -> true",
			setup: func(c *Config) { c.Env = constants.E2EEnv },
			want:  true,
		},
		{
			name:  "test env (integration_tests), no Twilio -> false",
			setup: func(c *Config) { c.Env = constants.TestEnv },
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			tt.setup(c)
			c.Finalize()

			if c.IsSMSServiceEnabled != tt.want {
				t.Errorf("IsSMSServiceEnabled = %v, want %v", c.IsSMSServiceEnabled, tt.want)
			}
		})
	}
}
