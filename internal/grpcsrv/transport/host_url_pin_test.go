package transport

import (
	"context"
	"net/http"
	"testing"

	"google.golang.org/grpc/metadata"

	"github.com/authorizerdev/authorizer/internal/parsers"
)

// TestHostURLPinnedToTrustedURL is the regression guard for the trusted-URL pin
// on the gRPC and REST-gateway transports.
//
// HostURL becomes the host of the emailed password-reset / magic-link URL and
// the `iss` claim of the tokens minted from them, so a caller who can choose it
// can have a victim emailed a genuine-looking reset link pointing at a host the
// caller controls. parsers.GetHostFromRequest pins that for the gin path; this
// transport read the metadata key directly and so never consulted --url.
//
// Not parallel, and restores the value in Cleanup: parsers.trustedURL is a
// set-once-at-startup package global.
func TestHostURLPinnedToTrustedURL(t *testing.T) {
	const trusted = "https://auth.example.com"
	const attacker = "https://evil.example.net"

	parsers.SetTrustedURL(trusted)
	t.Cleanup(func() { parsers.SetTrustedURL("") })

	// Control: the gin path already ignores the caller's value. If this ever
	// fails the pin itself regressed, not just this transport.
	req, err := http.NewRequest(http.MethodGet, "http://listener.internal/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Authorizer-URL", attacker)
	if got := parsers.GetHostFromRequest(req); got != trusted {
		t.Fatalf("gin control path regressed: GetHostFromRequest = %q, want %q", got, trusted)
	}

	t.Run("direct gRPC caller cannot choose the host", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("x-authorizer-url", attacker))
		if got := MetaFromGRPC(ctx).HostURL; got != trusted {
			t.Errorf("HostURL = %q, want the trusted URL %q", got, trusted)
		}
	})

	t.Run("gateway-smuggled value cannot win on ordering", func(t *testing.T) {
		// A REST client sending Grpc-Metadata-X-Authorizer-Url lands a real
		// x-authorizer-url entry BEFORE the gateway appends its own, so reading
		// index 0 returned the caller's. gateway.Handler now also refuses the
		// key outright; this asserts the transport holds even if it arrives.
		md := metadata.MD{}
		md.Append("x-authorizer-url", attacker)
		md.Append("x-authorizer-url", trusted)
		ctx := metadata.NewIncomingContext(context.Background(), md)
		if got := MetaFromGRPC(ctx).HostURL; got != trusted {
			t.Errorf("HostURL = %q, want the trusted URL %q", got, trusted)
		}
	})

	t.Run("grpcgateway- prefixed spelling is also pinned", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("grpcgateway-x-authorizer-url", attacker))
		if got := MetaFromGRPC(ctx).HostURL; got != trusted {
			t.Errorf("HostURL = %q, want the trusted URL %q", got, trusted)
		}
	})
}

// TestHostURLFallbackWithoutTrustedURL covers the embedder/test path. A shipped
// binary refuses to start without --url (cmd.validateAuthorizerURL), but a
// library embedder constructing a Config directly can reach this, and the
// gateway's own injected value must still arrive — otherwise every REST-minted
// token gets "http://bufconn" as its issuer and JWT validation rejects it.
func TestHostURLFallbackWithoutTrustedURL(t *testing.T) {
	parsers.SetTrustedURL("")

	t.Run("gateway's own value is honoured", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("x-authorizer-url", "https://auth.example.com"))
		if got := MetaFromGRPC(ctx).HostURL; got != "https://auth.example.com" {
			t.Errorf("HostURL = %q, want the gateway's injected value", got)
		}
	})

	t.Run("last value wins over a smuggled first", func(t *testing.T) {
		md := metadata.MD{}
		md.Append("x-authorizer-url", "https://evil.example.net")
		md.Append("x-authorizer-url", "https://auth.example.com")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		if got := MetaFromGRPC(ctx).HostURL; got != "https://auth.example.com" {
			t.Errorf("HostURL = %q, want the LAST (gateway-appended) value", got)
		}
	})

	t.Run("falls back to :authority when no value is present", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs(":authority", "listener.internal"))
		if got := MetaFromGRPC(ctx).HostURL; got != "http://listener.internal" {
			t.Errorf("HostURL = %q, want the :authority fallback", got)
		}
	})
}
