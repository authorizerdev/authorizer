package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"strconv"
	"text/template"

	log "github.com/sirupsen/logrus"
	gomail "gopkg.in/mail.v2"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

func getDefaultTemplate(event string) *model.EmailTemplate {
	switch event {
	case constants.VerificationTypeBasicAuthSignup, constants.VerificationTypeMagicLinkLogin, constants.VerificationTypeUpdateEmail:
		return &model.EmailTemplate{
			Subject:  emailVerificationSubject,
			Template: emailVerificationTemplate,
		}
	case constants.VerificationTypeForgotPassword:
		return &model.EmailTemplate{
			Subject:  forgotPasswordSubject,
			Template: forgotPasswordTemplate,
		}
	case constants.VerificationTypeInviteMember:
		return &model.EmailTemplate{
			Subject:  inviteEmailSubject,
			Template: inviteEmailTemplate,
		}
	case constants.VerificationTypeOTP:
		return &model.EmailTemplate{
			Subject:  otpEmailSubject,
			Template: otpEmailTemplate,
		}
	default:
		return nil
	}
}

func getEmailTemplate(event string, data map[string]interface{}) (*model.EmailTemplate, error) {
	ctx := context.Background()
	tmp, err := db.Provider.GetEmailTemplateByEventName(ctx, event)
	if err != nil || tmp == nil {
		tmp = getDefaultTemplate(event)
	}

	templ, err := template.New(event + "_template.tmpl").Parse(tmp.Template)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	err = templ.Execute(buf, data)
	if err != nil {
		return nil, err
	}
	templateString := buf.String()

	subject, err := template.New(event + "_subject.tmpl").Parse(tmp.Subject)
	if err != nil {
		return nil, err
	}
	buf = &bytes.Buffer{}
	err = subject.Execute(buf, data)
	if err != nil {
		return nil, err
	}
	subjectString := buf.String()

	return &model.EmailTemplate{
		Template: templateString,
		Subject:  subjectString,
	}, nil
}

// SendEmail function to send mail
func SendEmail(to []string, event string, data map[string]interface{}) error {
	// dont trigger email sending in case of test
	envKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyEnv)
	if err != nil {
		return err
	}
	if envKey == constants.TestEnv {
		return nil
	}

	tmp, err := getEmailTemplate(event, data)
	if err != nil {
		log.Errorf("Failed to get event template: ", err)
		return err
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
	m.SetHeader("Subject", tmp.Subject)
	m.SetBody("text/html", tmp.Template)
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
