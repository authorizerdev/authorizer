// Package service contains the transport-agnostic business logic for
// Authorizer's public API operations. Each operation accepts a context, a
// RequestMetadata describing the inbound request, and a typed request object;
// it returns a typed response, a ResponseSideEffects describing artifacts the
// transport must apply (cookies, etc.), and an error.
//
// GraphQL resolvers, gRPC handlers, and REST handlers all construct
// RequestMetadata from their transport and apply ResponseSideEffects back to
// it. The service layer itself never touches gin.Context, grpc.ServerStream,
// or any other transport-specific type.
package service

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// RequestMetadata is the transport-derived context every service method
// receives. Fields are populated by the transport (see MetaFromGin) and read
// by handlers; the underlying *http.Request is exposed for legacy helpers
// that haven't yet been refactored to take this struct directly.
type RequestMetadata struct {
	// HostURL is the authorizer-server base URL as derived from the request
	// (X-Authorizer-URL header, then X-Forwarded-Proto + X-Forwarded-Host,
	// finally Request.Host). Always populated.
	HostURL string

	// IPAddress is the best-effort client IP (honors X-Forwarded-For).
	IPAddress string

	// UserAgent is the User-Agent header value.
	UserAgent string

	// AuthorizationHeader is the raw `Authorization` header (typically
	// "Bearer <token>"). Empty when absent.
	AuthorizationHeader string

	// Cookies sent on the request. Use the typed cookie helpers when reading
	// session/mfa cookies — never reach for these unless you know what you
	// want.
	Cookies []*http.Cookie

	// Request is the raw inbound *http.Request. Provided as an escape hatch
	// for token-provider helpers that still take a gin.Context internally;
	// new service code should prefer the typed fields above.
	Request *http.Request

	// Protocol is the transport the request came in on — one of
	// constants.Protocol{GraphQL,GRPC,REST}. Surfaced in audit logs and the
	// authorizer_api_operations_total metric so each operation is attributable
	// to its protocol. Empty when the transport did not set it.
	Protocol string
}

// ResponseSideEffects collects out-of-band artifacts produced by a service
// method that the transport must apply to its response. Today that's just
// cookies; future additions may include redirect targets or trailing headers.
type ResponseSideEffects struct {
	// Cookies to set on the response. Each cookie's Domain, Path, Secure,
	// SameSite, and MaxAge fields are honored as set; the transport adds them
	// verbatim (gin: gc.SetSameSite + gc.SetCookie; net/http: http.SetCookie).
	Cookies []*http.Cookie

	// OfferMFASetupQuiet is true when the MFA gate decided the user already
	// skipped setup before — no enrollment payload, no offer flag, just a
	// normal login.
	OfferMFASetupQuiet bool
}

// AddCookie appends a cookie to the side-effects. Convenience over manual
// slice ops; safe on a zero-value receiver.
func (s *ResponseSideEffects) AddCookie(c *http.Cookie) {
	if c == nil {
		return
	}
	s.Cookies = append(s.Cookies, c)
}

// MetaFromGin builds a RequestMetadata from a gin.Context. The gin-aware
// transport (GraphQL resolver, REST handler mounted under Gin) calls this
// once per request before invoking a service method.
func MetaFromGin(gc *gin.Context) RequestMetadata {
	if gc == nil || gc.Request == nil {
		return RequestMetadata{}
	}
	meta := RequestMetadata{
		HostURL:             parsers.GetHostFromRequest(gc.Request),
		IPAddress:           utils.GetIP(gc.Request),
		UserAgent:           utils.GetUserAgent(gc.Request),
		AuthorizationHeader: gc.Request.Header.Get("Authorization"),
		Cookies:             gc.Request.Cookies(),
		Request:             gc.Request,
		Protocol:            constants.ProtocolGraphQL,
	}
	return meta
}

// ApplyToGin writes the response side-effects to a gin.Context. The gin-aware
// transport calls this once per request after a service method returns
// successfully. A nil receiver is a no-op (service methods that have no
// side-effects may return nil).
func ApplyToGin(gc *gin.Context, side *ResponseSideEffects) {
	if side == nil || gc == nil {
		return
	}
	for _, c := range side.Cookies {
		if c == nil {
			continue
		}
		// gin.SetSameSite is per-call and must be set before SetCookie.
		gc.SetSameSite(c.SameSite)
		gc.SetCookie(c.Name, c.Value, c.MaxAge, c.Path, c.Domain, c.Secure, c.HttpOnly)
	}
}
