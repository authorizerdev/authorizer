package validators

import (
	"fmt"
	"net"
	"net/url"
)

// ValidateEndpointURL checks the webhook endpoint URL for SSRF at registration
// time. Rejects non-http(s) schemes and missing hosts always; rejects
// private/loopback/link-local IPs unless allowPrivate is set. allowPrivate
// (Config.Env == constants.E2EEnv) is the registration-time counterpart to
// the delivery-time SafeHTTPClientAllowPrivate escape hatch: it exists solely so
// e2e-playground can register a webhook pointing at its docker-private
// webhook-sink mock. The scheme allow-list stays enforced either way. Must remain
// off (allowPrivate=false) in production - never true unless --env=e2e.
func ValidateEndpointURL(endpoint string, allowPrivate bool) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("malformed URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http and https schemes are allowed")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("missing host")
	}

	// Resolve the hostname to IP addresses
	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("failed to resolve host: %s", err.Error())
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return fmt.Errorf("invalid IP address resolved")
		}
		if !allowPrivate && isPrivateIP(ip) {
			return fmt.Errorf("requests to private/internal networks are not allowed")
		}
	}
	return nil
}

// isPrivateIP returns true if the IP is in a private, loopback, link-local,
// or otherwise non-routable range.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{parseCIDR("10.0.0.0/8")},
		{parseCIDR("172.16.0.0/12")},
		{parseCIDR("192.168.0.0/16")},
		{parseCIDR("127.0.0.0/8")},
		{parseCIDR("169.254.0.0/16")},  // link-local
		{parseCIDR("100.64.0.0/10")},   // CGN
		{parseCIDR("::1/128")},         // IPv6 loopback
		{parseCIDR("fc00::/7")},        // IPv6 ULA
		{parseCIDR("fe80::/10")},       // IPv6 link-local
		{parseCIDR("0.0.0.0/8")},       // "this" network
		{parseCIDR("192.0.0.0/24")},    // IETF protocol assignments
		{parseCIDR("192.0.2.0/24")},    // TEST-NET-1
		{parseCIDR("198.51.100.0/24")}, // TEST-NET-2
		{parseCIDR("203.0.113.0/24")},  // TEST-NET-3
		// The two below sat in the same IANA special-purpose registry as every
		// range above and were the only members of it still reachable. All three
		// TEST-NETs were blocked while these were not, which was an oversight
		// rather than a decision.
		//
		// 198.18.0.0/15 is the one that mattered: it is not globally routable, so
		// organisations use it internally EXACTLY BECAUSE it looks public — as a
		// lab range, or to number a network they do not want colliding with RFC
		// 1918. An SSRF there reaches a real internal service, and reaches it via
		// an address every "is this private?" intuition says is fine.
		{parseCIDR("198.18.0.0/15")},  // RFC 2544 benchmarking
		{parseCIDR("192.88.99.0/24")}, // 6to4 relay anycast (deprecated, RFC 7526)
		{parseCIDR("224.0.0.0/4")},    // multicast
		{parseCIDR("240.0.0.0/4")},    // reserved
		// IPv6 transition mechanisms EMBED an arbitrary IPv4 address inside an
		// IPv6 one, so a v6 literal in these ranges reaches a v4 destination
		// that every check above would have refused — 2002:7f00:0001:: is
		// 127.0.0.1, and 64:ff9b::a9fe:a9fe is the 169.254.169.254 cloud
		// metadata endpoint. Blocking the ranges outright is stronger and
		// simpler than decoding the embedded address, and costs nothing: none
		// of them is a legitimate destination for a webhook or an OIDC issuer.
		{parseCIDR("2002::/16")},      // RFC 3056 6to4
		{parseCIDR("2001::/32")},      // RFC 4380 Teredo
		{parseCIDR("64:ff9b::/96")},   // RFC 6052 NAT64 well-known prefix
		{parseCIDR("64:ff9b:1::/48")}, // RFC 8215 NAT64 local-use prefix
		{parseCIDR("::/128")},         // unspecified
		{parseCIDR("100::/64")},       // RFC 6666 discard-only
	}
	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseCIDR(cidr string) *net.IPNet {
	_, network, _ := net.ParseCIDR(cidr)
	return network
}
