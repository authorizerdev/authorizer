package utils

import (
	"errors"
	"os"
)

// CreateFolder creates a folder in Current working dir
func CreateFolder(dir string) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	path := pwd + "/" + dir
	err = os.Mkdir(path, 0o755)
	if err == nil {
		return path, nil
	}
	if os.IsExist(err) {
		// check that the existing path is a directory
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			return "", errors.New("path exists but is not a directory")
		}
		return path, nil
	}
	return path, err
}

// CreateFile creates a file on given path with given content
func CreateFile(filePath string, content string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.WriteString(content)

	if err != nil {
		return err
	}
	return nil
}
