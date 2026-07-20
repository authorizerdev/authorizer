// Package handlers — projection helpers in this file convert the
// GraphQL/storage model types returned by service.* into the proto wire types.
// Centralised here so each handler can stay focused on its delegation pattern.
package handlers

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"

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
		Id:                       u.ID,
		Email:                    refs.StringValue(u.Email),
		EmailVerified:            u.EmailVerified,
		SignupMethods:            u.SignupMethods,
		GivenName:                refs.StringValue(u.GivenName),
		FamilyName:               refs.StringValue(u.FamilyName),
		MiddleName:               refs.StringValue(u.MiddleName),
		Nickname:                 refs.StringValue(u.Nickname),
		PreferredUsername:        refs.StringValue(u.PreferredUsername),
		Gender:                   refs.StringValue(u.Gender),
		Birthdate:                refs.StringValue(u.Birthdate),
		PhoneNumber:              refs.StringValue(u.PhoneNumber),
		PhoneNumberVerified:      u.PhoneNumberVerified,
		Picture:                  refs.StringValue(u.Picture),
		Roles:                    u.Roles,
		CreatedAt:                refs.Int64Value(u.CreatedAt),
		UpdatedAt:                refs.Int64Value(u.UpdatedAt),
		RevokedTimestamp:         refs.Int64Value(u.RevokedTimestamp),
		IsMultiFactorAuthEnabled: refs.BoolValue(u.IsMultiFactorAuthEnabled),
		HasSkippedMfaSetupAt:     refs.Int64Value(u.HasSkippedMfaSetupAt),
		MfaLockedAt:              refs.Int64Value(u.MfaLockedAt),
	}
	if u.AppData != nil {
		out.AppData = mapToAppData(u.AppData)
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
		Message:                      a.Message,
		ShouldShowEmailOtpScreen:     refs.BoolValue(a.ShouldShowEmailOtpScreen),
		ShouldShowMobileOtpScreen:    refs.BoolValue(a.ShouldShowMobileOtpScreen),
		ShouldShowTotpScreen:         refs.BoolValue(a.ShouldShowTotpScreen),
		AccessToken:                  refs.StringValue(a.AccessToken),
		IdToken:                      refs.StringValue(a.IDToken),
		RefreshToken:                 refs.StringValue(a.RefreshToken),
		ExpiresIn:                    refs.Int64Value(a.ExpiresIn),
		User:                         projectUser(a.User),
		AuthenticatorScannerImage:    refs.StringValue(a.AuthenticatorScannerImage),
		AuthenticatorSecret:          refs.StringValue(a.AuthenticatorSecret),
		AuthenticatorRecoveryCodes:   derefStringSlice(a.AuthenticatorRecoveryCodes),
		ShouldOfferWebauthnMfaVerify: refs.BoolValue(a.ShouldOfferWebauthnMfaVerify),
		ShouldOfferWebauthnMfaSetup:  refs.BoolValue(a.ShouldOfferWebauthnMfaSetup),
		ShouldOfferEmailOtpMfaSetup:  refs.BoolValue(a.ShouldOfferEmailOtpMfaSetup),
		ShouldOfferSmsOtpMfaSetup:    refs.BoolValue(a.ShouldOfferSmsOtpMfaSetup),
	}
}

// mapToAppData converts a free-form map (GraphQL's `Map` for user app_data, or
// a JWT claims bag) into the proto AppData wrapper around
// google.protobuf.Struct. JSON is the lingua franca — it matches how AppData is
// persisted today and tolerates anything the JWT library or storage layer
// produces.
//
// A conversion failure (unmarshalable value, or a value Struct cannot
// represent) collapses to nil rather than failing the whole response: app_data
// is advisory metadata and must never take down an otherwise-valid User or
// claims payload. The inputs are produced by our own storage/JWT layers, so a
// failure here indicates corrupt data worth surfacing out-of-band, not a
// client error.
func mapToAppData(m map[string]any) *authorizerv1.AppData {
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
	return &authorizerv1.AppData{Value: s}
}

// appDataToMap is the inverse of mapToAppData: it unwraps the proto AppData
// (google.protobuf.Struct) back into a free-form map for the model layer.
// Returns nil for an absent/empty AppData so optional semantics are preserved.
func appDataToMap(in *authorizerv1.AppData) map[string]any {
	if in == nil || in.GetValue() == nil {
		return nil
	}
	return in.GetValue().AsMap()
}

// optionalString maps a proto3 scalar string onto the model layer's nullable
// *string: an empty wire value means "unset" and collapses to nil, matching
// how GraphQL omits an absent optional input field.
func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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

// protoToModelRequiredRelations converts the proto FgaRelationInput repeated
// field into the GraphQL model.FgaRelationInput slice used by service methods.
func protoToModelRequiredRelations(in []*authorizerv1.FgaRelationInput) []*model.FgaRelationInput {
	if len(in) == 0 {
		return nil
	}
	out := make([]*model.FgaRelationInput, 0, len(in))
	for _, r := range in {
		if r == nil {
			continue
		}
		out = append(out, &model.FgaRelationInput{
			Relation: r.Relation,
			Object:   r.Object,
		})
	}
	return out
}

// protoToModelPermissionChecks converts the proto PermissionCheckInput
// repeated field (including request-scoped contextual tuples) into the
// GraphQL model slice used by service.CheckPermissions.
func protoToModelPermissionChecks(in []*authorizerv1.PermissionCheckInput) []*model.PermissionCheckInput {
	if len(in) == 0 {
		return nil
	}
	out := make([]*model.PermissionCheckInput, 0, len(in))
	for _, c := range in {
		if c == nil {
			continue
		}
		check := &model.PermissionCheckInput{
			Relation: c.Relation,
			Object:   c.Object,
		}
		for _, t := range c.ContextualTuples {
			if t == nil {
				continue
			}
			check.ContextualTuples = append(check.ContextualTuples, &model.FgaTupleInput{
				User:     t.User,
				Relation: t.Relation,
				Object:   t.Object,
			})
		}
		out = append(out, check)
	}
	return out
}
