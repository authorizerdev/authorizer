package config

import "github.com/rs/zerolog"

// Config is the configuration for the database
type Config struct {
	// DatabaseType is the type of database to use
	DatabaseType string
	// DatabaseURL is the URL of the database
	DatabaseURL string
}

// Dependencies for the database
type Dependencies struct {
	Log zerolog.Logger
}
