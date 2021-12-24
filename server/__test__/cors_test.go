package test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCors(t *testing.T) {
	allowedOrigin := "http://localhost:8080" // The allowed origin that you want to check
	notAllowedOrigin := "http://myapp.com"

	s := testSetup()
	defer s.Server.Close()
	client := &http.Client{}

	req, _ := createContext(s)
	req.Header.Add("Origin", allowedOrigin)
	res, _ := client.Do(req)

	// You should get your origin (or a * depending on your config) if the
	// passed origin is allowed.
	o := res.Header.Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, o, notAllowedOrigin, "Origins should not match")
	assert.Equal(t, o, allowedOrigin, "Origins do match")
}
