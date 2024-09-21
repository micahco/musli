package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	MusicDir string
	ExecCmd  string
	Debug    bool
}

// Default config values
var DefaultConfig = Config{
	MusicDir: "$HOME/Music",
	ExecCmd:  "mpv",
	Debug:    false,
}

func Load(appDir string) (Config, error) {
	// Start with default values
	conf := DefaultConfig

	configDir, err := os.UserConfigDir()
	if err != nil {
		return conf, err
	}

	path := filepath.Join(configDir, appDir, "config.toml")

	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return conf, err
	}

	_, err = toml.DecodeFile(path, &conf)
	// Disregard if file does not exist
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return conf, err
	}

	// Allows the use of $HOME and other env vars in the config.toml
	conf.MusicDir = os.ExpandEnv(conf.MusicDir)

	return conf, nil
}
