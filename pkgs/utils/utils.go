package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const maxFileNameLength = 255

var sanitizeRe = regexp.MustCompile(`[<>:"'/\\|?*\s]+`)

func Mkdir(path string) error {
	if err := os.Mkdir(path, 0775); err != nil && !os.IsExist(err) {
		println(err)
		return err
	}
	return nil
}

func SanitizeFilename(filename string) string {
	sanitized := sanitizeRe.ReplaceAllString(filename, "_")
	sanitized = strings.TrimSpace(sanitized)

	if len(sanitized) > maxFileNameLength {
		sanitized = sanitized[:maxFileNameLength]
	}

	return sanitized
}

func ClearTmp(tempDirPath string) error {
	err := filepath.Walk(tempDirPath, func(path string, info fs.FileInfo, err error) error {
		if info.Name() == tempDirPath {
			return nil
		}
		if err != nil {
			return err
		}

		if info.IsDir() {
			err = os.RemoveAll(path)
			return err
		}

		if !info.IsDir() {
			err = os.Remove(path)
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
