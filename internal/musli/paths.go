package musli

import (
	"os"
	"path/filepath"
	"runtime"
)

func GetDefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(configDir, "musli", "config.toml")
	return path, nil
}

func GetLibraryPath() (string, error) {
	appDir, err := GetAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "library.db"), nil
}

func GetAppDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var appDir string
	appName := "musli"
	if runtime.GOOS == "linux" {
		xdgUserDir := os.Getenv("XDG_STATE_HOME")
		if xdgUserDir != "" {
			appDir = filepath.Join(xdgUserDir, appName)
		} else {
			appDir = filepath.Join(homeDir, ".state", appName)
		}
	} else {
		appDir = filepath.Join(homeDir, "."+appName)
	}

	err = os.MkdirAll(appDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	return appDir, nil
}
