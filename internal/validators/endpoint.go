package validators

import (
	"fmt"
	"net"
	"net/url"
)

// ValidateEndpointURL checks the webhook endpoint URL for SSRF.
// Rejects private/loopback/link-local IPs and non-http(s) schemes.
func ValidateEndpointURL(endpoint string) error {
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
		if isPrivateIP(ip) {
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
		{parseCIDR("224.0.0.0/4")},     // multicast
		{parseCIDR("240.0.0.0/4")},     // reserved
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
