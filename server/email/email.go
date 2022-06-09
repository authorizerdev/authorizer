package email

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"strconv"
	"text/template"

	log "github.com/sirupsen/logrus"
	gomail "gopkg.in/mail.v2"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
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
	// dont trigger email sending in case of test
	envKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyEnv)
	if err != nil {
		return err
	}
	if envKey == constants.TestEnv {
		return nil
	}
	m := gomail.NewMessage()
	senderEmail, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeySenderEmail)
	if err != nil {
		log.Errorf("Error while getting sender email from env variable: %v", err)
		return err
	}

	smtpPort, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeySmtpPort)
	if err != nil {
		log.Errorf("Error while getting smtp port from env variable: %v", err)
		return err
	}

	smtpHost, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeySmtpHost)
	if err != nil {
		log.Errorf("Error while getting smtp host from env variable: %v", err)
		return err
	}

	smtpUsername, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeySmtpUsername)
	if err != nil {
		log.Errorf("Error while getting smtp username from env variable: %v", err)
		return err
	}

	smtpPassword, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeySmtpPassword)
	if err != nil {
		log.Errorf("Error while getting smtp password from env variable: %v", err)
		return err
	}

	isProd, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsProd)
	if err != nil {
		log.Errorf("Error while getting env variable: %v", err)
		return err
	}

	m.SetHeader("From", senderEmail)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", Subject)
	m.SetBody("text/html", bodyMessage)
	port, _ := strconv.Atoi(smtpPort)
	d := gomail.NewDialer(smtpHost, port, smtpUsername, smtpPassword)
	if !isProd {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if err := d.DialAndSend(m); err != nil {
		log.Debug("SMTP Failed: ", err)
		return err
	}
	return nil
}
