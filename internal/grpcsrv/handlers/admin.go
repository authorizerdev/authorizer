// Package handlers — admin.go hosts AdminHandler, the single gRPC service
// handler for Authorizer's admin (super-admin-only) API. It embeds the
// generated UnimplementedAuthorizerAdminServiceServer so any not-yet-migrated
// RPC returns codes.Unimplemented; methods are filled in one domain group at a
// time, each delegating to service.AdminProvider following the public
// AuthorizerHandler pattern.
package handlers

import (
	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/service"
)

// AdminHandler implements authorizer.v1.AuthorizerAdminService. The single
// struct satisfies the entire admin service interface; methods become real
// one domain group at a time. Service is the transport-agnostic admin API.
type AdminHandler struct {
	authorizerv1.UnimplementedAuthorizerAdminServiceServer
	Service service.AdminProvider
}
