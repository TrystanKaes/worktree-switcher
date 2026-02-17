package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))  // cyan
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // gray
	branchStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))             // yellow
	timeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // gray
	filterStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true)  // magenta
	prevStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))             // blue
)

type model struct {
	worktrees   []Worktree
	filtered    []int // indices into worktrees
	cursor      int
	filter      string
	selected    string // result path
	quitting    bool
	previousIdx int // index of previous worktree (-1 if none)
}

func newModel(worktrees []Worktree, previousIdx int) model {
	indices := make([]int, len(worktrees))
	for i := range worktrees {
		indices[i] = i
	}
	return model{
		worktrees:   worktrees,
		filtered:    indices,
		previousIdx: previousIdx,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil

		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				idx := m.filtered[m.cursor]
				m.selected = m.worktrees[idx].Path
			}
			m.quitting = true
			return m, tea.Quit

		case tea.KeyBackspace, tea.KeyDelete:
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
			}
			return m, nil

		case tea.KeyRunes:
			m.filter += string(msg.Runes)
			m.applyFilter()
			return m, nil
		}
	}
	return m, nil
}

func (m *model) applyFilter() {
	if m.filter == "" {
		m.filtered = make([]int, len(m.worktrees))
		for i := range m.worktrees {
			m.filtered[i] = i
		}
	} else {
		lower := strings.ToLower(m.filter)
		m.filtered = nil
		for i, ws := range m.worktrees {
			if fuzzyMatch(lower, strings.ToLower(ws.Path)) ||
				fuzzyMatch(lower, strings.ToLower(ws.ShortPath())) ||
				fuzzyMatch(lower, strings.ToLower(ws.Branch)) {
				m.filtered = append(m.filtered, i)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// fuzzyMatch checks if all characters in pattern appear in str in order.
func fuzzyMatch(pattern, str string) bool {
	pi := 0
	for si := 0; si < len(str) && pi < len(pattern); si++ {
		if str[si] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Filter line
	if m.filter != "" {
		b.WriteString(filterStyle.Render("filter: " + m.filter))
		b.WriteString("\n\n")
	}

	if len(m.filtered) == 0 {
		b.WriteString(dimStyle.Render("  no matching worktrees"))
		b.WriteString("\n")
		return b.String()
	}

	// Compute column widths for alignment
	maxPath := 0
	maxBranch := 0
	for _, idx := range m.filtered {
		ws := m.worktrees[idx]
		sp := ws.ShortPath()
		if len(sp) > maxPath {
			maxPath = len(sp)
		}
		if len(ws.Branch) > maxBranch {
			maxBranch = len(ws.Branch)
		}
	}

	for i, idx := range m.filtered {
		ws := m.worktrees[idx]
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		path := ws.ShortPath()
		branch := ws.Branch
		relTime := ws.RelativeTime()

		// Pad for alignment
		pathPadded := path + strings.Repeat(" ", maxPath-len(path))
		branchPadded := branch + strings.Repeat(" ", maxBranch-len(branch))

		prevLabel := ""
		if idx == m.previousIdx {
			prevLabel = "  " + prevStyle.Render("(prev)")
		}

		if i == m.cursor {
			line := fmt.Sprintf("%s%s  %s  %s%s",
				cursor,
				selectedStyle.Render(pathPadded),
				branchStyle.Render(branchPadded),
				timeStyle.Render(relTime),
				prevLabel,
			)
			b.WriteString(line)
		} else {
			line := fmt.Sprintf("%s%s  %s  %s%s",
				cursor,
				dimStyle.Render(pathPadded),
				dimStyle.Render(branchPadded),
				dimStyle.Render(relTime),
				prevLabel,
			)
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("↑/↓ navigate • enter select • type to filter • esc quit"))
	b.WriteString("\n")

	return b.String()
}

// RunTUI launches the interactive worktree picker.
// Returns the selected path, or empty string if cancelled.
func RunTUI(worktrees []Worktree, previousIdx int) (string, error) {
	m := newModel(worktrees, previousIdx)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("TUI error: %w", err)
	}
	final := result.(model)
	return final.selected, nil
}
