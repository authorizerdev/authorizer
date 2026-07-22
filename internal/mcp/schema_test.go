package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// TestSchemaForMessage_FlatScalars covers the most common case: a request
// message with only scalar fields. SignupRequest is a good representative —
// string / repeated string / message-typed (AppData). Boolean is checked
// separately against VerifyOtpRequest.is_totp: SignupRequest no longer has a
// bool field (is_multi_factor_auth_enabled was removed — security fix, see
// internal/service/signup.go).
func TestSchemaForMessage_FlatScalars(t *testing.T) {
	md := (&authorizerv1.SignupRequest{}).ProtoReflect().Descriptor()
	s := schemaForMessage(md)

	assert.Equal(t, "object", s.Type)
	require.NotNil(t, s.Properties)

	assert.Equal(t, "string", s.Properties["email"].Type)
	assert.Equal(t, "string", s.Properties["password"].Type)

	// repeated string → array of strings
	roles := s.Properties["roles"]
	require.Equal(t, "array", roles.Type)
	require.NotNil(t, roles.Items)
	assert.Equal(t, "string", roles.Items.Type)

	// Nested message field (AppData) — recurses into its sub-schema.
	app := s.Properties["app_data"]
	assert.Equal(t, "object", app.Type)

	otpMd := (&authorizerv1.VerifyOtpRequest{}).ProtoReflect().Descriptor()
	otpSchema := schemaForMessage(otpMd)
	assert.Equal(t, "boolean", otpSchema.Properties["is_totp"].Type)
}

// TestSchemaForMessage_EmptyRequest — MetaRequest has no fields.
func TestSchemaForMessage_EmptyRequest(t *testing.T) {
	md := (&authorizerv1.MetaRequest{}).ProtoReflect().Descriptor()
	s := schemaForMessage(md)
	assert.Equal(t, "object", s.Type)
	assert.Empty(t, s.Properties)
}

// TestSchemaForMessage_CycleSafe — google.protobuf.Value references itself
// via repeated Value (ListValue.values). Before the cycle-guard fix, exposing
// any tool whose request includes a Struct or Value field would stack-overflow
// at boot. The visited-set short-circuits and emits an opaque `object`.
func TestSchemaForMessage_CycleSafe(t *testing.T) {
	// AppData wraps google.protobuf.Struct, which contains a
	// map<string, Value>, where Value can hold a ListValue of more Values.
	// That's the exact recursion that would stack-overflow without the guard.
	app := (&authorizerv1.SignupRequest{}).ProtoReflect().Descriptor().Fields().ByName("app_data")
	require.NotNil(t, app)

	schema := schemaForField(app, map[protoreflect.FullName]struct{}{})
	// Doesn't panic / overflow. The deeply-nested Value type collapses to
	// an opaque object once the cycle is detected.
	assert.Equal(t, "object", schema.Type)
}

// TestSchemaForMessage_ScalarOnly walks a request that's purely scalars
// (no nested message). Profile takes no arguments at all; Session takes
// a few list-of-string + nested PermissionInput.
func TestSchemaForMessage_AllScalarKinds(t *testing.T) {
	md := (&authorizerv1.ValidateJwtTokenRequest{}).ProtoReflect().Descriptor()
	s := schemaForMessage(md)
	assert.Equal(t, "string", s.Properties["token_type"].Type)
	assert.Equal(t, "string", s.Properties["token"].Type)
	assert.Equal(t, "array", s.Properties["roles"].Type)
}

// TestSchemaForMessage_CheckPermissionsNesting proves the generator handles
// the two-level nesting introduced by the OpenFGA dual-API:
// CheckPermissionsRequest.checks[] is a repeated PermissionCheckInput, and
// each PermissionCheckInput.contextual_tuples[] is a repeated FgaTupleInput.
// We assert the schema walks array → object → array → object down to the
// leaf scalars, so an MCP host sees the full input shape (not an opaque
// object) for a batch permission check with contextual tuples.
func TestSchemaForMessage_CheckPermissionsNesting(t *testing.T) {
	md := (&authorizerv1.CheckPermissionsRequest{}).ProtoReflect().Descriptor()
	s := schemaForMessage(md)

	assert.Equal(t, "object", s.Type)
	require.NotNil(t, s.Properties)

	// Optional explicit subject — a plain scalar alongside the nested list.
	assert.Equal(t, "string", s.Properties["user"].Type)

	// checks → array of PermissionCheckInput objects.
	checks := s.Properties["checks"]
	require.Equal(t, "array", checks.Type)
	require.NotNil(t, checks.Items, "checks must describe its element schema")
	assert.Equal(t, "object", checks.Items.Type)

	// PermissionCheckInput scalar fields.
	assert.Equal(t, "string", checks.Items.Properties["relation"].Type)
	assert.Equal(t, "string", checks.Items.Properties["object"].Type)

	// contextual_tuples → array of FgaTupleInput objects (the second level
	// of nesting; this is what regressed without proper recursion).
	tuples := checks.Items.Properties["contextual_tuples"]
	require.Equal(t, "array", tuples.Type)
	require.NotNil(t, tuples.Items, "contextual_tuples must describe its element schema")
	assert.Equal(t, "object", tuples.Items.Type)

	// FgaTupleInput leaf scalars — proves we reached the bottom of the tree.
	assert.Equal(t, "string", tuples.Items.Properties["user"].Type)
	assert.Equal(t, "string", tuples.Items.Properties["relation"].Type)
	assert.Equal(t, "string", tuples.Items.Properties["object"].Type)
}
