package metrics

import (
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var initOnce sync.Once

// Auth event names used as label values for AuthEventsTotal.
const (
	EventLogin         = "login"
	EventSignup        = "signup"
	EventLogout        = "logout"
	EventForgotPwd     = "forgot_password"
	EventResetPwd      = "reset_password"
	EventVerifyEmail   = "verify_email"
	EventVerifyOTP     = "verify_otp"
	EventMagicLink     = "magic_link_login"
	EventAdminLogin    = "admin_login"
	EventAdminLogout   = "admin_logout"
	EventOAuthLogin    = "oauth_login"
	EventOAuthCallback = "oauth_callback"
	EventTokenRefresh  = "token_refresh"
	EventTokenRevoke   = "token_revoke"

	StatusSuccess = "success"
	StatusFailure = "failure"
)

var (
	// HTTPRequestsTotal is the total number of HTTP requests received.
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authorizer_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration tracks the duration of HTTP requests in seconds.
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "authorizer_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// AuthEventsTotal is the total number of authentication events.
	AuthEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authorizer_auth_events_total",
			Help: "Total number of authentication events",
		},
		[]string{"event", "status"},
	)

	// ActiveSessions is the current number of active sessions.
	ActiveSessions = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "authorizer_active_sessions",
			Help: "Number of active sessions",
		},
	)

	// SecurityEventsTotal tracks security-sensitive events for alerting.
	SecurityEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authorizer_security_events_total",
			Help: "Total number of security-relevant events (failed logins, invalid tokens, etc.)",
		},
		[]string{"event", "reason"},
	)

	// GraphQLErrorsTotal tracks GraphQL responses that contain errors (HTTP 200 but with errors).
	GraphQLErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authorizer_graphql_errors_total",
			Help: "Total number of GraphQL responses containing errors",
		},
		[]string{"operation"},
	)

	// GraphQLRequestDuration tracks GraphQL operation latency.
	GraphQLRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "authorizer_graphql_request_duration_seconds",
			Help:    "GraphQL operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// DBHealthCheckTotal tracks database health check outcomes.
	DBHealthCheckTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authorizer_db_health_check_total",
			Help: "Total number of database health checks by result",
		},
		[]string{"status"},
	)

	// ClientIDNotFoundTotal tracks client ID not found events.
	ClientIDNotFoundTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "authorizer_client_id_not_found_total",
			Help: "Total number of client ID not found events",
		},
	)
)

// staticAssetPathSuffixes are path suffixes (after lowercasing) treated as static files
// for HTTP metrics filtering (images, icons, fonts, source maps, PWA manifest).
var staticAssetPathSuffixes = []string{
	".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".bmp", ".avif", ".jfif",
	".woff", ".woff2", ".ttf", ".otf", ".eot",
	".webmanifest",
	".map",
}

// SkipHTTPRequestMetrics reports whether a request path should be omitted from
// HTTP request counters and histograms (UI routes, static assets, favicons, images, fonts).
func SkipHTTPRequestMetrics(path string) bool {
	if path == "" {
		return false
	}
	if path == "/app" || strings.HasPrefix(path, "/app/") {
		return true
	}
	if path == "/dashboard" || strings.HasPrefix(path, "/dashboard/") {
		return true
	}
	if path == "/metrics" {
		return true
	}
	for _, seg := range strings.Split(path, "/") {
		if strings.HasPrefix(seg, "chunk-") {
			return true
		}
	}
	return skipHTTPRequestMetricsStaticAsset(path)
}

func skipHTTPRequestMetricsStaticAsset(path string) bool {
	p := strings.ToLower(path)
	if i := strings.Index(p, "?"); i >= 0 {
		p = p[:i]
	}
	switch p {
	case "/robots.txt", "/sitemap.xml", "/humans.txt", "/security.txt":
		return true
	}
	for _, suf := range staticAssetPathSuffixes {
		if strings.HasSuffix(p, suf) {
			return true
		}
	}
	file := p
	if i := strings.LastIndex(p, "/"); i >= 0 {
		file = p[i+1:]
	}
	if file == "" {
		return false
	}
	if strings.HasPrefix(file, "favicon") {
		return true
	}
	// Common browser / PWA icon filenames without matching suffix rules above.
	if strings.Contains(file, "apple-touch-icon") ||
		strings.Contains(file, "android-chrome") ||
		strings.Contains(file, "safari-pinned-tab") ||
		strings.Contains(file, "mstile-") {
		return true
	}
	return false
}

// Init registers all metrics with the default prometheus registry.
// It is safe to call multiple times; registration happens only once.
func Init() {
	initOnce.Do(func() {
		prometheus.MustRegister(HTTPRequestsTotal)
		prometheus.MustRegister(HTTPRequestDuration)
		prometheus.MustRegister(AuthEventsTotal)
		prometheus.MustRegister(ActiveSessions)
		prometheus.MustRegister(SecurityEventsTotal)
		prometheus.MustRegister(GraphQLErrorsTotal)
		prometheus.MustRegister(GraphQLRequestDuration)
		prometheus.MustRegister(DBHealthCheckTotal)
	})
}

// RecordAuthEvent records an authentication event with given status.
func RecordAuthEvent(event, status string) {
	AuthEventsTotal.WithLabelValues(event, status).Inc()
}

// RecordSecurityEvent records a security-relevant event for alerting.
func RecordSecurityEvent(event, reason string) {
	SecurityEventsTotal.WithLabelValues(event, reason).Inc()
}

// RecordGraphQLError records a GraphQL error for the given operation.
func RecordGraphQLError(operation string) {
	GraphQLErrorsTotal.WithLabelValues(operation).Inc()
}

// RecordClientIDNotFound records a client ID not found event.
func RecordClientIDNotFound() {
	ClientIDNotFoundTotal.Inc()
}
