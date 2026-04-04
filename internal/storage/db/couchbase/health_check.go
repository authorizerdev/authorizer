package couchbase

import (
	"context"
	"fmt"

	"github.com/couchbase/gocb/v2"
)

// HealthCheck verifies that the Couchbase backend is reachable and responsive
func (p *provider) HealthCheck(ctx context.Context) error {
	query := fmt.Sprintf("SELECT 1 FROM %s LIMIT 1", p.scopeName)
	_, err := p.db.Query(query, &gocb.QueryOptions{
		Context: ctx,
	})
	return err
}
