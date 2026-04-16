package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server string `toml:"server"`
}

var Cfg Config

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".rankedb", "etc", "ranke-cli.toml")
}

func LoadConfig() error {
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return initConfig(path)
		}
		return fmt.Errorf("read config %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, &Cfg); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	return nil
}

func initConfig(path string) error {
	fmt.Println("No configuration found. Let's set one up.")
	fmt.Print("RankeDB server URL [http://localhost:8000]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		input = "http://localhost:8000"
	}

	Cfg.Server = input

	content := fmt.Sprintf("# RankeDB CLI configuration\nserver = %q\n", input)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}

	fmt.Printf("Config saved to %s\n\n", path)
	return nil
}
