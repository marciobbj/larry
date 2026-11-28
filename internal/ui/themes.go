package ui

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
)

// ColorScheme describes a built-in curated colorscheme with full syntax and UI colors.
type ColorScheme struct {
	ID          string
	Name        string
	Family      string
	Category    string // "Modern", "Classic", "Editors", "Vibrant"
	Description string
	Mode        string // "dark" or "light"
	Colors      OmarchyColors
}

// ColorSchemeCategories lists the filter categories in display order.
var ColorSchemeCategories = []string{"All", "Modern", "Classic", "Editors", "Vibrant", "Dark", "Light"}

type schemeSpec struct {
	id, name, family, category, desc, mode                  string
	bg, darkBg, darkerBg, lightBg                           string
	fg, subtleFg, mutedFg                                   string
	accent, selection                                       string
	red, green, yellow, orange, blue, magenta, cyan         string
}

func buildScheme(s schemeSpec) ColorScheme {
	brightFg := "#ffffff"
	if s.mode == "light" {
		brightFg = "#000000"
	}
	return ColorScheme{
		ID:          s.id,
		Name:        s.name,
		Family:      s.family,
		Category:    s.category,
		Description: s.desc,
		Mode:        s.mode,
		Colors: OmarchyColors{
			Mode:              s.mode,
			Background:        s.bg,
			DarkBackground:    s.darkBg,
			DarkerBackground:  s.darkerBg,
			LighterBackground: s.lightBg,
			Foreground:        s.fg,
			LightForeground:   s.subtleFg,
			DarkForeground:    s.mutedFg,
			BrightForeground:  brightFg,
			Accent:            s.accent,
			Selection:         s.selection,
			Muted:             s.mutedFg,
			Cursor:            s.accent,
			ActiveBorder:      s.accent,
			Red:               s.red,
			Green:             s.green,
			Yellow:            s.yellow,
			Orange:            s.orange,
			Blue:              s.blue,
			Magenta:           s.magenta,
			Cyan:              s.cyan,
			BrightRed:         s.red,
			BrightGreen:       s.green,
			BrightYellow:      s.yellow,
			BrightBlue:        s.blue,
			BrightMagenta:     s.magenta,
			BrightCyan:        s.cyan,
		},
	}
}

