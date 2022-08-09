package utils

import (
	"reflect"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

// StringSliceContains checks if a string slice contains a particular string
func StringSliceContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// RemoveDuplicateString removes duplicate strings from a string slice
func RemoveDuplicateString(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

// ConvertInterfaceToSlice to convert interface to slice interface
func ConvertInterfaceToSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return nil
	}

	// Keep the distinction between nil and empty slice input
	if s.IsNil() {
		return nil
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}

// ConvertInterfaceToStringSlice to convert interface to string slice
func ConvertInterfaceToStringSlice(slice interface{}) []string {
	data := slice.([]interface{})
	var resSlice []string

	for _, v := range data {
		resSlice = append(resSlice, v.(string))
	}
	return resSlice
}

// GetOrganization to get organization object
func GetOrganization() map[string]interface{} {
	orgLogo, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyOrganizationLogo)
	if err != nil {
		return nil
	}
	orgName, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyOrganizationName)
	if err != nil {
		return nil
	}
	organization := map[string]interface{}{
		"name": orgName,
		"logo": orgLogo,
	}

	return organization
}

// GetForgotPasswordURL to get url for given token and hostname
func GetForgotPasswordURL(token, hostname string) string {
	resetPasswordUrl, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyResetPasswordURL)
	if err != nil {
		return ""
	}
	if resetPasswordUrl == "" {
		if err := memorystore.Provider.UpdateEnvVariable(constants.EnvKeyResetPasswordURL, hostname+"/app/reset-password"); err != nil {
			return ""
		}
	}
	verificationURL := resetPasswordUrl + "?token=" + token
	return verificationURL
}

// GetInviteVerificationURL to get url for invite email verification
func GetInviteVerificationURL(verificationURL, token, redirectURI string) string {
	return verificationURL + "?token=" + token + "&redirect_uri=" + redirectURI
}

// GetEmailVerificationURL to get url for invite email verification
func GetEmailVerificationURL(token, hostname string) string {
	return hostname + "/verify_email?token=" + token
}
