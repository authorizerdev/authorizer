package dynamodb

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// HealthCheck verifies that the DynamoDB backend is reachable and responsive
func (p *provider) HealthCheck(ctx context.Context) error {
	var envs []schemas.Env
	return p.db.Table(schemas.Collections.Env).Scan().Limit(1).AllWithContext(ctx, &envs)
}
