package graphql

import (
	"context"
	"errors"
	"strings"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// enforceRequiredRelations gates a request on fine-grained authorization. For
// each required (relation, object) it asks the engine whether the caller
// (subject "user:<userID>") holds that relation. Semantics:
//
//   - AND: every relation must be allowed.
//   - Fail-closed: an engine error OR any deny => "unauthorized".
//   - Empty list => authorized (preserves the prior common-case behavior where
//     no fine-grained gating was requested).
//   - Non-empty list with a nil engine => error (FGA not enabled but required).
//
// The subject is always derived server-side from the resolved userID, never
// from client input.
func enforceRequiredRelations(ctx context.Context, eng engine.AuthorizationEngine, userID string, required []*model.FgaRelationInput) error {
	if len(required) == 0 {
		return nil
	}
	if eng == nil {
		return errFgaNotEnabled
	}
	if strings.TrimSpace(userID) == "" {
		return errors.New("unauthorized")
	}
	subject := "user:" + userID
	for _, r := range required {
		if r == nil || strings.TrimSpace(r.Relation) == "" || strings.TrimSpace(r.Object) == "" {
			return errors.New("each required relation needs relation and object")
		}
		allowed, err := eng.Check(ctx, subject, r.Relation, r.Object)
		if err != nil {
			// Fail closed.
			return errors.New("unauthorized")
		}
		if !allowed {
			return errors.New("unauthorized")
		}
	}
	return nil
}
