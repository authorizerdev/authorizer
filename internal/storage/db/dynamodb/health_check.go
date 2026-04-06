package dynamodb

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// HealthCheck verifies that the DynamoDB backend is reachable and responsive
func (p *provider) HealthCheck(ctx context.Context) error {
	var envs []schemas.Env
	items, err := p.scanFilteredLimit(ctx, schemas.Collections.Env, nil, nil, 1)
	if err != nil {
		return err
	}
	for _, it := range items {
		var e schemas.Env
		if err := unmarshalItem(it, &e); err != nil {
			return err
		}
		envs = append(envs, e)
	}
	_ = envs
	return nil
}
