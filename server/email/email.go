package email

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"log"
	"strconv"
	"text/template"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	gomail "gopkg.in/mail.v2"
)

// addEmailTemplate is used to add html template in email body
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

// SendMail function to send mail
func SendMail(to []string, Subject, bodyMessage string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeySenderEmail))
	m.SetHeader("To", to...)
	m.SetHeader("Subject", Subject)
	m.SetBody("text/html", bodyMessage)
	port, _ := strconv.Atoi(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeySmtpPort))
	d := gomail.NewDialer(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeySmtpHost), port, envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeySmtpUsername), envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeySmtpPassword))
	if envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyEnv) == "development" {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if err := d.DialAndSend(m); err != nil {
		log.Printf("smtp error: %s", err)
		return err
	}
	return nil
}
