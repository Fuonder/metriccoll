package numericvalidation

import (
	"fmt"
	"strconv"
)

func ValidateNonNegativeString(interval string) error {
	i, err := strconv.ParseInt(interval, 10, 64)
	if err != nil {
		return fmt.Errorf("malformed interval value: \"%s\": %w", interval, err)
	}
	return ValidateNonNegativeInt64(i)
}

func ValidateNonNegativeInt64(interval int64) error {
	if interval <= 0 {
		return fmt.Errorf("interval out of range: %d", interval)
	}
	return nil
}

func ValidatePositiveString(interval string) error {
	i, err := strconv.ParseInt(interval, 10, 64)
	if err != nil {
		return fmt.Errorf("malformed interval value: \"%s\": %w", interval, err)
	}
	return ValidatePositiveInt64(i)
}

func ValidatePositiveInt64(interval int64) error {
	if interval < 0 {
		return fmt.Errorf("interval out of range: %d", interval)
	}
	return nil
}
