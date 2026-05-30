// Package-internal projection helpers: convert the GraphQL/storage model
// types returned by service.* into the proto wire types. Centralised here
// so each handler can stay focused on its delegation pattern.
package handlers

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"

	commonv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/common/v1"
	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// projectUser converts the GraphQL User model into the proto User message.
// Optional fields (nil pointers / nil maps) collapse to zero values; the
// gateway's UseProtoNames + EmitUnpopulated configuration makes them
// visible to REST clients regardless.
func projectUser(u *model.User) *authorizerv1.User {
	if u == nil {
		return nil
	}
	out := &authorizerv1.User{
		Id:                  u.ID,
		Email:               refs.StringValue(u.Email),
		EmailVerified:       u.EmailVerified,
		SignupMethods:       u.SignupMethods,
		GivenName:           refs.StringValue(u.GivenName),
		FamilyName:          refs.StringValue(u.FamilyName),
		MiddleName:          refs.StringValue(u.MiddleName),
		Nickname:            refs.StringValue(u.Nickname),
		PreferredUsername:   refs.StringValue(u.PreferredUsername),
		Gender:              refs.StringValue(u.Gender),
		Birthdate:           refs.StringValue(u.Birthdate),
		PhoneNumber:         refs.StringValue(u.PhoneNumber),
		PhoneNumberVerified: u.PhoneNumberVerified,
		Picture:             refs.StringValue(u.Picture),
		Roles:               u.Roles,
		CreatedAt:           refs.Int64Value(u.CreatedAt),
		UpdatedAt:           refs.Int64Value(u.UpdatedAt),
		RevokedTimestamp:    refs.Int64Value(u.RevokedTimestamp),
		IsMultiFactorAuthEnabled: refs.BoolValue(u.IsMultiFactorAuthEnabled),
	}
	if u.AppData != nil {
		out.AppData = projectAppData(u.AppData)
	}
	return out
}

// projectAuthResponse converts the GraphQL AuthResponse model into the
// proto AuthResponse. Used by the Session handler; the credential fields
// (tokens, authenticator secret/recovery codes) are passed through to gRPC
// + REST callers but the proto annotation on Session intentionally keeps
// it OFF the MCP surface (security audit C1).
func projectAuthResponse(a *model.AuthResponse) *authorizerv1.AuthResponse {
	if a == nil {
		return nil
	}
	return &authorizerv1.AuthResponse{
		Message:                    a.Message,
		ShouldShowEmailOtpScreen:   refs.BoolValue(a.ShouldShowEmailOtpScreen),
		ShouldShowMobileOtpScreen:  refs.BoolValue(a.ShouldShowMobileOtpScreen),
		ShouldShowTotpScreen:       refs.BoolValue(a.ShouldShowTotpScreen),
		AccessToken:                refs.StringValue(a.AccessToken),
		IdToken:                    refs.StringValue(a.IDToken),
		RefreshToken:               refs.StringValue(a.RefreshToken),
		ExpiresIn:                  refs.Int64Value(a.ExpiresIn),
		User:                       projectUser(a.User),
		AuthenticatorScannerImage:  refs.StringValue(a.AuthenticatorScannerImage),
		AuthenticatorSecret:        refs.StringValue(a.AuthenticatorSecret),
		AuthenticatorRecoveryCodes: derefStringSlice(a.AuthenticatorRecoveryCodes),
	}
}

// projectAppData converts the free-form GraphQL Map into the proto AppData
// (which wraps a google.protobuf.Struct). The conversion uses JSON as the
// lingua franca, matching how AppData is persisted today.
func projectAppData(m map[string]any) *commonv1.AppData {
	if len(m) == 0 {
		return nil
	}
	// Round-trip via JSON so anything Struct can't represent natively
	// (e.g. nested numbers > int64) gets surfaced consistently.
	b, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	var generic map[string]any
	if err := json.Unmarshal(b, &generic); err != nil {
		return nil
	}
	s, err := structpb.NewStruct(generic)
	if err != nil {
		return nil
	}
	return &commonv1.AppData{Value: s}
}

// derefStringSlice converts []*string (GraphQL's nullable string list shape)
// into []string, dropping nil entries.
func derefStringSlice(in []*string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != nil {
			out = append(out, *s)
		}
	}
	return out
}

// protoToModelPermissions converts the proto PermissionInput repeated field
// into the GraphQL model.PermissionInput slice used by service methods.
func protoToModelPermissions(in []*authorizerv1.PermissionInput) []*model.PermissionInput {
	if len(in) == 0 {
		return nil
	}
	out := make([]*model.PermissionInput, 0, len(in))
	for _, p := range in {
		if p == nil {
			continue
		}
		out = append(out, &model.PermissionInput{
			Resource: p.Resource,
			Scope:    p.Scope,
		})
	}
	return out
}

// claimsToAppData converts a free-form JWT claims map (interface{}-valued)
// into the proto AppData wrapper around google.protobuf.Struct. JSON is the
// lingua franca — matches projectAppData's strategy and tolerates anything
// the underlying JWT library produces.
func claimsToAppData(claims map[string]any) *commonv1.AppData {
	if len(claims) == 0 {
		return nil
	}
	b, err := json.Marshal(claims)
	if err != nil {
		return nil
	}
	var generic map[string]any
	if err := json.Unmarshal(b, &generic); err != nil {
		return nil
	}
	s, err := structpb.NewStruct(generic)
	if err != nil {
		return nil
	}
	return &commonv1.AppData{Value: s}
}
