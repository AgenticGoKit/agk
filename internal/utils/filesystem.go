package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// CreateDir creates a directory with all parent directories
func CreateDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// WriteFile writes content to a file, creating directories if needed
func WriteFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := CreateDir(dir); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(path, content, 0644)
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// IsEmptyDir checks if a directory is empty
func IsEmptyDir(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// ReadFile reads a file and returns its content
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// ReadFileAsString reads a file and returns it as a string
func ReadFileAsString(path string) (string, error) {
	content, err := ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ListDir lists all entries in a directory
func ListDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

// RemoveDir removes a directory and all its contents
func RemoveDir(path string) error {
	return os.RemoveAll(path)
}

// RemoveFile removes a file
func RemoveFile(path string) error {
	return os.Remove(path)
}

// EnsureDir ensures a directory exists, creating it if necessary
func EnsureDir(path string) error {
	if DirExists(path) {
		return nil
	}
	return CreateDir(path)
}
