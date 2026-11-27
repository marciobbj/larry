package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func hasVisibleStyle(s lipgloss.Style) bool {
	if _, ok := s.GetForeground().(lipgloss.NoColor); !ok && s.GetForeground() != nil {
		return true
	}
	return s.GetBold() || s.GetItalic() || s.GetUnderline()
}

func TestMarkdownSyntaxHighlightingAcrossThemes(t *testing.T) {
	themes := []string{"github", "dracula", "nord"}
	lines := []string{
		`<div align="center">`,
		`  <img src="assets/logo.png" width="160">`,
		`</div>`,
		`## Overview`,
		`### Core Features`,
		`- **Fast** playback with ` + "`MediaKit`" + ` and [Docs](https://example.com)`,
		"```go",
		`func main() { return }`,
		"```",
	}

	for _, th := range themes {
		t.Run(th, func(t *testing.T) {
			SetTheme(th)
			t.Cleanup(func() { SetTheme("dracula") })

			styled := TokenizeLines(lines, "README.md")
			if len(styled) != len(lines) {
				t.Fatalf("styled line count = %d, want %d", len(styled), len(lines))
			}

			// HTML tag inside Markdown: '<div' should be styled
			if !hasVisibleStyle(styled[0][1]) {
				t.Fatalf("theme %s: expected HTML tag <div to be styled", th)
			}
			// HTML attribute value "center" should be styled
			if !hasVisibleStyle(styled[0][12]) {
				t.Fatalf("theme %s: expected HTML attribute value to be styled", th)
			}
			// Heading '## Overview' should be styled (bold/colored)
			if !hasVisibleStyle(styled[3][0]) || !hasVisibleStyle(styled[3][4]) {
				t.Fatalf("theme %s: expected Markdown heading ## Overview to be styled", th)
			}
			// Subheading '### Core Features' should be styled
			if !hasVisibleStyle(styled[4][0]) {
				t.Fatalf("theme %s: expected Markdown subheading ### to be styled", th)
			}
			// List marker '-' should be styled
			if !hasVisibleStyle(styled[5][0]) {
				t.Fatalf("theme %s: expected Markdown list marker to be styled", th)
			}
			// Fenced Go keyword 'func' should be styled
			if !hasVisibleStyle(styled[7][0]) {
				t.Fatalf("theme %s: expected fenced Go code keyword to be styled", th)
			}
		})
	}
}

func TestXMLSyntaxHighlightingSingleAndMultiline(t *testing.T) {
	SetTheme("nord")
	t.Cleanup(func() { SetTheme("dracula") })

	lines := []string{
		`<?xml version="1.0" encoding="utf-8"?>`,
		`<!--`,
		`  multiline comment`,
		`-->`,
		`<activity`,
		`    android:name=".MainActivity"`,
		`    android:exported="true">`,
		`    <intent-filter/>`,
		`</activity>`,
	}

	for _, name := range []string{"AndroidManifest.xml", "app.iml", "Info.plist", "Main.xaml", "noext"} {
		t.Run(name, func(t *testing.T) {
			styled := TokenizeLines(lines, name)
			// XML declaration
			if !hasVisibleStyle(styled[0][0]) {
				t.Fatalf("%s: expected <?xml declaration to be styled", name)
			}
			// Multiline comment inner line
			if !hasVisibleStyle(styled[2][4]) {
				t.Fatalf("%s: expected multiline XML comment to be styled", name)
			}
			// Multiline tag opening
			if !hasVisibleStyle(styled[4][1]) {
				t.Fatalf("%s: expected <activity tag to be styled", name)
			}
			// Multiline tag attribute & value on continuation line
			if !hasVisibleStyle(styled[5][5]) || !hasVisibleStyle(styled[5][20]) {
				t.Fatalf("%s: expected multiline XML attribute and value to be styled", name)
			}
			// Closing tag
			if !hasVisibleStyle(styled[8][1]) {
				t.Fatalf("%s: expected </activity> closing tag to be styled", name)
			}
		})
	}
}

func TestGetLineStylesSupportsMarkdownHeadingsWithoutTrailingNewline(t *testing.T) {
	SetTheme("github")
	t.Cleanup(func() { SetTheme("dracula") })

	styles := GetLineStyles("## Overview", "README.md")
	if len(styles) != len("## Overview") {
		t.Fatalf("len(styles) = %d, want %d", len(styles), len("## Overview"))
	}
	if !hasVisibleStyle(styles[0]) || !hasVisibleStyle(styles[4]) {
		t.Fatal("expected single-line GetLineStyles to highlight Markdown heading")
	}
}
