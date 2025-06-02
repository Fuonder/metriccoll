package filevalidation

import (
	"fmt"
	"github.com/Fuonder/metriccoll.git/internal/storage"
	"os"
	"path/filepath"
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

func findFile(root string, fileName string) (string, error) {
	var result string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == fileName {
			result = path
			return filepath.SkipDir // stop walking once found
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error walking directory: %w", err)
	}

	if result == "" {
		return "", fmt.Errorf("file %s not found", fileName)
	}

	return result, nil
}

func FindCRTFile() (string, error) {
	return findFile(".", "server.crt")
}

func FindKEYFile() (string, error) {
	return findFile(".", "server.key")
}
