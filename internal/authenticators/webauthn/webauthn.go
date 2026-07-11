package webauthn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// challengeKeyPrefix namespaces WebAuthn challenges in the shared state store so
// they can never collide with OAuth state keys.
const challengeKeyPrefix = "webauthn_challenge:"

// ceremonyTimeout bounds how long a challenge is valid. Enforced server-side via
// the go-webauthn session Expires field (see newRP) — the memory-store key self
// cleans on its own longer TTL, but a stale challenge is rejected here first.
const ceremonyTimeout = 60 * time.Second

// newRP builds a Relying Party instance bound to the request host. For a
// self-hosted Authorizer, web/app is served from the same origin as the API, so
// the RP id is the request hostname and the sole permitted origin is the
// request's scheme+host. Deriving these from the request (rather than a static
// config value) keeps a single binary correct across every deployment host.
func (p *provider) newRP(host string) (*gowebauthn.WebAuthn, error) {
	u, err := url.Parse(host)
	if err != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("webauthn: invalid host %q", host)
	}
	return gowebauthn.New(&gowebauthn.Config{
		RPID:                  u.Hostname(),
		RPDisplayName:         "Authorizer",
		RPOrigins:             []string{u.Scheme + "://" + u.Host},
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationRequired,
		},
		Timeouts: gowebauthn.TimeoutsConfig{
			Login:        gowebauthn.TimeoutConfig{Enforce: true, Timeout: ceremonyTimeout, TimeoutUVD: ceremonyTimeout},
			Registration: gowebauthn.TimeoutConfig{Enforce: true, Timeout: ceremonyTimeout, TimeoutUVD: ceremonyTimeout},
		},
	})
}

// storeChallenge persists the ceremony session keyed by its challenge value.
func (p *provider) storeChallenge(session *gowebauthn.SessionData) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return p.deps.MemoryStoreProvider.SetState(challengeKeyPrefix+session.Challenge, string(data))
}

// consumeChallenge atomically retrieves and deletes the stored session for the
// challenge carried in the client response. Single-use: a replayed response
// finds no session and is rejected.
func (p *provider) consumeChallenge(challenge string) (*gowebauthn.SessionData, error) {
	raw, err := p.deps.MemoryStoreProvider.GetAndRemoveState(challengeKeyPrefix + challenge)
	if err != nil || raw == "" {
		return nil, fmt.Errorf("webauthn: challenge not found or already used")
	}
	var session gowebauthn.SessionData
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

// BeginRegistration starts a registration ceremony for user.
func (p *provider) BeginRegistration(ctx context.Context, host string, user *schemas.User) (string, error) {
	log := p.deps.Log.With().Str("func", "webauthn.BeginRegistration").Logger()
	rp, err := p.newRP(host)
	if err != nil {
		return "", err
	}
	waUser, err := p.newWebauthnUser(ctx, user)
	if err != nil {
		return "", err
	}
	// Exclude already-registered credentials so the same authenticator isn't
	// enrolled twice.
	creation, session, err := rp.BeginRegistration(waUser, gowebauthn.WithExclusions(waUser.credentialDescriptors()))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to begin registration")
		return "", err
	}
	if err := p.storeChallenge(session); err != nil {
		return "", err
	}
	out, err := json.Marshal(creation.Response)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// FinishRegistration verifies the attestation and persists the new credential.
func (p *provider) FinishRegistration(ctx context.Context, host string, user *schemas.User, name, responseJSON string) (*schemas.WebauthnCredential, error) {
	log := p.deps.Log.With().Str("func", "webauthn.FinishRegistration").Logger()
	rp, err := p.newRP(host)
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes([]byte(responseJSON))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse attestation response")
		return nil, fmt.Errorf("invalid credential response")
	}
	session, err := p.consumeChallenge(parsed.Response.CollectedClientData.Challenge)
	if err != nil {
		return nil, err
	}
	waUser, err := p.newWebauthnUser(ctx, user)
	if err != nil {
		return nil, err
	}
	credential, err := rp.CreateCredential(waUser, *session, parsed)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify attestation")
		return nil, fmt.Errorf("failed to verify passkey registration")
	}
	stored := fromWebauthnCredential(user.ID, strings.TrimSpace(name), credential)
	return p.deps.StorageProvider.AddWebauthnCredential(ctx, stored)
}

// BeginLogin starts a login ceremony scoped to user's own credentials.
func (p *provider) BeginLogin(ctx context.Context, host string, user *schemas.User) (string, error) {
	log := p.deps.Log.With().Str("func", "webauthn.BeginLogin").Logger()
	rp, err := p.newRP(host)
	if err != nil {
		return "", err
	}
	waUser, err := p.newWebauthnUser(ctx, user)
	if err != nil {
		return "", err
	}
	assertion, session, err := rp.BeginLogin(waUser)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to begin login")
		return "", err
	}
	if err := p.storeChallenge(session); err != nil {
		return "", err
	}
	out, err := json.Marshal(assertion.Response)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// BeginDiscoverableLogin starts a usernameless login ceremony.
