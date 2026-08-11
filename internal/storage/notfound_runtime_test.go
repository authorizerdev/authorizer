package storage

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// TestNotFoundIsRecognisableOnEveryBackend asserts that a single-entity getter
// reports an absent row with an error storage.IsNotFound actually recognises.
//
// TestNotFoundContractIsUniform, its static sibling, only detects a getter that
// returns (nil, nil). It cannot see the other half of the contract: a getter
// that correctly returns an ERROR, but one so shaped that IsNotFound says false.
// That gap shipped real bugs. DynamoDB's GetUserByID returned a bare
// errors.New("no documets found") and Couchbase's returned gocb.ErrNoResult from
// the N1QL path while its IsNotFound matched only ErrDocumentNotFound — so on
// exactly those two backends, callers branching on IsNotFound to separate "no
// such row" from "the query failed" got the wrong answer.
//
// The consequence was not theoretical. token.subjectLiveness used that branch to
// decide whether to look the subject up as a client; on DynamoDB and Couchbase
// it never got there, which both rejected every delegated service-account token
// and stopped service-account deactivation from revoking live tokens. Neither
// failed in CI, because CI runs SQLite.
//
// This test is deliberately a RUNTIME one: the defect lives in what the driver
// returns, which no static check over the source can see. It is therefore only
// meaningful under `make test-all-db` (or TEST_DBS naming a real backend);
// SQLite alone proves almost nothing here, which is precisely how the bug
// survived.
//
// Scoped to the two getters the liveness path depends on. Widening it to every
// getter would be better and is worth doing separately; asserting the two that
// caused a live bug is the part that must not regress.
func TestNotFoundIsRecognisableOnEveryBackend(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	for _, dbType := range getTestDBTypes() {
		t.Run(dbType, func(t *testing.T) {
			if dbType == constants.DbTypeDynamoDB {
				_ = os.Unsetenv("AWS_ACCESS_KEY_ID")
				_ = os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			}
			cfg := getTestDBConfig(dbType)
			if dbType == constants.DbTypeCouchbaseDB {
				conn, err := net.DialTimeout("tcp", "127.0.0.1:8091", 2*time.Second)
				if err != nil {
					t.Skipf("Couchbase not reachable on 127.0.0.1:8091: %v", err)
				}
				_ = conn.Close()
				cfg.DatabaseUsername = "Administrator"
				cfg.DatabasePassword = "password"
			}

			provider, err := New(cfg, &Dependencies{Log: &logger})
			require.NoError(t, err, "could not open %s", dbType)

			ctx := context.Background()
			// An id no row can have, so the ONLY correct answer is "not found".
			absent := "definitely-absent-" + uuid.NewString()

			t.Run("GetUserByID", func(t *testing.T) {
				user, uErr := provider.GetUserByID(ctx, absent)
				assert.Nil(t, user, "a missing row must not also yield a value")
				require.Error(t, uErr, "a single-entity getter must report absence as an error, never (nil, nil)")
				assert.True(t, IsNotFound(uErr),
					"absence must be distinguishable from failure: IsNotFound said false for %q. "+
						"Callers branch on this to tell a missing row from a database outage, and getting it "+
						"wrong turns one into the other.", uErr)
			})

			t.Run("GetClientByID", func(t *testing.T) {
				client, cErr := provider.GetClientByID(ctx, absent)
				assert.Nil(t, client)
				require.Error(t, cErr)
				assert.True(t, IsNotFound(cErr),
					"absence must be distinguishable from failure: IsNotFound said false for %q", cErr)
			})
		})
	}
}
