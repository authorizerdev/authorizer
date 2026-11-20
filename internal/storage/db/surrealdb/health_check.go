package surrealdb

import "context"

// HealthCheck checks that the selected namespace and database are reachable
// and readable with the session's credentials, by reading the migrations
// table every database Authorizer manages has.
func (p *provider) HealthCheck(ctx context.Context) error {
	return p.exec(ctx, "SELECT * FROM type::table($mt) LIMIT 1;", map[string]any{"mt": migrationsTable})
}
