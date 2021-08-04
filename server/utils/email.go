package utils

import (
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/email"
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
	`, constants.AUTHORIZER_URL+"/verify_email"+"?token="+token)
	bodyMessage := sender.WriteHTMLEmail(Receiver, Subject, message)

	return sender.SendMail(Receiver, Subject, bodyMessage)
}

// SendForgotPasswordMail to send verification email
func SendForgotPasswordMail(toEmail, token, host string) error {
	sender := email.NewSender()

	// The receiver needs to be in slice as the receive supports multiple receiver
	Receiver := []string{toEmail}

	Subject := "Reset Password"
	message := fmt.Sprintf(`
	<!DOCTYPE HTML PULBLIC "-//W3C//DTD HTML 4.01 Transitional//EN">
	<html>
	<head>
	<meta http-equiv="content-type" content="text/html"; charset=ISO-8859-1">
	</head>
	<body>
		<h1>Please use the link below to reset password </h1><br/>
		<a href="%s">Reset Password</a>
	</body>
	</html>
	`, host+"/"+constants.FORGOT_PASSWORD_URI+"?token="+token)
	bodyMessage := sender.WriteHTMLEmail(Receiver, Subject, message)

	return sender.SendMail(Receiver, Subject, bodyMessage)
}
