package utils

import (
	"fmt"

	"github.com/yauthdev/yauth/server/constants"
	"github.com/yauthdev/yauth/server/email"
)

// SendVerificationMail to send verification email
func SendVerificationMail(toEmail, token string) error {
	sender := email.NewSender()

	// The receiver needs to be in slice as the receive supports multiple receiver
	Receiver := []string{toEmail}

	Subject := "Please verify your email"
	message := fmt.Sprintf(`
	<!DOCTYPE HTML PULBLIC "-//W3C//DTD HTML 4.01 Transitional//EN">
	<html>
	<head>
	<meta http-equiv="content-type" content="text/html"; charset=ISO-8859-1">
	</head>
	<body>
		<h1>Please verify your email by clicking on the link below </h1><br/>
		<a href="%s">Click here to verify</a>
	</body>
	</html>
	`, constants.FRONTEND_URL+"/verify?token="+token)
	bodyMessage := sender.WriteHTMLEmail(Receiver, Subject, message)

	return sender.SendMail(Receiver, Subject, bodyMessage)
}
