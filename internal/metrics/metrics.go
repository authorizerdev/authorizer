package metrics

import (
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
)

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
