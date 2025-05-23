package utils

import (
	"errors"
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