var builtinColorSchemes = []ColorScheme{
	// ── Modernos & Populares ──
	buildScheme(schemeSpec{
		id: "catppuccin-mocha", name: "Catppuccin Mocha", family: "Catppuccin", category: "Modern",
		desc: "Soothing pastel dark palette", mode: "dark",
		bg: "#1e1e2e", darkBg: "#181825", darkerBg: "#11111b", lightBg: "#313244",
		fg: "#cdd6f4", subtleFg: "#a6adc8", mutedFg: "#6c7086",
		accent: "#cba6f7", selection: "#45475a",
		red: "#f38ba8", green: "#a6e3a1", yellow: "#f9e2af", orange: "#fab387",
		blue: "#89b4fa", magenta: "#cba6f7", cyan: "#94e2d5",
	}),
	buildScheme(schemeSpec{
		id: "catppuccin-macchiato", name: "Catppuccin Macchiato", family: "Catppuccin", category: "Modern",
		desc: "Warm muted pastel slate dark", mode: "dark",
		bg: "#24273a", darkBg: "#1e2030", darkerBg: "#181926", lightBg: "#363a4f",
		fg: "#cad3f5", subtleFg: "#a5adcb", mutedFg: "#6e738d",
		accent: "#c6a0f6", selection: "#494d64",
		red: "#ed8796", green: "#a6da95", yellow: "#eed49f", orange: "#f5a97f",
		blue: "#8aadf4", magenta: "#c6a0f6", cyan: "#8bd5ca",
	}),
	buildScheme(schemeSpec{
		id: "catppuccin-frappe", name: "Catppuccin Frappé", family: "Catppuccin", category: "Modern",
		desc: "Soft muted pastel blue-gray", mode: "dark",
		bg: "#303446", darkBg: "#292c3c", darkerBg: "#232634", lightBg: "#414559",
		fg: "#c6d0f5", subtleFg: "#a5adce", mutedFg: "#737994",
		accent: "#ca9ee6", selection: "#51576d",
		red: "#e78284", green: "#a6d189", yellow: "#e5c890", orange: "#ef9f76",
		blue: "#8caaee", magenta: "#ca9ee6", cyan: "#81c8be",
	}),
	buildScheme(schemeSpec{
		id: "catppuccin-latte", name: "Catppuccin Latte", family: "Catppuccin", category: "Modern",
		desc: "Cozy pastel light theme", mode: "light",
		bg: "#eff1f5", darkBg: "#e6e9ef", darkerBg: "#dce0e8", lightBg: "#ccd0da",
		fg: "#4c4f69", subtleFg: "#6c6f85", mutedFg: "#8c8fa1",
		accent: "#8839ef", selection: "#bcc0cc",
		red: "#d20f39", green: "#40a02b", yellow: "#df8e1d", orange: "#fe640b",
		blue: "#1e66f5", magenta: "#8839ef", cyan: "#179299",
	}),
	buildScheme(schemeSpec{
		id: "tokyo-night", name: "Tokyo Night", family: "Tokyo Night", category: "Modern",
		desc: "Deep indigo night with vibrant purple, cyan & orange", mode: "dark",
		bg: "#1a1b26", darkBg: "#16161e", darkerBg: "#101014", lightBg: "#24283b",
		fg: "#c0caf5", subtleFg: "#a9b1d6", mutedFg: "#565f89",
		accent: "#7aa2f7", selection: "#283457",
		red: "#f7768e", green: "#9ece6a", yellow: "#e0af68", orange: "#ff9e64",
		blue: "#7aa2f7", magenta: "#bb9af7", cyan: "#7dcfff",
	}),
	buildScheme(schemeSpec{
		id: "tokyo-night-storm", name: "Tokyo Night Storm", family: "Tokyo Night", category: "Modern",
		desc: "Softer stormy blue night variant", mode: "dark",
		bg: "#24283b", darkBg: "#1f2335", darkerBg: "#1a1b26", lightBg: "#292e42",
		fg: "#c0caf5", subtleFg: "#a9b1d6", mutedFg: "#565f89",
		accent: "#7aa2f7", selection: "#364a82",
		red: "#f7768e", green: "#9ece6a", yellow: "#e0af68", orange: "#ff9e64",
		blue: "#7aa2f7", magenta: "#bb9af7", cyan: "#7dcfff",
	}),
	buildScheme(schemeSpec{
		id: "tokyo-night-day", name: "Tokyo Night Day", family: "Tokyo Night", category: "Modern",
		desc: "Crisp daylight Tokyo variant", mode: "light",
		bg: "#e1e2e7", darkBg: "#d5d6db", darkerBg: "#c4c8da", lightBg: "#cfd0d7",
		fg: "#3760bf", subtleFg: "#6172b0", mutedFg: "#848cb5",
		accent: "#2e7de9", selection: "#b6bfe2",
		red: "#f52a65", green: "#587539", yellow: "#8c6c3e", orange: "#b15c00",
		blue: "#2e7de9", magenta: "#9854f1", cyan: "#007197",
	}),
	buildScheme(schemeSpec{
		id: "rose-pine", name: "Rosé Pine", family: "Rosé Pine", category: "Modern",
		desc: "Minimalist earthy matte dark with soft rose tones", mode: "dark",
		bg: "#191724", darkBg: "#1f1d2e", darkerBg: "#13111c", lightBg: "#26233a",
		fg: "#e0def4", subtleFg: "#908caa", mutedFg: "#6e6a86",
		accent: "#ebbcba", selection: "#403d52",
		red: "#eb6f92", green: "#31748f", yellow: "#f6c177", orange: "#ebbcba",
		blue: "#9ccfd8", magenta: "#c4a7e7", cyan: "#9ccfd8",
	}),
	buildScheme(schemeSpec{
		id: "rose-pine-moon", name: "Rosé Pine Moon", family: "Rosé Pine", category: "Modern",
		desc: "Muted twilight violet Rosé Pine", mode: "dark",
		bg: "#232136", darkBg: "#2a273f", darkerBg: "#1d1b2d", lightBg: "#393552",
		fg: "#e0def4", subtleFg: "#908caa", mutedFg: "#6e6a86",
		accent: "#ea9a97", selection: "#44415a",
		red: "#eb6f92", green: "#3e8fb0", yellow: "#f6c177", orange: "#ea9a97",
		blue: "#9ccfd8", magenta: "#c4a7e7", cyan: "#9ccfd8",
	}),
	buildScheme(schemeSpec{
		id: "rose-pine-dawn", name: "Rosé Pine Dawn", family: "Rosé Pine", category: "Modern",
		desc: "Warm parchment and soft rose light theme", mode: "light",
		bg: "#faf4ed", darkBg: "#f2e9e1", darkerBg: "#e9dfd5", lightBg: "#dfdad9",
		fg: "#575279", subtleFg: "#797593", mutedFg: "#9893a5",
		accent: "#d7827e", selection: "#cecacd",
		red: "#b4637a", green: "#286983", yellow: "#ea9d34", orange: "#d7827e",
		blue: "#56949f", magenta: "#907aa9", cyan: "#56949f",
	}),
	buildScheme(schemeSpec{
		id: "kanagawa-wave", name: "Kanagawa Wave", family: "Kanagawa", category: "Modern",
		desc: "Great Wave inspired slate blue & warm beige", mode: "dark",
		bg: "#1f1f28", darkBg: "#16161d", darkerBg: "#121218", lightBg: "#2a2a37",
		fg: "#dcd7ba", subtleFg: "#c8c093", mutedFg: "#727169",
		accent: "#7e9cd8", selection: "#2d4f67",
		red: "#c34043", green: "#76946a", yellow: "#dca561", orange: "#ffa066",
		blue: "#7e9cd8", magenta: "#957fb8", cyan: "#6a9589",
	}),
	buildScheme(schemeSpec{
		id: "kanagawa-dragon", name: "Kanagawa Dragon", family: "Kanagawa", category: "Modern",
		desc: "Deep ink-black Kanagawa for late-night sessions", mode: "dark",
		bg: "#181616", darkBg: "#0d0c0c", darkerBg: "#090808", lightBg: "#282727",
		fg: "#c5c9c5", subtleFg: "#a6a69c", mutedFg: "#737c73",
		accent: "#8ba4b0", selection: "#2d4f67",
		red: "#c4746e", green: "#8a9a7b", yellow: "#c4b28a", orange: "#b6927b",
		blue: "#8ba4b0", magenta: "#a292a3", cyan: "#8ea4a2",
	}),
	buildScheme(schemeSpec{
		id: "kanagawa-lotus", name: "Kanagawa Lotus", family: "Kanagawa", category: "Modern",
		desc: "Warm Japanese paper light variant", mode: "light",
		bg: "#f2ecbc", darkBg: "#e7dba0", darkerBg: "#d5cea3", lightBg: "#e5ddb0",
		fg: "#545464", subtleFg: "#6f6f7f", mutedFg: "#8a8980",
		accent: "#4d699b", selection: "#c7d7e0",
		red: "#c84053", green: "#6f894e", yellow: "#77713f", orange: "#cc6d00",
		blue: "#4d699b", magenta: "#b35b79", cyan: "#597b75",
	}),
	buildScheme(schemeSpec{
		id: "everforest-dark", name: "Everforest Dark", family: "Everforest", category: "Modern",
		desc: "Warm low-contrast organic forest green palette", mode: "dark",
		bg: "#2d353b", darkBg: "#232a2e", darkerBg: "#1e2326", lightBg: "#343f44",
		fg: "#d3c6aa", subtleFg: "#9da9a0", mutedFg: "#7a8478",
		accent: "#a7c080", selection: "#4f5b58",
		red: "#e67e80", green: "#a7c080", yellow: "#dbbc7f", orange: "#e69875",
		blue: "#7fbbb3", magenta: "#d699b6", cyan: "#83c092",
	}),
	buildScheme(schemeSpec{
		id: "everforest-light", name: "Everforest Light", family: "Everforest", category: "Modern",
		desc: "Soft warm cream & sage low-strain light theme", mode: "light",
		bg: "#fdf6e3", darkBg: "#f4f0d9", darkerBg: "#efebd4", lightBg: "#e6e2cc",
		fg: "#5c6a72", subtleFg: "#829181", mutedFg: "#939f91",
		accent: "#8da101", selection: "#eaedc8",
		red: "#f85552", green: "#8da101", yellow: "#dfa000", orange: "#f57d26",
		blue: "#3a94c5", magenta: "#df69ba", cyan: "#35a77c",
	}),
	buildScheme(schemeSpec{
		id: "nord", name: "Nord", family: "Nord", category: "Modern",
		desc: "Arctic, north-bluish clean minimalist palette", mode: "dark",
		bg: "#2e3440", darkBg: "#272c36", darkerBg: "#212630", lightBg: "#3b4252",
		fg: "#eceff4", subtleFg: "#d8dee9", mutedFg: "#616e88",
		accent: "#88c0d0", selection: "#434c5e",
		red: "#bf616a", green: "#a3be8c", yellow: "#ebcb8b", orange: "#d08770",
		blue: "#81a1c1", magenta: "#b48ead", cyan: "#88c0d0",
	}),

	// ── Clássicos & Retrô ──
	buildScheme(schemeSpec{
		id: "gruvbox-dark", name: "Gruvbox Dark", family: "Gruvbox", category: "Classic",
		desc: "Retro warm groove with amber, olive & orange", mode: "dark",
		bg: "#282828", darkBg: "#1d2021", darkerBg: "#141617", lightBg: "#3c3836",
		fg: "#ebdbb2", subtleFg: "#d5c4a1", mutedFg: "#928374",
		accent: "#fabd2f", selection: "#504945",
		red: "#fb4934", green: "#b8bb26", yellow: "#fabd2f", orange: "#fe8019",
		blue: "#83a598", magenta: "#d3869b", cyan: "#8ec07c",
	}),
	buildScheme(schemeSpec{
		id: "gruvbox-light", name: "Gruvbox Light", family: "Gruvbox", category: "Classic",
		desc: "Retro warm cream parchment light groove", mode: "light",
		bg: "#fbf1c7", darkBg: "#f2e5bc", darkerBg: "#ebdbb2", lightBg: "#e0cfa9",
		fg: "#3c3836", subtleFg: "#504945", mutedFg: "#7c6f64",
		accent: "#b57614", selection: "#d5c4a1",
		red: "#cc241d", green: "#79740e", yellow: "#b57614", orange: "#af3a03",
		blue: "#076678", magenta: "#8f3f71", cyan: "#427b58",
	}),
	buildScheme(schemeSpec{
		id: "dracula", name: "Dracula", family: "Dracula", category: "Classic",
		desc: "High-contrast purple-gray dark with neon accents", mode: "dark",
		bg: "#282a36", darkBg: "#21222c", darkerBg: "#191a21", lightBg: "#343746",
		fg: "#f8f8f2", subtleFg: "#d6d8e6", mutedFg: "#6272a4",
		accent: "#bd93f9", selection: "#44475a",
		red: "#ff5555", green: "#50fa7b", yellow: "#f1fa8c", orange: "#ffb86c",
		blue: "#8be9fd", magenta: "#ff79c6", cyan: "#8be9fd",
	}),
	buildScheme(schemeSpec{
		id: "solarized-dark", name: "Solarized Dark", family: "Solarized", category: "Classic",
		desc: "Mathematically balanced deep teal precision contrast", mode: "dark",
		bg: "#002b36", darkBg: "#00212b", darkerBg: "#001921", lightBg: "#073642",
		fg: "#93a1a1", subtleFg: "#839496", mutedFg: "#586e75",
		accent: "#268bd2", selection: "#0b4452",
		red: "#dc322f", green: "#859900", yellow: "#b58900", orange: "#cb4b16",
		blue: "#268bd2", magenta: "#d33682", cyan: "#2aa198",
	}),
	buildScheme(schemeSpec{
		id: "solarized-light", name: "Solarized Light", family: "Solarized", category: "Classic",
		desc: "Precision warm cream light theme", mode: "light",
		bg: "#fdf6e3", darkBg: "#eee8d5", darkerBg: "#e3dcc8", lightBg: "#e6dfc9",
		fg: "#586e75", subtleFg: "#657b83", mutedFg: "#93a1a1",
		accent: "#268bd2", selection: "#d9d2be",
		red: "#dc322f", green: "#859900", yellow: "#b58900", orange: "#cb4b16",
		blue: "#268bd2", magenta: "#d33682", cyan: "#2aa198",
	}),
	buildScheme(schemeSpec{
		id: "monokai", name: "Monokai", family: "Monokai", category: "Classic",
		desc: "Sublime Text classic charcoal & vibrant syntax", mode: "dark",
		bg: "#272822", darkBg: "#1e1f1c", darkerBg: "#171814", lightBg: "#3e3d32",
		fg: "#f8f8f2", subtleFg: "#cfcfc2", mutedFg: "#75715e",
		accent: "#a6e22e", selection: "#49483e",
		red: "#f92672", green: "#a6e22e", yellow: "#e6db74", orange: "#fd971f",
		blue: "#66d9ef", magenta: "#ae81ff", cyan: "#66d9ef",
	}),
	buildScheme(schemeSpec{
		id: "monokai-pro", name: "Monokai Pro", family: "Monokai", category: "Classic",
		desc: "Refined warm charcoal with balanced modern tones", mode: "dark",
		bg: "#2d2a2e", darkBg: "#221f22", darkerBg: "#19181a", lightBg: "#403e41",
		fg: "#fcfcfa", subtleFg: "#c1c0c0", mutedFg: "#727072",
		accent: "#ffd866", selection: "#5b595c",
		red: "#ff6188", green: "#a9dc76", yellow: "#ffd866", orange: "#fc9867",
		blue: "#78dce8", magenta: "#ab9df2", cyan: "#78dce8",
	}),
	buildScheme(schemeSpec{
		id: "zenburn", name: "Zenburn", family: "Zenburn", category: "Classic",
		desc: "Low-contrast pioneer designed for long reading", mode: "dark",
		bg: "#3f3f3f", darkBg: "#353535", darkerBg: "#2b2b2b", lightBg: "#4f4f4f",
		fg: "#dcdccc", subtleFg: "#c0c0b0", mutedFg: "#7f9f7f",
		accent: "#f0dfaf", selection: "#5f5f5f",
		red: "#cc9393", green: "#7f9f7f", yellow: "#f0dfaf", orange: "#dfaf8f",
		blue: "#8cd0d3", magenta: "#dc8cc3", cyan: "#93e0e3",
	}),

	// ── Padrões de Editores & Ecossistemas ──
	buildScheme(schemeSpec{
		id: "one-dark", name: "One Dark", family: "One Dark", category: "Editors",
		desc: "Classic Atom neutral & balanced dark theme", mode: "dark",
		bg: "#282c34", darkBg: "#21252b", darkerBg: "#1b1f27", lightBg: "#31353f",
		fg: "#abb2bf", subtleFg: "#9da5b4", mutedFg: "#5c6370",
		accent: "#61afef", selection: "#3e4451",
		red: "#e06c75", green: "#98c379", yellow: "#e5c07b", orange: "#d19a66",
		blue: "#61afef", magenta: "#c678dd", cyan: "#56b6c2",
	}),
	buildScheme(schemeSpec{
		id: "one-dark-pro", name: "One Dark Pro", family: "One Dark", category: "Editors",
		desc: "VS Code & Neovim iconic balanced dark palette", mode: "dark",
		bg: "#23272e", darkBg: "#1e2227", darkerBg: "#181a1f", lightBg: "#2c313a",
		fg: "#abb2bf", subtleFg: "#9ca3b2", mutedFg: "#5c6370",
		accent: "#528bff", selection: "#3e4452",
		red: "#ef596f", green: "#89ca78", yellow: "#e5c07b", orange: "#d19a66",
		blue: "#61afef", magenta: "#d55fde", cyan: "#2bbac5",
	}),
	buildScheme(schemeSpec{
		id: "github-dark", name: "GitHub Dark Default", family: "GitHub", category: "Editors",
		desc: "Official GitHub dark interface palette", mode: "dark",
		bg: "#0d1117", darkBg: "#090d12", darkerBg: "#05080c", lightBg: "#161b22",
		fg: "#e6edf3", subtleFg: "#b1bac4", mutedFg: "#7d8590",
		accent: "#58a6ff", selection: "#1f6feb",
		red: "#ff7b72", green: "#7ee787", yellow: "#d29922", orange: "#ffa657",
		blue: "#79c0ff", magenta: "#d2a8ff", cyan: "#a5d6ff",
	}),
	buildScheme(schemeSpec{
		id: "github-dimmed", name: "GitHub Dark Dimmed", family: "GitHub", category: "Editors",
		desc: "Softer low-glare GitHub dark variant", mode: "dark",
		bg: "#22272e", darkBg: "#1c2128", darkerBg: "#161a20", lightBg: "#2d333b",
		fg: "#adbac7", subtleFg: "#909dab", mutedFg: "#768390",
		accent: "#539bf5", selection: "#373e47",
		red: "#f47067", green: "#8ddb8c", yellow: "#c69026", orange: "#f69d50",
		blue: "#6cb6ff", magenta: "#dcbdfb", cyan: "#96d0ff",
	}),
	buildScheme(schemeSpec{
		id: "github-light", name: "GitHub Light", family: "GitHub", category: "Editors",
		desc: "Crisp official GitHub light interface palette", mode: "light",
		bg: "#ffffff", darkBg: "#f6f8fa", darkerBg: "#eaeef2", lightBg: "#edf1f5",
		fg: "#24292f", subtleFg: "#57606a", mutedFg: "#6e7781",
		accent: "#0969da", selection: "#ddf4ff",
		red: "#cf222e", green: "#116329", yellow: "#9a6700", orange: "#bc4c00",
		blue: "#0550ae", magenta: "#8250df", cyan: "#1b7c83",
	}),
	buildScheme(schemeSpec{
		id: "material-ocean", name: "Material Ocean", family: "Material", category: "Editors",
		desc: "Deep ocean blue Google Material theme", mode: "dark",
		bg: "#0f111a", darkBg: "#090b10", darkerBg: "#06070b", lightBg: "#1a1c25",
		fg: "#a6accd", subtleFg: "#8f93a2", mutedFg: "#464b5d",
		accent: "#84ffff", selection: "#1f2233",
		red: "#f07178", green: "#c3e88d", yellow: "#ffcb6b", orange: "#f78c6c",
		blue: "#82aaff", magenta: "#c792ea", cyan: "#89ddff",
	}),
	buildScheme(schemeSpec{
		id: "material-darker", name: "Material Darker", family: "Material", category: "Editors",
		desc: "Crisp carbon charcoal Material palette", mode: "dark",
		bg: "#212121", darkBg: "#1a1a1a", darkerBg: "#141414", lightBg: "#2c2c2c",
		fg: "#eeffff", subtleFg: "#b2ccd6", mutedFg: "#616161",
		accent: "#82aaff", selection: "#353535",
		red: "#f07178", green: "#c3e88d", yellow: "#ffcb6b", orange: "#f78c6c",
		blue: "#82aaff", magenta: "#c792ea", cyan: "#89ddff",
	}),
	buildScheme(schemeSpec{
		id: "material-lighter", name: "Material Lighter", family: "Material", category: "Editors",
		desc: "Clean Material Design light theme", mode: "light",
		bg: "#fafafa", darkBg: "#eeeeee", darkerBg: "#e2e2e2", lightBg: "#e7e7e8",
		fg: "#546e7a", subtleFg: "#78909c", mutedFg: "#90a4ae",
		accent: "#6182b8", selection: "#cceae7",
		red: "#e53935", green: "#91b859", yellow: "#f6a434", orange: "#f76d47",
		blue: "#6182b8", magenta: "#7c4dff", cyan: "#39adb5",
	}),
	buildScheme(schemeSpec{
		id: "ayu-dark", name: "Ayu Dark", family: "Ayu", category: "Editors",
		desc: "Minimal deep obsidian & warm gold highlights", mode: "dark",
		bg: "#0b0e14", darkBg: "#07090d", darkerBg: "#05070a", lightBg: "#151a24",
		fg: "#bfbdb6", subtleFg: "#9d9b94", mutedFg: "#636a72",
		accent: "#e6b450", selection: "#1b3a5b",
		red: "#f07178", green: "#aad94c", yellow: "#ffb454", orange: "#ff8f40",
		blue: "#59c2ff", magenta: "#d2a6ff", cyan: "#95e6cb",
	}),
	buildScheme(schemeSpec{
		id: "ayu-mirage", name: "Ayu Mirage", family: "Ayu", category: "Editors",
		desc: "Sharp blue-gray mirage Ayu variant", mode: "dark",
		bg: "#1f2430", darkBg: "#191e2a", darkerBg: "#141822", lightBg: "#2b3242",
		fg: "#cccac2", subtleFg: "#a6a49c", mutedFg: "#707a8c",
		accent: "#ffcc66", selection: "#33415e",
		red: "#f28779", green: "#d5ff80", yellow: "#ffd173", orange: "#ffad66",
		blue: "#73d0ff", magenta: "#dfbfff", cyan: "#95e6cb",
	}),
	buildScheme(schemeSpec{
		id: "ayu-light", name: "Ayu Light", family: "Ayu", category: "Editors",
		desc: "Bright crisp minimalist daylight theme", mode: "light",
		bg: "#f8f9fa", darkBg: "#eff0f1", darkerBg: "#e5e7e9", lightBg: "#e8eaed",
		fg: "#5c6166", subtleFg: "#787b80", mutedFg: "#8a9199",
		accent: "#ffaa33", selection: "#d1e4f4",
		red: "#f07171", green: "#86b300", yellow: "#f2ae49", orange: "#fa8d3e",
		blue: "#399ee6", magenta: "#a37acc", cyan: "#4cbf99",
	}),

	// ── Vibrantes & Alta Legibilidade ──
	buildScheme(schemeSpec{
		id: "night-owl", name: "Night Owl", family: "Night Owl", category: "Vibrant",
		desc: "Deep navy night palette tuned for colorblind accessibility", mode: "dark",
		bg: "#011627", darkBg: "#01111d", darkerBg: "#010c15", lightBg: "#0b2942",
		fg: "#d6deeb", subtleFg: "#a9b7c6", mutedFg: "#637777",
		accent: "#82aaff", selection: "#1d3b53",
		red: "#ef5350", green: "#addb67", yellow: "#ecc48d", orange: "#f78c6c",
		blue: "#82aaff", magenta: "#c792ea", cyan: "#7fdbca",
	}),
	buildScheme(schemeSpec{
		id: "night-owl-light", name: "Night Owl Light", family: "Night Owl", category: "Vibrant",
		desc: "Accessible bright daytime Night Owl variant", mode: "light",
		bg: "#fbfbfb", darkBg: "#f0f0f0", darkerBg: "#e5e5e5", lightBg: "#e8edf2",
		fg: "#403f53", subtleFg: "#697098", mutedFg: "#989fb1",
		accent: "#4876d6", selection: "#d3e8f8",
		red: "#de3d3b", green: "#08916a", yellow: "#daaa01", orange: "#aa0982",
		blue: "#4876d6", magenta: "#994cc3", cyan: "#0c969b",
	}),
	buildScheme(schemeSpec{
		id: "synthwave-84", name: "SynthWave '84", family: "SynthWave '84", category: "Vibrant",
		desc: "Retro 80s neon magenta, purple & electric cyan", mode: "dark",
		bg: "#262335", darkBg: "#1e1b2a", darkerBg: "#171520", lightBg: "#34294f",
		fg: "#ffffff", subtleFg: "#d0c8e8", mutedFg: "#848bbd",
		accent: "#ff7edb", selection: "#463465",
		red: "#fe4450", green: "#72f1b8", yellow: "#fede5d", orange: "#f97e72",
		blue: "#36f9f6", magenta: "#ff7edb", cyan: "#36f9f6",
	}),
	buildScheme(schemeSpec{
		id: "palenight", name: "Palenight", family: "Palenight", category: "Vibrant",
		desc: "Soft purple-blue dark with gentle colorful syntax", mode: "dark",
		bg: "#292d3e", darkBg: "#232635", darkerBg: "#1c1f2b", lightBg: "#34394e",
		fg: "#a6accd", subtleFg: "#959dcb", mutedFg: "#676e95",
		accent: "#c792ea", selection: "#434758",
		red: "#f07178", green: "#c3e88d", yellow: "#ffcb6b", orange: "#f78c6c",
		blue: "#82aaff", magenta: "#c792ea", cyan: "#89ddff",
	}),
	buildScheme(schemeSpec{
		id: "cyberpunk", name: "Cyberpunk", family: "Cyberpunk / Eldritch", category: "Vibrant",
		desc: "Ultra-dark high-impact neon pink, cyan & purple", mode: "dark",
		bg: "#0b0c15", darkBg: "#070810", darkerBg: "#04050a", lightBg: "#171a2b",
		fg: "#e6f1ff", subtleFg: "#a9bcd0", mutedFg: "#5e6a8c",
		accent: "#ff2a6d", selection: "#2a1d3d",
		red: "#ff2a6d", green: "#05d9e8", yellow: "#f9f871", orange: "#ff9e00",
		blue: "#00b8ff", magenta: "#d300c5", cyan: "#05d9e8",
	}),
	buildScheme(schemeSpec{
		id: "eldritch", name: "Eldritch", family: "Cyberpunk / Eldritch", category: "Vibrant",
		desc: "Dark cosmic palette with teal, purple & neon green", mode: "dark",
		bg: "#212337", darkBg: "#191b2b", darkerBg: "#131521", lightBg: "#2e314d",
		fg: "#ebfafa", subtleFg: "#abb4da", mutedFg: "#7081d0",
		accent: "#37f499", selection: "#323449",
		red: "#f16c75", green: "#37f499", yellow: "#f1fc79", orange: "#f7c67f",
		blue: "#04d1f9", magenta: "#a48cf2", cyan: "#04d1f9",
	}),
	buildScheme(schemeSpec{
		id: "modus-vivendi", name: "Modus Vivendi", family: "Modus Themes", category: "Vibrant",
		desc: "WCAG AAA maximum contrast dark accessibility theme", mode: "dark",
		bg: "#000000", darkBg: "#0d0e1c", darkerBg: "#000000", lightBg: "#1e1e1e",
		fg: "#ffffff", subtleFg: "#c6daff", mutedFg: "#989898",
		accent: "#2fafff", selection: "#3c3c3c",
		red: "#ff5f59", green: "#44bc44", yellow: "#d0bc00", orange: "#fec43f",
		blue: "#2fafff", magenta: "#feacd0", cyan: "#00d3d0",
	}),
	buildScheme(schemeSpec{
		id: "modus-operandi", name: "Modus Operandi", family: "Modus Themes", category: "Vibrant",
		desc: "WCAG AAA maximum contrast light accessibility theme", mode: "light",
		bg: "#ffffff", darkBg: "#f2f2f2", darkerBg: "#e0e0e0", lightBg: "#e8e8e8",
		fg: "#000000", subtleFg: "#3b3b3b", mutedFg: "#595959",
		accent: "#0031a9", selection: "#c0deff",
		red: "#a60000", green: "#006800", yellow: "#6f5500", orange: "#884900",
		blue: "#0031a9", magenta: "#721045", cyan: "#005e8b",
	}),
}

