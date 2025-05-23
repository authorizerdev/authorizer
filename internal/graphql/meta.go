package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Meta returns the meta information about the server.
// Permissions: none
func (g *graphqlProvider) Meta(ctx context.Context) (*model.Meta, error) {
	clientID := g.Config.ClientID

	googleClientID := g.Config.GoogleClientID
	googleClientSecret := g.Config.GoogleClientSecret

	facebookClientID := g.Config.FacebookClientID
	facebookClientSecret := g.Config.FacebookClientSecret

	linkedClientID := g.Config.LinkedinClientID
	linkedInClientSecret := g.Config.LinkedinClientSecret

	appleClientID := g.Config.AppleClientID
	appleClientSecret := g.Config.AppleClientSecret

	githubClientID := g.Config.GithubClientID
	githubClientSecret := g.Config.GithubClientSecret

	twitterClientID := g.Config.TwitterClientID
	twitterClientSecret := g.Config.TwitterClientSecret

	microsoftClientID := g.Config.MicrosoftClientID
	microsoftClientSecret := g.Config.MicrosoftClientSecret

	twitchClientID := g.Config.TwitchClientID
	twitchClientSecret := g.Config.TwitchClientSecret

	robloxClientID := g.Config.RobloxClientID
	robloxClientSecret := g.Config.RobloxClientSecret

	isBasicAuthDisabled := g.Config.DisableBasicAuthentication
	isMobileBasicAuthDisabled := g.Config.DisableMobileBasicAuthentication
	isMobileVerificationDisabled := g.Config.DisablePhoneVerification
	isMagicLinkLoginDisabled := g.Config.DisableMagicLinkLogin
	isEmailVerificationDisabled := g.Config.DisableEmailVerification
	isMultiFactorAuthenticationDisabled := g.Config.DisableMFA
	isStrongPasswordDisabled := g.Config.DisableStrongPassword
	isSignUpDisabled := g.Config.DisableSignup

	metaInfo := model.Meta{
		Version:                            constants.VERSION,
		ClientID:                           clientID,
		IsGoogleLoginEnabled:               googleClientID != "" && googleClientSecret != "",
		IsGithubLoginEnabled:               githubClientID != "" && githubClientSecret != "",
		IsFacebookLoginEnabled:             facebookClientID != "" && facebookClientSecret != "",
		IsLinkedinLoginEnabled:             linkedClientID != "" && linkedInClientSecret != "",
		IsAppleLoginEnabled:                appleClientID != "" && appleClientSecret != "",
		IsTwitterLoginEnabled:              twitterClientID != "" && twitterClientSecret != "",
		IsMicrosoftLoginEnabled:            microsoftClientID != "" && microsoftClientSecret != "",
		IsBasicAuthenticationEnabled:       !isBasicAuthDisabled,
		IsEmailVerificationEnabled:         !isEmailVerificationDisabled,
		IsMagicLinkLoginEnabled:            !isMagicLinkLoginDisabled,
		IsSignUpEnabled:                    !isSignUpDisabled,
		IsStrongPasswordEnabled:            !isStrongPasswordDisabled,
		IsMultiFactorAuthEnabled:           !isMultiFactorAuthenticationDisabled,
		IsMobileBasicAuthenticationEnabled: !isMobileBasicAuthDisabled,
		IsPhoneVerificationEnabled:         !isMobileVerificationDisabled,
		IsTwitchLoginEnabled:               twitchClientID != "" && twitchClientSecret != "",
		IsRobloxLoginEnabled:               robloxClientID != "" && robloxClientSecret != "",
	}
	return &metaInfo, nil
}
