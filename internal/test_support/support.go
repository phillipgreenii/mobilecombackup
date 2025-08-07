package test_support

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// based on https://stackoverflow.com/a/64733815/388006
// and https://golangbyexample.com/copy-file-go/
func CopyDir(source, destination string) error {
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		relPath := strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(destination, relPath), 0755)
		} else {
			return CopyFile(filepath.Join(source, relPath),
				filepath.Join(destination, relPath))
		}
	})
	return err
}

func CopyFile(source, destination string) error {
	s, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	d, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()

	//This will copy
	_, err = io.Copy(d, s)

	return err
}

func CountLines(source string) (int, error) {
	file, err := os.Open(source)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()
	fileScanner := bufio.NewScanner(file)
	lineCount := 0
	for fileScanner.Scan() {
		lineCount++
	}
	return lineCount, nil
}
