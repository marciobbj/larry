package ui

import "github.com/charmbracelet/lipgloss"

var (
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
)

func initStyles() {
	isDark := lipgloss.HasDarkBackground()

	if isDark {
		styleCursor = lipgloss.NewStyle().Background(lipgloss.Color("252")).Foreground(lipgloss.Color("0"))
		styleSelected = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255"))
		styleSearch = lipgloss.NewStyle().Background(lipgloss.Color("226")).Foreground(lipgloss.Color("0"))
		styleFile = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("235"))
		styleDir = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true).Background(lipgloss.Color("235"))
		lineNumStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		statusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("237"))
		modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Background(lipgloss.Color("235"))
		modalTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Background(lipgloss.Color("235")).
			Bold(true).
			Align(lipgloss.Center)

		helpStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252"))
		helpTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Background(lipgloss.Color("236")).
			Bold(true)
		categoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Background(lipgloss.Color("236")).
			Bold(true)
		keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("118")).
			Background(lipgloss.Color("236")).
			Bold(true)
		descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248")).
			Background(lipgloss.Color("236"))
		footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("236"))

		keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("118")).
			Background(lipgloss.Color("236")).
			Bold(true)
		descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248")).
			Background(lipgloss.Color("236"))
		footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Background(lipgloss.Color("236")).
			MarginTop(1)

		textInputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))

		treeHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
		treeHeaderFocusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
		treeDividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		treeDividerFocusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
		treeGuideStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
		treeDirStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
		treeFileStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		treeSymlinkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
		treeActiveFileStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("118")).Bold(true)
		treeCursorFocusedStyle = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255")).Bold(true)
		treeCursorUnfocusedStyle = lipgloss.NewStyle().Background(lipgloss.Color("237")).Foreground(lipgloss.Color("250"))
		treeEmptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
		treeRuleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	} else {
		styleCursor = lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("255"))
		styleSelected = lipgloss.NewStyle().Background(lipgloss.Color("153")).Foreground(lipgloss.Color("0"))
		styleSearch = lipgloss.NewStyle().Background(lipgloss.Color("226")).Foreground(lipgloss.Color("0"))
		styleFile = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("254"))
		styleDir = lipgloss.NewStyle().Foreground(lipgloss.Color("27")).Bold(true).Background(lipgloss.Color("254"))
		lineNumStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		statusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("235")).Background(lipgloss.Color("252"))
		modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Background(lipgloss.Color("254"))
		modalTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("125")).
			Background(lipgloss.Color("254")).
			Bold(true).
			Align(lipgloss.Center)

		helpStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Background(lipgloss.Color("254")).
			Foreground(lipgloss.Color("235"))
		helpTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("125")).
			Background(lipgloss.Color("254")).
			Bold(true)
		categoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("25")).
			Background(lipgloss.Color("254")).
			Bold(true)
		keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("22")).
			Background(lipgloss.Color("254")).
			Bold(true)
		descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("254"))
		footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Background(lipgloss.Color("254"))

		keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("22")).
			Background(lipgloss.Color("254")).
			Bold(true)
		descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("254"))
		footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Background(lipgloss.Color("254")).
			MarginTop(1)

		textInputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0"))

		treeHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
		treeHeaderFocusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("125")).Bold(true)
		treeDividerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		treeDividerFocusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
		treeGuideStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
		treeDirStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("27")).Bold(true)
		treeFileStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("235"))
		treeSymlinkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("31"))
		treeActiveFileStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("22")).Bold(true)
		treeCursorFocusedStyle = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255")).Bold(true)
		treeCursorUnfocusedStyle = lipgloss.NewStyle().Background(lipgloss.Color("252")).Foreground(lipgloss.Color("236"))
		treeEmptyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true)
		treeRuleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	}
}
