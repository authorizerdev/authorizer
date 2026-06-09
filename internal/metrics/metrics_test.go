package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGraphQLOperationPrometheusLabel(t *testing.T) {
	assert.Equal(t, "anonymous", GraphQLOperationPrometheusLabel(""))
	assert.Equal(t, "anonymous", GraphQLOperationPrometheusLabel("   "))
	got := GraphQLOperationPrometheusLabel("LoginOp")
	assert.True(t, strings.HasPrefix(got, "op_"))
	assert.Len(t, got, len("op_")+16) // 8 bytes hex
}

func TestSkipHTTPRequestMetrics(t *testing.T) {
	tests := []struct {
		path string
		skip bool
	}{
		{path: "", skip: false},
		{path: "/api/v1", skip: false},
		{path: "/app", skip: true},
		{path: "/app/", skip: true},
		{path: "/app/static/chunk-abc.js", skip: true},
		{path: "/dashboard", skip: true},
		{path: "/dashboard/", skip: true},
		{path: "/dashboard/users", skip: true},
		{path: "/metrics", skip: true},
		{path: "/static/chunk-vendors.js", skip: true},
		{path: "/assets/chunk-main.hash.js", skip: true},
		{path: "/favicon.ico", skip: true},
		{path: "/icons/favicon-32x32.png", skip: true},
		{path: "/apple-touch-icon.png", skip: true},
		{path: "/PWA/android-chrome-192x192.png", skip: true},
		{path: "/file.woff2", skip: true},
		{path: "/site.webmanifest", skip: true},
		{path: "/app/bundle.js.map", skip: true},
		{path: "/robots.txt", skip: true},
		{path: "/sitemap.xml", skip: true},
		{path: "/humans.txt", skip: true},
		{path: "/security.txt", skip: true},
		{path: "/logo.PNG", skip: true},
		{path: "/image.JPG?query=1", skip: true},
		{path: "/path?query=/app/foo", skip: false},
	}
	for _, tt := range tests {
		name := tt.path
		if name == "" {
			name = "(empty)"
		}
		t.Run(name, func(t *testing.T) {
			got := SkipHTTPRequestMetrics(tt.path)
			assert.Equal(t, tt.skip, got, "path=%q", tt.path)
		})
	}
}

func TestSkipHTTPRequestMetrics_chunkSegment(t *testing.T) {
	// Path segment must be prefixed with "chunk-", not merely contain it.
	assert.False(t, SkipHTTPRequestMetrics("/foo/mychunk-file.js"))
	assert.True(t, SkipHTTPRequestMetrics("/chunk-xyz"))
}

func TestRecordFgaCheck(t *testing.T) {
	// RecordFgaCheckResult maps the boolean decision to the right label.
	allowBefore := testutil.ToFloat64(FgaChecksTotal.WithLabelValues(FgaOpCheck, FgaResultAllowed))
	RecordFgaCheckResult(FgaOpCheck, true)
	assert.Equal(t, allowBefore+1,
		testutil.ToFloat64(FgaChecksTotal.WithLabelValues(FgaOpCheck, FgaResultAllowed)))

	denyBefore := testutil.ToFloat64(FgaChecksTotal.WithLabelValues(FgaOpBatchCheck, FgaResultDenied))
	RecordFgaCheckResult(FgaOpBatchCheck, false)
	assert.Equal(t, denyBefore+1,
		testutil.ToFloat64(FgaChecksTotal.WithLabelValues(FgaOpBatchCheck, FgaResultDenied)))

	errBefore := testutil.ToFloat64(FgaChecksTotal.WithLabelValues(FgaOpCheck, FgaResultError))
	RecordFgaCheck(FgaOpCheck, FgaResultError)
	assert.Equal(t, errBefore+1,
		testutil.ToFloat64(FgaChecksTotal.WithLabelValues(FgaOpCheck, FgaResultError)))
}

func TestRecordFgaOperation(t *testing.T) {
	before := testutil.ToFloat64(FgaOperationsTotal.WithLabelValues(FgaOpWriteModel, FgaResultSuccess))
	RecordFgaOperation(FgaOpWriteModel, FgaResultSuccess)
	assert.Equal(t, before+1,
		testutil.ToFloat64(FgaOperationsTotal.WithLabelValues(FgaOpWriteModel, FgaResultSuccess)))

	errBefore := testutil.ToFloat64(FgaOperationsTotal.WithLabelValues(FgaOpReset, FgaResultError))
	RecordFgaOperation(FgaOpReset, FgaResultError)
	assert.Equal(t, errBefore+1,
		testutil.ToFloat64(FgaOperationsTotal.WithLabelValues(FgaOpReset, FgaResultError)))
}

func TestObserveFgaCheckDuration(t *testing.T) {
	ObserveFgaCheckDuration(FgaOpListObjects, 0.01)
	ObserveFgaCheckDuration(FgaOpCheck, 0.02)
	// At least the two observed series are present in the histogram vec.
	assert.GreaterOrEqual(t, testutil.CollectAndCount(FgaCheckDuration), 1)
}
