package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
)

type Manager struct {
	Current *Config
	Path    string
	mu      sync.Mutex
}

type Config struct {
	Webhook     string      `toml:"Webhook"`
	RSSFeed     string      `toml:"RSSFeed"`
	ATOMFeed    string      `toml:"ATOMFeed"`
	ColorConfig ColorConfig `toml:"Colors"`
}
type ColorConfig struct {
	CriticalColor string `toml:"CriticalColor"`
	HighColor     string `toml:"HighColor"`
}

func DefaultConfig() *Config {
	return &Config{
		Webhook:  "",
		RSSFeed:  "",
		ATOMFeed: "",
		ColorConfig: ColorConfig{
			CriticalColor: "FA1414",
			HighColor:     "FAEF14",
		},
	}
}

func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config file: %s\n", err)
	}

	config := DefaultConfig()

	if err := toml.Unmarshal(data, config); err != nil {
		config.Save(path)
		return nil, fmt.Errorf("Failed to parse TOML: %s", err)
	}
	return config, nil
}

func (c *Config) Save(path string) error {
	// Marshal to TOML
	data, err := toml.Marshal(c)
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
