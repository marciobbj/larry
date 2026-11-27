package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/lipgloss"
)

// OmarchyColors holds the parsed color tokens from an Omarchy theme colors.toml file.
// It supports both the modern semantic schema (background, accent, red..magenta, bright_*)
// and the legacy terminal schema (color0..color15, selection_background, cursor).
type OmarchyColors struct {
	Mode string // "dark" or "light"

	Accent    string
	Selection string
	Muted     string
	Cursor    string

	Background        string
	DarkBackground    string
	DarkerBackground  string
	LighterBackground string

	Foreground       string
	DarkForeground   string
	LightForeground  string
	BrightForeground string

	ActiveBorder string

	Red     string
	Yellow  string
	Orange  string
	Green   string
	Cyan    string
	Blue    string
	Magenta string

	BrightRed     string
	BrightYellow  string
	BrightGreen   string
	BrightCyan    string
	BrightBlue    string
	BrightMagenta string
}

// ResolveOmarchyColorsPath locates the active Omarchy colors.toml file.
// Checks LARRY_OMARCHY_COLORS first, then XDG/state (~/.local/state/omarchy/current/theme/colors.toml)
// and legacy XDG/config (~/.config/omarchy/current/theme/colors.toml).
func ResolveOmarchyColorsPath() string {
	if envPath := strings.TrimSpace(os.Getenv("LARRY_OMARCHY_COLORS")); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	var candidates []string
	if stateHome := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); stateHome != "" {
		candidates = append(candidates, filepath.Join(stateHome, "omarchy", "current", "theme", "colors.toml"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".local", "state", "omarchy", "current", "theme", "colors.toml"))
	}
	if cfgHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); cfgHome != "" {
		candidates = append(candidates, filepath.Join(cfgHome, "omarchy", "current", "theme", "colors.toml"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, ".config", "omarchy", "current", "theme", "colors.toml"))
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// LoadCurrentOmarchyColors loads the currently active Omarchy theme colors from disk.
func LoadCurrentOmarchyColors() (OmarchyColors, string, error) {
	path := ResolveOmarchyColorsPath()
	if path == "" {
		return OmarchyColors{}, "", os.ErrNotExist
	}
	colors, err := LoadOmarchyColors(path)
	return colors, path, err
}

// LoadOmarchyColors reads and parses an Omarchy colors.toml file.
func LoadOmarchyColors(path string) (OmarchyColors, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return OmarchyColors{}, err
	}
	return ParseOmarchyColors(string(raw)), nil
}

