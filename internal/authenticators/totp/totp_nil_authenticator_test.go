package totp

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// nilAuthenticatorStore reproduces the DynamoDB provider's contract for a user
// with no enrolled TOTP authenticator: (nil, nil) rather than an error. Every
// other backend returns a not-found error here, and code written against that
// contract dereferences the nil row and panics.
type nilAuthenticatorStore struct{ storage.Provider }

func (nilAuthenticatorStore) GetAuthenticatorDetailsByUserId(_ context.Context, _, _ string) (*schemas.Authenticator, error) {
	return nil, nil
}

func newNilStoreProvider(t *testing.T) *provider {
	t.Helper()
	l := zerolog.Nop()
	p, err := NewProvider(&Dependencies{
		Log:             &l,
		StorageProvider: nilAuthenticatorStore{},
		EncryptionKey:   "test-key",
	})
	require.NoError(t, err)
	return p
}

// TestValidateHandlesMissingAuthenticator pins that a missing enrolment is a
// failed validation, not a panic.
func TestValidateHandlesMissingAuthenticator(t *testing.T) {
	p := newNilStoreProvider(t)

	require.NotPanics(t, func() {
		ok, err := p.Validate(context.Background(), "123456", "user-with-no-totp")
		assert.False(t, ok)
		assert.NoError(t, err)
	})
}

// TestValidateRecoveryCodeHandlesMissingAuthenticator pins the same contract on
// the recovery-code path, which dereferenced RecoveryCodes on the nil row.
func TestValidateRecoveryCodeHandlesMissingAuthenticator(t *testing.T) {
	p := newNilStoreProvider(t)

	require.NotPanics(t, func() {
		ok, err := p.ValidateRecoveryCode(context.Background(), "some-code", "user-with-no-totp")
		assert.False(t, ok)
		assert.NoError(t, err)
	})
}
