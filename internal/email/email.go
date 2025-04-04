package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"strings"
	"text/template"

	"github.com/rs/zerolog"
	gomail "gopkg.in/mail.v2"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/email/templates"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage"
)

// Provider interface for email provider
type Provider interface {
	SendEmail(to []string, event string, data map[string]interface{}) error
}

// Dependencies struct for email provider
type Dependencies struct {
	Log             *zerolog.Logger
	StorageProvider storage.Provider
}

// provider struct for email provider
type provider struct {
	config *config.Config
	deps   *Dependencies

	mailer *gomail.Dialer
}

// New returns a new email provider
func New(
	config *config.Config,
	deps *Dependencies,
) (Provider, error) {
	mailer := gomail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)
	if strings.TrimSpace(config.SMTPLocalName) != "" {
		mailer.LocalName = config.SMTPLocalName
	}
	if config.SkipTLSVerification {
		mailer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &provider{
		config: config,
		deps:   deps,
		mailer: mailer,
	}, nil
}

// SendEmail function to send mail
func (p *provider) SendEmail(to []string, event string, data map[string]interface{}) error {
	log := p.deps.Log.With().Str("func", "send_email").Str("event", event).Logger()
	// Don't trigger email sending in case of test
	if p.config.Env == constants.TestEnv {
		return nil
	}

	tmp, err := p.getEmailTemplate(event, data)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get email template")
		return err
	}

	m := gomail.NewMessage()
	m.SetAddressHeader("From", p.config.SMTPSenderEmail, p.config.SMTPSenderName)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", tmp.Subject)
	m.SetBody("text/html", tmp.Template)
	if err := p.mailer.DialAndSend(m); err != nil {
		log.Debug().Err(err).Msg("Failed to send email")
		return err
	}
	return nil
}

func getDefaultTemplate(event string) *model.EmailTemplate {
	switch event {
	case constants.VerificationTypeBasicAuthSignup, constants.VerificationTypeMagicLinkLogin, constants.VerificationTypeUpdateEmail:
		return &model.EmailTemplate{
			Subject:  templates.EmailVerificationSubject,
			Template: templates.EmailVerificationTemplate,
		}
	case constants.VerificationTypeForgotPassword:
		return &model.EmailTemplate{
			Subject:  templates.ForgotPasswordSubject,
			Template: templates.ForgotPasswordTemplate,
		}
	case constants.VerificationTypeInviteMember:
		return &model.EmailTemplate{
			Subject:  templates.InviteUserEmailSubject,
			Template: templates.InviteUserEmailTemplate,
		}
	case constants.VerificationTypeOTP:
		return &model.EmailTemplate{
			Subject:  templates.OtpEmailSubject,
			Template: templates.OtpEmailTemplate,
		}
	default:
		return nil
	}
}

func (p *provider) getEmailTemplate(event string, data map[string]interface{}) (*model.EmailTemplate, error) {
	ctx := context.Background()
	var tmp *model.EmailTemplate
	et, err := p.deps.StorageProvider.GetEmailTemplateByEventName(ctx, event)
	if err != nil || et == nil {
		tmp = getDefaultTemplate(event)
	} else {
		tmp = et.AsAPIEmailTemplate()
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
