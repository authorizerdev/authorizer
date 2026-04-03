package integration_tests

import (
	"os"
	"testing"

	"github.com/authorizerdev/authorizer/internal/metrics"
)

func TestMain(m *testing.M) {
	metrics.Init()
	os.Exit(m.Run())
}
