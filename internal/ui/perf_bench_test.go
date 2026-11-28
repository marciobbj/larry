package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"larry/internal/config"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

func loadBenchLines(b *testing.B, relPath string) []string {
	b.Helper()
	raw, err := os.ReadFile(relPath)
	if err != nil {
		b.Fatal(err)
	}
	return strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
}

func BenchmarkCursorNavigationFrame(b *testing.B) {
	prevProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prevProfile)

	lines := loadBenchLines(b, "themes.go")
	absPath, _ := filepath.Abs("themes.go")

	cases := []struct {
		name      string
		theme     string
		noPlugins bool
		fileTree  bool
	}{
		{name: "dracula_no_plugins", theme: "dracula", noPlugins: true},
		{name: "catppuccin_with_plugins", theme: "catppuccin-mocha", noPlugins: false},
		{name: "catppuccin_with_plugins_and_tree", theme: "catppuccin-mocha", noPlugins: false, fileTree: true},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			cfg := config.DefaultConfig()
			cfg.Theme = tc.theme
			if tc.noPlugins {
				disabled := false
				cfg.Plugins.Enabled = &disabled
			}

			m := InitialModel(absPath, lines, cfg)
			defer m.ClosePlugins()
			m.Width = 180
			m.Height = 50
			if tc.fileTree {
				m.OpenDirectory("../..")
				m.fileTree.Focused = false
			}
			m.Init()

			// Warm up caches once (syntax, git base, plugin init)
			_ = m.View()

			down := tea.KeyMsg{Type: tea.KeyDown}
			up := tea.KeyMsg{Type: tea.KeyUp}

			var totalBytes int64
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := down
				if m.CursorRow >= len(m.Lines)-2 {
					key = up
				} else if (i/30)%2 == 1 {
					key = up
				}
				updated, _ := m.Update(key)
				m = updated.(Model)
				out := m.View()
				totalBytes += int64(len(out))
			}
			b.StopTimer()
			if b.N > 0 {
				b.ReportMetric(float64(totalBytes)/float64(b.N), "view-bytes/op")
			}
		})
	}
}
