package config

import "testing"

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
		name      string
		setup     func(*Config)
		wantTOTP  bool
		wantEmail bool
		wantSMS   bool
		wantMFA   bool
	}{
		{
			name:      "defaults: TOTP on, no providers -> MFA available via TOTP",
			setup:     func(c *Config) {},
			wantTOTP:  true,
			wantEmail: true, // flag on, but email service unavailable
			wantSMS:   true, // flag on, but SMS service unavailable
			wantMFA:   true, // TOTP alone makes MFA available
		},
		{
			name:      "TOTP disabled, no providers -> no MFA available",
			setup:     func(c *Config) { c.DisableTOTPLogin = true },
			wantTOTP:  false,
			wantEmail: true,
			wantSMS:   true,
			wantMFA:   false, // email/SMS on but neither provider configured
		},
		{
			name:      "TOTP disabled, email service configured -> MFA via email OTP",
			setup:     func(c *Config) { c.DisableTOTPLogin = true; withSMTP(c) },
			wantTOTP:  false,
			wantEmail: true,
			wantSMS:   true,
			wantMFA:   true,
		},
		{
			name:      "TOTP+email disabled, SMS service configured -> MFA via SMS OTP",
			setup:     func(c *Config) { c.DisableTOTPLogin = true; c.DisableEmailOTP = true; withTwilio(c) },
			wantTOTP:  false,
			wantEmail: false,
			wantSMS:   true,
			wantMFA:   true,
		},
		{
			name: "all methods disabled -> no MFA even with providers configured",
			setup: func(c *Config) {
				c.DisableTOTPLogin = true
				c.DisableEmailOTP = true
				c.DisableSMSOTP = true
				withSMTP(c)
				withTwilio(c)
			},
			wantTOTP:  false,
			wantEmail: false,
			wantSMS:   false,
			wantMFA:   false,
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
