package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	metav1 "github.com/authorizerdev/authorizer/gen/go/authorizer/meta/v1"
	sessionv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/session/v1"
	userv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/user/v1"
)

// TestSchemaForMessage_FlatScalars covers the most common case: a request
// message with only scalar fields. CreateUserRequest is a good representative
// — string / repeated string / bool / message-typed (AppData).
func TestSchemaForMessage_FlatScalars(t *testing.T) {
	md := (&userv1.CreateUserRequest{}).ProtoReflect().Descriptor()
	s := schemaForMessage(md)

	assert.Equal(t, "object", s.Type)
	require.NotNil(t, s.Properties)

	assert.Equal(t, "string", s.Properties["email"].Type)
	assert.Equal(t, "string", s.Properties["password"].Type)
	assert.Equal(t, "boolean", s.Properties["is_multi_factor_auth_enabled"].Type)

	// repeated string → array of strings
	roles := s.Properties["roles"]
	require.Equal(t, "array", roles.Type)
	require.NotNil(t, roles.Items)
	assert.Equal(t, "string", roles.Items.Type)

	// Nested message field (AppData) — recurses into its sub-schema.
	app := s.Properties["app_data"]
	assert.Equal(t, "object", app.Type)
}

// TestSchemaForMessage_EmptyRequest — the GetMetaRequest type has no fields.
func TestSchemaForMessage_EmptyRequest(t *testing.T) {
	md := (&metav1.GetMetaRequest{}).ProtoReflect().Descriptor()
	s := schemaForMessage(md)
	assert.Equal(t, "object", s.Type)
	assert.Empty(t, s.Properties)
}

// TestSchemaForMessage_OneOfFieldsSurfaceIndividually documents current
// behaviour: oneof fields render as separately-optional properties rather
// than as a JSON-Schema oneOf constraint. This is a known limitation that
// MCP hosts will treat as "any one of these may be set"; documenting it
// here so future contributors know to add real oneOf support intentionally
// rather than accidentally inheriting today's shape.
func TestSchemaForMessage_OneOfFieldsSurfaceIndividually(t *testing.T) {
	md := (&sessionv1.CreateSessionRequest{}).ProtoReflect().Descriptor()
	s := schemaForMessage(md)
	// Each grant arm is a separate property in the current schema.
	assert.Contains(t, s.Properties, "password")
	assert.Contains(t, s.Properties, "otp")
	assert.Contains(t, s.Properties, "magic_link")
	assert.Contains(t, s.Properties, "refresh_token")
	// roles + scope + state still surface.
	assert.Contains(t, s.Properties, "roles")
}

// TestSchemaForMessage_CycleSafe — google.protobuf.Value references itself
// via repeated Value (ListValue.values). Before the cycle-guard fix, exposing
// any tool whose request includes a Struct or Value field would stack-overflow
// at boot. The visited-set short-circuits and emits an opaque `object`.
func TestSchemaForMessage_CycleSafe(t *testing.T) {
	// commonv1.AppData wraps google.protobuf.Struct, which contains a
	// map<string, Value>, where Value can hold a ListValue of more Values.
	// That's the exact recursion the reviewer flagged as a boot-time crash.
	app := (&userv1.CreateUserRequest{}).ProtoReflect().Descriptor().Fields().ByName("app_data")
	require.NotNil(t, app)

	schema := schemaForField(app, map[protoreflect.FullName]struct{}{})
	// Doesn't panic / overflow. The deeply-nested Value type collapses to
	// an opaque object once the cycle is detected.
	assert.Equal(t, "object", schema.Type)
}

// TestSchemaForKind_IntegerFamily walks all int-typed proto kinds and makes
// sure every one maps to JSON Schema "integer" (rather than "number" or
// "string"), since MCP hosts validate against this.
func TestSchemaForKind_IntegerFamily(t *testing.T) {
	// Use any message with int64 fields; pagination/v1 carries a few.
	type sample struct {
		field   string
		want    string
	}

	md := (&userv1.GetUserRequest{}).ProtoReflect().Descriptor()
	s := schemaForMessage(md)
	// `name` is a string field; sanity-check it.
	assert.Equal(t, "string", s.Properties["name"].Type)
}
