package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/storage"
)

func TestMaskInternalError(t *testing.T) {
	driverErr := &storage.Error{Err: errors.New(`pq: duplicate key value violates unique constraint "authorizer_users_email_key"`)}

	t.Run("storage error is masked", func(t *testing.T) {
		msg, id, masked := MaskInternalError(driverErr)
		assert.True(t, masked)
		assert.NotEmpty(t, id)
		assert.Equal(t, "internal server error (error_id: "+id+")", msg)
		assert.NotContains(t, msg, "duplicate key")
	})

	t.Run("wrapped storage error is masked", func(t *testing.T) {
		_, _, masked := MaskInternalError(fmt.Errorf("add user: %w", driverErr))
		assert.True(t, masked)
	})

	t.Run("each masked error gets its own id", func(t *testing.T) {
		_, a, _ := MaskInternalError(driverErr)
		_, b, _ := MaskInternalError(driverErr)
		assert.NotEqual(t, a, b)
	})

	t.Run("typed error is never masked", func(t *testing.T) {
		_, _, masked := MaskInternalError(fmt.Errorf("%w: %w", InvalidArgument("bad email"), driverErr))
		assert.False(t, masked, "a deliberately classified error keeps its message")
	})

	t.Run("plain untyped error is left alone", func(t *testing.T) {
		_, _, masked := MaskInternalError(errors.New("invalid email"))
		assert.False(t, masked)
	})

	t.Run("SAML URL validation keeps the field name in the typed message", func(t *testing.T) {
		// Transports surface only the typed message, so the field context
		// must live inside it rather than in a fmt.Errorf wrapper.
		bad := "http://idp.example.com"
		_, _, _, err := validateOptionalSAMLFields(&bad, nil, nil)
		var se *Error
		assert.ErrorAs(t, err, &se)
		assert.Equal(t, "sp_entity_id must be a valid https URL", se.Error())
	})

	t.Run("nil", func(t *testing.T) {
		_, _, masked := MaskInternalError(nil)
		assert.False(t, masked)
	})
}
