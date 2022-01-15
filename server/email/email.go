package email

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"log"
	"strconv"
	"text/template"

	"github.com/authorizerdev/authorizer/server/constants"
	gomail "gopkg.in/mail.v2"
)

// AddEmailTemplate is used to add html template in email body
func addEmailTemplate(a string, b map[string]interface{}, templateName string) string {
	tmpl, err := template.New(templateName).Parse(a)
	if err != nil {
		output, _ := json.Marshal(b)
		return string(output)
	}
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, b)
	if err != nil {
		panic(err)
	}
	s := buf.String()
	return s
}

func SendMail(to []string, Subject, bodyMessage string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", constants.EnvData.SENDER_EMAIL)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", Subject)
	m.SetBody("text/html", bodyMessage)
	port, _ := strconv.Atoi(constants.EnvData.SMTP_PORT)
	d := gomail.NewDialer(constants.EnvData.SMTP_HOST, port, constants.EnvData.SMTP_USERNAME, constants.EnvData.SMTP_PASSWORD)
	if constants.EnvData.ENV == "development" {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if err := d.DialAndSend(m); err != nil {
		log.Printf("smtp error: %s", err)
		return err
	}
	return nil
}
