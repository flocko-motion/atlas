package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server string `toml:"server"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".rankedb", "etc", "ranke-cli.toml")
}

func loadConfig() (Config, error) {
	var cfg Config
	path := configPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // no config file is fine — use --server flag
		}
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}
