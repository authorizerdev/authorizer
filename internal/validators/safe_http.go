package validators

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// SafeHTTPClient parses rawURL, resolves the host once, rejects any private,
// loopback, or otherwise non-routable IPs, and returns an *http.Client whose
// Transport.DialContext is pinned to dial the validated IP directly. This
// defeats SSRF DNS-rebinding TOCTOU because the http stack never re-resolves
// the hostname between validation and the actual dial. TLS still uses
// ServerName=host so SNI and certificate validation continue to work.
//
// timeout applies to both the dial and the overall request. If timeout is 0,
// a default of 30 seconds is used.
func SafeHTTPClient(ctx context.Context, rawURL string, timeout time.Duration) (*http.Client, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("malformed URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("only http and https schemes are allowed")
	}
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("missing host")
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	var safeIP net.IP
	if literal := net.ParseIP(host); literal != nil {
		if isPrivateIP(literal) {
			return nil, fmt.Errorf("requests to private/internal networks are not allowed")
		}
		safeIP = literal
	} else {
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve host: %w", err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no IP addresses resolved")
		}
		for _, ipa := range ips {
			if isPrivateIP(ipa.IP) {
				return nil, fmt.Errorf("requests to private/internal networks are not allowed")
			}
			if safeIP == nil {
				safeIP = ipa.IP
			}
		}
	}

	dialAddr := net.JoinHostPort(safeIP.String(), port)
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Force the dial to the validated IP, ignoring whatever address
			// the http machinery would otherwise re-resolve.
			return (&net.Dialer{Timeout: timeout}).DialContext(ctx, network, dialAddr)
		},
		TLSClientConfig: &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		},
		ResponseHeaderTimeout: timeout,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, nil
}
