package service

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Meta returns the meta information about the server.
// Permissions: none
func (s *service) Meta(ctx context.Context) (*model.Meta, error) {
	clientID := s.Config.ClientID

	googleClientID := s.Config.GoogleClientID
	googleClientSecret := s.Config.GoogleClientSecret

	facebookClientID := s.Config.FacebookClientID
	facebookClientSecret := s.Config.FacebookClientSecret

	linkedClientID := s.Config.LinkedinClientID
	linkedInClientSecret := s.Config.LinkedinClientSecret

	appleClientID := s.Config.AppleClientID
	appleClientSecret := s.Config.AppleClientSecret

	githubClientID := s.Config.GithubClientID
	githubClientSecret := s.Config.GithubClientSecret

	twitterClientID := s.Config.TwitterClientID
	twitterClientSecret := s.Config.TwitterClientSecret

	microsoftClientID := s.Config.MicrosoftClientID
	microsoftClientSecret := s.Config.MicrosoftClientSecret

	twitchClientID := s.Config.TwitchClientID
	twitchClientSecret := s.Config.TwitchClientSecret

	robloxClientID := s.Config.RoboloxClientID
	robloxClientSecret := s.Config.RoboloxClientSecret

	isBasicAuthDisabled := s.Config.DisableBasicAuthentication
	isMobileBasicAuthDisabled := s.Config.DisableMobileBasicAuthentication
	isMobileVerificationDisabled := s.Config.DisablePhoneVerification
	isMagicLinkLoginDisabled := s.Config.DisableMagicLinkLogin
	isEmailVerificationDisabled := s.Config.DisableEmailVerification
	isMultiFactorAuthenticationDisabled := s.Config.DisableMFA
	isStrongPasswordDisabled := s.Config.DisableStrongPassword
	isSignUpDisabled := s.Config.DisableSignup

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
