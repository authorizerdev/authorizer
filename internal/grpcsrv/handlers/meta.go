// Package handlers contains gRPC service handler implementations. Each
// service is a thin transport adapter: pull RequestMetadata out of the gRPC
// context, delegate to internal/service, project the result into the
// proto-generated response type.
package handlers

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/grpcsrv/transport"
	"github.com/authorizerdev/authorizer/internal/service"

	metav1 "github.com/authorizerdev/authorizer/gen/go/authorizer/meta/v1"
)

// MetaHandler implements authorizer.meta.v1.MetaService.
type MetaHandler struct {
	metav1.UnimplementedMetaServiceServer
	Service service.Provider
}

// GetMeta delegates to service.Meta and projects the GraphQL Meta model into
// the proto GetMetaResponse.
func (h *MetaHandler) GetMeta(ctx context.Context, _ *metav1.GetMetaRequest) (*metav1.GetMetaResponse, error) {
	m, _, err := h.Service.Meta(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	return &metav1.GetMetaResponse{
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
	}, nil
}
