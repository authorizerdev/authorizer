package metrics

import (
	"strings"
	"testing"

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
