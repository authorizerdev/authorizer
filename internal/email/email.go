package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"strconv"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
	gomail "gopkg.in/mail.v2"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
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
		log.Error("Failed to get event template: ", err)
		return err
	}

	m := gomail.NewMessage()
	senderEmail, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeySenderEmail)
	if err != nil {
		log.Errorf("Error while getting sender email from env variable: %v", err)
		return err
	}

	senderName, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeySenderName)
	if err != nil {
		log.Errorf("Error while getting sender name from env variable: %v", err)
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

	smtpLocalName, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeySmtpLocalName)
	if err != nil {
		log.Debugf("Error while getting smtp localname from env variable: %v", err)
		smtpLocalName = ""
	}

	isProd, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsProd)
	if err != nil {
		log.Errorf("Error while getting env variable: %v", err)
		return err
	}

	m.SetAddressHeader("From", senderEmail, senderName)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", tmp.Subject)
	m.SetBody("text/html", tmp.Template)
	port, _ := strconv.Atoi(smtpPort)
	d := gomail.NewDialer(smtpHost, port, smtpUsername, smtpPassword)
	if !isProd {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if strings.TrimSpace(smtpLocalName) != "" {
		d.LocalName = smtpLocalName
	}

	if err := d.DialAndSend(m); err != nil {
		log.Debug("SMTP Failed: ", err)
		return err
	}
	return nil
}
