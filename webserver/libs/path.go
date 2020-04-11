package libs

import (
	"os"
)

type FileType string

const (
	NotFound  FileType = "NOT_FOUND"
	File      FileType = "FILE"
	Directory FileType = "DIRECTORY"
)

// Checks that path or file exists
func PathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	} else {
		return err == nil
	}
}

// Checks if path type (file or directory)
func PathType(path string) FileType {
	if !PathExists(path) {
		return NotFound
	}
	f, _ := os.Stat(path)
	if f.IsDir() {
		return Directory
	} else {
		return File
	}
}
