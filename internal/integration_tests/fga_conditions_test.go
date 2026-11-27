package integration_tests

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// fgaConditionAgentModel gives agents time-limited access: an agent grant only
// holds while non_expired_grant is true. Users keep unconditional grants.
const fgaConditionAgentModel = `model
  schema 1.1
type user
type agent
type document
  relations
    define viewer: [user, agent with non_expired_grant]
    define can_view: viewer
condition non_expired_grant(current_time: timestamp, grant_time: timestamp, grant_duration: duration) {
  current_time < grant_time + grant_duration
}
`

func agentGrant(start time.Time, d string) *model.FgaConditionInput {
	return &model.FgaConditionInput{
		Name:    "non_expired_grant",
		Context: map[string]any{"grant_time": start.UTC().Format(time.RFC3339), "grant_duration": d},
	}
}

// TestFGAConditionsTimeLimitedAgentGrant drives conditional tuples end to end:
// admin writes through GraphQL, a real delegated token from /oauth/token
// checks through GraphQL, and the server clock decides expiry.
func TestFGAConditionsTimeLimitedAgentGrant(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaConditionAgentModel})
	require.NoError(t, err)

	delegated, agentID, userID := mintDelegatedViaEndpoint(t, ts, router, testAuthorizerHost(ts))
	now := time.Now()

	setAdminCookie(t, ts)
	_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{Tuples: []*model.FgaTupleInput{
		{User: "user:" + userID, Relation: "viewer", Object: "document:fresh"},
		{User: "user:" + userID, Relation: "viewer", Object: "document:stale"},
		{User: "agent:" + agentID, Relation: "viewer", Object: "document:fresh", Condition: agentGrant(now, "10m")},
		{User: "agent:" + agentID, Relation: "viewer", Object: "document:stale", Condition: agentGrant(now.Add(-time.Hour), "10m")},
	}})
	require.NoError(t, err)

	checkAs := func(t *testing.T, object string, in *model.CheckPermissionsInput) (*model.CheckPermissionsResponse, error) {
		t.Helper()
		if in == nil {
			in = &model.CheckPermissionsInput{}
		}
		in.Checks = []*model.PermissionCheckInput{{Relation: "can_view", Object: object}}
		return ts.GraphQLProvider.CheckPermissions(ctx, in)
	}

	t.Run("an agent grant inside its window allows", func(t *testing.T) {
		presentDelegatedToken(ts, delegated)
		res, err := checkAs(t, "document:fresh", nil)
		require.NoError(t, err)
		assert.True(t, res.Results[0].Allowed)
	})

	t.Run("an expired agent grant denies though the user still has access", func(t *testing.T) {
		presentDelegatedToken(ts, delegated)
		res, err := checkAs(t, "document:stale", nil)
		require.NoError(t, err)
		assert.False(t, res.Results[0].Allowed)
	})

	t.Run("a delegated token cannot supply condition context", func(t *testing.T) {
		presentDelegatedToken(ts, delegated)
		_, err := checkAs(t, "document:stale", &model.CheckPermissionsInput{
			Context: map[string]any{"grant_time": now.UTC().Format(time.RFC3339)},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "may not supply condition context")
	})

	t.Run("a delegated token cannot supply contextual tuples", func(t *testing.T) {
		presentDelegatedToken(ts, delegated)
		_, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{
				Relation: "can_view", Object: "document:stale",
				ContextualTuples: []*model.FgaTupleInput{{User: "agent:" + agentID, Relation: "viewer", Object: "document:stale"}},
			}},
		})
		require.Error(t, err, "an agent attaching a self-granting contextual tuple must be refused, not answered")
		assert.Contains(t, err.Error(), "may not supply contextual tuples")
	})

	t.Run("read returns the stored condition", func(t *testing.T) {
		setAdminCookie(t, ts)
		u, r, o := "agent:"+agentID, "viewer", "document:fresh"
		res, err := ts.GraphQLProvider.FgaReadTuples(ctx, &model.FgaReadTuplesInput{User: &u, Relation: &r, Object: &o})
		require.NoError(t, err)
		require.Len(t, res.Tuples, 1)
		require.NotNil(t, res.Tuples[0].Condition)
		assert.Equal(t, "non_expired_grant", res.Tuples[0].Condition.Name)
		assert.Equal(t, "10m", res.Tuples[0].Condition.Context["grant_duration"])
	})

	t.Run("a condition on delete is rejected", func(t *testing.T) {
		setAdminCookie(t, ts)
		_, err := ts.GraphQLProvider.FgaDeleteTuples(ctx, &model.FgaWriteTuplesInput{Tuples: []*model.FgaTupleInput{
			{User: "agent:" + agentID, Relation: "viewer", Object: "document:fresh", Condition: agentGrant(now, "10m")},
		}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "condition is not used when deleting")
	})

	t.Run("current_time cannot be stored on a tuple", func(t *testing.T) {
		setAdminCookie(t, ts)
		c := agentGrant(now, "10m")
		c.Context["current_time"] = now.UTC().Format(time.RFC3339)
		_, err := ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{Tuples: []*model.FgaTupleInput{
			{User: "agent:" + agentID, Relation: "viewer", Object: "document:frozen", Condition: c},
		}})
		require.Error(t, err, "a stored current_time would take precedence over the server clock and never expire")
		assert.Contains(t, err.Error(), "current_time cannot be stored")
	})

	t.Run("revoking by key removes the conditional grant", func(t *testing.T) {
		setAdminCookie(t, ts)
		_, err := ts.GraphQLProvider.FgaDeleteTuples(ctx, &model.FgaWriteTuplesInput{Tuples: []*model.FgaTupleInput{
			{User: "agent:" + agentID, Relation: "viewer", Object: "document:fresh"},
		}})
		require.NoError(t, err)
		presentDelegatedToken(ts, delegated)
		res, err := checkAs(t, "document:fresh", nil)
		require.NoError(t, err)
		assert.False(t, res.Results[0].Allowed)
	})
}

