package integration_tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestDelegatedActionIsAuditedAsTheAgent proves the audit attribution end to end
// on the GRAPHQL surface, with a token minted by the real /oauth/token endpoint.
//
// The surface matters more than the assertion here. authctx.Principal is
// populated by the gRPC interceptor and nothing else, so an earlier version that
// read the actor from the context attributed every GraphQL agent action to the
// human — silently, with all its unit tests passing, because they injected a
// Principal directly. This test constructs no principal at all.
func TestDelegatedActionIsAuditedAsTheAgent(t *testing.T) {
	cfg := getTestConfig()
	ts, _ := initFGATestSetup(t, cfg)
	_, ctx := createContext(ts)

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	delegated, agentID, userID := mintDelegatedViaEndpoint(t, ts, router, testAuthorizerHost(ts))

	httpReq, err := http.NewRequest(http.MethodPost, testAuthorizerHost(ts)+"/graphql", nil)
	require.NoError(t, err)
	httpReq.Header.Set("Authorization", "Bearer "+delegated)
	httpReq.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))

	// Logout is the smallest action that writes a delegation-aware audit event.
	meta := service.RequestMetadata{
		HostURL:   testAuthorizerHost(ts),
		Request:   httpReq,
		Protocol:  constants.ProtocolGraphQL,
		IPAddress: "127.0.0.1",
		UserAgent: "delegated-agent-test",
	}
	_, _, err = ts.ServiceProvider.Logout(ctx, meta)
	require.NoError(t, err, "a delegated caller must be able to act at all")

	// Audit writes are fire-and-forget (asyncutil.Go), so poll rather than
	// assume the row has landed.
	var logs []*schemas.AuditLog
	require.Eventually(t, func() bool {
		var lErr error
		logs, _, lErr = ts.StorageProvider.ListAuditLogs(ctx, &model.Pagination{Limit: 50, Page: 1},
			map[string]interface{}{"action": constants.AuditLogoutEvent})
		return lErr == nil && len(logs) > 0
	}, 5*time.Second, 25*time.Millisecond, "no logout audit entry was ever written")

	var found bool
	for _, l := range logs {
		if l.ActorID != agentID {
			continue
		}
		found = true
		assert.Equal(t, constants.AuditActorTypeAgent, l.ActorType,
			"the actor type must say an agent did this, not a user")
		assert.Empty(t, l.ActorEmail,
			"an agent has no mailbox; the delegating user's address must not sit in the actor field")
		assert.Contains(t, l.Metadata, "delegated_user_id="+userID,
			"'for whom' must survive alongside 'who' — an agent action with no trace of its "+
				"delegating user is as useless for incident response as one with no trace of the agent")
	}
	require.True(t, found,
		"no audit entry attributed to the agent: the delegation was recorded as the user acting "+
			"alone, which is exactly the impersonation RFC 8693 distinguishes delegation from")

	// The delegating user is minted fresh by this test and never logs out on its
	// own, so any logout entry naming it as the actor is this one mis-attributed.
	for _, l := range logs {
		require.NotEqual(t, userID, l.ActorID,
			"the delegated logout was attributed to the user, not the agent")
	}
}
