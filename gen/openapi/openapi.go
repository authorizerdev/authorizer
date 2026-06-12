// Package openapi exposes the generated OpenAPI 2.0 spec as a byte slice
// so HTTP handlers can serve it from any working directory (test, Docker
// container, etc.). The file is embedded at compile time via go:embed so
// builds fail loudly if `make proto-gen` hasn't been run.
package openapi

import _ "embed"

//go:embed authorizer.swagger.json
var spec []byte

// Spec returns the embedded OpenAPI 2.0 JSON.
func Spec() []byte { return spec }
