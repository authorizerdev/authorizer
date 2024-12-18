package utils

import (
	"reflect"

	"github.com/authorizerdev/authorizer/internal/config"
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
func GetOrganization(cfg *config.Config) map[string]interface{} {
	orgLogo := cfg.OrganizationLogo
	orgName := cfg.OrganizationName
	organization := map[string]interface{}{
		"name": orgName,
		"logo": orgLogo,
	}

	return organization
}

// GetForgotPasswordURL to get url for given token and hostname
func GetForgotPasswordURL(token, redirectURI string) string {
	verificationURL := redirectURI + "?token=" + token
	return verificationURL
}

// GetInviteVerificationURL to get url for invite email verification
func GetInviteVerificationURL(verificationURL, token, redirectURI string) string {
	return verificationURL + "?token=" + token + "&redirect_uri=" + redirectURI
}

// GetEmailVerificationURL to get url for invite email verification
func GetEmailVerificationURL(token, hostname, redirectURI string) string {
	return hostname + "/verify_email?token=" + token + "&redirect_uri=" + redirectURI
}

// FindDeletedValues find deleted values between original and updated one
func FindDeletedValues(original, updated []string) []string {
	deletedValues := make([]string, 0)

	// Create a map to store elements of the updated array for faster lookups
	updatedMap := make(map[string]bool)
	for _, value := range updated {
		updatedMap[value] = true
	}

	// Check for deleted values in the original array
	for _, value := range original {
		if _, found := updatedMap[value]; !found {
			deletedValues = append(deletedValues, value)
		}
	}

	return deletedValues
}

// DeleteFromArray will delete array from an array
func DeleteFromArray(original, valuesToDelete []string) []string {
	result := make([]string, 0)

	// Create a map to store values to delete for faster lookups
	valuesToDeleteMap := make(map[string]bool)
	for _, value := range valuesToDelete {
		valuesToDeleteMap[value] = true
	}

	// Check if each element in the original array should be deleted
	for _, value := range original {
		if _, found := valuesToDeleteMap[value]; !found {
			result = append(result, value)
		}
	}

	return result
}
