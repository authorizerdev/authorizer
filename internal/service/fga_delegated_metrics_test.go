package service

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/metrics"
)

// delegatedCount reads one outcome of the delegated-checks series.
func delegatedCount(op, outcome string) float64 {
	return testutil.ToFloat64(metrics.FgaDelegatedChecksTotal.WithLabelValues(op, outcome))
}

// TestNotEnforcedIsCounted covers the one outcome that reports a security
// property NOT being enforced.
//
// When the active model declares no `agent` type the intersection cannot be
// evaluated, so a delegated caller is authorized as the delegating user ALONE —
// with the agent constraint unevaluable. The request is now DENIED, but the
// counter still fires — it is what tells an operator that delegated traffic is
// arriving against a model that cannot express agent grants, whether that ends
// in a denial (default) or in unconstrained authority (the explicit opt-out).
func TestNotEnforcedIsCounted(t *testing.T) {
	p := &provider{Config: &config.Config{}}
	p.AuthzEngine = &advStubEngine{modelID: "m-no-agent", typeNames: []string{"user", "document"}}
	caller := advDelegatedCaller("alice", "bot")

	before := delegatedCount(metrics.FgaOpCheckPermissions, metrics.FgaDelegatedNotEnforced)
	_, err := p.delegationSubjects(context.Background(), caller, "user:alice", metrics.FgaOpCheckPermissions)
	require.Error(t, err, "the default is to deny when the agent half cannot be evaluated")

	assert.Equal(t, before+1, delegatedCount(metrics.FgaOpCheckPermissions, metrics.FgaDelegatedNotEnforced),
		"a delegated caller arriving at a model with no agent type must be counted either way, or "+
			"the misconfiguration is invisible until someone reports a broken integration")
}

// TestNotEnforcedIsLabelledPerOperation pins that the operation label is the
// CALLER's operation and not a hardcoded one — enumeration and yes/no checks
// must be distinguishable, since only one of them leaks resource names.
func TestNotEnforcedIsLabelledPerOperation(t *testing.T) {
	p := &provider{Config: &config.Config{}}
	p.AuthzEngine = &advStubEngine{modelID: "m-no-agent-2", typeNames: []string{"user"}}
	caller := advDelegatedCaller("alice", "bot")

	before := delegatedCount(metrics.FgaOpListPermissions, metrics.FgaDelegatedNotEnforced)
	_, err := p.delegationSubjects(context.Background(), caller, "user:alice", metrics.FgaOpListPermissions)
	require.Error(t, err)

	assert.Equal(t, before+1, delegatedCount(metrics.FgaOpListPermissions, metrics.FgaDelegatedNotEnforced))
}

// TestOrdinaryCallerIsNotCountedAsDelegated pins that the delegated series
// stays clean. If ordinary traffic leaked into it, the `not_enforced` alert an
// operator builds on it would fire constantly and be turned off.
func TestOrdinaryCallerIsNotCountedAsDelegated(t *testing.T) {
	p := &provider{}
	p.AuthzEngine = &advStubEngine{modelID: "m-no-agent-3", typeNames: []string{"user"}}

	var before float64
	for _, outcome := range []string{
		metrics.FgaDelegatedAllowed, metrics.FgaDelegatedDeniedByAgent,
		metrics.FgaDelegatedDeniedByUser, metrics.FgaDelegatedNotEnforced,
	} {
		before += delegatedCount(metrics.FgaOpCheckPermissions, outcome)
	}

	_, err := p.delegationSubjects(context.Background(), fgaCaller{subject: "user:alice"}, "user:alice", metrics.FgaOpCheckPermissions)
	require.NoError(t, err)

	var after float64
	for _, outcome := range []string{
		metrics.FgaDelegatedAllowed, metrics.FgaDelegatedDeniedByAgent,
		metrics.FgaDelegatedDeniedByUser, metrics.FgaDelegatedNotEnforced,
	} {
		after += delegatedCount(metrics.FgaOpCheckPermissions, outcome)
	}
	assert.Equal(t, before, after, "a non-delegated caller must never touch the delegated series")
}
