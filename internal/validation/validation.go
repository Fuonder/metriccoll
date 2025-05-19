package validation

import (
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"os"
	"strconv"
)

func CheckFilePresence(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func CheckPathWritable(path string) error {
	if path == "" {
		return fmt.Errorf("path can not be empty")
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("can not create file \"%s\": %w", path, err)
			}
			defer func(file *os.File) {
				err := file.Close()
				if err != nil {
					fmt.Printf("failed to close file \"%s\": %v\n", path, err)
				}
			}(file)
		} else {
			return fmt.Errorf("can not get information about path \"%s\": %w", path, err)
		}
	}

	file, err := os.OpenFile(path, os.O_RDWR, storage.OsAllRw)
	if err != nil {
		return fmt.Errorf("can not open file in Write mode: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("failed to close file \"%s\": %v\n", path, err)
		}
	}(file)

	return nil
}

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
