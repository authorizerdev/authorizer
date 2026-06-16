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

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/grpcsrv/transport"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/service"
)

// AuthorizerHandler implements authorizer.v1.AuthorizerService. The single
// struct satisfies the entire service interface; methods become real one at
// a time. The Go type name stays "AuthorizerHandler" (not "...ServiceHandler")
// because in Go we don't repeat the "Service" suffix at the call site.
type AuthorizerHandler struct {
	authorizerv1.UnimplementedAuthorizerServiceServer
	Service service.Provider
}

// Signup delegates to service.SignUp, applies session/MFA cookie side-effects
// to the outgoing stream (grpc-gateway lifts them to Set-Cookie for REST
// callers), and projects the AuthResponse. Proto3 scalars carry no presence,
// so optional string inputs collapse empty -> nil via optionalString to match
// the GraphQL "field omitted" semantics. Signup is intentionally NOT MCP-
// exposed (it returns credentials).
func (h *AuthorizerHandler) Signup(ctx context.Context, req *authorizerv1.SignupRequest) (*authorizerv1.AuthResponse, error) {
	res, side, err := h.Service.SignUp(ctx, transport.MetaFromGRPC(ctx), &model.SignUpRequest{
		Email:                    optionalString(req.Email),
		PhoneNumber:              optionalString(req.PhoneNumber),
		Password:                 req.Password,
		ConfirmPassword:          req.ConfirmPassword,
		GivenName:                optionalString(req.GivenName),
		FamilyName:               optionalString(req.FamilyName),
		MiddleName:               optionalString(req.MiddleName),
		Nickname:                 optionalString(req.Nickname),
		Gender:                   optionalString(req.Gender),
		Birthdate:                optionalString(req.Birthdate),
		Picture:                  optionalString(req.Picture),
		Roles:                    req.Roles,
		Scope:                    req.Scope,
		RedirectURI:              optionalString(req.RedirectUri),
		IsMultiFactorAuthEnabled: &req.IsMultiFactorAuthEnabled,
		State:                    optionalString(req.State),
		AppData:                  appDataToMap(req.AppData),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return projectAuthResponse(res), nil
}

// Revoke delegates to service.Revoke and projects the result.
func (h *AuthorizerHandler) Revoke(ctx context.Context, req *authorizerv1.RevokeRequest) (*authorizerv1.RevokeResponse, error) {
	res, _, err := h.Service.Revoke(ctx, transport.MetaFromGRPC(ctx), &model.OAuthRevokeRequest{RefreshToken: req.RefreshToken})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.RevokeResponse{Message: res.Message}, nil
}

// ValidateJwtToken delegates to service.ValidateJwtToken. The JWT claims
// map (free-form) is projected to AppData (which wraps Struct) to preserve
// the existing GraphQL semantics.
func (h *AuthorizerHandler) ValidateJwtToken(ctx context.Context, req *authorizerv1.ValidateJwtTokenRequest) (*authorizerv1.ValidateJwtTokenResponse, error) {
	res, _, err := h.Service.ValidateJwtToken(ctx, transport.MetaFromGRPC(ctx), &model.ValidateJWTTokenRequest{
		TokenType:         req.TokenType,
		Token:             req.Token,
		Roles:             req.Roles,
		RequiredRelations: protoToModelRequiredRelations(req.RequiredRelations),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.ValidateJwtTokenResponse{
		IsValid: res.IsValid,
		Claims:  mapToAppData(res.Claims),
	}, nil
}

// ValidateSession delegates to service.ValidateSession.
func (h *AuthorizerHandler) ValidateSession(ctx context.Context, req *authorizerv1.ValidateSessionRequest) (*authorizerv1.ValidateSessionResponse, error) {
	res, _, err := h.Service.ValidateSession(ctx, transport.MetaFromGRPC(ctx), &model.ValidateSessionRequest{
		Cookie:            req.Cookie,
		Roles:             req.Roles,
		RequiredRelations: protoToModelRequiredRelations(req.RequiredRelations),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.ValidateSessionResponse{
		IsValid: res.IsValid,
		User:    projectUser(res.User),
	}, nil
}

// Session delegates to service.Session, applies the rotated session cookie
// to the outgoing stream, and projects the AuthResponse. SessionResponse
// carries credentials and is intentionally NOT MCP-exposed (audit C1).
func (h *AuthorizerHandler) Session(ctx context.Context, req *authorizerv1.SessionRequest) (*authorizerv1.AuthResponse, error) {
	res, side, err := h.Service.Session(ctx, transport.MetaFromGRPC(ctx), &model.SessionQueryRequest{
		Roles:             req.Roles,
		Scope:             req.Scope,
		State:             refs.NewStringRef(req.State),
		RequiredRelations: protoToModelRequiredRelations(req.RequiredRelations),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return projectAuthResponse(res), nil
}

// Profile delegates to service.Profile and projects the result into the
// proto ProfileResponse. Requires session/bearer auth (handled inside the
// service via TokenProvider.GetUserIDFromSessionOrAccessToken).
func (h *AuthorizerHandler) Profile(ctx context.Context, _ *authorizerv1.ProfileRequest) (*authorizerv1.User, error) {
	u, _, err := h.Service.Profile(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	return projectUser(u), nil
}

// CheckPermissions delegates to service.CheckPermissions and projects the
// per-check results. The subject trust gate and fail-closed semantics live
// in the service layer.
func (h *AuthorizerHandler) CheckPermissions(ctx context.Context, req *authorizerv1.CheckPermissionsRequest) (*authorizerv1.CheckPermissionsResponse, error) {
	params := &model.CheckPermissionsInput{
		Checks: protoToModelPermissionChecks(req.Checks),
	}
	if req.User != "" {
		params.User = refs.NewStringRef(req.User)
	}
	res, _, err := h.Service.CheckPermissions(ctx, transport.MetaFromGRPC(ctx), params)
	if err != nil {
		return nil, err
	}
	out := make([]*authorizerv1.PermissionCheckResult, 0, len(res.Results))
	for _, r := range res.Results {
		out = append(out, &authorizerv1.PermissionCheckResult{
			Relation: r.Relation,
			Object:   r.Object,
			Allowed:  r.Allowed,
		})
	}
	return &authorizerv1.CheckPermissionsResponse{Results: out}, nil
}

// ListPermissions delegates to service.ListPermissions and projects the
// (object, relation) pairs plus the distinct object list.
func (h *AuthorizerHandler) ListPermissions(ctx context.Context, req *authorizerv1.ListPermissionsRequest) (*authorizerv1.ListPermissionsResponse, error) {
	params := &model.ListPermissionsInput{}
	if req.Relation != "" {
		params.Relation = refs.NewStringRef(req.Relation)
	}
	if req.ObjectType != "" {
		params.ObjectType = refs.NewStringRef(req.ObjectType)
	}
	if req.User != "" {
		params.User = refs.NewStringRef(req.User)
	}
	res, _, err := h.Service.ListPermissions(ctx, transport.MetaFromGRPC(ctx), params)
	if err != nil {
		return nil, err
	}
	perms := make([]*authorizerv1.Permission, 0, len(res.Permissions))
	for _, p := range res.Permissions {
		perms = append(perms, &authorizerv1.Permission{Object: p.Object, Relation: p.Relation})
	}
	return &authorizerv1.ListPermissionsResponse{
		Objects:     res.Objects,
		Permissions: perms,
		Truncated:   res.Truncated,
	}, nil
}

// Logout delegates to service.Logout, applies any cookie side-effects to
// the outgoing gRPC stream (grpc-gateway lifts them to Set-Cookie when
// the call came in via REST), then returns the typed response.
func (h *AuthorizerHandler) Logout(ctx context.Context, _ *authorizerv1.LogoutRequest) (*authorizerv1.LogoutResponse, error) {
	res, side, err := h.Service.Logout(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	// Best-effort: cookie application is out-of-band; a SendHeader failure
	// degrades to "user has to re-auth" rather than failing the request.
	_ = transport.ApplyToGRPC(ctx, side)
	return &authorizerv1.LogoutResponse{Message: res.Message}, nil
}

// ResendVerifyEmail delegates to service.ResendVerifyEmail. Public — the
// response is generic to avoid account enumeration.
func (h *AuthorizerHandler) ResendVerifyEmail(ctx context.Context, req *authorizerv1.ResendVerifyEmailRequest) (*authorizerv1.ResendVerifyEmailResponse, error) {
	res, side, err := h.Service.ResendVerifyEmail(ctx, transport.MetaFromGRPC(ctx), &model.ResendVerifyEmailRequest{
		Email:      req.Email,
		Identifier: req.Identifier,
		State:      optionalString(req.State),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return &authorizerv1.ResendVerifyEmailResponse{Message: res.Message}, nil
}

// ResendOtp delegates to service.ResendOTP. Public. Applies any MFA-session
// cookie side-effects to the outgoing stream.
func (h *AuthorizerHandler) ResendOtp(ctx context.Context, req *authorizerv1.ResendOtpRequest) (*authorizerv1.ResendOtpResponse, error) {
	res, side, err := h.Service.ResendOTP(ctx, transport.MetaFromGRPC(ctx), &model.ResendOTPRequest{
		Email:       optionalString(req.Email),
		PhoneNumber: optionalString(req.PhoneNumber),
		State:       optionalString(req.State),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return &authorizerv1.ResendOtpResponse{Message: res.Message}, nil
}

// ForgotPassword delegates to service.ForgotPassword. Public — the response is
// generic to avoid account enumeration. Applies any MFA-session cookie
// side-effects (SMS flow) to the outgoing stream.
func (h *AuthorizerHandler) ForgotPassword(ctx context.Context, req *authorizerv1.ForgotPasswordRequest) (*authorizerv1.ForgotPasswordResponse, error) {
	res, side, err := h.Service.ForgotPassword(ctx, transport.MetaFromGRPC(ctx), &model.ForgotPasswordRequest{
		Email:       optionalString(req.Email),
		PhoneNumber: optionalString(req.PhoneNumber),
		State:       optionalString(req.State),
		RedirectURI: optionalString(req.RedirectUri),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return &authorizerv1.ForgotPasswordResponse{
		Message:                   res.Message,
		ShouldShowMobileOtpScreen: refs.BoolValue(res.ShouldShowMobileOtpScreen),
	}, nil
}

// Login delegates to service.Login, applies session/MFA cookie side-effects
// to the outgoing stream (grpc-gateway lifts them to Set-Cookie for REST
// callers), and projects the AuthResponse. Login is intentionally NOT
// MCP-exposed (it returns credentials).
func (h *AuthorizerHandler) Login(ctx context.Context, req *authorizerv1.LoginRequest) (*authorizerv1.AuthResponse, error) {
	res, side, err := h.Service.Login(ctx, transport.MetaFromGRPC(ctx), &model.LoginRequest{
		Email:       optionalString(req.Email),
		PhoneNumber: optionalString(req.PhoneNumber),
		Password:    req.Password,
		Roles:       req.Roles,
		Scope:       req.Scope,
		State:       optionalString(req.State),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return projectAuthResponse(res), nil
}

// MagicLinkLogin delegates to service.MagicLinkLogin. Public — the response is
// generic to avoid account enumeration.
func (h *AuthorizerHandler) MagicLinkLogin(ctx context.Context, req *authorizerv1.MagicLinkLoginRequest) (*authorizerv1.MagicLinkLoginResponse, error) {
	res, side, err := h.Service.MagicLinkLogin(ctx, transport.MetaFromGRPC(ctx), &model.MagicLinkLoginRequest{
		Email:       req.Email,
		Roles:       req.Roles,
		Scope:       req.Scope,
		State:       optionalString(req.State),
		RedirectURI: optionalString(req.RedirectUri),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return &authorizerv1.MagicLinkLoginResponse{Message: res.Message}, nil
}

// VerifyEmail delegates to service.VerifyEmail, applies the session cookie
// side-effect to the outgoing stream, and projects the AuthResponse.
func (h *AuthorizerHandler) VerifyEmail(ctx context.Context, req *authorizerv1.VerifyEmailRequest) (*authorizerv1.AuthResponse, error) {
	res, side, err := h.Service.VerifyEmail(ctx, transport.MetaFromGRPC(ctx), &model.VerifyEmailRequest{
		Token: req.Token,
		State: optionalString(req.State),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return projectAuthResponse(res), nil
}

// VerifyOtp delegates to service.VerifyOTP, applies the session cookie
// side-effect to the outgoing stream, and projects the AuthResponse.
func (h *AuthorizerHandler) VerifyOtp(ctx context.Context, req *authorizerv1.VerifyOtpRequest) (*authorizerv1.AuthResponse, error) {
	res, side, err := h.Service.VerifyOTP(ctx, transport.MetaFromGRPC(ctx), &model.VerifyOTPRequest{
		Email:       optionalString(req.Email),
		PhoneNumber: optionalString(req.PhoneNumber),
		Otp:         req.Otp,
		IsTotp:      &req.IsTotp,
		State:       optionalString(req.State),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return projectAuthResponse(res), nil
}

// ResetPassword delegates to service.ResetPassword. Public.
func (h *AuthorizerHandler) ResetPassword(ctx context.Context, req *authorizerv1.ResetPasswordRequest) (*authorizerv1.ResetPasswordResponse, error) {
	res, side, err := h.Service.ResetPassword(ctx, transport.MetaFromGRPC(ctx), &model.ResetPasswordRequest{
		Token:           optionalString(req.Token),
		Otp:             optionalString(req.Otp),
		PhoneNumber:     optionalString(req.PhoneNumber),
		Password:        req.Password,
		ConfirmPassword: req.ConfirmPassword,
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return &authorizerv1.ResetPasswordResponse{Message: res.Message}, nil
}

// UpdateProfile delegates to service.UpdateProfile. Requires session/bearer
// auth (enforced inside the service). On email change the service rotates the
// session via cookie side-effects (lifted to Set-Cookie for REST callers).
func (h *AuthorizerHandler) UpdateProfile(ctx context.Context, req *authorizerv1.UpdateProfileRequest) (*authorizerv1.UpdateProfileResponse, error) {
	res, side, err := h.Service.UpdateProfile(ctx, transport.MetaFromGRPC(ctx), &model.UpdateProfileRequest{
		OldPassword:              optionalString(req.OldPassword),
		NewPassword:              optionalString(req.NewPassword),
		ConfirmNewPassword:       optionalString(req.ConfirmNewPassword),
		Email:                    optionalString(req.Email),
		GivenName:                optionalString(req.GivenName),
		FamilyName:               optionalString(req.FamilyName),
		MiddleName:               optionalString(req.MiddleName),
		Nickname:                 optionalString(req.Nickname),
		Gender:                   optionalString(req.Gender),
		Birthdate:                optionalString(req.Birthdate),
		PhoneNumber:              optionalString(req.PhoneNumber),
		Picture:                  optionalString(req.Picture),
		IsMultiFactorAuthEnabled: req.IsMultiFactorAuthEnabled,
		AppData:                  appDataToMap(req.AppData),
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return &authorizerv1.UpdateProfileResponse{Message: res.Message}, nil
}

// DeactivateAccount delegates to service.DeactivateAccount. Requires
// session/bearer auth (enforced inside the service).
func (h *AuthorizerHandler) DeactivateAccount(ctx context.Context, _ *authorizerv1.DeactivateAccountRequest) (*authorizerv1.DeactivateAccountResponse, error) {
	res, _, err := h.Service.DeactivateAccount(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeactivateAccountResponse{Message: res.Message}, nil
}

// Meta delegates to service.Meta and projects the GraphQL Meta model into
// the proto MetaResponse.
func (h *AuthorizerHandler) Meta(ctx context.Context, _ *authorizerv1.MetaRequest) (*authorizerv1.Meta, error) {
	m, _, err := h.Service.Meta(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	return &authorizerv1.Meta{
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
