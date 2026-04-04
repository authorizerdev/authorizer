package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// HealthCheck verifies that the MongoDB database is reachable and responsive
func (p *provider) HealthCheck(ctx context.Context) error {
	return p.db.Client().Ping(ctx, readpref.Primary())
}
