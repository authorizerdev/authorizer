package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHostName(t *testing.T) {
	authorizer_url := "http://test.herokuapp.com"

	got := GetHostName(authorizer_url)
	want := "test.herokuapp.com"

	assert.Equal(t, got, want, "hostname should be equal")
}

func TestGetDomainName(t *testing.T) {
	authorizer_url := "http://test.herokuapp.com"

	got := GetDomainName(authorizer_url)
	want := "herokuapp.com"

	assert.Equal(t, got, want, "domain name should be equal")
}
