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
