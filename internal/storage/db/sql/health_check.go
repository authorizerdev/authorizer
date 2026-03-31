package sql

import "context"

// HealthCheck verifies that the SQL database is reachable and responsive
func (p *provider) HealthCheck(ctx context.Context) error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
