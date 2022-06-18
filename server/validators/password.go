package validators

import (
	"errors"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

// ValidatePassword to validate the password against the following policy
// min char length: 6
// max char length: 36
// at least one upper case letter
// at least one lower case letter
// at least one digit
// at least one special character
func IsValidPassword(password string) error {
	if len(password) < 6 || len(password) > 36 {
		return errors.New("password must be of minimum 6 characters and maximum 36 characters")
	}

	// if strong password is disabled
	// just check for min 6 chars & max 36
	isStrongPasswordDisabled, _ := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableStrongPassword)
	if isStrongPasswordDisabled {
		return nil
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

	isValid := hasUpperCase && hasLowerCase && hasDigit && hasSpecialChar

	if isValid {
		return nil
	}

	return errors.New(`password is not valid. It needs to be at least 6 characters long and contain at least one number, one uppercase letter, one lowercase letter and one special character`)
}
