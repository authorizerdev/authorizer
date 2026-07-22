package provider_template

import (
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage"
)

// Dependencies struct the TODO(replace with new db name) data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

// TODO change following provider to new db provider
type provider struct {
	config       *config.Config
	dependencies *Dependencies
	db           *gorm.DB
}

// Compile-time check: provider must implement every method of storage.Provider.
// Deleting or renaming a method here without updating the interface (or vice
// versa) fails the build immediately instead of silently drifting.
var _ storage.Provider = (*provider)(nil)

// NewProvider returns a new provider for your database type.
// TODO: change provider struct and NewProvider to use your database client.
//
// This provider must implement every method of storage.Provider — see that
// interface (internal/storage/provider.go) for the authoritative, documented
// list. Use schemas.Collections for table/collection names (e.g.,
// schemas.Collections.SessionToken).
func NewProvider(
	config *config.Config,
	deps *Dependencies,
) (*provider, error) {
	var sqlDB *gorm.DB

	return &provider{
		config:       config,
		dependencies: deps,
		db:           sqlDB,
	}, nil
}

// Close closes the underlying database pool when initialized.
func (p *provider) Close() error {
	if p.db == nil {
		return nil
	}
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
