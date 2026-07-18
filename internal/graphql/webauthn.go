package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// WebauthnRegistrationOptions delegates to the transport-agnostic service layer.
// Permissions: authenticated:user, or an MFA-session-cookie caller mid-offer.
func (g *graphqlProvider) WebauthnRegistrationOptions(ctx context.Context, email, phoneNumber *string) (*model.WebauthnRegistrationOptionsResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	return g.ServiceProvider.WebauthnRegistrationOptions(ctx, service.MetaFromGin(gc), email, phoneNumber)
}

// WebauthnRegistrationVerify delegates to the transport-agnostic service layer.
// Permissions: authenticated:user, or an MFA-session-cookie caller mid-offer.
func (g *graphqlProvider) WebauthnRegistrationVerify(ctx context.Context, params *model.WebauthnRegistrationVerifyRequest) (*model.AuthResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, side, err := g.ServiceProvider.WebauthnRegistrationVerify(ctx, service.MetaFromGin(gc), params)
	if err != nil {
		return nil, err
	}
	service.ApplyToGin(gc, side)
	return res, nil
}

// WebauthnLoginOptions delegates to the transport-agnostic service layer.
// Permissions: none.
func (g *graphqlProvider) WebauthnLoginOptions(ctx context.Context, email *string) (*model.WebauthnLoginOptionsResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	return g.ServiceProvider.WebauthnLoginOptions(ctx, service.MetaFromGin(gc), email)
}

// WebauthnLoginVerify delegates to the transport-agnostic service layer.
// Permissions: none.
func (g *graphqlProvider) WebauthnLoginVerify(ctx context.Context, params *model.WebauthnLoginVerifyRequest) (*model.AuthResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, side, err := g.ServiceProvider.WebauthnLoginVerify(ctx, service.MetaFromGin(gc), params)
	if err != nil {
		return nil, err
	}
	service.ApplyToGin(gc, side)
	return res, nil
}

// WebauthnCredentials delegates to the transport-agnostic service layer.
// Permissions: authenticated:user.
func (g *graphqlProvider) WebauthnCredentials(ctx context.Context) ([]*model.WebauthnCredentialInfo, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	return g.ServiceProvider.WebauthnCredentials(ctx, service.MetaFromGin(gc))
}

// WebauthnDeleteCredential delegates to the transport-agnostic service layer.
// Permissions: authenticated:user.
func (g *graphqlProvider) WebauthnDeleteCredential(ctx context.Context, id string) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	return g.ServiceProvider.WebauthnDeleteCredential(ctx, service.MetaFromGin(gc), id)
}
