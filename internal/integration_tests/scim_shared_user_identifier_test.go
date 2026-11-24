package integration_tests

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestSCIMSharedUserIdentifierTakeover is the GHSA-p7g3-2258-8wmp regression:
// org A's SCIM token must not rewrite the global email/phone of a user who is
// also a member of org B, because that redirects B's password recovery to an
// address A controls. Exercises the real SCIM HTTP handler over SQLite, every
// PatchOp shape that reaches the email/phone, B's view, and the recovery path.
func TestSCIMSharedUserIdentifierTakeover(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)
	base := bootSCIMServer(t, ts)
	setAdminCookie(t, ts)

	newOrg := func(prefix string) (string, string) {
		org, err := ts.GraphQLProvider.CreateOrganization(ctx, &model.CreateOrganizationRequest{Name: prefix + "-" + uuid.NewString()})
		require.NoError(t, err)
		ep, err := ts.GraphQLProvider.CreateScimEndpoint(ctx, &model.CreateScimEndpointRequest{OrgID: org.ID})
		require.NoError(t, err)
		return org.ID, ep.Token
	}
	_, tokenA := newOrg("scim-a")
	orgB, tokenB := newOrg("scim-b")

	provision := func(token string) (string, string) {
		name := "shared-" + uuid.NewString() + "@authorizer.test"
		resp, decoded := scimDo(t, base, http.MethodPost, "/scim/v2/Users", token, map[string]any{"userName": name, "active": true})
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		return decoded["id"].(string), name
	}

	sharedID, sharedEmail := provision(tokenA)
	_, err := ts.StorageProvider.AddOrgMembership(ctx, &schemas.OrgMembership{OrgID: orgB, UserID: sharedID})
	require.NoError(t, err)

	attacker := "attacker-" + uuid.NewString() + "@evil.test"
	patches := map[string][]map[string]any{
		"userName path":    {{"op": "replace", "path": "userName", "value": attacker}},
		"emails path":      {{"op": "replace", "path": "emails", "value": []map[string]any{{"value": attacker, "primary": true}}}},
		"emails filter":    {{"op": "replace", "path": `emails[type eq "work"].value`, "value": attacker}},
		"no-path userName": {{"op": "replace", "value": map[string]any{"userName": attacker}}},
		"no-path emails":   {{"op": "Add", "value": map[string]any{"emails": []map[string]any{{"value": attacker}}}}},
		"phoneNumbers":     {{"op": "replace", "path": "phoneNumbers", "value": []map[string]any{{"value": "+15550001111"}}}},
	}
	for name, ops := range patches {
		t.Run(name+" via org A -> 400 mutability", func(t *testing.T) {
			resp, decoded := scimDo(t, base, http.MethodPatch, "/scim/v2/Users/"+sharedID, tokenA, map[string]any{"Operations": ops})
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
			require.Equal(t, "mutability", decoded["scimType"])
		})
	}

	t.Run("org B still sees the original identifiers", func(t *testing.T) {
		resp, decoded := scimDo(t, base, http.MethodGet, "/scim/v2/Users/"+sharedID, tokenB, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, sharedEmail, decoded["userName"])
		u, err := ts.StorageProvider.GetUserByID(ctx, sharedID)
		require.NoError(t, err)
		require.Nil(t, u.PhoneNumber)
	})

	t.Run("password recovery cannot be redirected to the attacker address", func(t *testing.T) {
		_, err := ts.GraphQLProvider.ForgotPassword(ctx, &model.ForgotPasswordRequest{Email: &attacker})
		require.NoError(t, err) // generic response, by design
		_, err = ts.StorageProvider.GetVerificationRequestByEmail(ctx, attacker, constants.VerificationTypeForgotPassword)
		require.Error(t, err, "no reset token may be issued for the attacker's address")
	})

	t.Run("backward compatible: non-identifier edits on the shared user still work", func(t *testing.T) {
		resp, decoded := scimDo(t, base, http.MethodPatch, "/scim/v2/Users/"+sharedID, tokenA, map[string]any{"Operations": []map[string]any{
			{"op": "replace", "path": "name.givenName", "value": "Sam"},
			{"op": "replace", "path": "userName", "value": sharedEmail}, // unchanged replay
		}})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, sharedEmail, decoded["userName"])
	})

	t.Run("backward compatible: single-org user email change still works", func(t *testing.T) {
		soloID, _ := provision(tokenA)
		renamed := "renamed-" + uuid.NewString() + "@authorizer.test"
		resp, decoded := scimDo(t, base, http.MethodPatch, "/scim/v2/Users/"+soloID, tokenA, map[string]any{"Operations": []map[string]any{
			{"op": "replace", "path": "userName", "value": renamed},
		}})
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, renamed, decoded["userName"])
	})
}
