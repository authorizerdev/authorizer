package utils

import "net/mail"

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
