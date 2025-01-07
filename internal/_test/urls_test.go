package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/stretchr/testify/assert"
)

func TestGetHostName(t *testing.T) {
	url := "http://test.herokuapp.com:80"

	host, port := parsers.GetHostParts(url)
	expectedHost := "test.herokuapp.com"

	assert.Equal(t, host, expectedHost, "hostname should be equal")
	assert.Equal(t, port, "80", "port should be 80")
}

func TestGetDomainName(t *testing.T) {
	url := "http://test.herokuapp.com"

	got := parsers.GetDomainName(url)
	want := "herokuapp.com"

	assert.Equal(t, got, want, "domain name should be equal")
}
