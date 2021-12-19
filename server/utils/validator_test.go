package utils

import "testing"

func TestIsValidEmail(t *testing.T) {
	validEmail := "lakhan@gmail.com"
	invalidEmail1 := "lakhan"
	invalidEmail2 := "lakhan.me"

	if IsValidEmail(validEmail) != true {
		t.Errorf("IsValidEmail Test failed got %t, wanted %t for %s", false, true, validEmail)
	}

	if IsValidEmail(invalidEmail1) != false {
		t.Errorf("IsValidEmail Test failed got %t, wanted %t for %s", true, false, invalidEmail1)
	}

	if IsValidEmail(invalidEmail2) != false {
		t.Errorf("IsValidEmail Test failed got %t, wanted %t for %s", true, false, invalidEmail2)
	}
}
