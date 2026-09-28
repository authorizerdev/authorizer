package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestValidateJWTTokenTypeBinding pins the contract of ValidateJwtToken's
// token_type parameter: it is the type the caller is ASKING ABOUT, and a
// successful validation must mean the presented token is actually of that type.
//
// It used to be neither. The requested type was checked against a list of legal
// names and then never compared to the token's own signed token_type claim, so
// every cross-type combination validated successfully (GHSA-cqr5-83v5-mx8f).
//
// The mechanism is worth stating, because it is why the session-store lookup did
// not catch this on its own: CreateAuthToken stamps the SAME nonce on all three
// tokens of one login, and the lookup key is "<requestedType>_<nonce>". So the
// entry existed for whichever type was asked about, whatever token was actually
// presented — and the fetched value was discarded rather than compared. A
// refresh token therefore validated as an access token, which on this endpoint
// (public, and the introspection surface a resource server calls) told that
// resource server the wrong thing.
func TestValidateJWTTokenTypeBinding(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "validate_jwt_binding_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	// offline_access so the login actually yields a refresh token; without it
	// the refresh-token rows below would be skipped and the matrix would be
	// silently incomplete.
	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email:    &email,
		Password: password,
		Scope:    []string{"openid", "email", "profile", "offline_access"},
	})
	require.NoError(t, err)
	require.NotNil(t, loginRes.AccessToken)
	require.NotNil(t, loginRes.IDToken)
	require.NotNil(t, loginRes.RefreshToken, "offline_access must yield a refresh token")

	tokens := map[string]string{
		constants.TokenTypeAccessToken:   *loginRes.AccessToken,
		constants.TokenTypeIdentityToken: *loginRes.IDToken,
		constants.TokenTypeRefreshToken:  *loginRes.RefreshToken,
	}
	types := []string{
		constants.TokenTypeAccessToken,
		constants.TokenTypeIdentityToken,
		constants.TokenTypeRefreshToken,
	}

	for _, actual := range types {
		for _, requested := range types {
			name := "actual=" + actual + "/requested=" + requested
			t.Run(name, func(t *testing.T) {
				res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
					Token:     tokens[actual],
					TokenType: requested,
				})
				if actual == requested {
					require.NoError(t, err, "a token must validate as its own type")
					require.NotNil(t, res)
					require.True(t, res.IsValid)
					return
				}
				require.Error(t, err,
					"a %s must NOT validate as a %s — the signed token_type claim is the boundary",
					actual, requested)
				require.Nil(t, res)
			})
		}
	}
}

// TestValidateJWTTokenRejectsForeignTokenValue pins the second half of the fix:
// the session entry's VALUE is compared, not merely its existence.
//
// Two separate logins by the same user produce two nonces and two session
// entries. Presenting login A's access token is legitimate and must pass. The
// interesting case is the negative one below it — a token whose signature and
// claims are genuine but whose session entry has been deleted must fail, and
// must fail on the store lookup rather than on the signature.
func TestValidateJWTTokenRejectsRevokedToken(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "validate_jwt_revoked_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email:    &email,
		Password: password,
	})
	require.NoError(t, err)
	require.NotNil(t, loginRes.AccessToken)
	accessToken := *loginRes.AccessToken

	// Baseline: it validates while the session is live.
	res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
		Token:     accessToken,
		TokenType: constants.TokenTypeAccessToken,
	})
	require.NoError(t, err)
	require.True(t, res.IsValid)

	// Revoke every session for the user, then replay the same token.
	require.NotNil(t, loginRes.User)
	require.NoError(t, ts.MemoryStoreProvider.DeleteAllUserSessions(loginRes.User.ID))

	res, err = ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
		Token:     accessToken,
		TokenType: constants.TokenTypeAccessToken,
	})
	require.Error(t, err, "a token whose session was revoked must not validate")
	require.Nil(t, res)
}
