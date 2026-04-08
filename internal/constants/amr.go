package constants

// Authentication Methods Reference (amr) values per RFC 8176.
// These are the values emitted in the OIDC ID Token `amr` claim
// (OIDC Core §2). Only the subset Authorizer currently supports is
// listed; extend as new authentication methods land.
const (
	// AMRPassword indicates a password-based authentication ("pwd"
	// per RFC 8176 §2). Used for username/password and mobile/password
	// login flows.
	AMRPassword = "pwd"
	// AMROTP indicates a one-time password authentication ("otp"
	// per RFC 8176 §2). Used for magic-link and mobile-OTP flows.
	AMROTP = "otp"
	// AMRFederated indicates a federated identity assertion ("fed"
	// per RFC 8176 §2). Used for all upstream OAuth2/OIDC social
	// providers (Google, GitHub, Apple, etc.).
	AMRFederated = "fed"
	// AMRMFA indicates that multi-factor authentication was used
	// ("mfa" per RFC 8176 §2). Reserved for future use when MFA
	// signal is plumbed through to the ID token issuer.
	AMRMFA = "mfa"
)
