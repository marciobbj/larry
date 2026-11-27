package ui

import "github.com/charmbracelet/lipgloss"

// UIPalette holds semantic colors shared across every Larry UI component.
type UIPalette struct {
	IsDark      bool
	Accent      lipgloss.Color
	AccentAlt   lipgloss.Color
	AccentFg    lipgloss.Color
	Background  lipgloss.Color
	Surface     lipgloss.Color
	Selection   lipgloss.Color
	SelectionFg lipgloss.Color
	Foreground  lipgloss.Color
	SubtleFg    lipgloss.Color
	MutedFg     lipgloss.Color
	Border      lipgloss.Color
	Red         lipgloss.Color
	Green       lipgloss.Color
	Yellow      lipgloss.Color
	Blue        lipgloss.Color
	Cyan        lipgloss.Color
	Magenta     lipgloss.Color
	CursorBg    lipgloss.Color
	CursorFg    lipgloss.Color
}

var (
	activePalette = defaultPalette(true)

	styleCursor   lipgloss.Style
	styleSelected lipgloss.Style
	styleSearch   lipgloss.Style
	styleFile     lipgloss.Style
	styleDir      lipgloss.Style
	lineNumStyle  lipgloss.Style

	borderStyle    lipgloss.Style
	statusBarStyle lipgloss.Style

	modalStyle      lipgloss.Style
	modalTitleStyle lipgloss.Style

	helpStyle      lipgloss.Style
	helpTitleStyle lipgloss.Style
	categoryStyle  lipgloss.Style
	keyStyle       lipgloss.Style
	descStyle      lipgloss.Style
	footerStyle    lipgloss.Style

	textInputStyle lipgloss.Style

	treeHeaderStyle          lipgloss.Style
	treeHeaderFocusedStyle   lipgloss.Style
	treeDividerStyle         lipgloss.Style
	treeDividerFocusedStyle  lipgloss.Style
	treeGuideStyle           lipgloss.Style
	treeDirStyle             lipgloss.Style
	treeFileStyle            lipgloss.Style
	treeSymlinkStyle         lipgloss.Style
	treeActiveFileStyle      lipgloss.Style
	treeCursorFocusedStyle   lipgloss.Style
	treeCursorUnfocusedStyle lipgloss.Style
	treeEmptyStyle           lipgloss.Style
	treeRuleStyle            lipgloss.Style

	gitAddedStyle     lipgloss.Style
	gitModifiedStyle  lipgloss.Style
	gitDeletedStyle   lipgloss.Style
	gitUntrackedStyle lipgloss.Style
	gitDirDirtyStyle  lipgloss.Style
)

func defaultPalette(isDark bool) UIPalette {
	if isDark {
		return UIPalette{
			IsDark:      true,
			Accent:      lipgloss.Color("62"),
			AccentAlt:   lipgloss.Color("170"),
			AccentFg:    lipgloss.Color("255"),
			Background:  lipgloss.Color("235"),
			Surface:     lipgloss.Color("236"),
			Selection:   lipgloss.Color("62"),
			SelectionFg: lipgloss.Color("255"),
			Foreground:  lipgloss.Color("252"),
			SubtleFg:    lipgloss.Color("248"),
			MutedFg:     lipgloss.Color("241"),
			Border:      lipgloss.Color("240"),
			Red:         lipgloss.Color("203"),
			Green:       lipgloss.Color("118"),
			Yellow:      lipgloss.Color("226"),
			Blue:        lipgloss.Color("39"),
			Cyan:        lipgloss.Color("81"),
			Magenta:     lipgloss.Color("170"),
			CursorBg:    lipgloss.Color("252"),
			CursorFg:    lipgloss.Color("0"),
		}
	}

	return UIPalette{
		IsDark:      false,
		Accent:      lipgloss.Color("62"),
		AccentAlt:   lipgloss.Color("125"),
		AccentFg:    lipgloss.Color("255"),
		Background:  lipgloss.Color("254"),
		Surface:     lipgloss.Color("252"),
		Selection:   lipgloss.Color("153"),
		SelectionFg: lipgloss.Color("0"),
		Foreground:  lipgloss.Color("235"),
		SubtleFg:    lipgloss.Color("240"),
		MutedFg:     lipgloss.Color("244"),
		Border:      lipgloss.Color("244"),
		Red:         lipgloss.Color("160"),
		Green:       lipgloss.Color("22"),
		Yellow:      lipgloss.Color("226"),
		Blue:        lipgloss.Color("27"),
		Cyan:        lipgloss.Color("25"),
		Magenta:     lipgloss.Color("125"),
		CursorBg:    lipgloss.Color("235"),
		CursorFg:    lipgloss.Color("255"),
	}
}

