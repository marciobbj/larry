package config

import (
	"encoding/json"
	"os"
)

// Config holds the editor configuration
type Config struct {
	Theme       string `json:"theme"`        // Syntax highlight theme (e.g., "dracula", "monokai")
	TabWidth    int    `json:"tab_width"`    // Number of spaces for tab
	LineNumbers bool   `json:"line_numbers"` // Show line numbers
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		Theme:       "dracula",
		TabWidth:    4,
		LineNumbers: true,
	}
}

// LoadConfig loads the configuration from a JSON file.
// If the file doesn't exist or is invalid, it returns the default config (and potentially an error).
// If only some fields are missing, it uses defaults for them (basic approach: defaults -> overwrite)
func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return DefaultConfig(), err
	}
	return cfg, nil
}
