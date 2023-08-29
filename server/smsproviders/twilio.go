package smsproviders

import (
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
	log "github.com/sirupsen/logrus"
	twilio "github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

// SendSMS util to send sms
// TODO: Should be restructured to interface when another provider is added
func SendSMS(sendTo, messageBody string) error {
	twilioAPISecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyTwilioAPISecret)
	if err != nil || twilioAPISecret == "" {
		log.Debug("Failed to get api secret: ", err)
		return err
	}
	twilioAPIKey, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyTwilioAPIKey)
	if err != nil || twilioAPIKey == "" {
		log.Debug("Failed to get api key: ", err)
		return err
	}
	twilioSenderFrom, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyTwilioSender)
	if err != nil || twilioSenderFrom == "" {
		log.Debug("Failed to get sender: ", err)
		return err
	}
	// accountSID is not a must to send sms on twilio
	twilioAccountSID, _ := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyTwilioAccountSID)
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username:   twilioAPIKey,
		Password:   twilioAPISecret,
		AccountSid: twilioAccountSID,
	})
	message := &api.CreateMessageParams{}
	message.SetBody(messageBody)
	message.SetFrom(twilioSenderFrom)
	message.SetTo(sendTo)

	_, err = client.Api.CreateMessage(message)

	if err != nil {
		log.Debug("Failed to send sms: ", err)
		return err
	}

	return nil
}
