package provider_template_test

import (
	"testing"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/db/provider_template"
)

// TestImplementsStorageProvider fails to compile — not just to run — the
// moment provider stops satisfying storage.Provider. It lives in the
// _test external package so it can import internal/storage without an
// import cycle (internal/storage will import this package's non-test code
// once it's wired into storage.New(); see provider.go).
func TestImplementsStorageProvider(t *testing.T) {
	log := zerolog.Nop()
	p, err := provider_template.NewProvider(&config.Config{}, &provider_template.Dependencies{Log: &log})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	var _ storage.Provider = p
}