// TestFGAConditionsRequestContextGate covers who may send request context, and
// that non-delegated callers keep contextual tuples.
func TestFGAConditionsRequestContextGate(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaConditionAgentModel})
	require.NoError(t, err)

	userToken := testAccessToken(t, ts)
	claims, err := ts.TokenProvider.ParseJWTToken(userToken)
	require.NoError(t, err)
	userID, _ := claims["sub"].(string)

	presentUser := func() {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Authorization", "Bearer "+userToken)
	}
	check := func(in *model.CheckPermissionsInput) (*model.CheckPermissionsResponse, error) {
		in.Checks = []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:1"}}
		return ts.GraphQLProvider.CheckPermissions(ctx, in)
	}

	t.Run("a regular user cannot send context", func(t *testing.T) {
		presentUser()
		_, err := check(&model.CheckPermissionsInput{Context: map[string]any{"grant_duration": "1h"}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "may only be supplied by a super-admin")
	})

	t.Run("a regular user keeps contextual tuples", func(t *testing.T) {
		presentUser()
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{
				Relation: "can_view", Object: "document:1",
				ContextualTuples: []*model.FgaTupleInput{{User: "user:" + userID, Relation: "viewer", Object: "document:1"}},
			}},
		})
		require.NoError(t, err, "contextual tuples stay available to non-delegated callers")
		assert.True(t, res.Results[0].Allowed)
	})

	t.Run("a contextual tuple cannot carry a condition", func(t *testing.T) {
		presentUser()
		_, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			Checks: []*model.PermissionCheckInput{{
				Relation: "can_view", Object: "document:1",
				ContextualTuples: []*model.FgaTupleInput{{User: "user:" + userID, Relation: "viewer", Object: "document:1",
					Condition: agentGrant(time.Now(), "1m")}},
			}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot carry a condition")
	})

	t.Run("a super-admin may send context but not current_time", func(t *testing.T) {
		setAdminCookie(t, ts)
		subject := "user:" + userID
		_, err := check(&model.CheckPermissionsInput{User: &subject, Context: map[string]any{"current_time": "2000-01-01T00:00:00Z"}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "current_time is supplied by the server")

		setAdminCookie(t, ts)
		res, err := check(&model.CheckPermissionsInput{User: &subject, Context: map[string]any{"grant_duration": "1h"}})
		require.NoError(t, err, "a super-admin's extra context is accepted")
		assert.False(t, res.Results[0].Allowed)
	})
}

// TestFGAConditionsOnEverySurface checks that a time-limited agent grant is
// honoured identically by list_permissions and required_relations, not only by
// check_permissions — those surfaces have no request context of their own.
func TestFGAConditionsOnEverySurface(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)
	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaConditionAgentModel})
	require.NoError(t, err)
	delegated, agentID, userID := mintDelegatedViaEndpoint(t, ts, router, testAuthorizerHost(ts))
	now := time.Now()

	setAdminCookie(t, ts)
	_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{Tuples: []*model.FgaTupleInput{
		{User: "user:" + userID, Relation: "viewer", Object: "document:fresh"},
		{User: "user:" + userID, Relation: "viewer", Object: "document:stale"},
		{User: "agent:" + agentID, Relation: "viewer", Object: "document:fresh", Condition: agentGrant(now, "10m")},
		{User: "agent:" + agentID, Relation: "viewer", Object: "document:stale", Condition: agentGrant(now.Add(-time.Hour), "10m")},
	}})
	require.NoError(t, err)

	t.Run("list_permissions shows only the unexpired grant", func(t *testing.T) {
		presentDelegatedToken(ts, delegated)
		rel, typ := "can_view", "document"
		res, err := ts.GraphQLProvider.ListPermissions(ctx, &model.ListPermissionsInput{Relation: &rel, ObjectType: &typ})
		require.NoError(t, err, "a conditional tuple must not make list_permissions error")
		objects := make([]string, 0, len(res.Permissions))
		for _, p := range res.Permissions {
			objects = append(objects, p.Object)
		}
		assert.Equal(t, []string{"document:fresh"}, objects)
	})

	t.Run("stored parameters win over a super-admin's context", func(t *testing.T) {
		// Admin only: drop the delegated bearer left by the previous subtest,
		// which (correctly) makes the gate refuse context.
		ts.GinContext.Request.Header.Del("Authorization")
		setAdminCookie(t, ts)
		subject := "user:" + userID
		res, err := ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			User:    &subject,
			Context: map[string]any{"grant_time": now.UTC().Format(time.RFC3339), "grant_duration": "100h"},
			Checks:  []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:stale"}},
		})
		require.NoError(t, err)
		assert.True(t, res.Results[0].Allowed, "the user's own grant is unconditional")

		agent := "agent:" + agentID
		res, err = ts.GraphQLProvider.CheckPermissions(ctx, &model.CheckPermissionsInput{
			User:    &agent,
			Context: map[string]any{"grant_time": now.UTC().Format(time.RFC3339), "grant_duration": "100h"},
			Checks:  []*model.PermissionCheckInput{{Relation: "can_view", Object: "document:stale"}},
		})
		require.NoError(t, err)
		assert.False(t, res.Results[0].Allowed, "request context must not override the tuple's stored grant_time/grant_duration")
	})
}

