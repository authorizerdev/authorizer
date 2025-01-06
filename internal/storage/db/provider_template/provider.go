package provider_template

import (
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/config"
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

// NewProvider returns a new SQL provider
// TODO change following provider to new db provider
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