// ParseOmarchyColors parses TOML key = "value" pairs from Omarchy's colors.toml.
func ParseOmarchyColors(data string) OmarchyColors {
	kv := make(map[string]string)

	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		val = stripInlineComment(val)
		val = strings.Trim(val, `"'`)
		if key != "" && val != "" {
			kv[key] = val
		}
	}

	pick := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := kv[k]; ok && isHexColor(v) {
				return normalizeHex(v)
			}
		}
		return ""
	}

	c := OmarchyColors{
		Mode: strings.ToLower(kv["mode"]),

		Background:        pick("background", "color0"),
		DarkBackground:    pick("dark_background", "darker_background", "background", "color0"),
		DarkerBackground:  pick("darker_background", "dark_background", "background", "color0"),
		LighterBackground: pick("lighter_background", "selection", "selection_background", "background", "color0"),

		Foreground:       pick("foreground", "color7"),
		DarkForeground:   pick("dark_foreground", "muted", "color8"),
		LightForeground:  pick("light_foreground", "foreground", "color7"),
		BrightForeground: pick("bright_foreground", "foreground", "color15", "color7"),

		Accent:       pick("accent", "active_border_color", "blue", "color4"),
		Selection:    pick("selection", "selection_background", "lighter_background", "color8"),
		Muted:        pick("muted", "dark_foreground", "color8"),
		Cursor:       pick("cursor", "accent", "foreground", "color15"),
		ActiveBorder: pick("active_border_color", "accent", "blue", "color4"),

		Red:     pick("red", "color1"),
		Green:   pick("green", "color2"),
		Yellow:  pick("yellow", "color3"),
		Orange:  pick("orange", "bright_yellow", "yellow", "color3"),
		Blue:    pick("blue", "color4", "accent"),
		Magenta: pick("magenta", "color5", "accent"),
		Cyan:    pick("cyan", "color6", "blue"),

		BrightRed:     pick("bright_red", "color9", "red", "color1"),
		BrightGreen:   pick("bright_green", "color10", "green", "color2"),
		BrightYellow:  pick("bright_yellow", "color11", "yellow", "color3"),
		BrightBlue:    pick("bright_blue", "color12", "blue", "color4"),
		BrightMagenta: pick("bright_magenta", "color13", "magenta", "color5"),
		BrightCyan:    pick("bright_cyan", "color14", "cyan", "color6"),
	}

	// Infer dark/light mode from background luminance when mode is not explicitly set.
	if c.Mode != "light" && c.Mode != "dark" {
		if c.Background != "" && colorLuminance(c.Background) > 0.5 {
			c.Mode = "light"
		} else {
			c.Mode = "dark"
		}
	}

	// Ensure sensible fallbacks for any missing keys.
	if c.Mode == "light" {
		fillDefault(&c.Background, "#eff1f5")
		fillDefault(&c.DarkBackground, "#e3e4e8")
		fillDefault(&c.DarkerBackground, "#d7d8dc")
		fillDefault(&c.LighterBackground, "#dce0e8")
		fillDefault(&c.Foreground, "#4c4f69")
		fillDefault(&c.DarkForeground, "#8c8fa1")
		fillDefault(&c.LightForeground, "#5c5f77")
		fillDefault(&c.BrightForeground, "#2c2f45")
		fillDefault(&c.Accent, "#1e66f5")
		fillDefault(&c.Selection, "#ccd0da")
		fillDefault(&c.Muted, "#9ca0b0")
		fillDefault(&c.Cursor, c.Foreground)
		fillDefault(&c.ActiveBorder, c.Accent)
		fillDefault(&c.Red, "#d20f39")
		fillDefault(&c.Green, "#40a02b")
		fillDefault(&c.Yellow, "#df8e1d")
		fillDefault(&c.Orange, "#fe640b")
		fillDefault(&c.Blue, "#1e66f5")
		fillDefault(&c.Magenta, "#ea76cb")
		fillDefault(&c.Cyan, "#179299")
	} else {
		fillDefault(&c.Background, "#1a1b26")
		fillDefault(&c.DarkBackground, "#13141c")
		fillDefault(&c.DarkerBackground, "#0e0e14")
		fillDefault(&c.LighterBackground, "#24283b")
		fillDefault(&c.Foreground, "#c0caf5")
		fillDefault(&c.DarkForeground, "#565f89")
		fillDefault(&c.LightForeground, "#a9b1d6")
		fillDefault(&c.BrightForeground, "#e0e6ff")
		fillDefault(&c.Accent, "#7aa2f7")
		fillDefault(&c.Selection, "#292e42")
		fillDefault(&c.Muted, "#565f89")
		fillDefault(&c.Cursor, c.Foreground)
		fillDefault(&c.ActiveBorder, c.Accent)
		fillDefault(&c.Red, "#f7768e")
		fillDefault(&c.Green, "#9ece6a")
		fillDefault(&c.Yellow, "#e0af68")
		fillDefault(&c.Orange, "#ff9e64")
		fillDefault(&c.Blue, "#7aa2f7")
		fillDefault(&c.Magenta, "#bb9af7")
		fillDefault(&c.Cyan, "#7dcfff")
	}

	fillDefault(&c.BrightRed, c.Red)
	fillDefault(&c.BrightGreen, c.Green)
	fillDefault(&c.BrightYellow, c.Yellow)
	fillDefault(&c.BrightBlue, c.Blue)
	fillDefault(&c.BrightMagenta, c.Magenta)
	fillDefault(&c.BrightCyan, c.Cyan)

	return c
}

// OmarchyThemeSignature returns a cheap fingerprint of the active Omarchy theme
// (using theme.generation when present, plus colors.toml mtime/size) for live reload.
func OmarchyThemeSignature(colorsPath string) string {
	if colorsPath == "" {
		colorsPath = ResolveOmarchyColorsPath()
	}
	if colorsPath == "" {
		return ""
	}

	// Omarchy writes ~/.local/state/omarchy/current/theme.generation on every theme switch.
	genPath := filepath.Join(filepath.Dir(filepath.Dir(colorsPath)), "theme.generation")
	gen := ""
	if raw, err := os.ReadFile(genPath); err == nil {
		gen = strings.TrimSpace(string(raw))
	}

	statPart := ""
	if info, err := os.Stat(colorsPath); err == nil {
		statPart = fmt.Sprintf("%d:%d", info.ModTime().UnixNano(), info.Size())
	}
	return colorsPath + "|" + gen + "|" + statPart
}

