package token

import (
	"crypto/subtle"
	"strconv"

	"github.com/authorizerdev/authorizer/internal/metrics"
)

const (
	// adminSecretMaxFailedAttempts / adminSecretLockoutWindowSeconds throttle
	// online guessing of the admin secret.
	//
	// The admin secret is the single highest-privilege credential in the system,
	// and until now the only thing standing between an attacker and unlimited
	// guesses was the shared 30rps request limiter — the same budget ordinary
	// traffic gets. The budget here is deliberately much tighter and much
	// longer-lived than the login lockout: nobody legitimately mistypes an
	// admin secret dozens of times, and unlike a user account there is no
	// self-service recovery an attacker could grief by tripping it.
	adminSecretMaxFailedAttempts    = 10
	adminSecretLockoutWindowSeconds = int64(15 * 60)
	// adminSecretLockoutPrefix namespaces the counter. Keyed by client IP:
	// there is only one admin secret, so a per-principal key would be a single
	// global counter that any attacker could use to lock out every operator.
	adminSecretLockoutPrefix = "admin_secret_failed_attempts:"
	// adminSecretLockoutUnknownIP is the bucket for callers whose address we
	// could not determine. It is deliberately a NAMED bucket rather than the
	// empty string: keying on "" silently merges every unidentifiable caller
	// into one counter, which is how a pure-gRPC deployment (no forwarded
	// headers, no peer address) turns 10 wrong guesses into an outage for every
	// admin client at once. Callers should ensure this is never needed —
	// transport.MetaFromGRPC falls back to the gRPC peer address for exactly
	// that reason — but if it is, the shared bucket must at least be visible in
	// the key rather than looking like a real client.
	adminSecretLockoutUnknownIP = "unknown"
)

// VerifyAdminSecret is the single gate for every admin-secret comparison —
// AdminLogin's cookie-establishing check and the x-authorizer-admin-secret
// header path both route through it, so neither can be brute-forced while the
// other is throttled.
//
// FAILED attempts are counted, not all attempts. The counter is read before the
// comparison and incremented only when the comparison fails. The alternative —
// increment-then-check, which the login and OTP lockouts use — is right for
// those because they gate a human typing a password a few times a minute, but
// wrong here: this same function runs on EVERY request that authenticates with
// the x-authorizer-admin-secret header, so counting successes means an
// integration issuing more than adminSecretMaxFailedAttempts concurrent admin
// calls from one address gets 401s while presenting the CORRECT secret. The
// concurrency argument for increment-then-check does not apply either: it exists
// to stop parallel requests reading one stale pre-increment count and all
// passing, which matters when each parallel request is an independent guess at a
// short secret. Here every parallel request carries the same operator-chosen
// secret, and overshooting the budget by the in-flight count costs an attacker
// nothing they did not already have.
//
// Returns (valid, locked). A locked caller never reaches the comparison at all,
// so the lockout cannot itself be used as a timing oracle for the secret.
//
// What this does NOT protect against, stated plainly so it is not mistaken for a
// boundary:
//
//   - clientIP comes from utils.GetIP / RequestMetadata, which prefer the
//     X-Real-Ip and X-Forwarded-For request headers. On a deployment that is not
//     behind a proxy that overwrites them, those are attacker-controlled: a
//     guesser rotates the header per request and never fills a bucket. Fixing
//     that needs trusted-proxy configuration this server does not yet have.
//   - a distributed guesser gets adminSecretMaxFailedAttempts per source
//     regardless.
//
// It is defence in depth that makes naive online guessing expensive, not a
// substitute for a high-entropy AdminSecret. The entropy is the control.
func (p *provider) VerifyAdminSecret(clientIP, candidate string) (valid bool, locked bool) {
	// An unconfigured secret must never authenticate anything, empty candidate
	// included.
	if p.config.AdminSecret == "" || candidate == "" {
		return false, false
	}

	// The throttle is defence in depth around the comparison, never a
	// precondition for it. A provider built without a memory store (unit tests,
	// and any future wiring that omits it) must still authenticate correctly
	// rather than nil-panic in a request path — an unrecovered panic here would
	// take down the whole process, turning a missing dependency into an outage.
	if p.dependencies == nil || p.dependencies.MemoryStoreProvider == nil {
		return subtle.ConstantTimeCompare([]byte(candidate), []byte(p.config.AdminSecret)) == 1, false
	}

	if clientIP == "" {
		clientIP = adminSecretLockoutUnknownIP
	}
	lockKey := adminSecretLockoutPrefix + clientIP
	// Fail open on a store fault: an outage must not lock every operator out of
	// their own admin console. Same stance as the login/OTP paths.
	if spent, err := p.dependencies.MemoryStoreProvider.GetCache(lockKey); err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Failed to read admin-secret failed-attempt counter")
	} else if attempts, _ := strconv.ParseInt(spent, 10, 64); attempts >= adminSecretMaxFailedAttempts {
		metrics.RecordSecurityEvent("admin_secret_locked", "admin_auth")
		p.dependencies.Log.Warn().Int64("attempts", attempts).Str("ip", clientIP).Msg("Admin secret verification locked: too many failed attempts")
		return false, true
	}

	if subtle.ConstantTimeCompare([]byte(candidate), []byte(p.config.AdminSecret)) != 1 {
		if _, err := p.dependencies.MemoryStoreProvider.IncrementCache(lockKey, adminSecretLockoutWindowSeconds); err != nil {
			p.dependencies.Log.Debug().Err(err).Msg("Failed to increment admin-secret failed-attempt counter")
		}
		return false, false
	}

	// Correct secret: clear the budget so a legitimate operator who fat-fingered
	// it a few times starts fresh.
	if err := p.dependencies.MemoryStoreProvider.DeleteCacheByPrefix(lockKey); err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("Failed to reset admin-secret failed-attempt counter")
	}
	return true, false
}