func initStyles() {
	p := activePalette

	styleCursor = lipgloss.NewStyle().Background(p.CursorBg).Foreground(p.CursorFg)
	styleSelected = lipgloss.NewStyle().Background(p.Selection).Foreground(p.SelectionFg)
	styleSearch = lipgloss.NewStyle().Background(p.Yellow).Foreground(lipgloss.Color("0"))
	styleFile = lipgloss.NewStyle().Foreground(p.Foreground).Background(p.Background)
	styleDir = lipgloss.NewStyle().Foreground(p.Blue).Bold(true).Background(p.Background)
	lineNumStyle = lipgloss.NewStyle().Foreground(p.MutedFg)
	borderStyle = lipgloss.NewStyle().Foreground(p.Border)
	statusBarStyle = lipgloss.NewStyle().Foreground(p.Foreground).Background(p.Surface)

	modalStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Accent).
		Padding(1, 2).
		Background(p.Background)
	modalTitleStyle = lipgloss.NewStyle().
		Foreground(p.AccentAlt).
		Background(p.Background).
		Bold(true).
		Align(lipgloss.Center)

	helpStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Accent).
		Padding(1, 2).
		Background(p.Surface).
		Foreground(p.Foreground)
	helpTitleStyle = lipgloss.NewStyle().
		Foreground(p.AccentAlt).
		Background(p.Surface).
		Bold(true)
	categoryStyle = lipgloss.NewStyle().
		Foreground(p.Cyan).
		Background(p.Surface).
		Bold(true)
	keyStyle = lipgloss.NewStyle().
		Foreground(p.Green).
		Background(p.Surface).
		Bold(true)
	descStyle = lipgloss.NewStyle().
		Foreground(p.SubtleFg).
		Background(p.Surface)
	footerStyle = lipgloss.NewStyle().
		Foreground(p.MutedFg).
		Background(p.Surface).
		MarginTop(1)

	textInputStyle = lipgloss.NewStyle().Foreground(p.Foreground)

	treeHeaderStyle = lipgloss.NewStyle().Foreground(p.SubtleFg).Bold(true)
	treeHeaderFocusedStyle = lipgloss.NewStyle().Foreground(p.Accent).Bold(true)
	treeDividerStyle = lipgloss.NewStyle().Foreground(p.Border)
	treeDividerFocusedStyle = lipgloss.NewStyle().Foreground(p.Accent)
	treeGuideStyle = lipgloss.NewStyle().Foreground(p.Border)
	treeDirStyle = lipgloss.NewStyle().Foreground(p.Blue).Bold(true)
	treeFileStyle = lipgloss.NewStyle().Foreground(p.Foreground)
	treeSymlinkStyle = lipgloss.NewStyle().Foreground(p.Cyan)
	treeActiveFileStyle = lipgloss.NewStyle().Foreground(p.Green).Bold(true)
	treeCursorFocusedStyle = lipgloss.NewStyle().Background(p.Accent).Foreground(p.AccentFg).Bold(true)
	treeCursorUnfocusedStyle = lipgloss.NewStyle().Background(p.Surface).Foreground(p.SubtleFg)
	treeEmptyStyle = lipgloss.NewStyle().Foreground(p.MutedFg).Italic(true)
	treeRuleStyle = lipgloss.NewStyle().Foreground(p.Border)

	gitAddedStyle = lipgloss.NewStyle().Foreground(p.Green).Bold(true)
	gitModifiedStyle = lipgloss.NewStyle().Foreground(p.Yellow).Bold(true)
	gitDeletedStyle = lipgloss.NewStyle().Foreground(p.Red).Bold(true)
	gitUntrackedStyle = lipgloss.NewStyle().Foreground(p.Cyan).Bold(true)
	gitDirDirtyStyle = lipgloss.NewStyle().Foreground(p.Yellow)
}
