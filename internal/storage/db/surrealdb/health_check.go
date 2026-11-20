package surrealdb

import "context"

// HealthCheck checks that the database answers a query on the selected
// namespace and database.
func (p *provider) HealthCheck(ctx context.Context) error {
	return p.exec(ctx, "RETURN true;", nil)
}
