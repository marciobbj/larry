package ui

import (
	"fmt"
	"larry/internal/search"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FinderMode int

const (
	FinderModeFile FinderMode = iota
	FinderModeGrep
)

type FinderModel struct {
	textInput textinput.Model
	mode      FinderMode
	results   []search.FinderResult
	cursor    int
	width     int
	height    int
	matcher   *search.FuzzyMatcher
	grep      *search.LiveGrep
	scanner   *search.DirectoryScanner
	allFiles  []string
	loading   bool
	root      string
}

func NewFinderModel(width, height int) FinderModel {
	ti := textinput.New()
	ti.Placeholder = "Search files or content..."
	ti.Prompt = " » "
	ti.Focus()

	return FinderModel{
		textInput: ti,
		mode:      FinderModeFile,
		matcher:   search.NewFuzzyMatcher(),
		grep:      search.NewLiveGrep(),
		scanner:   search.NewDirectoryScanner(),
		root:      ".",
		width:     width,
		height:    height,
	}
}

func (m FinderModel) Init() tea.Cmd {
	return nil
}

type searchMsg []search.FinderResult

func (m FinderModel) Update(msg tea.Msg) (FinderModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.mode == FinderModeFile {
				m.mode = FinderModeGrep
			} else {
				m.mode = FinderModeFile
			}
			return m, m.performSearch()

		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
		}

	case searchMsg:
		m.results = msg
		m.loading = false
		if m.cursor >= len(m.results) {
			m.cursor = 0
		}
		return m, nil
	}

	oldQuery := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)
	if m.textInput.Value() != oldQuery {
		return m, m.performSearch()
	}

	return m, cmd
}

func (m *FinderModel) performSearch() tea.Cmd {
	query := m.textInput.Value()
	m.loading = true

	return func() tea.Msg {
		if m.mode == FinderModeFile {
			if m.allFiles == nil {
				files, _ := m.scanner.Scan(m.root)
				m.allFiles = files
			}

			var results []search.FinderResult
			for _, f := range m.allFiles {
				if matched, _ := m.matcher.Match(query, f); matched {
					results = append(results, search.FinderResult{
						File: &search.FileResult{Path: f},
						Mode: search.ModeFiles,
					})
					if len(results) > 50 {
						break
					}
				}
			}
			return searchMsg(results)
		} else {
			results, _ := m.grep.Search(m.root, query)
			var finderResults []search.FinderResult
			for i := range results {
				finderResults = append(finderResults, search.FinderResult{
					Grep: &results[i],
					Mode: search.ModeGrep,
				})
				if len(finderResults) > 50 {
					break
				}
			}
			return searchMsg(finderResults)
		}
	}
}

func (m FinderModel) View() string {
	var modeStr string
	if m.mode == FinderModeFile {
		modeStr = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255")).Padding(0, 1).Render(" FILES ")
	} else {
		modeStr = lipgloss.NewStyle().Background(lipgloss.Color("160")).Foreground(lipgloss.Color("255")).Padding(0, 1).Render(" GREP ")
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center, modeStr, " ", m.textInput.View())

	maxResults := m.height - 10
	if maxResults < 5 {
		maxResults = 5
	}

	var resultsView strings.Builder
	count := 0

	start := 0
	if m.cursor >= maxResults {
		start = m.cursor - maxResults + 1
	}

	for i := start; i < len(m.results) && count < maxResults; i++ {
		res := m.results[i]
		cursor := "  "
		if i == m.cursor {
			cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render("» ")
		}

		var line string
		if res.Mode == search.ModeFiles {
			line = res.File.Path
		} else {
			line = fmt.Sprintf("%s:%d: %s", res.Grep.Path, res.Grep.Line, res.Grep.Content)
		}

		maxWidth := m.width - 15
		if maxWidth < 10 {
			maxWidth = 10
		}
		if len(line) > maxWidth {
			line = line[:maxWidth-3] + "..."
		}

		if i == m.cursor {
			resultsView.WriteString(cursor + lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("237")).Render(line) + "\n")
		} else {
			resultsView.WriteString(cursor + line + "\n")
		}
		count++
	}

	if count == 0 && !m.loading {
		resultsView.WriteString("\n  No results found.")
	} else if m.loading {
		resultsView.WriteString("\n  Searching...")
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, "\n", resultsView.String())
}
