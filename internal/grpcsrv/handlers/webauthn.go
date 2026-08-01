package handlers

import (
	"context"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/grpcsrv/transport"
)

// TOTP enrollment and WebAuthn/passkey RPCs. Thin delegations to the service
// layer, which owns permission resolution — in particular the dual-mode
// (bearer token OR MFA session cookie) identification shared with the
// Email/SMS OTP setup paths.

func projectWebauthnCredentialInfo(c *model.WebauthnCredentialInfo) *authorizerv1.WebauthnCredentialInfo {
	if c == nil {
		return nil
	}
	return &authorizerv1.WebauthnCredentialInfo{
		Id:         c.ID,
		Name:       c.Name,
		Transports: c.Transports,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
		LastUsedAt: c.LastUsedAt,
	}
}

// TotpMfaSetup delegates to service.TOTPMFASetup. Mirrors EmailOtpMfaSetup:
// empty email/phone_number collapse to nil so the service falls back to the
// authenticated caller rather than treating "" as an identifier.
func (h *AuthorizerHandler) TotpMfaSetup(ctx context.Context, req *authorizerv1.TotpMfaSetupRequest) (*authorizerv1.AuthResponse, error) {
	res, side, err := h.Service.TOTPMFASetup(ctx, transport.MetaFromGRPC(ctx), &model.OtpMfaSetupRequest{
		Email:       optionalString(req.GetEmail()),
		PhoneNumber: optionalString(req.GetPhoneNumber()),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return projectAuthResponse(res), nil
}

// WebauthnRegistrationOptions delegates to service.WebauthnRegistrationOptions.
func (h *AuthorizerHandler) WebauthnRegistrationOptions(ctx context.Context, req *authorizerv1.WebauthnRegistrationOptionsRequest) (*authorizerv1.WebauthnRegistrationOptionsResponse, error) {
	res, err := h.Service.WebauthnRegistrationOptions(ctx, transport.MetaFromGRPC(ctx),
		optionalString(req.GetEmail()), optionalString(req.GetPhoneNumber()))
	if err != nil {
		return nil, err
	}
	return &authorizerv1.WebauthnRegistrationOptionsResponse{Options: res.Options}, nil
}

// WebauthnRegistrationVerify delegates to service.WebauthnRegistrationVerify.
// access_token is populated only on the MFA-session-cookie enrollment path.
func (h *AuthorizerHandler) WebauthnRegistrationVerify(ctx context.Context, req *authorizerv1.WebauthnRegistrationVerifyRequest) (*authorizerv1.AuthResponse, error) {
	res, side, err := h.Service.WebauthnRegistrationVerify(ctx, transport.MetaFromGRPC(ctx), &model.WebauthnRegistrationVerifyRequest{
		Name:        req.Name,
		Credential:  req.GetCredential(),
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		State:       req.State,
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return projectAuthResponse(res), nil
}

// WebauthnLoginOptions delegates to service.WebauthnLoginOptions. An empty
// email selects the usernameless (discoverable credential) ceremony.
func (h *AuthorizerHandler) WebauthnLoginOptions(ctx context.Context, req *authorizerv1.WebauthnLoginOptionsRequest) (*authorizerv1.WebauthnLoginOptionsResponse, error) {
	res, err := h.Service.WebauthnLoginOptions(ctx, transport.MetaFromGRPC(ctx), optionalString(req.GetEmail()))
	if err != nil {
		return nil, err
	}
	return &authorizerv1.WebauthnLoginOptionsResponse{Options: res.Options}, nil
}

// WebauthnLoginVerify delegates to service.WebauthnLoginVerify.
func (h *AuthorizerHandler) WebauthnLoginVerify(ctx context.Context, req *authorizerv1.WebauthnLoginVerifyRequest) (*authorizerv1.AuthResponse, error) {
	res, side, err := h.Service.WebauthnLoginVerify(ctx, transport.MetaFromGRPC(ctx), &model.WebauthnLoginVerifyRequest{
		State:      req.State,
		Credential: req.GetCredential(),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return projectAuthResponse(res), nil
}

// WebauthnCredentials delegates to service.WebauthnCredentials, listing the
// authenticated caller's own passkeys.
func (h *AuthorizerHandler) WebauthnCredentials(ctx context.Context, _ *authorizerv1.WebauthnCredentialsRequest) (*authorizerv1.WebauthnCredentialsResponse, error) {
	res, err := h.Service.WebauthnCredentials(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	items := make([]*authorizerv1.WebauthnCredentialInfo, 0, len(res))
	for _, item := range res {
		items = append(items, projectWebauthnCredentialInfo(item))
	}
	return &authorizerv1.WebauthnCredentialsResponse{WebauthnCredentials: items}, nil
}

// WebauthnDeleteCredential delegates to service.WebauthnDeleteCredential,
// which scopes the delete to the authenticated caller's own credentials.
func (h *AuthorizerHandler) WebauthnDeleteCredential(ctx context.Context, req *authorizerv1.WebauthnDeleteCredentialRequest) (*authorizerv1.WebauthnDeleteCredentialResponse, error) {
	res, err := h.Service.WebauthnDeleteCredential(ctx, transport.MetaFromGRPC(ctx), req.GetId())
	if err != nil {
		return nil, err
	}
	return &authorizerv1.WebauthnDeleteCredentialResponse{Message: res.Message}, nil
}
