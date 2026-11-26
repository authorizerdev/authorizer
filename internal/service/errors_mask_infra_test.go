package service

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/internalerr"
	"github.com/authorizerdev/authorizer/internal/storage"
)

// TestMaskInternalError_InfraProviders covers issue #356's non-storage gap:
// Redis/memory-store, OpenFGA, JWT-signing, SMTP and SMS errors are marked at
// their provider boundary (internalerr.Wrap) and must be masked exactly like
// storage errors, while deliberate user-facing messages stay untouched.
func TestMaskInternalError_InfraProviders(t *testing.T) {
	cases := map[string]error{
		"memory_store": internalerr.Wrap("memory_store", errors.New("dial tcp 10.0.0.5:6379: connect: connection refused")),
		"openfga":      internalerr.Wrap("openfga", errors.New(`openfga.Check: pq: relation "tuple" does not exist`)),
		"token":        internalerr.Wrap("token", errors.New("asn1: structure error: tags don't match")),
		"email":        internalerr.Wrap("email", errors.New("gomail: could not send email 1: 535 5.7.8 auth failed for smtp.internal:587")),
		"sms":          internalerr.Wrap("sms", errors.New("twilio: status 401 account AC123 suspended")),
	}
	for source, err := range cases {
		t.Run(source, func(t *testing.T) {
			msg, id, masked := MaskInternalError(fmt.Errorf("login: %w", err))
			assert.True(t, masked)
			assert.Equal(t, "internal server error (error_id: "+id+")", msg)
			assert.NotContains(t, msg, err.Error())

			var buf bytes.Buffer
			log := zerolog.New(&buf)
			MaskedErrorEvent(&log, err, id).Msg("masked")
			assert.Contains(t, buf.String(), `"error_id":"`+id+`"`)
			assert.Contains(t, buf.String(), `"error_source":"`+source+`"`)
			assert.Contains(t, buf.String(), `"level":"error"`)
		})
	}

	t.Run("typed error wrapping an internal one keeps its message", func(t *testing.T) {
		_, _, masked := MaskInternalError(fmt.Errorf("%w: %w", Unauthenticated("unauthorized"), cases["memory_store"]))
		assert.False(t, masked)
	})

	t.Run("deliberate untyped message is left alone", func(t *testing.T) {
		_, _, masked := MaskInternalError(errors.New("unauthorized"))
		assert.False(t, masked)
	})

	t.Run("storage errors report source storage", func(t *testing.T) {
		var buf bytes.Buffer
		log := zerolog.New(&buf)
		MaskedErrorEvent(&log, &storage.Error{Err: errors.New("boom")}, "id").Msg("masked")
		assert.Contains(t, buf.String(), `"error_source":"storage"`)
	})
}
