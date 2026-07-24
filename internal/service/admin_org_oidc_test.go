package service

import "testing"

// TestValidateSSOIssuerURL_AllowInsecureFalse_RejectsHTTP proves the
// production-default no-op: with allowInsecure=false (Config.Env != E2EEnv,
// the production default), a plain http:// issuer_url is rejected - the
// SSRF-relaxation escape hatch (e2e-playground's mock IdP has no TLS
// termination) only opens when an operator explicitly passes --env=e2e.
func TestValidateSSOIssuerURL_AllowInsecureFalse_RejectsHTTP(t *testing.T) {
	if err := validateSSOIssuerURL("http://idp.example.com", false); err == nil {
		t.Fatal("expected http issuer_url to be rejected when allowInsecure is false")
	}
}

// TestValidateSSOIssuerURL_AllowInsecureTrue_AcceptsHTTP proves the escape
// hatch itself works when explicitly opted into.
func TestValidateSSOIssuerURL_AllowInsecureTrue_AcceptsHTTP(t *testing.T) {
	if err := validateSSOIssuerURL("http://mock-saml-idp.e2e-playground.local", true); err != nil {
		t.Fatalf("expected http issuer_url to be accepted when allowInsecure is true: %v", err)
	}
}

// TestValidateSSOIssuerURL_HTTPSAlwaysAccepted proves allowInsecure has no
// effect on the https path either way.
func TestValidateSSOIssuerURL_HTTPSAlwaysAccepted(t *testing.T) {
	for _, allowInsecure := range []bool{false, true} {
		if err := validateSSOIssuerURL("https://idp.example.com", allowInsecure); err != nil {
			t.Fatalf("expected https issuer_url to be accepted (allowInsecure=%v): %v", allowInsecure, err)
		}
	}
}

// TestValidateSSOIssuerURL_InvalidURL_AlwaysRejected proves an unparseable/opaque
// URL is rejected regardless of allowInsecure.
func TestValidateSSOIssuerURL_InvalidURL_AlwaysRejected(t *testing.T) {
	for _, allowInsecure := range []bool{false, true} {
		if err := validateSSOIssuerURL("not-a-url", allowInsecure); err == nil {
			t.Fatalf("expected invalid issuer_url to be rejected (allowInsecure=%v)", allowInsecure)
		}
	}
}
