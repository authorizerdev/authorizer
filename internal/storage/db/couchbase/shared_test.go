package couchbase

import (
	"encoding/json"
	"testing"

	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStructToDocumentPersistsSecretFields proves the write side no longer drops
// json:"-" fields. gocb Insert/Upsert marshal the document via encoding/json, which
// honors json:"-" and previously omitted User.Password from the stored document.
func TestStructToDocumentPersistsSecretFields(t *testing.T) {
	user := &schemas.User{
		ID:            "user-1",
		Email:         refs.NewStringRef("a@b.com"),
		Password:      refs.NewStringRef("hashed-secret"),
		SignupMethods: "basic_auth",
	}

	// Baseline: plain json.Marshal (what gocb did before the fix) drops the password.
	plain := map[string]interface{}{}
	raw, err := json.Marshal(user)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &plain))
	_, hadPassword := plain["password"]
	assert.False(t, hadPassword, "sanity: encoding/json must drop json:\"-\" password (this is the bug)")

	// Fixed path: structToDocument re-adds the password under its bson key.
	doc, err := structToDocument(user)
	require.NoError(t, err)
	require.Contains(t, doc, "password", "structToDocument must persist the password field")
	assert.Equal(t, "hashed-secret", refs.StringValue(doc["password"].(*string)))

	// Shape preservation: every key/value the old json.Marshal produced must be
	// unchanged; only the previously-dropped "password" key is added.
	docJSON, err := json.Marshal(doc)
	require.NoError(t, err)
	roundTripped := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(docJSON, &roundTripped))
	for k, v := range plain {
		assert.Equal(t, v, roundTripped[k], "key %q must serialize identically to before", k)
	}
	delete(roundTripped, "password")
	assert.Equal(t, plain, roundTripped, "no keys other than password may be added or changed")
}

// TestDecodeDocumentPopulatesSecretFields proves the read side no longer drops
// json:"-" fields: Row/One unmarshal via encoding/json, which ignores json:"-".
func TestDecodeDocumentPopulatesSecretFields(t *testing.T) {
	// Simulate a document as persisted by structToDocument -> gocb (JSON bytes).
	stored := []byte(`{"_id":"user-1","email":"a@b.com","password":"hashed-secret","signup_methods":"basic_auth"}`)

	// Baseline: plain unmarshal drops the password.
	var plain schemas.User
	require.NoError(t, json.Unmarshal(stored, &plain))
	assert.Nil(t, plain.Password, "sanity: encoding/json must ignore json:\"-\" password on read")

	// Fixed path: decodeDocument populates it from the bson key.
	var got schemas.User
	require.NoError(t, decodeDocument(stored, &got))
	require.NotNil(t, got.Password)
	assert.Equal(t, "hashed-secret", *got.Password)
	assert.Equal(t, "a@b.com", refs.StringValue(got.Email))
}

// TestSecretFieldFullRoundTrip exercises the exact write->read cycle end to end.
func TestSecretFieldFullRoundTrip(t *testing.T) {
	in := &schemas.User{
		ID:       "user-1",
		Email:    refs.NewStringRef("a@b.com"),
		Password: refs.NewStringRef("hashed-secret"),
	}
	doc, err := structToDocument(in)
	require.NoError(t, err)
	persisted, err := json.Marshal(doc) // gocb Insert serialization
	require.NoError(t, err)

	var out schemas.User
	require.NoError(t, decodeDocument(persisted, &out))
	require.NotNil(t, out.Password)
	assert.Equal(t, *in.Password, *out.Password)
}

// TestNilSecretFieldRoundTrip ensures a nil password persists as null and decodes
// back to nil (clearable-on-nil semantics must match other providers).
func TestNilSecretFieldRoundTrip(t *testing.T) {
	in := &schemas.User{ID: "user-1", Email: refs.NewStringRef("a@b.com")}
	doc, err := structToDocument(in)
	require.NoError(t, err)
	require.Contains(t, doc, "password")
	assert.Nil(t, doc["password"])

	persisted, err := json.Marshal(doc)
	require.NoError(t, err)
	assert.Contains(t, string(persisted), `"password":null`)

	var out schemas.User
	require.NoError(t, decodeDocument(persisted, &out))
	assert.Nil(t, out.Password)
}

// TestStructToDocumentNoOpForPlainStructs guarantees entities without json:"-"
// fields serialize byte-identically to the previous raw-struct path.
func TestStructToDocumentNoOpForPlainStructs(t *testing.T) {
	wh := &schemas.Webhook{ID: "wh-1", EventName: "user.created", EndPoint: "https://x/y", Enabled: true}

	doc, err := structToDocument(wh)
	require.NoError(t, err)
	got, err := json.Marshal(doc)
	require.NoError(t, err)

	want, err := json.Marshal(wh)
	require.NoError(t, err)

	// Compare as maps (key order independent).
	var gotMap, wantMap map[string]interface{}
	require.NoError(t, json.Unmarshal(got, &gotMap))
	require.NoError(t, json.Unmarshal(want, &wantMap))
	assert.Equal(t, wantMap, gotMap, "plain structs must serialize identically to raw json.Marshal")
}
