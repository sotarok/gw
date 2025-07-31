package ui

import "github.com/sotarok/gw/internal/git"

// Interface defines the UI operations used by the application
type Interface interface {
	// Selection operations
	SelectWorktree() (*git.WorktreeInfo, error)
	ShowSelector(title string, items []SelectorItem) (*SelectorItem, error)

	// Prompt operations
	ConfirmPrompt(message string) (bool, error)

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

func (u *DefaultUI) SelectWorktree() (*git.WorktreeInfo, error) {
	return SelectWorktree()
}

func (u *DefaultUI) ShowSelector(title string, items []SelectorItem) (*SelectorItem, error) {
	return ShowSelector(title, items)
}

func (u *DefaultUI) ConfirmPrompt(message string) (bool, error) {
	return ConfirmPrompt(message)
}

func (u *DefaultUI) ShowEnvFilesList(files []string) {
	ShowEnvFilesList(files)
}
