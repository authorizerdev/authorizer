package webauthn

import (
	"context"
	"encoding/base64"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// webauthnUser adapts an Authorizer user + its stored credentials to the
// go-webauthn User interface.
type webauthnUser struct {
	user  *schemas.User
	creds []gowebauthn.Credential
}

// newWebauthnUser loads a user's stored credentials and wraps them for the
// go-webauthn library.
func (p *provider) newWebauthnUser(ctx context.Context, user *schemas.User) (*webauthnUser, error) {
	stored, err := p.deps.StorageProvider.ListWebauthnCredentialsByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	creds := make([]gowebauthn.Credential, 0, len(stored))
	for _, c := range stored {
		cred, err := toWebauthnCredential(c)
		if err != nil {
			// A single unparsable row must not block the whole ceremony.
			p.deps.Log.Debug().Err(err).Str("credential_id", c.CredentialID).Msg("Skipping unparsable credential")
			continue
		}
		creds = append(creds, cred)
	}
	return &webauthnUser{user: user, creds: creds}, nil
}

// WebAuthnID is the stable user handle. The user's UUID is used verbatim so it
// resolves consistently across registration and login ceremonies.
func (u *webauthnUser) WebAuthnID() []byte { return []byte(u.user.ID) }

// WebAuthnName is a human-palatable identifier; email, falling back to the id.
func (u *webauthnUser) WebAuthnName() string {
	if email := refs.StringValue(u.user.Email); email != "" {
		return email
	}
	return u.user.ID
}

// WebAuthnDisplayName is shown by the authenticator UI.
func (u *webauthnUser) WebAuthnDisplayName() string {
	if name := refs.StringValue(u.user.GivenName); name != "" {
		return name
	}
	return u.WebAuthnName()
}

// WebAuthnCredentials returns the user's registered credentials.
func (u *webauthnUser) WebAuthnCredentials() []gowebauthn.Credential { return u.creds }

// credentialDescriptors returns the exclusion/allow list for this user.
func (u *webauthnUser) credentialDescriptors() []protocol.CredentialDescriptor {
	descriptors := make([]protocol.CredentialDescriptor, len(u.creds))
	for i, c := range u.creds {
		descriptors[i] = c.Descriptor()
	}
	return descriptors
}

// toWebauthnCredential reconstructs a go-webauthn credential from storage. The
// Flags byte is restored so the library's backup-eligibility consistency check
// (a login-time invariant) passes for synced passkeys.
func toWebauthnCredential(c *schemas.WebauthnCredential) (gowebauthn.Credential, error) {
	id, err := base64.StdEncoding.DecodeString(c.CredentialID)
	if err != nil {
		return gowebauthn.Credential{}, err
	}
	pub, err := base64.StdEncoding.DecodeString(c.PublicKey)
	if err != nil {
		return gowebauthn.Credential{}, err
	}
	aaguid, err := base64.StdEncoding.DecodeString(c.AAGUID)
	if err != nil {
		return gowebauthn.Credential{}, err
	}
	transports := []protocol.AuthenticatorTransport{}
	for _, t := range c.ParsedTransports() {
		transports = append(transports, protocol.AuthenticatorTransport(t))
	}
	return gowebauthn.Credential{
		ID:        id,
		PublicKey: pub,
		Transport: transports,
		Flags:     gowebauthn.CredentialFlagsFromMsgpByte(byte(c.Flags)),
		Authenticator: gowebauthn.Authenticator{
			AAGUID:    aaguid,
			SignCount: uint32(c.SignCount),
		},
	}, nil
}