func (p *provider) BeginDiscoverableLogin(ctx context.Context, host string) (string, error) {
	log := p.deps.Log.With().Str("func", "webauthn.BeginDiscoverableLogin").Logger()
	rp, err := p.newRP(host)
	if err != nil {
		return "", err
	}
	assertion, session, err := rp.BeginDiscoverableLogin()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to begin discoverable login")
		return "", err
	}
	if err := p.storeChallenge(session); err != nil {
		return "", err
	}
	out, err := json.Marshal(assertion.Response)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// FinishLogin verifies an assertion for both the usernameless (discoverable)
// and scoped ceremonies, dispatching on whether the stored session pins a user.
func (p *provider) FinishLogin(ctx context.Context, host, responseJSON string) (*schemas.User, *schemas.WebauthnCredential, error) {
	log := p.deps.Log.With().Str("func", "webauthn.FinishLogin").Logger()
	rp, err := p.newRP(host)
	if err != nil {
		return nil, nil, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes([]byte(responseJSON))
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse assertion response")
		return nil, nil, fmt.Errorf("invalid credential response")
	}
	session, err := p.consumeChallenge(parsed.Response.CollectedClientData.Challenge)
	if err != nil {
		return nil, nil, err
	}

	// Scoped ceremony: the session pins a user (BeginLogin). Verify against that
	// user's credentials directly.
	if len(session.UserID) != 0 {
		user, err := p.deps.StorageProvider.GetUserByID(ctx, string(session.UserID))
		if err != nil || user == nil {
			return nil, nil, fmt.Errorf("user not found")
		}
		waUser, err := p.newWebauthnUser(ctx, user)
		if err != nil {
			return nil, nil, err
		}
		credential, err := rp.ValidateLogin(waUser, *session, parsed)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to verify scoped assertion")
			return nil, nil, fmt.Errorf("failed to verify passkey")
		}
		stored, err := p.persistLoginResult(ctx, credential)
		if err != nil {
			return nil, nil, err
		}
		return user, stored, nil
	}

	// Usernameless (discoverable) ceremony: resolve the user from the credential
	// id carried in the assertion.
	var resolvedUser *schemas.User
	handler := func(rawID, userHandle []byte) (gowebauthn.User, error) {
		credentialID := base64.StdEncoding.EncodeToString(rawID)
		stored, err := p.deps.StorageProvider.GetWebauthnCredentialByCredentialID(ctx, credentialID)
		if err != nil || stored == nil {
			return nil, fmt.Errorf("unknown credential")
		}
		user, err := p.deps.StorageProvider.GetUserByID(ctx, stored.UserID)
		if err != nil || user == nil {
			return nil, fmt.Errorf("user not found")
		}
		resolvedUser = user
		return p.newWebauthnUser(ctx, user)
	}

	_, credential, err := rp.ValidatePasskeyLogin(handler, *session, parsed)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify discoverable assertion")
		return nil, nil, fmt.Errorf("failed to verify passkey")
	}
	stored, err := p.persistLoginResult(ctx, credential)
	if err != nil {
		return nil, nil, err
	}
	return resolvedUser, stored, nil
}

// persistLoginResult writes back the mutable post-login credential state
// (sign_count, flags, last_used_at) that go-webauthn updates on the returned
// credential — this is what makes cloned-authenticator detection durable.
func (p *provider) persistLoginResult(ctx context.Context, credential *gowebauthn.Credential) (*schemas.WebauthnCredential, error) {
	credentialID := base64.StdEncoding.EncodeToString(credential.ID)
	stored, err := p.deps.StorageProvider.GetWebauthnCredentialByCredentialID(ctx, credentialID)
	if err != nil || stored == nil {
		return nil, fmt.Errorf("credential not found")
	}
	now := time.Now().Unix()
	stored.SignCount = int64(credential.Authenticator.SignCount)
	stored.Flags = int64(credential.Flags.MsgpByte())
	stored.LastUsedAt = &now
	if _, err := p.deps.StorageProvider.UpdateWebauthnCredential(ctx, stored); err != nil {
		p.deps.Log.Debug().Err(err).Msg("Failed to update credential after login")
	}
	return stored, nil
}

// fromWebauthnCredential maps a freshly registered go-webauthn credential to the
// storage schema. Binary values are base64-standard encoded and transports are
// comma joined to keep the row uniform across every backend.
func fromWebauthnCredential(userID, name string, c *gowebauthn.Credential) *schemas.WebauthnCredential {
	transports := make([]string, len(c.Transport))
	for i, t := range c.Transport {
		transports[i] = string(t)
	}
	if name == "" {
		name = "Passkey"
	}
	return &schemas.WebauthnCredential{
		UserID:       userID,
		CredentialID: base64.StdEncoding.EncodeToString(c.ID),
		PublicKey:    base64.StdEncoding.EncodeToString(c.PublicKey),
		SignCount:    int64(c.Authenticator.SignCount),
		Flags:        int64(c.Flags.MsgpByte()),
		Transports:   strings.Join(transports, ","),
		AAGUID:       base64.StdEncoding.EncodeToString(c.Authenticator.AAGUID),
		Name:         name,
	}
}
