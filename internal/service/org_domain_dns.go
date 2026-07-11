package service

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"net"
	"strings"
	"time"
)

const (
	// domainChallengeTTL is how long a pending DNS verification challenge lives in
	// the memory store before it must be re-requested (~24h).
	domainChallengeTTL = 24 * time.Hour

	// domainChallengeKeyPrefix namespaces the pending-challenge entries in the
	// memory store: org_domain_challenge:<org_id>:<domain> → <token>.
	domainChallengeKeyPrefix = "org_domain_challenge:"

	// domainChallengeRecordPrefix is the DNS label the TXT proof is published at:
	// _authorizer-challenge.<domain>.
	domainChallengeRecordPrefix = "_authorizer-challenge."

	// domainChallengeValuePrefix prefixes the TXT record value the tenant must
	// publish: authorizer-domain-verification=<token>.
	domainChallengeValuePrefix = "authorizer-domain-verification="

	// domainDNSLookupTimeout bounds the resolver call so a slow/hostile
	// nameserver cannot stall the request.
	domainDNSLookupTimeout = 5 * time.Second
)

// DNSResolver is the minimal resolver surface the domain-verification flow
// needs. *net.Resolver satisfies it; tests inject a mock so no real DNS is hit.
type DNSResolver interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
}

// domainResolver returns the injected resolver, or the process default. Kept as
// a method so a single nil-check lives in one place.
func (p *provider) domainResolver() DNSResolver {
	if p.DNSResolver != nil {
		return p.DNSResolver
	}
	return net.DefaultResolver
}

// challengeKey builds the memory-store key for an (org, domain) pending
// challenge. domain must already be normalized.
func challengeKey(orgID, domain string) string {
	return domainChallengeKeyPrefix + orgID + ":" + domain
}

// generateDomainChallengeToken returns a 32-byte crypto/rand token, base32
// (lowercase, unpadded) so it is safe to publish verbatim in a DNS TXT record.
func generateDomainChallengeToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)), nil
}

// challengeRecordValue is the exact TXT value the tenant must publish.
func challengeRecordValue(token string) string {
	return domainChallengeValuePrefix + token
}

// challengeRecordName is the DNS name the TXT record lives at.
func challengeRecordName(domain string) string {
	return domainChallengeRecordPrefix + domain
}

// lookupDomainTXTMatches resolves the challenge TXT record for domain and
// reports whether any record exactly equals authorizer-domain-verification=token.
// The lookup is TXT-only with a bounded deadline; no HTTP is ever fetched.
func (p *provider) lookupDomainTXTMatches(ctx context.Context, domain, token string) (bool, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, domainDNSLookupTimeout)
	defer cancel()
	records, err := p.domainResolver().LookupTXT(lookupCtx, challengeRecordName(domain))
	if err != nil {
		return false, err
	}
	want := challengeRecordValue(token)
	for _, r := range records {
		// Exact match only — a substring/prefix match would let an attacker who
		// controls unrelated TXT content on the domain satisfy the challenge.
		if strings.TrimSpace(r) == want {
			return true, nil
		}
	}
	return false, nil
}
