package utils

import "testing"

func TestGetHostName(t *testing.T) {
	authorizer_url := "http://test.herokuapp.com"

	got := GetHostName(authorizer_url)
	want := "test.herokuapp.com"

	if got != want {
		t.Errorf("GetHostName Test failed got %q, wanted %q", got, want)
	}
}

func TestGetDomainName(t *testing.T) {
	authorizer_url := "http://test.herokuapp.com"

	got := GetDomainName(authorizer_url)
	want := "herokuapp.com"

	if got != want {
		t.Errorf("GetHostName Test failed got %q, wanted %q", got, want)
	}
}