// BuildOmarchyChromaStyle synthesizes a Chroma syntax highlighting theme from OmarchyColors.
func BuildOmarchyChromaStyle(c OmarchyColors) (*chroma.Style, error) {
	entries := chroma.StyleEntries{
		chroma.Background:          c.Foreground + " bg:" + c.Background,
		chroma.Text:                c.Foreground,
		chroma.Error:               c.Red,
		chroma.Comment:             "italic " + c.DarkForeground,
		chroma.CommentPreproc:      c.Cyan,
		chroma.Keyword:             "bold " + c.Magenta,
		chroma.KeywordConstant:     c.Orange,
		chroma.KeywordDeclaration:  "bold " + c.Magenta,
		chroma.KeywordNamespace:    c.Cyan,
		chroma.KeywordType:         c.Cyan,
		chroma.Operator:            c.Cyan,
		chroma.OperatorWord:        "bold " + c.Magenta,
		chroma.Punctuation:         c.LightForeground,
		chroma.Name:                c.Foreground,
		chroma.NameAttribute:       c.Accent,
		chroma.NameBuiltin:         c.Cyan,
		chroma.NameBuiltinPseudo:   c.Orange,
		chroma.NameClass:           "bold " + c.Yellow,
		chroma.NameConstant:        c.Orange,
		chroma.NameDecorator:         c.Accent,
		chroma.NameEntity:            c.Yellow,
		chroma.NameException:         c.Red,
		chroma.NameFunction:          c.Blue,
		chroma.NameFunctionMagic:     c.BrightBlue,
		chroma.NameNamespace:         c.Cyan,
		chroma.NameTag:               c.Red,
		chroma.NameVariable:          c.BrightForeground,
		chroma.LiteralNumber:         c.Orange,
		chroma.LiteralString:         c.Green,
		chroma.LiteralStringBacktick: c.Cyan,
		chroma.LiteralStringDoc:      "italic " + c.Green,
		chroma.LiteralStringEscape:   c.Cyan,
		chroma.LiteralStringInterpol: c.Cyan,
		chroma.LiteralStringRegex:    c.Yellow,
		chroma.GenericDeleted:        c.Red,
		chroma.GenericEmph:           "italic " + c.LightForeground,
		chroma.GenericError:          c.Red,
		chroma.GenericHeading:        "bold " + c.Accent,
		chroma.GenericInserted:       c.Green,
		chroma.GenericPrompt:         "bold " + c.Accent,
		chroma.GenericStrong:         "bold " + c.BrightForeground,
		chroma.GenericSubheading:     "bold " + c.Blue,
	}
	return chroma.NewStyle("omarchy", entries)
}

func paletteFromOmarchy(c OmarchyColors) UIPalette {
	isDark := c.Mode != "light"

	// Pick a distinct surface shade for status bar & modals.
	surface := c.LighterBackground
	if surface == c.Background {
		surface = c.DarkBackground
	}
	modalBg := c.DarkBackground
	if modalBg == c.Background {
		modalBg = c.LighterBackground
	}

	accentFg := contrastingTextColor(c.Accent, c.DarkerBackground, c.BrightForeground)
	cursorFg := contrastingTextColor(c.Cursor, c.DarkerBackground, c.BrightForeground)
	selectionFg := contrastingTextColor(c.Selection, c.DarkerBackground, c.BrightForeground)

	return UIPalette{
		IsDark:      isDark,
		Accent:      lipgloss.Color(c.Accent),
		AccentAlt:   lipgloss.Color(c.Magenta),
		AccentFg:    accentFg,
		Background:  lipgloss.Color(modalBg),
		Surface:     lipgloss.Color(surface),
		Selection:   lipgloss.Color(c.Selection),
		SelectionFg: selectionFg,
		Foreground:  lipgloss.Color(c.Foreground),
		SubtleFg:    lipgloss.Color(c.LightForeground),
		MutedFg:     lipgloss.Color(c.DarkForeground),
		Border:      lipgloss.Color(c.Muted),
		Red:         lipgloss.Color(c.Red),
		Green:       lipgloss.Color(c.Green),
		Yellow:      lipgloss.Color(c.Yellow),
		Blue:        lipgloss.Color(c.Blue),
		Cyan:        lipgloss.Color(c.Cyan),
		Magenta:     lipgloss.Color(c.Magenta),
		CursorBg:    lipgloss.Color(c.Cursor),
		CursorFg:    cursorFg,
	}
}

func fillDefault(target *string, fallback string) {
	if *target == "" {
		*target = fallback
	}
}

func stripInlineComment(val string) string {
	inQuotes := byte(0)
	for i := 0; i < len(val); i++ {
		ch := val[i]
		if inQuotes != 0 {
			if ch == inQuotes {
				inQuotes = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			inQuotes = ch
			continue
		}
		if ch == '#' {
			return strings.TrimSpace(val[:i])
		}
	}
	return val
}

func isHexColor(s string) bool {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 && len(s) != 3 {
		return false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		isHex := (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
		if !isHex {
			return false
		}
	}
	return true
}

func normalizeHex(s string) string {
	raw := strings.TrimPrefix(s, "#")
	if len(raw) == 3 {
		raw = string([]byte{raw[0], raw[0], raw[1], raw[1], raw[2], raw[2]})
	}
	return "#" + strings.ToLower(raw)
}

func colorLuminance(hex string) float64 {
	raw := strings.TrimPrefix(normalizeHex(hex), "#")
	if len(raw) != 6 {
		return 0
	}
	r, _ := strconv.ParseUint(raw[0:2], 16, 8)
	g, _ := strconv.ParseUint(raw[2:4], 16, 8)
	b, _ := strconv.ParseUint(raw[4:6], 16, 8)
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255.0
}

func contrastingTextColor(bgHex, darkCandidate, lightCandidate string) lipgloss.Color {
	if colorLuminance(bgHex) > 0.52 {
		if darkCandidate != "" && colorLuminance(darkCandidate) < 0.3 {
			return lipgloss.Color(darkCandidate)
		}
		return lipgloss.Color("#0c0b0c")
	}
	if lightCandidate != "" && colorLuminance(lightCandidate) > 0.7 {
		return lipgloss.Color(lightCandidate)
	}
	return lipgloss.Color("#ffffff")
}
