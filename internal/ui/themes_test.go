package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"larry/internal/config"
)

func TestAllBuiltinColorSchemesAreValidAndApplicable(t *testing.T) {
	schemes := BuiltinColorSchemes()
	if len(schemes) < 40 {
		t.Fatalf("expected at least 40 curated colorschemes, got %d", len(schemes))
	}

	seen := make(map[string]bool, len(schemes))
	validCats := map[string]bool{"Modern": true, "Classic": true, "Editors": true, "Vibrant": true}
	for _, s := range schemes {
		if s.ID == "" || s.Name == "" || s.Family == "" {
			t.Fatalf("invalid metadata on colorscheme %+v", s)
		}
		if seen[s.ID] {
			t.Fatalf("duplicate colorscheme ID %q", s.ID)
		}
		seen[s.ID] = true
		if !validCats[s.Category] {
			t.Fatalf("unexpected category %q for %q", s.Category, s.ID)
		}
		if s.Mode != "dark" && s.Mode != "light" {
			t.Fatalf("invalid mode %q for %q", s.Mode, s.ID)
		}
		if !isHexColor(s.Colors.Background) || !isHexColor(s.Colors.Foreground) || !isHexColor(s.Colors.Accent) {
			t.Fatalf("invalid core hex colors for %q", s.ID)
		}

		SetTheme(s.ID)
		if theme == nil {
			t.Fatalf("SetTheme(%q) left nil chroma theme", s.ID)
		}
		wantDark := s.Mode == "dark"
		if activePalette.IsDark != wantDark {
			t.Fatalf("SetTheme(%q): activePalette.IsDark=%v, want %v", s.ID, activePalette.IsDark, wantDark)
		}
	}

	// Verify common aliases resolve cleanly
	for alias, wantID := range map[string]string{
		"catppuccin":  "catppuccin-mocha",
		"tokyonight":  "tokyo-night",
		"rosepine":    "rose-pine",
		"kanagawa":    "kanagawa-wave",
		"everforest":  "everforest-dark",
		"gruvbox":     "gruvbox-dark",
		"solarized":   "solarized-dark",
		"onedark":     "one-dark",
		"github":      "github-light",
		"material":    "material-ocean",
		"ayu":         "ayu-dark",
		"nightowl":    "night-owl",
		"synthwave84": "synthwave-84",
		"modus":       "modus-vivendi",
	} {
		got, ok := LookupColorScheme(alias)
		if !ok || got.ID != wantID {
			t.Fatalf("LookupColorScheme(%q) = (%q, %v), want %q", alias, got.ID, ok, wantID)
		}
	}
}

func TestThemePickerLivePreviewRevertFilterAndPersist(t *testing.T) {
	isolatePluginsDir(t)

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"theme":"dracula","tab_width":2,"leader_key":"ctrl"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	m := InitialModel("main.go", []string{"package main", "func main() {}"}, cfg)
	defer m.ClosePlugins()
	m.Width = 110
	m.Height = 32

	// Open built-in Colorscheme Switcher via Alt+T
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}, Alt: true})
	m = updated.(Model)
	if !m.themePickerActive {
		t.Fatal("expected Alt+T to open built-in Colorscheme Picker")
	}

	// Verify responsive rendering across wide, medium, and narrow terminals
	for _, dims := range [][2]int{{110, 32}, {80, 24}, {52, 20}} {
		m.Width, m.Height = dims[0], dims[1]
		view := m.View()
		rows := strings.Split(view, "\n")
		if len(rows) > m.Height {
			t.Fatalf("theme picker height %d > %d at %dx%d", len(rows), m.Height, dims[0], dims[1])
		}
		for i, r := range rows {
			if w := lipgloss.Width(r); w > m.Width {
				t.Fatalf("theme picker row %d width %d > %d at %dx%d", i, w, m.Width, dims[0], dims[1])
			}
		}
	}
	m.Width, m.Height = 110, 32

	// Filter to "latte" (Catppuccin Latte, a light theme) -> live preview switches palette to light!
	for _, r := range "latte" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	if hovered, ok := m.themePicker.Hovered(); !ok || hovered.ID != "catppuccin-latte" {
		t.Fatalf("expected filter 'latte' to select catppuccin-latte, got %+v (%v)", hovered, ok)
	}
	if activePalette.IsDark {
		t.Fatal("expected live preview of catppuccin-latte to switch activePalette to light")
	}

	// Press Esc -> closes picker and reverts back to original dark Dracula theme!
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.themePickerActive {
		t.Fatal("expected Esc to close theme picker")
	}
	if m.Config.Theme != "dracula" || !activePalette.IsDark {
		t.Fatalf("expected Esc to revert to dracula (dark), got theme=%q isDark=%v", m.Config.Theme, activePalette.IsDark)
	}

	// Re-open, filter with Tab to Light, search "modus", press Enter -> applies & saves to config.json!
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}, Alt: true})
	m = updated.(Model)
	// Cycle Tab until "Light"
	for ColorSchemeCategories[m.themePicker.CategoryIdx] != "Light" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = updated.(Model)
	}
	for _, r := range "modus" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.themePickerActive {
		t.Fatal("expected Enter to close theme picker")
	}
	if m.Config.Theme != "modus-operandi" {
		t.Fatalf("expected modus-operandi applied, got %q", m.Config.Theme)
	}
	if activePalette.IsDark {
		t.Fatal("expected activePalette.IsDark=false after applying modus-operandi")
	}

	// Verify config.json persisted "theme" while preserving "tab_width" and "leader_key"
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved map[string]any
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved["theme"] != "modus-operandi" {
		t.Fatalf("expected persisted theme modus-operandi, got %#v", saved["theme"])
	}
	if saved["tab_width"] != float64(2) || saved["leader_key"] != "ctrl" {
		t.Fatalf("expected existing config keys preserved, got %#v", saved)
	}
}

func TestBuiltinColorschemesPluginAppearsInLoadedPluginsAndRespectsDisabled(t *testing.T) {
	isolatePluginsDir(t)

	m := InitialModel("main.go", []string{"package main"}, config.DefaultConfig())
	defer m.ClosePlugins()
	m = m.showLoadedPluginsPanel()
	joined := strings.Join(m.pluginPanel.Lines, "\n")
	if !strings.Contains(joined, "colorschemes [built-in]") {
		t.Fatalf("expected built-in colorschemes plugin in Loaded Plugins panel, got:\n%s", joined)
	}

	// When disabled in config, Alt+T should not open picker
	cfg := config.DefaultConfig()
	cfg.Plugins.Disabled = []string{"colorschemes"}
	m2 := InitialModel("main.go", []string{"package main"}, cfg)
	defer m2.ClosePlugins()
	updated, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}, Alt: true})
	m2 = updated.(Model)
	if m2.themePickerActive {
		t.Fatal("expected disabled colorschemes plugin not to open picker")
	}
}
