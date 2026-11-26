package sms

import (
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/internalerr"
)

type failingSMS struct{ err error }

func (f failingSMS) SendSMS(string, string) error { return f.err }

func TestErrorBoundary_MarksErrorsInternal(t *testing.T) {
	cause := errors.New("twilio: status 401 account AC123 suspended")
	err := (&errorBoundary{next: failingSMS{err: cause}}).SendSMS("+10000000000", "hi")
	assert.True(t, internalerr.Is(err))
	assert.ErrorIs(t, err, cause)
	assert.NoError(t, (&errorBoundary{next: failingSMS{}}).SendSMS("+10000000000", "hi"))
}

// TestNew_UnconfiguredStaysNil: wrapping must not turn "SMS not configured"
// (a nil Provider) into a non-nil interface.
func TestNew_UnconfiguredStaysNil(t *testing.T) {
	log := zerolog.Nop()
	p, err := New(&config.Config{}, &Dependencies{Log: &log})
	require.NoError(t, err)
	assert.Nil(t, p)
}
