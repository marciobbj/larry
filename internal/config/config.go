package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Theme       string `json:"theme"`
	TabWidth    int    `json:"tab_width"`
	LineNumbers bool   `json:"line_numbers"`
}

func DefaultConfig() Config {
	return Config{
		Theme:       "dracula",
		TabWidth:    4,
		LineNumbers: true,
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		configDir, err := os.UserConfigDir()
		if err == nil {
			defaultPath := filepath.Join(configDir, "larry", "config.json")
			if _, err := os.Stat(defaultPath); err == nil {
				path = defaultPath
			}
		}
	}

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
