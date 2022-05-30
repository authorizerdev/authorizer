package validators

import "net/mail"

// IsValidEmail validates email
func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