// TestFGAConditionsWriteRules covers the write-time guards that keep every
// stored condition evaluable with the server clock alone.
func TestFGAConditionsWriteRules(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: fgaConditionAgentModel})
	require.NoError(t, err)

	write := func(c *model.FgaConditionInput) error {
		setAdminCookie(t, ts)
		_, err := ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{Tuples: []*model.FgaTupleInput{
			{User: "agent:a", Relation: "viewer", Object: "document:1", Condition: c},
		}})
		return err
	}

	t.Run("a missing stored parameter is rejected", func(t *testing.T) {
		err := write(&model.FgaConditionInput{Name: "non_expired_grant", Context: map[string]any{"grant_time": time.Now().UTC().Format(time.RFC3339)}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing parameters grant_duration")
	})

	t.Run("an undeclared condition is rejected", func(t *testing.T) {
		err := write(&model.FgaConditionInput{Name: "no_such_condition"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not declared")
	})

	t.Run("a fully bound condition is accepted", func(t *testing.T) {
		require.NoError(t, write(agentGrant(time.Now(), "1m")))
	})

	t.Run("a model declaring current_time with another type is rejected", func(t *testing.T) {
		setAdminCookie(t, ts)
		_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: `model
  schema 1.1
type user
type document
  relations
    define viewer: [user with after]
condition after(current_time: int, start: int) {
  current_time > start
}
`})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a timestamp")
	})
}

// TestFGAConditionsRequiredRelations checks that required_relations (session,
// validate_session, validate_jwt_token) evaluates a conditional grant with the
// server clock, the same way check_permissions does.
func TestFGAConditionsRequiredRelations(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	setAdminCookie(t, ts)
	_, err := ts.GraphQLProvider.FgaWriteModel(ctx, &model.FgaWriteModelInput{Dsl: `model
  schema 1.1
type user
type document
  relations
    define viewer: [user with non_expired_grant]
    define can_view: viewer
condition non_expired_grant(current_time: timestamp, grant_time: timestamp, grant_duration: duration) {
  current_time < grant_time + grant_duration
}
`})
	require.NoError(t, err)

	userToken := testAccessToken(t, ts)
	claims, err := ts.TokenProvider.ParseJWTToken(userToken)
	require.NoError(t, err)
	userID, _ := claims["sub"].(string)
	now := time.Now()

	setAdminCookie(t, ts)
	_, err = ts.GraphQLProvider.FgaWriteTuples(ctx, &model.FgaWriteTuplesInput{Tuples: []*model.FgaTupleInput{
		{User: "user:" + userID, Relation: "viewer", Object: "document:fresh", Condition: agentGrant(now, "10m")},
		{User: "user:" + userID, Relation: "viewer", Object: "document:stale", Condition: agentGrant(now.Add(-time.Hour), "10m")},
	}})
	require.NoError(t, err)

	for obj, wantOK := range map[string]bool{"document:fresh": true, "document:stale": false} {
		clearCookies(ts)
		ts.GinContext.Request.Header.Set("Authorization", "Bearer "+userToken)
		res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
			TokenType:         "access_token",
			Token:             userToken,
			RequiredRelations: []*model.FgaRelationInput{{Relation: "can_view", Object: obj}},
		})
		if wantOK {
			require.NoError(t, err, obj)
			assert.True(t, res.IsValid, obj)
		} else {
			assert.Error(t, err, obj)
		}
	}
}
