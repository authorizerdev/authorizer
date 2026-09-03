package utils

import (
	"net"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
)

// trustedProxyCIDRs holds the operator-configured reverse-proxy networks whose
// forwarded headers may be believed. Package-level and set once at startup,
// mirroring parsers.SetTrustedURL — the alternative, threading configuration
// through GetIP's signature, would touch ~25 call sites across the HTTP
// handlers, GraphQL services and audit paths for no behavioural gain.
//
// atomic.Pointer rather than a plain var: cmd/root.go writes it during startup
// while tests write it between cases, and GetIP is read from every request
// goroutine. A plain var is a data race the race detector will (rightly) fail
// on.
//
// nil (the default) means TRUST NOTHING, matching gin's own default. That is
// the safe direction: forwarded headers are ignored and the connection's peer
// address is used.
var trustedProxyCIDRs atomic.Pointer[[]*net.IPNet]

// SetTrustedProxies configures which peers' forwarded headers GetIP will
// believe. It takes the same []string of CIDRs and bare IPs that
// gin's Engine.SetTrustedProxies takes, and MUST be called with the same value
// — internal/server.NewRouter does exactly that, one line apart, so the
// router's view of the client address and GetIP's view cannot drift.
//
// That drift was the vulnerability (GHSA-93hc-xq3w-xw87): the router was
// correctly hardened with SetTrustedProxies while GetIP read X-Forwarded-For
// unconditionally, so the admin-secret lockout bucketed on an
// attacker-chosen label and never filled.
//
// An empty or nil list trusts no proxies. An invalid entry is reported as an
// error and leaves the previous configuration untouched, so a typo cannot
// silently widen trust.
func SetTrustedProxies(proxies []string) error {
	cidrs := make([]*net.IPNet, 0, len(proxies))
	for _, entry := range proxies {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		// A bare address is a /32 (or /128) of itself — the same normalisation
		// gin's prepareTrustedCIDRs applies, so both accept identical input.
		if !strings.Contains(entry, "/") {
			if ip := net.ParseIP(entry); ip != nil {
				if ip.To4() != nil {
					entry += "/32"
				} else {
					entry += "/128"
				}
			}
		}
		_, cidr, err := net.ParseCIDR(entry)
		if err != nil {
			return err
		}
		cidrs = append(cidrs, cidr)
	}
	trustedProxyCIDRs.Store(&cidrs)
	return nil
}

// isTrustedProxy reports whether ip falls in a configured trusted network.
func isTrustedProxy(ip net.IP) bool {
	cidrs := trustedProxyCIDRs.Load()
	if cidrs == nil || ip == nil {
		return false
	}
	for _, cidr := range *cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// GetIP returns the client IP address for a request.
//
// It is the identity used for the admin-secret lockout bucket
// (token.VerifyAdminSecret) and for the IPAddress recorded on every audit
// event, so "who is this request from" has to be a fact about the connection,
// not a claim the caller makes about itself.
//
// The resolution deliberately mirrors gin's Context.ClientIP:
//
//  1. Split the bare IP out of RemoteAddr. This is the peer that actually
//     opened the TCP connection and is the only unforgeable signal available.
//     The PORT IS DROPPED — a lockout keyed on "IP:port" gets a fresh bucket
//     for every new connection, which defeated the throttle even with no
//     forwarded headers in play at all.
//  2. If that peer is NOT a configured trusted proxy, it IS the client.
//     Forwarded headers are attacker-controlled here and are ignored.
//  3. If it IS trusted, walk X-Forwarded-For right-to-left and return the
//     first entry that is not itself a trusted proxy — the handoff point
//     between infrastructure we trust and the network we do not. With a
//     CDN in front of a load balancer, BOTH must appear in the trusted list
//     or the walk stops at the outermost unlisted hop and returns that hop's
//     address instead of the end client's. Returning a real proxy address is
//     the correct failure: the alternative is believing a header no
//     configured party vouched for.
//  4. Fall back to X-Real-IP, then to the peer address.
//
// gin's Context.ClientIP is deliberately NOT called here even though it
// implements the same rule: it dereferences the unexported Context.engine
// field, and this codebase builds ~20 synthetic contexts as
// &gin.Context{Request: meta.Request} (internal/service, internal/grpcsrv) to
// reach request-scoped helpers off the gin request path. Calling ClientIP on
// one of those nil-panics in an auth path. Taking an *http.Request keeps every
// existing call site working. TestGetIPMatchesGinClientIP pins the two
// implementations to the same answers so they cannot drift.
func GetIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	// RemoteAddr is "IP:port" for net/http, but a test or a non-TCP transport
	// may hand over a bare address; fall back to the raw value rather than
	// returning nothing.
	peer := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(peer); err == nil {
		peer = host
	}

	if !isTrustedProxy(net.ParseIP(peer)) {
		return peer
	}

	// Same header precedence and same right-to-left walk as gin.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		items := strings.Split(xff, ",")
		for i, raw := range slices.Backward(items) {
			ipStr := strings.TrimSpace(raw)
			ip := net.ParseIP(ipStr)
			if ip == nil {
				// A malformed entry poisons everything to its left: we can no
				// longer tell which hop appended what. Stop and fall through.
				break
			}
			// The leftmost entry is the originating client by definition, so it
			// is returned even if it happens to sit in a trusted range.
			if i == 0 || !isTrustedProxy(ip) {
				return ipStr
			}
		}
	}
	// Parsed, not returned raw: gin runs this header through the same
	// validation as X-Forwarded-For and falls back to the peer on garbage.
	// Returning it unchecked would let a trusted proxy's malformed value become
	// a lockout bucket key that is not an address at all.
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-Ip")); realIP != "" {
		if net.ParseIP(realIP) != nil {
			return realIP
		}
	}
	return peer
}

// GetUserAgent helps in getting the user agent from the request
func GetUserAgent(r *http.Request) string {
	return r.UserAgent()
}
