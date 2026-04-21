package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/trystankaes/worktree-switcher/columns"
)

// deleteResultMsg is sent back from an async RemoveWorktree call.
type deleteResultMsg struct {
	path  string
	err   error
	force bool // whether this was a force attempt
}

func removeWorktreeCmd(path string, force bool) tea.Cmd {
	return func() tea.Msg {
		err := RemoveWorktree(path, force)
		return deleteResultMsg{path: path, err: err, force: force}
	}
}

// Use a renderer targeting stderr since the TUI outputs there
// (stdout is captured by the shell wrapper for cd).
var renderer = lipgloss.NewRenderer(os.Stderr)

var (
	headerStyle   = renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("2")) // green
	selectedStyle = renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("6")) // cyan
	dimStyle      = renderer.NewStyle().Foreground(lipgloss.Color("8"))            // gray
	branchStyle   = renderer.NewStyle().Foreground(lipgloss.Color("3"))            // yellow
	filterStyle   = renderer.NewStyle().Foreground(lipgloss.Color("5")).Bold(true) // magenta
	prevStyle     = renderer.NewStyle().Foreground(lipgloss.Color("4"))            // blue
	deleteStyle   = renderer.NewStyle().Bold(true).Foreground(lipgloss.Color("1")) // red
	dangerStyle   = renderer.NewStyle().Foreground(lipgloss.Color("1"))            // red dim
)

type model struct {
	worktrees     []Worktree
	rows          []columns.Row // parallel to worktrees; rich data for rendering
	layout        columns.Layout
	footer        string
	width         int
	filtered      []int // indices into worktrees
	cursor        int
	filter        string
	selected      string // result path
	quitting      bool
	previousIdx   int  // index of previous worktree (-1 if none)
	deleteMode    bool // true when 'd' has been pressed
	confirmDelete bool // true when showing confirmation dialog
	confirmChoice int  // 0 = Cancel, 1 = Delete
	deleteErr     string
	forcePrompt   bool   // true when showing "force delete?" prompt
	forceChoice   int    // 0 = No, 1 = Yes
	forceErr      string // the original error message shown in the force prompt
	creating      bool   // true when in create-new input mode
	createInput   string // branch name being typed
	createErr     string
}

func newModel(worktrees []Worktree, rows []columns.Row, previousIdx int) model {
	const defaultWidth = 100
	indices := make([]int, len(worktrees))
	for i := range worktrees {
		indices[i] = i
	}
	layout := columns.Allocate(rows, defaultWidth)
	footer := columns.Format(columns.Aggregate(rows, layout.HiddenCount))
	return model{
		worktrees:   worktrees,
		rows:        rows,
		layout:      layout,
		footer:      footer,
		width:       defaultWidth,
		filtered:    indices,
		previousIdx: previousIdx,
	}
}

