package util

import (
	"os"
)

// CanLoadFile returns true if a file can be opened or false if otherwise.
func CanLoadFile(path string) bool {
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return false
	}
	return true
}

func FileSize(path string) (int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func IsDirectory(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

// Cwd returns the current working directory or panics.
func Cwd() string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return pwd
}
