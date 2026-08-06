package integration_tests

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestRequiredRelationsIntersectsForDelegatedCallers pins the THIRD
// authorization-decision surface.
//
// enforceRequiredRelations backs `session`, `validate_session` and
// `validate_jwt_token`. It hardcoded "user:<sub>" and never expanded the
// delegated subjects, so it answered a different question from
// CheckPermissions for the very same token: an agent with no grant of its own
// was reported as SATISFYING a required relation that check_permissions denied.
//
// Two answers to one authority question is worse than either answer alone — a
// gateway gating on required_relations would admit exactly the requests the
// permission API refuses.
func TestRequiredRelationsIntersectsForDelegatedCallers(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaAgentModel})
	require.NoError(t, err)

	delegated, _, userID := mintDelegatedViaEndpoint(t, ts, router, testAuthorizerHost(ts))

	const obj = "document:required-relations"
	setAdminCookie(t, ts)
	_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{
		// The USER can view it. The agent holds nothing.
		Tuples: []*model.FgaTupleInput{{User: "user:" + userID, Relation: "viewer", Object: obj}},
	})
	require.NoError(t, err)

	required := []*model.FgaRelationInput{{Relation: "can_view", Object: obj}}

	// Control: the permission API denies, because the agent has no grant.
	presentDelegatedToken(ts, delegated)
	chk, cErr := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
		Checks: []*model.PermissionCheckInput{{Relation: "can_view", Object: obj}},
	})
	require.NoError(t, cErr)
	require.False(t, chk.Results[0].Allowed, "control: the intersection denies the agent")

	// The same question through required_relations must give the SAME answer.
	presentDelegatedToken(ts, delegated)
	_, vErr := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
		// id_token, not access_token: the access/refresh branch requires a
		// `nonce` + live session entry, which a stateless delegated token has
		// not got, so it is rejected before required_relations is ever reached.
		// The id_token branch skips that, which is how a delegated token gets
		// to this decision surface at all.
		TokenType:         "id_token",
		Token:             delegated,
		RequiredRelations: required,
	})
	assert.Error(t, vErr,
		"validate_jwt_token must not report a relation satisfied that check_permissions denies "+
			"for the same token, relation and object")
}
