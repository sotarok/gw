package ui

import (
	"fmt"
	"gw/internal/git"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)
	normalStyle = lipgloss.NewStyle()
	dimStyle    = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
	currentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))
)

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Quit   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type worktreeSelector struct {
	worktrees []git.WorktreeInfo
	cursor    int
	selected  *git.WorktreeInfo
	err       error
}

func (m worktreeSelector) Init() tea.Cmd {
	return nil
}

func (m worktreeSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.worktrees)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Select):
			m.selected = &m.worktrees[m.cursor]
			return m, tea.Quit
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m worktreeSelector) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if len(m.worktrees) == 0 {
		return "No worktrees found.\n"
	}

	s := strings.Builder{}
	s.WriteString("Select a worktree:\n\n")

	for i, wt := range m.worktrees {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		line := fmt.Sprintf("%s%s", cursor, wt.Path)

		if wt.Branch != "" {
			line += fmt.Sprintf(" (%s)", wt.Branch)
		}

		style := normalStyle
		if m.cursor == i {
			style = selectedStyle
		}
		if wt.IsCurrent {
			line += " [current]"
			if m.cursor != i {
				style = currentStyle
			}
		}

		s.WriteString(style.Render(line) + "\n")
	}

	s.WriteString("\n")
	s.WriteString(dimStyle.Render("↑/k: up • ↓/j: down • enter: select • q: quit"))

	return s.String()
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// SelectWorktree shows an interactive UI to select a worktree
func SelectWorktree() (*git.WorktreeInfo, error) {
	worktrees, err := git.ListWorktrees()
	if err != nil {
		return nil, err
	}

	if len(worktrees) == 0 {
		return nil, fmt.Errorf("no worktrees found")
	}

	// Filter out the main worktree (usually the first one)
	var filteredWorktrees []git.WorktreeInfo
	for _, wt := range worktrees {
		// Skip if it's the main worktree (check for issue number pattern)
		// Accept branches like "123/impl", "456/fix", etc.
		parts := strings.Split(wt.Branch, "/")
		if len(parts) >= 2 && isNumeric(parts[0]) {
			filteredWorktrees = append(filteredWorktrees, wt)
		}
	}

	if len(filteredWorktrees) == 0 {
		return nil, fmt.Errorf("no issue worktrees found")
	}

	m := worktreeSelector{
		worktrees: filteredWorktrees,
		cursor:    0,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	model := result.(worktreeSelector)
	if model.selected == nil {
		return nil, fmt.Errorf("no worktree selected")
	}

	return model.selected, nil
}
