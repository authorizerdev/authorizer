package service

import (
	"context"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
)

// EmailVerification is the outcome of redeeming a verification token: who the
// principal is, which flow the token belonged to, and whether this redemption is
// the one that verified the address.
type EmailVerification struct {
	User *schemas.User
	// Request is the consumed verification row. The caller deletes it once it
	// has finished with it — deletion is deliberately NOT done here, because the
	// two callers finish at different points.
	Request *schemas.VerificationRequest
	// LoginMethod is basic_auth, or magic_link_login when the token came from a
	// magic link.
	LoginMethod string
	// IsSignUp reports whether THIS redemption flipped the address to verified,
	// which is what the callers key their signup-vs-login webhook on.
	IsSignUp bool
	// RedirectURI is the `redirect_uri` claim the token was minted with. The
	// REST handler falls back to it when the query string carries none; it is
	// returned here so the claim never has to leave this function.
	RedirectURI string
}

// ConsumeEmailVerificationToken is the single source of truth for what it means
// to redeem an email-verification token.
//
// It exists because there are two implementations of this flow over the same
// token table, and they have now drifted twice:
//
//   - the MFA gate was added to the GraphQL mutation and missed the REST
//     handler, so the emailed link bypassed MFA entirely (noted in that
//     handler's own comment);
//   - the email_verified_at write was moved above the MFA gate in the mutation
//     and missed the REST handler, so users who clicked the emailed button were
//     never marked verified — surfacing only on passkey login, because that is
//     the sole login path that checks the column. Password login checks it too
//     but silently self-heals via an email-OTP detour, which is why TOTP and
//     email-OTP users appeared unaffected.
//
// GET /verify_email is what the button in the email literally points to, so the
// REST handler is the path essentially every real user takes — and it was the
// copy that kept missing fixes. GraphQL and gRPC both already delegate to the
// service; this pulls the REST handler onto the same decision logic so a third
// divergence cannot happen.
//
// What stays with the callers is presentation, which legitimately differs: the
// mutation returns an AuthResponse, the handler redirects with tokens in the
// query string. What lives here is every decision that must be identical.
//
// ORDER IS LOAD-BEARING. The address is marked verified BEFORE this returns, so
// that a caller which then withholds tokens — the MFA gate redirecting to setup
// — cannot lose the fact that the user proved control of their mailbox.
// Clicking the link is the proof; whether MFA interrupts token issuance
// afterwards is a separate question.
func (p *provider) ConsumeEmailVerificationToken(ctx context.Context, hostname, rawToken string) (*EmailVerification, error) {
	log := p.Log.With().Str("func", "ConsumeEmailVerificationToken").Logger()

	verificationRequest, err := p.StorageProvider.GetVerificationRequestByToken(ctx, rawToken)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetVerificationRequestByToken")
		return nil, InvalidArgument(`invalid verification token`)
	}

	claim, err := p.TokenProvider.ParseJWTToken(rawToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse jwt token")
		return nil, InvalidArgument(`invalid verification token`)
	}

	if ok, err := p.TokenProvider.ValidateJWTClaims(claim, &token.AuthTokenConfig{
		HostName: hostname,
		Nonce:    verificationRequest.Nonce,
		User: &schemas.User{
			Email: &verificationRequest.Email,
		},
	}); !ok || err != nil {
		log.Debug().Err(err).Msg("Failed to validate jwt claims")
		return nil, InvalidArgument(`invalid verification token`)
	}

	// Purpose binding: only the email-verification family completes here. A
	// forgot-password token redeemed at this endpoint would otherwise hand out a
	// full session AND mark the address verified. Generic error so it is not an
	// oracle for which flow a leaked token belongs to.
	if !IsVerifyEmailPurpose(verificationRequest, claim) {
		log.Debug().Str("identifier", verificationRequest.Identifier).Msg("Verification token used for the wrong purpose")
		return nil, InvalidArgument(`invalid verification token`)
	}

	// `sub` must be a non-empty string before it is used to select an account.
	//
	// ValidateJWTClaims above does NOT guarantee that. Its subject check is
	//
	//	claims["sub"] != cfg.User.ID && claims["sub"] != cfg.User.Email
	//
	// and this call site sets only Email, leaving User.ID as "". A token whose
	// `sub` is the empty STRING therefore satisfies the first comparison and the
	// whole check passes. The lookup would then run as GetUserByEmail(ctx, ""),
	// whose result depends entirely on whether a given backend stores a missing
	// email as NULL or as "" — SQL keeps it NULL today, so nothing matches, but
	// that is an accident of one backend's representation and not a property
	// anyone declared. Six storage backends is too many places for a security
	// boundary to be implicit.
	//
	// Rejected explicitly so this does not depend on the empty-User.ID quirk
	// above, nor on NULL-vs-empty-string behaviour below.
	email, _ := claim["sub"].(string)
	if strings.TrimSpace(email) == "" {
		log.Debug().Msg("verification token has no subject")
		return nil, InvalidArgument(`invalid verification token`)
	}

	user, err := p.StorageProvider.GetUserByEmail(ctx, email)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetUserByEmail")
		return nil, err
	}

	if user.RevokedTimestamp != nil {
		log.Debug().Msg("User access has been revoked")
		return nil, FailedPrecondition("user access has been revoked")
	}

	loginMethod := constants.AuthRecipeMethodBasicAuth
	if verificationRequest.Identifier == constants.VerificationTypeMagicLinkLogin {
		loginMethod = constants.AuthRecipeMethodMagicLinkLogin
	}

	// See the ORDER IS LOAD-BEARING note above.
	isSignUp := user.EmailVerifiedAt == nil
	if isSignUp {
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
		user, err = p.StorageProvider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug().Err(err).Msg("failed UpdateUser")
			return nil, err
		}
	}

	redirectURI, _ := claim["redirect_uri"].(string)

	log.Debug().Str("email", refs.StringValue(user.Email)).Msg("Email verified successfully")
	return &EmailVerification{
		User:        user,
		Request:     verificationRequest,
		LoginMethod: loginMethod,
		IsSignUp:    isSignUp,
		RedirectURI: redirectURI,
	}, nil
}
