package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sotarok/gw/internal/git"
)

// Interface defines the UI operations used by the application
type Interface interface {
	// Selection operations
	SelectWorktree() (*git.WorktreeInfo, error)
	ShowSelector(title string, items []SelectorItem) (*SelectorItem, error)

	// Prompt operations
	ConfirmPrompt(message string) (bool, error)

	// TrustPrompt asks the user whether to trust and run the given project
	// hook values. It defaults to "no" (fail closed) and must never write to
	// stdout — see the DefaultUI implementation for why.
	TrustPrompt(projectPath string, hookLines []string) (bool, error)

	// Display operations
	ShowEnvFilesList(files []string)
}

// DefaultUI implements Interface using actual UI components
type DefaultUI struct{}

// Ensure DefaultUI implements Interface
var _ Interface = (*DefaultUI)(nil)

// NewDefaultUI creates a new default UI
func NewDefaultUI() *DefaultUI {
	return &DefaultUI{}
}

// SelectWorktree shows an interactive UI to select a worktree
func (u *DefaultUI) SelectWorktree() (*git.WorktreeInfo, error) {
	worktrees, err := git.NewClient().ListWorktrees()
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
func (u *DefaultUI) ShowSelector(title string, items []SelectorItem) (*SelectorItem, error) {
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

// ConfirmPrompt shows a yes/no prompt to the user
func (u *DefaultUI) ConfirmPrompt(message string) (bool, error) {
	m := confirmModel{
		message: message,
		cursor:  0, // Default to "yes"
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return false, err
	}

	model := result.(confirmModel)
	return model.confirmed, nil
}

// ShowEnvFilesList displays a list of environment files
func (u *DefaultUI) ShowEnvFilesList(files []string) {
	if len(files) == 0 {
		fmt.Println("No environment files found.")
		return
	}

	fmt.Println("\nThe following environment files will be copied:")
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("242"))

	for _, file := range files {
		fmt.Printf("  %s %s\n", headerStyle.Render("→"), fileStyle.Render(file))
	}
	fmt.Println()
}
