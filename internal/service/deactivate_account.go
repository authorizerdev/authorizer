package service

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// DeactivateAccount marks the authenticated caller's account as revoked and
// drops all of their sessions. Transport-agnostic port of
// graphqlProvider.DeactivateAccount.
//
// Permissions: authenticated user.
func (p *provider) DeactivateAccount(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeactivateAccount").Logger()

	tokenData, err := p.callerTokenData(ctx, meta)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user id from session or access token")
		return nil, nil, Unauthenticated("unauthorized")
	}
	if tokenData == nil || tokenData.UserID == "" {
		return nil, nil, Unauthenticated("unauthorized")
	}
	log = log.With().Str("userID", tokenData.UserID).Logger()
	user, err := p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by id")
		return nil, nil, err
	}
	now := time.Now().Unix()
	user.RevokedTimestamp = &now
	user, err = p.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, nil, err
	}
	go func() {
		_ = p.MemoryStoreProvider.DeleteAllUserSessions(user.ID)
		_ = p.EventsProvider.RegisterEvent(ctx, constants.UserDeactivatedWebhookEvent, "", user)
	}()
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditUserDeactivatedEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{Message: "user account deactivated successfully"}, nil, nil
}
