package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PluginsConfig struct {
	Enabled  *bool                     `json:"enabled,omitempty"`
	Dir      string                    `json:"dir,omitempty"`
	Disabled []string                  `json:"disabled,omitempty"`
	Settings map[string]map[string]any `json:"settings,omitempty"`
}

func (p PluginsConfig) IsEnabled() bool {
	if p.Enabled == nil {
		return true
	}
	return *p.Enabled
}

func (p PluginsConfig) IsDisabled(name string) bool {
	for _, d := range p.Disabled {
		if d == name {
			return true
		}
	}
	return false
}

type Config struct {
	Theme       string        `json:"theme"`
	TabWidth    int           `json:"tab_width"`
	LineNumbers bool          `json:"line_numbers"`
	WordWrap    bool          `json:"word_wrap"`
	LeaderKey   string        `json:"leader_key"`
	Plugins     PluginsConfig `json:"plugins,omitempty"`
}

// ResolvePluginsDir returns the directory where Larry discovers Lua plugins.
// Priority: LARRY_PLUGINS_DIR env var > cfg.Plugins.Dir > <UserConfigDir>/larry/plugins > ~/.config/larry/plugins.
func ResolvePluginsDir(cfg Config) string {
	if envDir := os.Getenv("LARRY_PLUGINS_DIR"); envDir != "" {
		return envDir
	}
	if cfg.Plugins.Dir != "" {
		return cfg.Plugins.Dir
	}
	if configDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(configDir, "larry", "plugins")
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".config", "larry", "plugins")
	}
	return ""
}

func DefaultConfig() Config {
	return Config{
		Theme:       "dracula",
		TabWidth:    4,
		LineNumbers: true,
		WordWrap:    false,
		LeaderKey:   "ctrl",
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

		if path == "" {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				fallbackPath := filepath.Join(homeDir, ".config", "larry", "config.json")
				if _, err := os.Stat(fallbackPath); err == nil {
					path = fallbackPath
				}
			}
		}
	}

	if path == "" {
		if hasOmarchyTheme() {
			cfg.Theme = "omarchy"
		}
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

	if cfg.Theme == "" && hasOmarchyTheme() {
		cfg.Theme = "omarchy"
	}

	return cfg, nil
}

func hasOmarchyTheme() bool {
	if envPath := os.Getenv("LARRY_OMARCHY_COLORS"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return true
		}
	}
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		if _, err := os.Stat(filepath.Join(stateHome, "omarchy", "current", "theme", "colors.toml")); err == nil {
			return true
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if _, err := os.Stat(filepath.Join(home, ".local", "state", "omarchy", "current", "theme", "colors.toml")); err == nil {
			return true
		}
		if _, err := os.Stat(filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml")); err == nil {
			return true
		}
	}
	return false
}