// tuiStyles returns a Styles set targeting the TUI renderer (stderr).
// Colours match the existing ui.go palette.
func tuiStyles() columns.Styles {
	return columns.Styles{
		Dim:      dimStyle,
		Addition: renderer.NewStyle().Foreground(lipgloss.Color("2")),
		Deletion: renderer.NewStyle().Foreground(lipgloss.Color("1")),
		Selected: selectedStyle,
		Delete:   deleteStyle,
		Branch:   branchStyle,
		Danger:   dangerStyle,
		Header:   headerStyle,
		Prev:     prevStyle,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// onCreateNew returns true when the cursor is on the "+ Create new" row.
func (m model) onCreateNew() bool {
	return !m.deleteMode && m.cursor == len(m.filtered)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case deleteResultMsg:
		if msg.err != nil {
			errMsg := msg.err.Error()
			if !msg.force && strings.Contains(errMsg, "--force") {
				m.confirmDelete = false
				m.confirmChoice = 0
				m.forcePrompt = true
				m.forceChoice = 0
				m.forceErr = errMsg
				return m, nil
			}
			m.deleteErr = errMsg
		} else {
			// Remove the deleted worktree by matching on path; keep rows in sync.
			for i, wt := range m.worktrees {
				if wt.Path == msg.path {
					m.worktrees = append(m.worktrees[:i], m.worktrees[i+1:]...)
					m.rows = append(m.rows[:i], m.rows[i+1:]...)
					if m.previousIdx == i {
						m.previousIdx = -1
					} else if m.previousIdx > i {
						m.previousIdx--
					}
					break
				}
			}
			m.applyFilter()
			if m.cursor >= len(m.filtered) {
				m.cursor = max(0, len(m.filtered)-1)
			}
		}
		m.confirmDelete = false
		m.confirmChoice = 0
		m.forcePrompt = false
		m.forceChoice = 0
		m.forceErr = ""
		return m, nil

	case tea.KeyMsg:
		// Create input mode handling
		if m.creating {
			switch msg.Type {
			case tea.KeyEsc, tea.KeyCtrlC:
				m.creating = false
				m.createInput = ""
				m.createErr = ""
				return m, nil
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.createInput) > 0 {
					m.createInput = m.createInput[:len(m.createInput)-1]
				}
				return m, nil
			case tea.KeyEnter:
				branch := strings.TrimSpace(m.createInput)
				if branch == "" {
					m.createErr = "branch name cannot be empty"
					return m, nil
				}
				repo, err := RepoName()
				if err != nil {
					m.createErr = err.Error()
					return m, nil
				}
				path, _, err := CreateWorktreeForBranch(repo, branch, false)
				if err != nil {
					m.createErr = err.Error()
					return m, nil
				}
				m.selected = path
				m.quitting = true
				return m, tea.Quit
			case tea.KeyRunes:
				m.createInput += string(msg.Runes)
				m.createErr = ""
				return m, nil
			}
			return m, nil
		}

		// Force delete prompt handling
		if m.forcePrompt {
			switch msg.Type {
			case tea.KeyEsc, tea.KeyCtrlC:
				m.forcePrompt = false
				m.forceChoice = 0
				m.forceErr = ""
				return m, nil
			case tea.KeyLeft, tea.KeyRight:
				m.forceChoice = 1 - m.forceChoice
				return m, nil
			case tea.KeyEnter:
				if m.forceChoice == 1 && len(m.filtered) > 0 {
					idx := m.filtered[m.cursor]
					ws := m.worktrees[idx]
					return m, removeWorktreeCmd(ws.Path, true)
				}
				// "No" selected — go back to delete mode list
				m.forcePrompt = false
				m.forceChoice = 0
				m.forceErr = ""
				return m, nil
			case tea.KeyRunes:
				r := string(msg.Runes)
				if r == "h" {
					m.forceChoice = 0
				} else if r == "l" {
					m.forceChoice = 1
				}
				return m, nil
			}
			return m, nil
		}

		// Confirmation dialog handling
		if m.confirmDelete {
			switch msg.Type {
			case tea.KeyEsc, tea.KeyCtrlC:
				m.confirmDelete = false
				m.confirmChoice = 0
				return m, nil
			case tea.KeyLeft, tea.KeyRight:
				m.confirmChoice = 1 - m.confirmChoice
				return m, nil
			case tea.KeyEnter:
				if m.confirmChoice == 1 && len(m.filtered) > 0 {
					idx := m.filtered[m.cursor]
					ws := m.worktrees[idx]
					return m, removeWorktreeCmd(ws.Path, false)
				}
				// "Cancel" selected
				m.confirmDelete = false
				m.confirmChoice = 0
				return m, nil
			case tea.KeyRunes:
				r := string(msg.Runes)
				if r == "h" {
					m.confirmChoice = 0
				} else if r == "l" {
					m.confirmChoice = 1
				}
				return m, nil
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			if m.deleteMode {
				m.deleteMode = false
				m.deleteErr = ""
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case tea.KeyDown:
			maxCursor := len(m.filtered) - 1
			if !m.deleteMode {
				maxCursor = len(m.filtered) // allow "create new" row
			}
			if m.cursor < maxCursor {
				m.cursor++
			}
			return m, nil

		case tea.KeyEnter:
			if m.deleteMode {
				if len(m.filtered) > 0 {
					m.confirmDelete = true
					m.confirmChoice = 0
					m.deleteErr = ""
				}
				return m, nil
			}
			if m.onCreateNew() {
				m.creating = true
				m.createInput = ""
				m.createErr = ""
				return m, nil
			}
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
			r := string(msg.Runes)
			if (r == "d" || r == "D") && m.filter == "" {
				m.deleteMode = !m.deleteMode
				m.deleteErr = ""
				// Clamp cursor when entering delete mode (no "create new" row)
				if m.deleteMode && m.cursor >= len(m.filtered) {
					m.cursor = max(0, len(m.filtered)-1)
				}
				return m, nil
			}
			m.filter += r
			m.applyFilter()
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.layout = columns.Allocate(m.rows, m.width)
		m.footer = columns.Format(columns.Aggregate(m.rows, m.layout.HiddenCount))
		return m, nil
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
	maxCursor := len(m.filtered) // "create new" row
	if m.deleteMode {
		maxCursor = max(0, len(m.filtered)-1)
	}
	if m.cursor > maxCursor {
		m.cursor = maxCursor
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

	// Delete mode header
	if m.deleteMode {
		b.WriteString(headerStyle.Render("WORKTREE SWITCHER"))
		b.WriteString(deleteStyle.Render(" (delete mode)"))
		b.WriteString("\n\n")
	} else {
		b.WriteString(headerStyle.Render("WORKTREE SWITCHER"))
		b.WriteString("\n\n")
	}

	// Filter line
	if m.filter != "" {
		b.WriteString(filterStyle.Render("filter: " + m.filter))
		b.WriteString("\n\n")
	}

	// Create input mode
	if m.creating {
		b.WriteString(selectedStyle.Render("+ Create new worktree"))
		b.WriteString("\n\n")
		b.WriteString("  Branch name: " + filterStyle.Render(m.createInput))
		b.WriteString(selectedStyle.Render("▎"))
		b.WriteString("\n")
		if m.createErr != "" {
			b.WriteString("\n" + dangerStyle.Render("  error: "+m.createErr) + "\n")
		}
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("enter create • esc cancel"))
		b.WriteString("\n")
		return b.String()
	}

	if len(m.filtered) == 0 {
		b.WriteString(dimStyle.Render("  no matching worktrees"))
		b.WriteString("\n")
		if m.deleteMode {
			return b.String()
		}
		b.WriteString("\n")
		// cursor is at 0 which == len(m.filtered), so onCreateNew() is true
		b.WriteString(selectedStyle.Render("> + Create new worktree"))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("↑/↓ navigate • enter select • type to filter • d delete • esc quit"))
		b.WriteString("\n")
		return b.String()
	}

	// Confirmation dialog
	if m.confirmDelete && len(m.filtered) > 0 {
		idx := m.filtered[m.cursor]
		ws := m.worktrees[idx]
		b.WriteString(fmt.Sprintf("  Delete %s?\n\n", deleteStyle.Render(ws.ShortPath())))
		cancel := "  Cancel  "
		del := "  Delete  "
		if m.confirmChoice == 0 {
			cancel = selectedStyle.Render("▸ Cancel  ")
			del = dimStyle.Render("  Delete  ")
		} else {
			cancel = dimStyle.Render("  Cancel  ")
			del = deleteStyle.Render("▸ Delete  ")
		}
		b.WriteString("  " + cancel + del + "\n\n")
		b.WriteString(dimStyle.Render("←/→ choose • enter confirm • esc cancel"))
		b.WriteString("\n")
		return b.String()
	}

	// Force delete prompt
	if m.forcePrompt {
		b.WriteString(dangerStyle.Render("  "+m.forceErr) + "\n\n")
		b.WriteString("  Force delete?\n\n")
		no := "  No  "
		yes := "  Yes  "
		if m.forceChoice == 0 {
			no = selectedStyle.Render("▸ No  ")
			yes = dimStyle.Render("  Yes  ")
		} else {
			no = dimStyle.Render("  No  ")
			yes = deleteStyle.Render("▸ Yes  ")
		}
		b.WriteString("  " + no + yes + "\n\n")
		b.WriteString(dimStyle.Render("←/→ choose • enter confirm • esc cancel"))
		b.WriteString("\n")
		return b.String()
	}

	// Error message from failed delete
	if m.deleteErr != "" {
		b.WriteString(dangerStyle.Render("  error: "+m.deleteErr) + "\n\n")
	}

	styles := tuiStyles()

	// Column header row
	b.WriteString(columns.RenderHeader(m.layout, styles))
	b.WriteString("\n")

	// Worktree rows
	for i, idx := range m.filtered {
		row := m.rows[idx]
		isCursor := i == m.cursor
		isPrev := idx == m.previousIdx
		b.WriteString(columns.RenderRow(row, m.layout, styles, isCursor, m.deleteMode, isPrev))
		b.WriteString("\n")
	}

	// "Create new" row (not shown in delete mode)
	if !m.deleteMode {
		b.WriteString("\n")
		if m.onCreateNew() {
			b.WriteString(selectedStyle.Render(" + Create new worktree"))
		} else {
			b.WriteString(dimStyle.Render("  + Create new worktree"))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.deleteMode {
		b.WriteString(dangerStyle.Render("d exit delete mode • enter delete • esc cancel"))
	} else {
		b.WriteString(dimStyle.Render("↑/↓ navigate • enter select • type to filter • d delete • esc quit"))
	}
	b.WriteString("\n")

	// Footer summary line
	b.WriteString(dimStyle.Render(m.footer))
	b.WriteString("\n")

	return b.String()
}

// RunTUI launches the interactive worktree picker.
// rows must be the parallel rich-data slice produced by columns.Collect
// for the same worktrees slice (same length, same order).
// Returns the selected path, or empty string if cancelled.
func RunTUI(worktrees []Worktree, rows []columns.Row, previousIdx int) (string, error) {
	m := newModel(worktrees, rows, previousIdx)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("TUI error: %w", err)
	}
	final, ok := result.(model)
	if !ok {
		return "", nil
	}
	return final.selected, nil
}
