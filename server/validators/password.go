package validators

// ValidatePassword to validate the password against the following policy
// min char length: 6
// max char length: 36
// at least one upper case letter
// at least one lower case letter
// at least one digit
// at least one special character
func IsValidPassword(password string) bool {
	if len(password) < 6 || len(password) > 36 {
		return false
	}

	hasUpperCase := false
	hasLowerCase := false
	hasDigit := false
	hasSpecialChar := false

	for _, char := range password {
		if char >= 'A' && char <= 'Z' {
			hasUpperCase = true
		} else if char >= 'a' && char <= 'z' {
			hasLowerCase = true
		} else if char >= '0' && char <= '9' {
			hasDigit = true
		} else {
			hasSpecialChar = true
		}
	}

	return hasUpperCase && hasLowerCase && hasDigit && hasSpecialChar
}
