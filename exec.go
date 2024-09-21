package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func playAlbum(execCmd string, paths []string, debug bool) error {
	// Extract the cmd name and args
	c := strings.Split(execCmd, " ")
	args := append(c[1:], paths...)
	cmd := exec.Command(c[0], args...)

	if debug {
		// Generate log filename
		ct := time.Now()
		ft := ct.Format("2006-01-02_15-04-05")
		name := fmt.Sprintf("%s_%s.txt", APP_NAME, ft)

		// Create file
		file, err := os.Create(name)
		if err != nil {
			return err
		}
		defer file.Close()

		// Pipe the commands output to the file
		cmd.Stdout = file
		cmd.Stderr = file
	}

	return cmd.Run()
}
