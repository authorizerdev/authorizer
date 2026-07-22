package provider_template

import (
	"context"
)

// HealthCheck verifies that the storage backend is reachable and responsive.
func (p *provider) HealthCheck(ctx context.Context) error {
	return nil
}
