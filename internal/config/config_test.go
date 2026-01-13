package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_DefaultLocation(t *testing.T) {
	tmpConfigDir, err := os.MkdirTemp("", "larry-test-config-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpConfigDir)

	os.Setenv("XDG_CONFIG_HOME", tmpConfigDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	larryConfigDir := filepath.Join(tmpConfigDir, "larry")
	err = os.MkdirAll(larryConfigDir, 0755)
	if err != nil {
		t.Fatalf("failed to create larry config dir: %v", err)
	}

	configFilePath := filepath.Join(larryConfigDir, "config.json")
	configContent := `{"theme": "monokai", "tab_width": 2, "line_numbers": false}`
	err = os.WriteFile(configFilePath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Theme != "monokai" {
		t.Errorf("expected theme monokai, got %s", cfg.Theme)
	}
	if cfg.TabWidth != 2 {
		t.Errorf("expected tab_width 2, got %d", cfg.TabWidth)
	}
	if cfg.LineNumbers != false {
		t.Errorf("expected line_numbers false, got %v", cfg.LineNumbers)
	}
}

func TestLoadConfig_ExplicitPath(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `{"theme": "nord", "tab_width": 8, "line_numbers": true}`
	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Theme != "nord" {
		t.Errorf("expected theme nord, got %s", cfg.Theme)
	}
}
