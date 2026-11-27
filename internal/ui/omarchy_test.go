package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/lipgloss"

	"larry/internal/config"
)

func TestParseOmarchyColorsModernAndLegacy(t *testing.T) {
	modern := `
mode = "dark"
accent = "#7aa2f7"
selection = "#292e42"
muted = "#414868"
background = "#1a1b26"
dark_background = "#13141c"
lighter_background = "#24283b"
foreground = "#a9b1d6"
dark_foreground = "#565f89"
red = "#f7768e"
green = "#9ece6a"
yellow = "#e0af68"
blue = "#7aa2f7"
magenta = "#ad8ee6"
cyan = "#449dab"
`
	c := ParseOmarchyColors(modern)
	if c.Mode != "dark" {
		t.Fatalf("mode = %q, want dark", c.Mode)
	}
	if c.Accent != "#7aa2f7" || c.Background != "#1a1b26" || c.Green != "#9ece6a" {
		t.Fatalf("unexpected parsed modern colors: %+v", c)
	}

	legacy := `
accent = "#1e66f5"
cursor = "#4c4f69"
foreground = "#4c4f69"
background = "#eff1f5"
selection_background = "#ccd0da"
color0 = "#bcc0cc"
color1 = "#d20f39"
color2 = "#40a02b"
color3 = "#df8e1d"
color4 = "#1e66f5"
color5 = "#ea76cb"
color6 = "#179299"
color8 = "#acb0be"
`
	lc := ParseOmarchyColors(legacy)
	if lc.Mode != "light" {
		t.Fatalf("expected inferred light mode from #eff1f5, got %q", lc.Mode)
	}
	if lc.Accent != "#1e66f5" || lc.Red != "#d20f39" || lc.Selection != "#ccd0da" {
		t.Fatalf("unexpected parsed legacy colors: %+v", lc)
	}
}

func TestSetThemeOmarchyLoadsPaletteSyntaxAndLiveReload(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "theme")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	colorsPath := filepath.Join(themeDir, "colors.toml")
	genPath := filepath.Join(dir, "theme.generation")

	initialToml := `
mode = "dark"
accent = "#b59790"
selection = "#584e51"
muted = "#584e51"
background = "#0c0b0c"
dark_background = "#090809"
foreground = "#FAFCFB"
red = "#c38b7b"
green = "#87a9b0"
blue = "#b59790"
magenta = "#c4d8e2"
cyan = "#a5a0b6"
`
	if err := os.WriteFile(colorsPath, []byte(initialToml), 0644); err != nil {
		t.Fatalf("write colors.toml: %v", err)
	}
	if err := os.WriteFile(genPath, []byte("gen-1\n"), 0644); err != nil {
		t.Fatalf("write gen: %v", err)
	}

	t.Setenv("LARRY_OMARCHY_COLORS", colorsPath)
	t.Cleanup(func() { SetTheme("dracula") })

	cfg := config.DefaultConfig()
	cfg.Theme = "omarchy"
	m := InitialModel("main.go", []string{"package main"}, cfg)

	if !omarchyActive {
		t.Fatal("expected omarchyActive to be true")
	}
	if activePalette.Accent != lipgloss.Color("#b59790") {
		t.Fatalf("activePalette.Accent = %q, want #b59790", activePalette.Accent)
	}
	if !activePalette.IsDark {
		t.Fatal("expected dark palette")
	}

	// Verify Chroma syntax style uses Omarchy colors
	entry := theme.Get(chroma.LiteralString)
	if entry.Colour.String() != "#87a9b0" {
		t.Fatalf("syntax LiteralString color = %q, want #87a9b0", entry.Colour.String())
	}

	// Simulate an Omarchy theme switch to a light theme and fire a tick
	switchedToml := `
mode = "light"
accent = "#1e66f5"
selection = "#ccd0da"
muted = "#acb0be"
background = "#eff1f5"
dark_background = "#e3e4e8"
foreground = "#4c4f69"
green = "#40a02b"
`
	if err := os.WriteFile(colorsPath, []byte(switchedToml), 0644); err != nil {
		t.Fatalf("write switched colors.toml: %v", err)
	}
	if err := os.WriteFile(genPath, []byte("gen-2\n"), 0644); err != nil {
		t.Fatalf("write gen-2: %v", err)
	}

	updated, cmd := m.Update(omarchyThemeTickMsg{})
	_ = updated.(Model)
	if cmd == nil {
		t.Fatal("expected live-reload tick to re-arm itself")
	}
	if activePalette.Accent != lipgloss.Color("#1e66f5") {
		t.Fatalf("expected live-reloaded Accent #1e66f5, got %q", activePalette.Accent)
	}
	if activePalette.IsDark {
		t.Fatal("expected live-reloaded palette to switch to light mode")
	}
}
