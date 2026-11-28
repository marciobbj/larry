package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PluginsConfig struct {
	Enabled  *bool                     `json:"enabled,omitempty"`
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
	ConfigPath  string        `json:"-"`
}

// PluginsDir returns the fixed, cross-platform directory where Larry discovers Lua plugins:
//   - Linux:   $XDG_CONFIG_HOME/larry/plugins (default: ~/.config/larry/plugins)
//   - macOS:   ~/Library/Application Support/larry/plugins (falls back to ~/.config/larry/plugins if present)
//   - Windows: %AppData%\larry\plugins
func PluginsDir() string {
	var primary string
	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		primary = filepath.Join(configDir, "larry", "plugins")
		if info, statErr := os.Stat(primary); statErr == nil && info.IsDir() {
			return primary
		}
	}
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		fallback := filepath.Join(homeDir, ".config", "larry", "plugins")
		if fallback != primary {
			if info, statErr := os.Stat(fallback); statErr == nil && info.IsDir() {
				return fallback
			}
		}
		if primary == "" {
			return fallback
		}
	}
	return primary
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
		cfg.ConfigPath = DefaultConfigPath()
		if hasOmarchyTheme() {
			cfg.Theme = "omarchy"
		}
		return cfg, nil
	}

	cfg.ConfigPath = path

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return DefaultConfig(), err
	}
	cfg.ConfigPath = path

	if cfg.Theme == "" && hasOmarchyTheme() {
		cfg.Theme = "omarchy"
	}

	return cfg, nil
}

// DefaultConfigPath returns the preferred path for persisting user configuration.
func DefaultConfigPath() string {
	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		p := filepath.Join(configDir, "larry", "config.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		fallback := filepath.Join(homeDir, ".config", "larry", "config.json")
		if _, err := os.Stat(fallback); err == nil {
			return fallback
		}
		return fallback
	}
	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		return filepath.Join(configDir, "larry", "config.json")
	}
	return ""
}

// SaveTheme persists the selected theme name into configPath while preserving
// all other existing keys in config.json.
func SaveTheme(configPath, themeName string) error {
	if configPath == "" {
		return nil
	}
	raw := make(map[string]any)
	if data, err := os.ReadFile(configPath); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &raw)
	}
	raw["theme"] = themeName

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(configPath, out, 0o644)
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