var schemeIndex map[string]ColorScheme

var schemeAliases = map[string]string{
	"catppuccin":     "catppuccin-mocha",
	"tokyonight":     "tokyo-night",
	"tokyo":          "tokyo-night",
	"rosepine":       "rose-pine",
	"rose-pine-main": "rose-pine",
	"kanagawa":       "kanagawa-wave",
	"everforest":     "everforest-dark",
	"gruvbox":        "gruvbox-dark",
	"solarized":      "solarized-dark",
	"onedark":        "one-dark",
	"onedarkpro":     "one-dark-pro",
	"github":         "github-light",
	"material":       "material-ocean",
	"ayu":            "ayu-dark",
	"nightowl":       "night-owl",
	"synthwave84":    "synthwave-84",
	"synthwave":      "synthwave-84",
	"modus":          "modus-vivendi",
}

func init() {
	schemeIndex = make(map[string]ColorScheme, len(builtinColorSchemes))
	for _, s := range builtinColorSchemes {
		schemeIndex[s.ID] = s
	}
}

// BuiltinColorSchemes returns all curated colorschemes in display order.
func BuiltinColorSchemes() []ColorScheme {
	return builtinColorSchemes
}

// LookupColorScheme resolves a colorscheme by canonical ID or common alias.
func LookupColorScheme(name string) (ColorScheme, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return ColorScheme{}, false
	}
	if canonical, ok := schemeAliases[key]; ok {
		key = canonical
	}
	cs, ok := schemeIndex[key]
	return cs, ok
}

