package main

import (
	"io/fs"
	"path/filepath"
	"strings"
)

func isAudioFile(path string) bool {
	ext := filepath.Ext(path)
	switch strings.ToUpper(ext) {
	case
		".MP3", ".M4A", ".M4B", ".M4P", ".ALAC", ".FLAC", ".OGG", ".DSF":
		return true
	}
	return false
}

func findAudioFilePaths(dir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, di fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !di.IsDir() && isAudioFile(path) {
			paths = append(paths, path)
		}

		return nil
	})
	return paths, err
}
