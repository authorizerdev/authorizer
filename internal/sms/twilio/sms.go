package twilio

import (
	twilio "github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

// SendSMS util to send sms
func (p *provider) SendSMS(sendTo, messageBody string) error {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username:   p.config.TwilioAPIKey,
		Password:   p.config.TwilioAPISecret,
		AccountSid: p.config.TwilioAccountSID,
	})
	message := &api.CreateMessageParams{}
	message.SetBody(messageBody)
	message.SetFrom(p.config.TwilioSender)
	message.SetTo(sendTo)
	_, err := client.Api.CreateMessage(message)
	if err != nil {
		p.dependencies.Log.Debug().Err(err).Msg("error sending sms")
		return err
	}

	return nil
}
