package resolvers

import (
	"context"

	"github.com/authorizerdev/authorizer/server/graph/model"
)

// VerifyOtpResolver resolver for verify otp mutation
func VerifyOtpResolver(ctx context.Context, params model.VerifyOTPRequest) (*model.AuthResponse, error) {
	return nil, nil
}
