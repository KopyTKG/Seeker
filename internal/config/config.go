package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/kopytkg/golog/v2"
)

type Manager struct {
	Current *Config
	Path    string
}

type Config struct {
	Configuration  Configuration `toml:"Configuration"`
	Authentication Auth          `toml:"Authentication"`
	Severities     Severities    `toml:"Severities"`
	Logging        Logging       `toml:"Logging"`
}

type Configuration struct {
	BaseUri  string `toml:"BaseUri"`
	ValkeyIP string `toml:"ValkeyIP"`
	Database string `toml:"Database"`
	Cron     string `toml:"Cron"`
	TimeZone string `toml:"TimeZone"`
}

type Auth struct {
	Token     string   `toml:"Token"`
	ChannelID string   `toml:"Channel"`
	Guild     string   `toml:"Guild"`
	Admins    []string `toml:"Admins"`
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
		Configuration: Configuration{
			BaseUri:  "",
			ValkeyIP: "",
			Cron:     "*/30 * * * *",
			Database: "",
			TimeZone: "Europe/Prague",
		},
		Authentication: Auth{
			Token:     "",
			ChannelID: "",
			Guild:     "",
			Admins:    []string{},
		},
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

func LoadFromFile(path string) (*Manager, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		golog.APPWarnf("Attempting to create config file at %s", path)
		return nil, fmt.Errorf("Create cfg file ")
	}

	config := DefaultConfig()

	if err := toml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("Failed to parse TOML: %s", err)
	}

	if config.Configuration.ValkeyIP == "" {
		return nil, fmt.Errorf("Empty config detected, please edit config at %s", path)
	}

	return &Manager{Current: config, Path: path}, nil
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

func (m *Manager) Save() error {
	// Marshal to TOML
	data, err := toml.Marshal(m.Current)
	if err != nil {
		return fmt.Errorf("failed to marshal TOML: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(m.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file with restricted permissions
	if err := os.WriteFile(m.Path, data, 0640); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
