// Package handlers contains the AuthorizerHandler, the single gRPC service
// handler for Authorizer's public API. Methods that have already been
// migrated into internal/service (currently just Meta) delegate there; the
// rest embed the proto-generated UnimplementedAuthorizerServer so they
// return codes.Unimplemented until their underlying service method lands.
//
// As each follow-up PR migrates one GraphQL op into internal/service, the
// corresponding stub here is replaced with a real delegation following the
// Meta pattern. Tests in internal/integration_tests/grpc_surface_test.go
// guard the Unimplemented contract until each migration ships.
package handlers

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/grpcsrv/transport"
	"github.com/authorizerdev/authorizer/internal/service"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// AuthorizerHandler implements authorizer.v1.Authorizer. The single struct
// satisfies the entire service interface; methods become real one at a time.
type AuthorizerHandler struct {
	authorizerv1.UnimplementedAuthorizerServer
	Service service.Provider
}

// Meta delegates to service.Meta and projects the GraphQL Meta model into
// the proto MetaResponse.
func (h *AuthorizerHandler) Meta(ctx context.Context, _ *authorizerv1.MetaRequest) (*authorizerv1.MetaResponse, error) {
	m, _, err := h.Service.Meta(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	return &authorizerv1.MetaResponse{
		Meta: &authorizerv1.Meta{
			Version:                            m.Version,
			ClientId:                           m.ClientID,
			IsGoogleLoginEnabled:               m.IsGoogleLoginEnabled,
			IsFacebookLoginEnabled:             m.IsFacebookLoginEnabled,
			IsGithubLoginEnabled:               m.IsGithubLoginEnabled,
			IsLinkedinLoginEnabled:             m.IsLinkedinLoginEnabled,
			IsAppleLoginEnabled:                m.IsAppleLoginEnabled,
			IsDiscordLoginEnabled:              m.IsDiscordLoginEnabled,
			IsTwitterLoginEnabled:              m.IsTwitterLoginEnabled,
			IsMicrosoftLoginEnabled:            m.IsMicrosoftLoginEnabled,
			IsTwitchLoginEnabled:               m.IsTwitchLoginEnabled,
			IsRobloxLoginEnabled:               m.IsRobloxLoginEnabled,
			IsEmailVerificationEnabled:         m.IsEmailVerificationEnabled,
			IsBasicAuthenticationEnabled:       m.IsBasicAuthenticationEnabled,
			IsMagicLinkLoginEnabled:            m.IsMagicLinkLoginEnabled,
			IsSignUpEnabled:                    m.IsSignUpEnabled,
			IsStrongPasswordEnabled:            m.IsStrongPasswordEnabled,
			IsMultiFactorAuthEnabled:           m.IsMultiFactorAuthEnabled,
			IsMobileBasicAuthenticationEnabled: m.IsMobileBasicAuthenticationEnabled,
			IsPhoneVerificationEnabled:         m.IsPhoneVerificationEnabled,
		},
	}, nil
}
