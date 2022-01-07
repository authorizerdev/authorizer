package email

import (
	"bytes"
	"fmt"
	"log"
	"mime/quotedprintable"
	"net/smtp"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
)

/**
	Using: https://github.com/tangingw/go_smtp/blob/master/send_mail.go
	For gmail add instruction to enable less security
	// https://myaccount.google.com/u/0/lesssecureapps
	// https://www.google.com/settings/security/lesssecureapps
	// https://stackoverflow.com/questions/19877246/nodemailer-with-gmail-and-nodejs
**/

// TODO -> try using gomail.v2

type Sender struct {
	User     string
	Password string
}

func NewSender() Sender {
	return Sender{User: constants.SMTP_USERNAME, Password: constants.SMTP_PASSWORD}
}

func (sender Sender) SendMail(Dest []string, Subject, bodyMessage string) error {
	msg := "From: " + constants.SENDER_EMAIL + "\n" +
		"To: " + strings.Join(Dest, ",") + "\n" +
		"Subject: " + Subject + "\n" + bodyMessage

	err := smtp.SendMail(constants.SMTP_HOST+":"+constants.SMTP_PORT,
		smtp.PlainAuth("", sender.User, sender.Password, constants.SMTP_HOST),
		constants.SENDER_EMAIL, Dest, []byte(msg))
	if err != nil {
		log.Printf("smtp error: %s", err)
		return err
	}

	return nil
}

func (sender Sender) WriteEmail(dest []string, contentType, subject, bodyMessage string) string {
	header := make(map[string]string)
	header["From"] = sender.User

	receipient := ""

	for _, user := range dest {
		receipient = receipient + user
	}

	header["To"] = receipient
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = fmt.Sprintf("%s; charset=\"utf-8\"", contentType)
	header["Content-Transfer-Encoding"] = "quoted-printable"
	header["Content-Disposition"] = "inline"

	message := ""

	for key, value := range header {
		message += fmt.Sprintf("%s: %s\r\n", key, value)
	}

	var encodedMessage bytes.Buffer

	finalMessage := quotedprintable.NewWriter(&encodedMessage)
	finalMessage.Write([]byte(bodyMessage))
	finalMessage.Close()

	message += "\r\n" + encodedMessage.String()

	return message
}

func (sender *Sender) WriteHTMLEmail(dest []string, subject, bodyMessage string) string {
	return sender.WriteEmail(dest, "text/html", subject, bodyMessage)
}

func (sender *Sender) WritePlainEmail(dest []string, subject, bodyMessage string) string {
	return sender.WriteEmail(dest, "text/plain", subject, bodyMessage)
}
