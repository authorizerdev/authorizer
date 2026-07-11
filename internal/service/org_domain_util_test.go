package service

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeDomain(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "already normalized", in: "acme.com", want: "acme.com"},
		{name: "uppercase", in: "Acme.COM", want: "acme.com"},
		{name: "surrounding whitespace", in: "  acme.com \t", want: "acme.com"},
		{name: "trailing dot (FQDN)", in: "acme.com.", want: "acme.com"},
		{name: "leading @ (email artifact)", in: "@acme.com", want: "acme.com"},
		{name: "wildcard prefix stripped", in: "*.acme.com", want: "acme.com"},
		{name: "subdomain kept exact", in: "eng.acme.com", want: "eng.acme.com"},
		{name: "IDNA unicode to punycode", in: "münchen.de", want: "xn--mnchen-3ya.de"},
		{name: "empty", in: "", wantErr: true},
		{name: "whitespace only", in: "   ", wantErr: true},
		{name: "bare label / no dot", in: "localhost", wantErr: true},
		{name: "scheme rejected", in: "https://acme.com", wantErr: true},
		{name: "path rejected", in: "acme.com/login", wantErr: true},
		{name: "port rejected", in: "acme.com:8080", wantErr: true},
		{name: "embedded wildcard rejected", in: "foo.*.acme.com", wantErr: true},
		{name: "full email rejected", in: "jane@acme.com", wantErr: true},
		{name: "space in host rejected", in: "ac me.com", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeDomain(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("normalizeDomain(%q) = %q, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeDomain(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("normalizeDomain(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestGuardVerifiableDomain(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr error
	}{
		{name: "registrable domain allowed", in: "acme.com"},
		{name: "subdomain allowed", in: "eng.acme.com"},
		{name: "bare TLD rejected", in: "com", wantErr: errPublicSuffixDomain},
		{name: "multi-label public suffix rejected", in: "co.uk", wantErr: errPublicSuffixDomain},
		{name: "consumer gmail rejected", in: "gmail.com", wantErr: errConsumerDomain},
		{name: "consumer outlook rejected", in: "outlook.com", wantErr: errConsumerDomain},
		{name: "consumer yahoo rejected", in: "yahoo.com", wantErr: errConsumerDomain},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := guardVerifiableDomain(tc.in)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("guardVerifiableDomain(%q) unexpected error: %v", tc.in, err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("guardVerifiableDomain(%q) = %v, want %v", tc.in, err, tc.wantErr)
			}
		})
	}
}

// TestNormalizeEmailDomainParity locks review M3: the domain a login email
// resolves to (Phase-3 HRD lookup) must be byte-identical to what Phase-2 wrote
// via normalizeDomain — one normalizer, no split routing.
func TestNormalizeEmailDomainParity(t *testing.T) {
	cases := []struct {
		name    string
		email   string
		want    string
		wantErr bool
	}{
		{name: "simple", email: "jane@acme.com", want: "acme.com"},
		{name: "uppercased", email: "JANE@Acme.COM", want: "acme.com"},
		{name: "whitespace", email: "  jane@acme.com \t", want: "acme.com"},
		{name: "subdomain kept exact", email: "jane@eng.acme.com", want: "eng.acme.com"},
		{name: "idna to punycode", email: "jane@münchen.de", want: "xn--mnchen-3ya.de"},
		{name: "plus-addressing local", email: "jane+tag@acme.com", want: "acme.com"},
		{name: "last-at wins", email: "weird@local@acme.com", want: "acme.com"},
		{name: "empty", email: "", wantErr: true},
		{name: "no at", email: "acme.com", wantErr: true},
		{name: "empty local", email: "@acme.com", wantErr: true},
		{name: "empty domain", email: "jane@", wantErr: true},
		{name: "bare label domain", email: "jane@localhost", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeEmailDomain(tc.email)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("NormalizeEmailDomain(%q) = %q, want error", tc.email, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeEmailDomain(%q) unexpected error: %v", tc.email, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeEmailDomain(%q) = %q, want %q", tc.email, got, tc.want)
			}
			// Parity: same output as normalizing the raw domain part directly.
			if raw, rerr := normalizeDomain(tc.email[strings.LastIndex(tc.email, "@")+1:]); rerr == nil && raw != got {
				t.Fatalf("parity broken: email path %q vs domain path %q", got, raw)
			}
		})
	}
}

func TestChallengeTokenIsRandomBase32(t *testing.T) {
	a, err := generateDomainChallengeToken()
	if err != nil {
		t.Fatalf("generateDomainChallengeToken error: %v", err)
	}
	b, err := generateDomainChallengeToken()
	if err != nil {
		t.Fatalf("generateDomainChallengeToken error: %v", err)
	}
	if a == b {
		t.Fatal("two tokens collided — not random")
	}
	// 32 bytes base32 (no padding) = 52 chars.
	if len(a) != 52 {
		t.Fatalf("token length = %d, want 52", len(a))
	}
}
