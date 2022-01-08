package email

import (
	"crypto/tls"
	"log"
	"strconv"

	"github.com/authorizerdev/authorizer/server/constants"
	gomail "gopkg.in/mail.v2"
)

type Sender struct {
	User     string
	Password string
}

func NewSender() Sender {
	return Sender{User: constants.SMTP_USERNAME, Password: constants.SMTP_PASSWORD}
}

func (sender Sender) SendMail(to []string, Subject, bodyMessage string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", constants.SENDER_EMAIL)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", Subject)
	m.SetBody("text/html", bodyMessage)
	port, _ := strconv.Atoi(constants.SMTP_PORT)
	d := gomail.NewDialer(constants.SMTP_HOST, port, constants.SMTP_USERNAME, constants.SMTP_PASSWORD)
	if constants.ENV == "development" {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if err := d.DialAndSend(m); err != nil {
		log.Printf("smtp error: %s", err)
		return err
	}
	return nil
}
