package provider_template

import (
	"gorm.io/gorm"
)

// TODO change following provider to new db provider
type provider struct {
	db *gorm.DB
}

// NewProvider returns a new SQL provider
// TODO change following provider to new db provider
func NewProvider() (*provider, error) {
	var sqlDB *gorm.DB

	return &provider{
		db: sqlDB,
	}, nil
}
