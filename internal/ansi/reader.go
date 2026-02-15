package ansi

import (
	"os"
	"path/filepath"
)

// ReadFile reads an ANSI file and returns its contents.
func ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

