package constants

// Authentication Context Class Reference (acr) values per OIDC Core §2.
// The acr claim communicates the strength of the authentication context
// to the relying party. Authorizer currently emits a fixed baseline
// value; richer ACR handling (driven by `acr_values` request parameter
// and MFA awareness) is a future enhancement.
const (
	// ACRBaseline is the no-op baseline acr value. It matches the
	// "0" level of assurance described by ISO/IEC 29115 and noted
	// in OIDC Core §2 as the lowest meaningful claim. Emitted so
	// that clients which require the claim's presence keep working.
	ACRBaseline = "0"
)
