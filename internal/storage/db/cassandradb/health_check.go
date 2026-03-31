package cassandradb

import "context"

// HealthCheck verifies that the Cassandra database is reachable and responsive
func (p *provider) HealthCheck(ctx context.Context) error {
	return p.db.Query("SELECT now() FROM system.local").Exec()
}
