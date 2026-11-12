package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/kopytkg/golog/v2"
)

type Manager struct {
	Current *Config
	Path    string
	mu      sync.Mutex
}

type Config struct {
	Token      string     `toml:"Token"`
	BaseUri    string     `toml:"BaseUri"`
	ValkeyIP   string     `toml:"ValkeyIP"`
	Cron       string     `toml:"Cron"`
	ChannelID  string     `toml:"Channel"`
	Database   string     `toml:"Database"`
	Guild      string     `toml:"Guild"`
	Severities Severities `toml:"Severities"`
	Logging    Logging    `toml:"Logging"`
}

type Severities struct {
	Critical string `toml:"CRITICAL"`
	High     string `toml:"HIGH"`
	Medium   string `toml:"MEDUIM"`
	Low      string `toml:"LOW"`
	None     string `toml:"NONE"`
}

type Logging struct {
	Logfile  string `toml:"Logfile"`
	Loglevel string `toml:"Loglevel"`
}

func DefaultConfig() *Config {
	return &Config{
		Token:     "",
		BaseUri:   "",
		ValkeyIP:  "",
		Cron:      "*/30 * * * *",
		ChannelID: "",
		Database:  "",
		Guild:     "",
		Severities: Severities{
			Critical: "FF2929",
			High:     "F79511",
			Medium:   "FAE95C",
			Low:      "C7FFFB",
			None:     "CCCCCC",
		},
		Logging: Logging{
			Logfile:  "",
			Loglevel: "ERROR",
		},
	}
}

func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		golog.APPWarnf("Attempting to create config file at %s", path)
		return nil, fmt.Errorf("Create cfg file ")
	}

	config := DefaultConfig()

	if err := toml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("Failed to parse TOML: %s", err)
	}

	if config.ValkeyIP == "" {
		return nil, fmt.Errorf("Empty config detected, please edit config at %s", path)
	}

	return config, nil
}

func SaveDefault(path string) error {
	// Marshal to TOML
	data, err := toml.Marshal(DefaultConfig())
	if err != nil {
		return fmt.Errorf("failed to marshal TOML: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with restricted permissions
	if err := os.WriteFile(path, data, 0640); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
