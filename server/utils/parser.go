package utils

import (
	"errors"
	"strings"
	"time"
)

// ParseDurationInSeconds parses input s, removes ms/us/ns and returns result duration
func ParseDurationInSeconds(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}

	d = d.Truncate(time.Second)
	if d <= 0 {
		return 0, errors.New(`duration must be greater than 0s`)
	}

	return d, nil
}

// Helper function to parse string array values
func ParseStringArray(value string) []*string {
	if value == "" {
		return nil
	}
	splitValues := strings.Split(value, "|")

	var result []*string
	for _, s := range splitValues {
		temp := s
		result = append(result, &temp)
	}

	return result
}

// Helper function to parse reference string array values
func ParseReferenceStringArray(value *string) []string {
	if value == nil {
		return nil
	}

	// Dereference the pointer to get the string value
	strValue := *value

	// Remove JSON brackets
	strValue = strings.Trim(strValue, "{}")

	splitValues := strings.Split(strValue, ",")

	var result []string
	for _, s := range splitValues {
		// Split each key-value pair by colon ':'
		parts := strings.SplitN(s, ":", 2)
		if len(parts) > 0 {
			unquoted := strings.Trim(strings.TrimSpace(parts[0]), `"`)

			// Extract and append only the key (UUID)
			result = append(result, unquoted)
		}
	}

	return result
}
