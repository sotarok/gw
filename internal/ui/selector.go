package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sotarok/gw/internal/git"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectorItem represents an item in the selector
type SelectorItem struct {
	ID   string
	Name string
}

const (
	mainBranch   = "main"
	masterBranch = "master"
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

	// Filter out the main worktree and master/main branches
	filteredWorktrees := make([]git.WorktreeInfo, 0, len(worktrees))
	for _, wt := range worktrees {
		// Skip main/master branches and current worktree if it's main/master
		if wt.Branch == mainBranch || wt.Branch == masterBranch {
			continue
		}
		// Include all other worktrees
		filteredWorktrees = append(filteredWorktrees, wt)
	}

	if len(filteredWorktrees) == 0 {
		return nil, fmt.Errorf("no worktrees found (excluding main/master)")
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

// ShowSelector displays a generic selector with the given items
func ShowSelector(title string, items []SelectorItem) (*SelectorItem, error) {
	m := &genericSelector{
		items:  items,
		title:  title,
		cursor: 0,
		keyMap: keys,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	model := result.(*genericSelector)
	if model.selected == nil {
		return nil, fmt.Errorf("no item selected")
	}

	return model.selected, nil
}

// genericSelector is a generic selector model
type genericSelector struct {
	items    []SelectorItem
	cursor   int
	selected *SelectorItem
	title    string
	keyMap   keyMap
}

func (m *genericSelector) Init() tea.Cmd {
	return nil
}

func (m *genericSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keyMap.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keyMap.Select):
			m.selected = &m.items[m.cursor]
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *genericSelector) View() string {
	s := fmt.Sprintf("%s\n\n", m.title)

	for i, item := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		line := fmt.Sprintf("%s %s", cursor, item.Name)
		if m.cursor == i {
			s += selectedStyle.Render(line) + "\n"
		} else {
			s += normalStyle.Render(line) + "\n"
		}
	}

	s += "\n" + dimStyle.Render("↑/k: up • ↓/j: down • enter: select • q: quit")
	return s
}
