package config

import (
	"fmt"
	"os"

	"ssh-bastion/models"

	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = "config.yaml"

type Config struct {
	ListenAddr    string           `yaml:"listen_addr"`
	HostKeyPath   string           `yaml:"host_key_path"`
	AuthorizedKey string           `yaml:"authorized_key"`
	Targets       []models.Target  `yaml:"targets"`
}

// Cfg holds the loaded configuration. Call Load() before accessing.
var Cfg Config

func Load(path string) error {
	Cfg = Config{}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &Cfg); err != nil {
		return fmt.Errorf("parsing config file: %w", err)
	}

	if Cfg.ListenAddr == "" {
		Cfg.ListenAddr = ":2222"
	}
	if Cfg.HostKeyPath == "" {
		Cfg.HostKeyPath = "host_key"
	}
	if Cfg.AuthorizedKey == "" {
		return fmt.Errorf("authorized_key is required in config")
	}
	if len(Cfg.Targets) == 0 {
		return fmt.Errorf("at least one target is required in config")
	}

	return nil
}
