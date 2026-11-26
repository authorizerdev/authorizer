package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/service"
)

// TestSignup_PhoneConflictOnEmailSignup: an email signup that also carries a
// phone number already held by another account is rejected by the backend
// ("user with given phone number already exists"), which since #924 is a
// masked storage error. The service must report it as a typed client error
// with the phone pre-check's wording, not as an internal server error
// (issue #356).
func TestSignup_PhoneConflictOnEmailSignup(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableEmailVerification = false
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	password := "Password@123"
	phone := "+19" + randomDigits(9)

	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		PhoneNumber:     refs.NewStringRef(phone),
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	email := "signup_conflict_" + uuid.New().String() + "@authorizer.dev"
	res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           refs.NewStringRef(email),
		PhoneNumber:     refs.NewStringRef(phone),
		Password:        password,
		ConfirmPassword: password,
	})
	require.Error(t, err)
	assert.Nil(t, res)
	var se *service.Error
	require.ErrorAs(t, err, &se, "conflict must be a typed error, not a masked storage error")
	assert.Equal(t, service.KindInvalidArgument, se.Kind)
	assert.Equal(t, "signup failed. please check your credentials or try a different method", se.Error())
}

func randomDigits(n int) string {
	digits := make([]byte, 0, n)
	for _, r := range uuid.New().String() + uuid.New().String() {
		if r >= '0' && r <= '9' {
			digits = append(digits, byte(r))
			if len(digits) == n {
				break
			}
		}
	}
	for len(digits) < n {
		digits = append(digits, '7')
	}
	return string(digits)
}
