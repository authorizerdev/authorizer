package models

// CollectionList / Tables available for authorizer in the database
type CollectionList struct {
	User                   string
	VerificationRequest    string
	Session                string
	Env                    string
	Webhook                string
	WebhookLog             string
	EmailTemplate          string
	OTP                    string
	SMSVerificationRequest string
	Authenticators         string
}

var (
	// Prefix for table name / collection names
	Prefix = "authorizer_"
	// Collections / Tables available for authorizer in the database (used for dbs other than gorm)
	Collections = CollectionList{
		User:                   Prefix + "users",
		VerificationRequest:    Prefix + "verification_requests",
		Session:                Prefix + "sessions",
		Env:                    Prefix + "env",
		Webhook:                Prefix + "webhooks",
		WebhookLog:             Prefix + "webhook_logs",
		EmailTemplate:          Prefix + "email_templates",
		OTP:                    Prefix + "otps",
		SMSVerificationRequest: Prefix + "sms_verification_requests",
		Authenticators:         Prefix + "authenticators",
	}
)
