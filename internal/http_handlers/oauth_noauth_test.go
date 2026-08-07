package http_handlers

import (
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Regression tests for the nOAuth account-takeover class (audit findings #1
// and #2).
//
// The attack: a federated login resolves a local account by email, and the
// email an IdP hands back is not necessarily one the principal controls.
// Microsoft Entra is the sharp case — its v2 ID tokens carry NO
// `email_verified` claim at all and `email` is a mutable profile attribute, so
// anyone with a free Entra tenant can set a user's email to the victim's
// address and sign in as them. Three guards close it, and each is pinned here:
//
//  1. claims decode `email_verified` at all (it used to be absent from
//     oidcClaims entirely, so it was never consulted);
//  2. the quoted-string form some IdPs emit still reads as verified;
//  3. Microsoft tokens are constrained to a tenant the operator trusts, and an
//     untrusted tenant's email is never treated as attested.

func TestOIDCClaims_EmailVerifiedIsDecoded(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		payload string
		want    bool
	}{
		{"boolean true", `{"email":"a@b.com","email_verified":true}`, true},
		{"boolean false", `{"email":"a@b.com","email_verified":false}`, false},
		// Apple documents email_verified as "a string or Boolean value";
		// LinkedIn has shipped both. Decoding either into a plain bool would
		// fail the whole claim set and silently downgrade a verified email.
		{"apple string true", `{"email":"a@b.com","email_verified":"true"}`, true},
		{"apple string false", `{"email":"a@b.com","email_verified":"false"}`, false},
		// Absent is the Entra case, and the one that made nOAuth work.
		{"absent", `{"email":"a@b.com"}`, false},
		{"unexpected type", `{"email":"a@b.com","email_verified":1}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			claims := &oidcClaims{}
			require.NoError(t, json.Unmarshal([]byte(tc.payload), claims))
			assert.Equal(t, tc.want, bool(claims.EmailVerified))
		})
	}
}

// TestValidateMicrosoftTenant_RejectsForeignTenant is the direct nOAuth
// reproduction. An attacker's own tenant mints a structurally valid token —
// real Microsoft signature, `aud` equal to the target's client id — because the
// multi-tenant endpoints sign with Microsoft's global keys. Only the tenant
// distinguishes it from a legitimate login.
func TestValidateMicrosoftTenant_RejectsForeignTenant(t *testing.T) {
	t.Parallel()
	const victimTenant = "11111111-1111-1111-1111-111111111111"
	const attackerTenant = "99999999-9999-9999-9999-999999999999"

	attacker := &oidcClaims{
		Email:    "victim@example.com",
		TenantID: attackerTenant,
		Issuer:   "https://login.microsoftonline.com/" + attackerTenant + "/v2.0",
	}

	// Deployment pinned to one tenant: a foreign tenant is refused outright.
	trusted, err := validateMicrosoftTenant(attacker, victimTenant, nil)
	require.Error(t, err, "a token from an unexpected tenant must not authenticate")
	assert.False(t, trusted)

	// Deployment with an allowlist: same outcome.
	trusted, err = validateMicrosoftTenant(attacker, "common", []string{victimTenant})
	require.Error(t, err, "a tenant outside the allowlist must not authenticate")
	assert.False(t, trusted)

	// Multi-tenant with no allowlist: the login proceeds (a documented
	// deployment mode) but the tenant is NOT trusted, so the caller must not
	// treat the email as proof of anything. This false is what stops the
	// takeover — the callback refuses to resolve a local account from it.
	trusted, err = validateMicrosoftTenant(attacker, "common", nil)
	require.NoError(t, err)
	assert.False(t, trusted, "an arbitrary tenant's email must never be treated as attested")
}

func TestValidateMicrosoftTenant_AcceptsTrustedTenant(t *testing.T) {
	t.Parallel()
	const tenant = "11111111-1111-1111-1111-111111111111"
	claims := &oidcClaims{
		Email:    "ada@contoso.com",
		TenantID: tenant,
		Issuer:   "https://login.microsoftonline.com/" + tenant + "/v2.0",
	}

	for _, tc := range []struct {
		name       string
		configured string
		allowed    []string
	}{
		{"pinned tenant", tenant, nil},
		{"pinned tenant, case-insensitive", "11111111-1111-1111-1111-111111111111", nil},
		{"allowlisted under common", "common", []string{"other-tenant", tenant}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			trusted, err := validateMicrosoftTenant(claims, tc.configured, tc.allowed)
			require.NoError(t, err)
			assert.True(t, trusted, "a tenant the operator trusts may assert an email")
		})
	}
}

// TestValidateMicrosoftTenant_RejectsIssuerTenantMismatch pins the
// defence-in-depth check: a token may not claim one tenant in `iss` and another
// in `tid`. Without it, `tid` could be forged to match the allowlist while the
// token actually came from elsewhere.
func TestValidateMicrosoftTenant_RejectsIssuerTenantMismatch(t *testing.T) {
	t.Parallel()
	const tenant = "11111111-1111-1111-1111-111111111111"

	_, err := validateMicrosoftTenant(&oidcClaims{
		TenantID: tenant,
		Issuer:   "https://login.microsoftonline.com/99999999-9999-9999-9999-999999999999/v2.0",
	}, tenant, nil)
	assert.Error(t, err, "iss and tid must agree")

	_, err = validateMicrosoftTenant(&oidcClaims{
		TenantID: tenant,
		Issuer:   "https://evil.example.com/" + tenant + "/v2.0",
	}, tenant, nil)
	assert.Error(t, err, "iss must be a Microsoft issuer")

	_, err = validateMicrosoftTenant(&oidcClaims{
		Issuer: "https://login.microsoftonline.com/" + tenant + "/v2.0",
	}, tenant, nil)
	assert.Error(t, err, "a token with no tid cannot be placed in any tenant")
}

// TestAllowUnverifiedProviderEmail pins the compatibility escape hatch. The
// point of these cases is that the flag is NOT "turn the check off" — if it
// were, setting it would restore the CVE verbatim. An unattested address may
// sign up fresh or return to an account its own provider already owns, and
// nothing else.
func TestAllowUnverifiedProviderEmail(t *testing.T) {
	t.Parallel()

	newProvider := func(allow bool) *httpProvider {
		logger := zerolog.Nop()
		return &httpProvider{
			Config:       &config.Config{OAuthAllowUnverifiedProviderEmail: allow},
			Dependencies: Dependencies{Log: &logger},
		}
	}

	passwordAccount := &schemas.User{SignupMethods: constants.AuthRecipeMethodBasicAuth}
	googleAccount := &schemas.User{SignupMethods: constants.AuthRecipeMethodGoogle}
	microsoftAccount := &schemas.User{SignupMethods: constants.AuthRecipeMethodMicrosoft}
	mixedAccount := &schemas.User{
		SignupMethods: constants.AuthRecipeMethodBasicAuth + "," + constants.AuthRecipeMethodMicrosoft,
	}

	t.Run("default refuses everything unattested", func(t *testing.T) {
		t.Parallel()
		h := newProvider(false)
		assert.False(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodMicrosoft, nil, false),
			"no signup from an unattested address by default")
		assert.False(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodMicrosoft, microsoftAccount, true),
			"not even a returning user of the same provider by default")
	})

	t.Run("compatibility mode still blocks cross-credential takeover", func(t *testing.T) {
		t.Parallel()
		h := newProvider(true)

		// This IS nOAuth: an unattested Entra address reaching for an account a
		// password (or another provider) owns. Must stay refused with the flag on.
		assert.False(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodMicrosoft, passwordAccount, true),
			"an unattested email must never reach a password account")
		assert.False(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodMicrosoft, googleAccount, true),
			"an unattested email must never reach another provider's account")
	})

	t.Run("compatibility mode keeps existing deployments working", func(t *testing.T) {
		t.Parallel()
		h := newProvider(true)

		assert.True(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodMicrosoft, nil, false),
			"a brand-new account selects nobody, so it harms nobody")
		assert.True(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodMicrosoft, microsoftAccount, true),
			"a returning user of this same provider must keep working")
		assert.True(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodMicrosoft, mixedAccount, true),
			"an account that already linked this provider is already this provider's")
	})

	t.Run("an inconsistent storage result fails closed", func(t *testing.T) {
		t.Parallel()
		h := newProvider(true)
		// found==true with no row is a storage inconsistency, not a signup.
		// Guessing "signup" here would hand an unattested address a free pass.
		assert.False(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodMicrosoft, nil, true))
	})

	t.Run("signup-method matching is exact, not substring", func(t *testing.T) {
		t.Parallel()
		h := newProvider(true)
		// twitch/twitter are the near-miss pair; a substring check would let one
		// provider inherit the other's accounts.
		twitchAccount := &schemas.User{SignupMethods: constants.AuthRecipeMethodTwitch}
		assert.False(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodTwitter, twitchAccount, true))
		assert.True(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodTwitch, twitchAccount, true))

		// Whitespace in a stored list must not defeat the match either way.
		spaced := &schemas.User{SignupMethods: "basic_auth, microsoft"}
		assert.True(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodMicrosoft, spaced, true))
		assert.False(t, h.allowUnverifiedProviderEmail(constants.AuthRecipeMethodGoogle, spaced, true))
	})
}

// TestValidateMicrosoftTenant_XmsEdovIsTheEmailSignal documents that on the
// multi-tenant endpoints the ONLY per-token attestation Microsoft offers is
// xms_edov ("email domain owner verified") — there is no `email_verified` to
// fall back on, which is exactly why the original code trusted an unattested
// address.
func TestValidateMicrosoftTenant_XmsEdovIsTheEmailSignal(t *testing.T) {
	t.Parallel()
	const tenant = "99999999-9999-9999-9999-999999999999"
	payload := `{
		"iss": "https://login.microsoftonline.com/` + tenant + `/v2.0",
		"tid": "` + tenant + `",
		"email": "victim@example.com",
		"xms_edov": true
	}`
	claims := &oidcClaims{}
	require.NoError(t, json.Unmarshal([]byte(payload), claims))

	trusted, err := validateMicrosoftTenant(claims, "common", nil)
	require.NoError(t, err)
	assert.False(t, trusted, "an unknown tenant is still not trusted...")
	assert.True(t, bool(claims.XmsEdov), "...but xms_edov attests the address independently")

	// And a v2 token without it — the nOAuth payload — attests nothing.
	claims = &oidcClaims{}
	require.NoError(t, json.Unmarshal([]byte(`{"email":"victim@example.com"}`), claims))
	assert.False(t, bool(claims.XmsEdov))
	assert.False(t, bool(claims.EmailVerified))
}
