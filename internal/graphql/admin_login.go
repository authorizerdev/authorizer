package graphql

import (
	"context"
	"crypto/subtle"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AdminLogin is the method to login as admin.
// Permissions: none
func (g *graphqlProvider) AdminLogin(ctx context.Context, params *model.AdminLoginRequest) (*model.Response, error) {
	log := g.Log.With().Str("func", "AdminLogin").Logger()
	var res *model.Response
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return res, fmt.Errorf("internal server error")
	}
	if subtle.ConstantTimeCompare([]byte(params.AdminSecret), []byte(g.Config.AdminSecret)) != 1 {
		log.Debug().Msg("Invalid admin secret")
		g.AuditProvider.LogEvent(audit.Event{
			Action:       constants.AuditAdminLoginFailedEvent,
			ActorType:    constants.AuditActorTypeAdmin,
			ResourceType: constants.AuditResourceTypeAdminSession,
			IPAddress:    utils.GetIP(gc.Request),
			UserAgent:    utils.GetUserAgent(gc.Request),
		})
		return res, fmt.Errorf(`invalid admin secret`)
	}

	hashedKey, err := crypto.EncryptPassword(g.Config.AdminSecret)
	if err != nil {
		return res, err
	}
	cookie.SetAdminCookie(gc, hashedKey, g.Config.AdminCookieSecure)

	g.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminLoginSuccessEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeAdminSession,
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
	})
	res = &model.Response{
		Message: "admin logged in successfully",
	}
	return res, nil
}
