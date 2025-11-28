package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/detect"
	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/ui"
)

// Symbol constants for consistent output formatting across commands
const (
	symbolSuccess = "✓"
	symbolError   = "✗"
	symbolWarning = "⚠"
	symbolArrow   = "→"
)

// Symbol styles for colored output (lipgloss handles NO_COLOR automatically)
var (
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // Green
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // Red
	styleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // Yellow
	styleArrow   = lipgloss.NewStyle().Foreground(lipgloss.Color("4")) // Blue
)

// Colored symbol functions return symbols with appropriate colors
func coloredSuccess() string { return styleSuccess.Render(symbolSuccess) }
func coloredError() string   { return styleError.Render(symbolError) }
func coloredWarning() string { return styleWarning.Render(symbolWarning) }
func coloredArrow() string   { return styleArrow.Render(symbolArrow) }

// Dependencies holds all the dependencies for commands
type Dependencies struct {
	Git    git.Interface
	UI     ui.Interface
	Detect detect.Interface
	Stdout io.Writer
	Stderr io.Writer
}

// DefaultDependencies returns the default dependencies
func DefaultDependencies() *Dependencies {
	return &Dependencies{
		Git:    git.NewDefaultClient(),
		UI:     ui.NewDefaultUI(),
		Detect: detect.NewDefaultDetector(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// handleEnvFiles is a common function for handling environment files
// Priority order:
// 1. If --copy-envs flag is set, always copy
// 2. If config.CopyEnvs is set (true/false), use that value (unless flag overrides)
// 3. If neither is set, prompt user (interactive mode)
func handleEnvFiles(deps *Dependencies, cfg *config.Config, copyEnvsFlag bool, originalDir, worktreePath string) error {
	envFiles, err := deps.Git.FindUntrackedEnvFiles(originalDir)
	if err != nil {
		return fmt.Errorf("failed to find env files: %w", err)
	}

	if len(envFiles) == 0 {
		return nil
	}

	// Prepare file list
	filePaths := make([]string, len(envFiles))
	for i, f := range envFiles {
		filePaths[i] = f.Path
	}

	// Determine whether to copy based on priority
	var shouldCopy bool
	var needsPrompt bool

	if copyEnvsFlag {
		// Priority 1: Flag is set, always copy
		shouldCopy = true
		needsPrompt = false
	} else if cfg != nil && cfg.CopyEnvs != nil {
		// Priority 2: Config is set, use config value
		shouldCopy = *cfg.CopyEnvs
		needsPrompt = false
	} else {
		// Priority 3: Neither flag nor config is set, prompt user
		needsPrompt = true
	}

	if needsPrompt {
		fmt.Fprintf(deps.Stdout, "\nFound %d untracked environment file(s):\n", len(envFiles))
		deps.UI.ShowEnvFilesList(filePaths)

		fmt.Fprintf(deps.Stdout, "\nCopy them to the new worktree?")
		confirmed, err := deps.UI.ConfirmPrompt("")
		if err != nil {
			return fmt.Errorf("failed to get user input: %w", err)
		}
		shouldCopy = confirmed
	} else if shouldCopy {
		// When copy decision is made without prompting, show the files being copied
		fmt.Fprintf(deps.Stdout, "\nCopying environment files:\n")
		deps.UI.ShowEnvFilesList(filePaths)
	}

	if shouldCopy {
		// Copy files
		if err := deps.Git.CopyEnvFiles(envFiles, originalDir, worktreePath); err != nil {
			return fmt.Errorf("failed to copy env files: %w", err)
		}
		fmt.Fprintf(deps.Stdout, "%s Environment files copied successfully\n", coloredSuccess())
	}

	return nil
}
