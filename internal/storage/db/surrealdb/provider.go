// Package surrealdb stores Authorizer's data in SurrealDB.
//
// Every table is SCHEMALESS: field types come from the Go structs in
// internal/storage/schemas, as with the MongoDB, ArangoDB and Couchbase
// providers. Indexes are created by the versioned migrations in schema.go.
//
// Three SurrealDB behaviours shape the code (verified against 4.0.1 with the
// Go SDK v2.0.1; see specs/2026-11-20-surrealdb-storage-provider-design.md in
// the docs repo):
//
//   - A query for a missing record returns no rows and no error, so getters
//     report ErrNotFound themselves (errors.go).
//   - Unique-index violations and transaction conflicts arrive as a plain
//     *QueryError with no typed kind, so they are recognised by message
//     (errors.go); tests pin the messages.
//   - Concurrent multi-statement transactions on the same rows abort with a
//     retryable conflict, so transactions are replayed whole (query.go).
package surrealdb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog"
	surrealdb "github.com/surrealdb/surrealdb.go/v2"

	"github.com/authorizerdev/authorizer/internal/config"
)

// Dependencies struct for the surrealdb data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

type provider struct {
	config       *config.Config
	dependencies *Dependencies
	db           *surrealdb.DB
	// ns and dbName are the namespace and database the session uses.
	ns, dbName string
	// authMu serialises re-authentication (see reauthenticate).
	authMu sync.Mutex
}

// NewProvider connects to SurrealDB, selects the namespace and database, and
// applies the schema migrations.
//
// --database-url may use http(s):// (recommended: stateless, nothing to
// restore after a dropped connection) or ws(s)://.
func NewProvider(cfg *config.Config, deps *Dependencies) (*provider, error) {
	ctx := context.Background()
	ns := strings.TrimSpace(cfg.SurrealDBNamespace)
	if ns == "" {
		ns = "authorizer"
	}
	dbName := strings.TrimSpace(cfg.DatabaseName)
	if dbName == "" {
		return nil, errors.New("surrealdb: --database-name is required")
	}

	// No WithTokenRenewal: it only renews token pairs from SignInWithRefresh,
	// which system users do not get. Expiry is handled by reauthenticate.
	db, err := surrealdb.FromEndpointURLString(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("surrealdb: connect: %w", err)
	}
	if err := signIn(ctx, db, cfg, ns, dbName); err != nil {
		_ = db.Close(ctx)
		return nil, err
	}
	// Create the namespace and database when the credentials allow it; a
	// namespace or database user cannot, and gets them from an operator.
	_, _ = surrealdb.Query[any](ctx, db, "DEFINE NAMESPACE IF NOT EXISTS "+ident(ns)+";", nil)
	if err := db.Use(ctx, ns, ""); err == nil {
		_, _ = surrealdb.Query[any](ctx, db, "DEFINE DATABASE IF NOT EXISTS "+ident(dbName)+";", nil)
	}
	if err := db.Use(ctx, ns, dbName); err != nil {
		_ = db.Close(ctx)
		return nil, fmt.Errorf("surrealdb: use %s/%s: %w", ns, dbName, err)
	}

	p := &provider{config: cfg, dependencies: deps, db: db, ns: ns, dbName: dbName}
	if err := p.migrate(ctx, migrations); err != nil {
		_ = db.Close(ctx)
		return nil, err
	}
	return p, nil
}

// signIn tries the narrowest credential level first: a database user, then a
// namespace user, then a root user, so least-privilege deployments work
// without a separate flag for the level.
func signIn(ctx context.Context, db *surrealdb.DB, cfg *config.Config, ns, dbName string) error {
	user, pass := cfg.DatabaseUsername, cfg.DatabasePassword
	if user == "" {
		return nil // unauthenticated server, e.g. `surreal start --unauthenticated`
	}
	var lastErr error
	for _, auth := range []*surrealdb.Auth{
		{Namespace: ns, Database: dbName, Username: user, Password: pass},
		{Namespace: ns, Username: user, Password: pass},
		{Username: user, Password: pass},
	} {
		if _, err := db.SignIn(ctx, auth); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return fmt.Errorf("surrealdb: sign in as %q: %w", user, lastErr)
}

// reauthenticate signs in again with the configured credentials. A system
// user's sign-in token expires (after 1h by default). Over WebSocket the
// session outlives it, but over HTTP every request carries the token, so once
// it expires every query is refused with 401. The SDK's token renewal only
// covers sign-ins that return a refresh token, which system users do not get.
// Serialised so a burst of requests hitting the expiry signs in once each in
// turn rather than all at once.
func (p *provider) reauthenticate(ctx context.Context) error {
	p.authMu.Lock()
	defer p.authMu.Unlock()
	if err := signIn(ctx, p.db, p.config, p.ns, p.dbName); err != nil {
		return err
	}
	return p.db.Use(ctx, p.ns, p.dbName)
}

// ident quotes a namespace or database name for DDL, where SurrealQL takes
// identifiers, not parameters. Backticks delimit the identifier; a backtick in
// the name is escaped. Only operator-supplied flag values reach here.
func ident(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "\\`") + "`"
}

// Close closes the connection.
func (p *provider) Close() error {
	if p.db == nil {
		return nil
	}
	return p.db.Close(context.Background())
}
