package arangodb

import "context"

// HealthCheck verifies that the ArangoDB database is reachable and responsive
func (p *provider) HealthCheck(ctx context.Context) error {
	_, err := p.db.Info(ctx)
	return err
}
