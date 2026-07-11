package service

import (
	"errors"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
)

// Domain-validation errors. Kept distinct so callers/tests can assert the exact
// reason, and so the same messages flow to the API uniformly.
var (
	errInvalidDomain      = errors.New("invalid domain")
	errPublicSuffixDomain = errors.New("cannot verify a public suffix or bare TLD")
	errConsumerDomain     = errors.New("cannot verify a shared consumer email domain")
)

// consumerDomainBlocklist is a small set of well-known consumer email providers
// that are registrable (so the public-suffix guard would allow them) but must
// never be claimable as a tenant's routing domain. Not exhaustive by design —
// DNS TXT proof is still the trust anchor; this only blocks the obvious ones.
var consumerDomainBlocklist = map[string]bool{
	"gmail.com":      true,
	"googlemail.com": true,
	"outlook.com":    true,
	"hotmail.com":    true,
	"live.com":       true,
	"msn.com":        true,
	"yahoo.com":      true,
	"ymail.com":      true,
	"aol.com":        true,
	"icloud.com":     true,
	"me.com":         true,
	"mac.com":        true,
	"proton.me":      true,
	"protonmail.com": true,
	"gmx.com":        true,
	"gmx.net":        true,
	"mail.com":       true,
	"zoho.com":       true,
	"yandex.com":     true,
	"yandex.ru":      true,
	"fastmail.com":   true,
	"hey.com":        true,
	"pm.me":          true,
	"qq.com":         true,
	"163.com":        true,
	"126.com":        true,
}

// normalizeDomain is the SINGLE canonical domain normalizer used by every domain
// operation (Phase-2 writes AND any future home-realm-discovery lookup). Keeping
// it as one function is deliberate: a split implementation could let a homograph
// verify under one form and route under another. It:
//   - lowercases and trims,
//   - strips a leading "@" (email-local artifact) or "*." (wildcard) label,
//   - rejects anything carrying a scheme, path, port, wildcard, or whitespace,
//   - converts to punycode via the IDNA Lookup profile (UTS-46,
//     non-transitional), and
//   - requires at least one dot (a bare label / TLD is not a verifiable domain).
//
// It does NOT apply the public-suffix / consumer guard — that is a separate
// policy check (guardVerifiableDomain) applied only to WRITES, so a lookup of an
// already-verified consumer-ish row (should one ever exist) still resolves.
func normalizeDomain(input string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(input))
	if d == "" {
		return "", errInvalidDomain
	}
	if strings.Contains(d, "://") {
		return "", errInvalidDomain
	}
	d = strings.TrimPrefix(d, "@")
	d = strings.TrimPrefix(d, "*.")
	d = strings.TrimSuffix(d, ".")
	// Reject leftover unsafe characters: path, port, wildcard, whitespace, or a
	// stray email "@". Punycode/ASCII domains never legitimately contain these.
	if strings.ContainsAny(d, "/*:@ \t\r\n?#") {
		return "", errInvalidDomain
	}
	ascii, err := idna.Lookup.ToASCII(d)
	if err != nil {
		return "", errInvalidDomain
	}
	if !strings.Contains(ascii, ".") {
		return "", errInvalidDomain
	}
	return ascii, nil
}

// NormalizeEmailDomain extracts the domain part of an email address and
// normalizes it through the SAME canonical normalizeDomain used for Phase-2
// domain verification writes, so a home-realm-discovery lookup resolves to the
// exact value that was stored (review M3 — one normalizer, no split routing).
// It splits on the LAST "@" (a valid address has exactly one, but this is
// robust to quirky local-parts) and requires a non-empty local part.
func NormalizeEmailDomain(email string) (string, error) {
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "", errInvalidDomain
	}
	return normalizeDomain(email[at+1:])
}

// guardVerifiableDomain enforces the write-time policy: a tenant may not verify a
// public suffix / bare TLD (co.uk, com) nor a shared consumer email provider
// (gmail.com, …). The input MUST already be normalized (punycode).
func guardVerifiableDomain(ascii string) error {
	suffix, _ := publicsuffix.PublicSuffix(ascii)
	if suffix == ascii {
		// The whole string is itself a public suffix (e.g. "com", "co.uk").
		return errPublicSuffixDomain
	}
	if consumerDomainBlocklist[ascii] {
		return errConsumerDomain
	}
	return nil
}