// BuildChromaStyleFromColors synthesizes a full Chroma syntax theme from an OmarchyColors palette.
func BuildChromaStyleFromColors(name string, c OmarchyColors) (*chroma.Style, error) {
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
		chroma.NameVariable:          c.Foreground,
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
	return chroma.NewStyle(name, entries)
}

// OmarchyColorScheme returns a live ColorScheme entry representing the active
// Omarchy desktop theme when an Omarchy colors.toml file is available.
func OmarchyColorScheme() (ColorScheme, bool) {
	colors, _, err := LoadCurrentOmarchyColors()
	if err != nil {
		return ColorScheme{}, false
	}
	mode := colors.Mode
	if mode != "light" {
		mode = "dark"
	}
	return ColorScheme{
		ID:          "omarchy",
		Name:        "Omarchy (Auto)",
		Family:      "Omarchy",
		Category:    "Modern",
		Description: "Follows active Omarchy desktop theme (live reload)",
		Mode:        mode,
		Colors:      colors,
	}, true
}

// ThemePickerColorSchemes returns all selectable colorschemes for the Theme
// Picker, placing the live Omarchy (Auto) theme first when Omarchy is present.
func ThemePickerColorSchemes() []ColorScheme {
	builtin := BuiltinColorSchemes()
	if om, ok := OmarchyColorScheme(); ok {
		out := make([]ColorScheme, 0, len(builtin)+1)
		out = append(out, om)
		out = append(out, builtin...)
		return out
	}
	return builtin
}
