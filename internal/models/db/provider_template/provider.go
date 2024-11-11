package provider_template

import (
	"github.com/authorizerdev/authorizer/internal/models/config"
	"gorm.io/gorm"
)

// TODO change following provider to new db provider
type provider struct {
	Dependencies config.Dependencies
	db           *gorm.DB
}

// NewProvider returns a new SQL provider
// TODO change following provider to new db provider
func NewProvider(
	config config.Config,
	deps config.Dependencies,
) (*provider, error) {
	var sqlDB *gorm.DB

	return &provider{
		Dependencies: deps,
		db:           sqlDB,
	}, nil
}
