package surrealdb_test

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/db/surrealdb"
)

// implementsProvider only compiles when New returns a storage.Provider: the
// type parameter's constraint is checked at compile time. It lives in the
// _test external package so it can import internal/storage without an import
// cycle, and needs no running database (NewProvider connects).
func implementsProvider[P storage.Provider](func(*config.Config, *surrealdb.Dependencies) (P, error)) {
}

func TestImplementsStorageProvider(t *testing.T) {
	implementsProvider(surrealdb.NewProvider)
}
