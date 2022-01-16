package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func TestGetHostName(t *testing.T) {
	authorizer_url := "http://test.herokuapp.com:80"

	host, port := utils.GetHostParts(authorizer_url)
	expectedHost := "test.herokuapp.com"

	assert.Equal(t, host, expectedHost, "hostname should be equal")
	assert.Equal(t, port, "80", "port should be 80")
}

func TestGetDomainName(t *testing.T) {
	authorizer_url := "http://test.herokuapp.com"

	got := utils.GetDomainName(authorizer_url)
	want := "herokuapp.com"

	assert.Equal(t, got, want, "domain name should be equal")
}
