package http_handlers

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// accountHasState reports whether a user account holds state that would be
// silently destroyed by deleting it, and names the first thing found.
//
// This exists to bound the pre-hijack guard in OAuthCallbackHandler. That guard
// deletes an *unverified* pre-existing account rather than linking a federated
// identity to it, which is correct for its intended target — an attacker who
// signed up with someone else's address and never verified it, squatting to
// intercept their later social login. A squatter's account is empty by
// definition: created seconds ago, never used.
//
// A real account is not. StorageProvider.DeleteUser cascades to every
// user-keyed table (schemas.UserOwnedCollections, #749), so nothing is left
// dangling — but "cleanly destroyed" is not "safe to destroy". The replacement
// account gets a fresh id, so deleting a real account silently drops its org
// memberships, its enrolled authenticators and passkeys, and its federated
// identities, and the user gets none of it back. Two things are worse than
// merely lost:
//
//   - FGA grants are NOT covered by that cascade. The tuple store lives outside
//     StorageProvider, and the purge that admin _delete_user runs
//     (service.purgeFgaTuplesForUser) is in the service layer, which this
//     handler's direct StorageProvider.DeleteUser call does not go through. So
//     `user:<dead-id>` grants persist forever while the new account inherits
//     none of them.
//   - the whole thing is triggered by an UNAUTHENTICATED OAuth callback. Nobody
//     proved they own the account being destroyed; they merely presented a
//     provider assertion for the same address.
//
// So an account carrying any of this is not a squatter, and must never be
// deleted to resolve an email collision. The caller refuses the login instead,
// which is recoverable; deletion is not.
//
// Fail-closed by design: a storage or FGA fault reports "has state" (the second
// return names it), because the safe answer when we cannot tell is to refuse
// rather than destroy.
func (h *httpProvider) accountHasState(ctx context.Context, user *schemas.User) (bool, string) {
	if user == nil || user.ID == "" {
		return false, ""
	}

	if memberships, _, err := h.StorageProvider.ListOrgMembershipsByUser(ctx, user.ID, nil); err != nil {
		return true, "org membership lookup failed"
	} else if len(memberships) > 0 {
		return true, "org membership"
	}

	if creds, err := h.StorageProvider.ListWebauthnCredentialsByUserID(ctx, user.ID); err != nil {
		return true, "passkey lookup failed"
	} else if len(creds) > 0 {
		return true, "passkey"
	}

	for _, authenticatorType := range []string{
		constants.EnvKeyTOTPAuthenticator,
		constants.EnvKeyEmailOTPAuthenticator,
		constants.EnvKeySMSOTPAuthenticator,
	} {
		// "Not enrolled" is the normal case and every backend reports it as an
		// error (the not-found contract), so it must be told apart from "the
		// lookup failed" — swallowing both would make a transient storage fault
		// look like an empty account and delete somebody's MFA enrollment.
		a, err := h.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, authenticatorType)
		switch {
		case storage.IsNotFound(err):
			continue
		case err != nil:
			return true, "authenticator lookup failed"
		case a != nil:
			return true, "enrolled authenticator"
		}
	}

	// FGA grants are the ones the user notices losing and the ones no cascade
	// would ever clean up. Only ask the engine when one is configured.
	if h.AuthzEngine != nil {
		res, err := h.AuthzEngine.ReadTuples(ctx, engine.ReadTuplesFilter{
			User:     "user:" + user.ID,
			PageSize: 1,
		})
		if err != nil {
			return true, "fga tuple lookup failed"
		}
		if res != nil && len(res.Tuples) > 0 {
			return true, "fga grant"
		}
	}

	return false, ""
}
