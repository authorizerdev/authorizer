package handlers

import (
	"github.com/authorizerdev/authorizer/internal/service"

	authzv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/authz/v1"
	sessionv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/session/v1"
	tokenv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/token/v1"
	userv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/user/v1"
	verificationv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/verification/v1"
)

// The handlers below embed the proto-generated UnimplementedXServer types,
// which makes them all return codes.Unimplemented for every RPC. As each
// service's operations migrate into internal/service in subsequent PRs,
// individual methods are added here and override the unimplemented stubs.
//
// Keeping every service registered (even as a stub) means:
//   - gRPC reflection lists the complete API surface from day one
//   - clients can discover capability rather than getting "service not found"
//   - the grpc-gateway mount registers all REST routes, returning
//     codes.Unimplemented → HTTP 501 for unimplemented ops
//
// All handlers receive the shared service.Provider so the wire-up is in
// place for the migration; the Service field is unused on stubs today.

type UserHandler struct {
	userv1.UnimplementedUserServiceServer
	Service service.Provider
}

type SessionHandler struct {
	sessionv1.UnimplementedSessionServiceServer
	Service service.Provider
}

type MagicLinkHandler struct {
	sessionv1.UnimplementedMagicLinkServiceServer
	Service service.Provider
}

type EmailVerificationHandler struct {
	verificationv1.UnimplementedEmailVerificationServiceServer
	Service service.Provider
}

type PasswordResetHandler struct {
	verificationv1.UnimplementedPasswordResetServiceServer
	Service service.Provider
}

type OtpChallengeHandler struct {
	verificationv1.UnimplementedOtpChallengeServiceServer
	Service service.Provider
}

type TokenHandler struct {
	tokenv1.UnimplementedTokenServiceServer
	Service service.Provider
}

type AuthzHandler struct {
	authzv1.UnimplementedAuthzServiceServer
	Service service.Provider
}
