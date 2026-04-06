package couchbase

import (
	"context"
	"fmt"

	"github.com/couchbase/gocb/v2"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// HealthCheck verifies that the Couchbase backend is reachable and responsive
func (p *provider) HealthCheck(ctx context.Context) error {
	query := fmt.Sprintf("SELECT 1 FROM %s.%s LIMIT 1", p.scopeName, schemas.Collections.User)
	_, err := p.db.Query(query, &gocb.QueryOptions{
		Context: ctx,
	})
	return err
}
