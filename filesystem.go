package gosolc

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type fileRef struct {
	path    string
	modTime time.Time
}

func readDir(dirPath string) ([]*fileRef, error) {
	files := []*fileRef{}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// filter by solidity files
		if !strings.HasSuffix(path, ".sol") {
			return nil
		}

		// use the relative path with respect to the contracts dir
		path = strings.TrimPrefix(path, dirPath+"/")

		files = append(files, &fileRef{
			path:    path,
			modTime: info.ModTime(),
		})
		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}
