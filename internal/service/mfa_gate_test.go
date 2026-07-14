// internal/service/mfa_gate_test.go
package service

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

func TestResolveMFAGate(t *testing.T) {
	cases := []struct {
		name                  string
		userMFAEnabled        bool
		enforceMFA            bool
		authenticatorVerified bool
		hasSkippedSetup       bool
		want                  mfaGateDecision
	}{
		{"mfa off for user", false, false, false, false, mfaGateNone},
		{"mfa off for user, enforced anyway (inconsistent state defends safe)", false, true, false, false, mfaGateNone},
		{"enforced, not yet enrolled", true, true, false, false, mfaGateBlockEnroll},
		{"enforced, already verified", true, true, true, false, mfaGateBlockVerify},
		{"enforced, skip flag present but ignored", true, true, false, true, mfaGateBlockEnroll},
		{"optional, already verified -> still verify every time", true, false, true, false, mfaGateBlockVerify},
		{"optional, already verified, skip flag stale -> still verify", true, false, true, true, mfaGateBlockVerify},
		{"optional, not enrolled, never skipped -> offer", true, false, false, false, mfaGateOfferSetup},
		{"optional, not enrolled, already skipped -> quiet login", true, false, false, true, mfaGateSkippedSetup},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolveMFAGate(c.userMFAEnabled, c.enforceMFA, c.authenticatorVerified, c.hasSkippedSetup)
			if got != c.want {
				t.Errorf("resolveMFAGate(%v,%v,%v,%v) = %v, want %v", c.userMFAEnabled, c.enforceMFA, c.authenticatorVerified, c.hasSkippedSetup, got, c.want)
			}
		})
	}
}

func TestEffectiveMFAEnabled(t *testing.T) {
	cases := []struct {
		name         string
		cfgEnableMFA bool
		userOptIn    *bool // nil = never explicitly set
		want         bool
	}{
		{"MFA available server-wide, user never set it explicitly -> follows config", true, nil, true},
		{"MFA unavailable server-wide, user never set it explicitly -> follows config", false, nil, false},
		{"MFA available server-wide, user explicitly opted out -> respects opt-out", true, boolPtr(false), false},
		{"MFA unavailable server-wide, user explicitly opted in -> respects opt-in", false, boolPtr(true), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &config.Config{EnableMFA: c.cfgEnableMFA}
			user := &schemas.User{IsMultiFactorAuthEnabled: c.userOptIn}
			got := effectiveMFAEnabled(cfg, user)
			if got != c.want {
				t.Errorf("effectiveMFAEnabled(EnableMFA=%v, opt-in=%v) = %v, want %v", c.cfgEnableMFA, c.userOptIn, got, c.want)
			}
		})
	}
}

func boolPtr(b bool) *bool { return &b }
