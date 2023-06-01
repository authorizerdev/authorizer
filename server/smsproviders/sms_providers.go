package smsproviders

type SMSProviders interface {
	// Authenticate
	SetCredentials(APIkey string, APIsecret string) 
}
