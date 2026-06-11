package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/detect"
	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/hook"
	"github.com/sotarok/gw/internal/spinner"
	"github.com/sotarok/gw/internal/ui"
)

// defaultBaseBranch is the base branch used when none is supplied (start) and
// the branch treated as the integration target by the safety checks and the
// clean command's protection list.
const defaultBaseBranch = "main"

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
	Config *config.Config // always non-nil
	Stdout io.Writer
	Stderr io.Writer
}

// DefaultDependencies returns the default dependencies.
//
// Config is loaded once here. A load failure falls back to defaults with a
// warning on stderr (previously the failure was swallowed silently), so
// Config is guaranteed to be non-nil.
func DefaultDependencies() *Dependencies {
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Could not load ~/.gwrc, using defaults: %v\n", symbolWarning, err)
		cfg = config.New()
	}
	return &Dependencies{
		Git:    git.NewClient(),
		UI:     ui.NewDefaultUI(),
		Detect: detect.NewDefaultDetector(),
		Config: cfg,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// runPreEndHook runs pre_end_hook with cwd set to worktreePath, then restores
// the original directory regardless of hook outcome. Hook failures are
// reported as warnings on stderr; commandLabel ("end" or "clean") flows into
// GW_COMMAND for the hook process.
func runPreEndHook(deps *Dependencies, hookCmd, worktreePath, branchName, repoName, commandLabel string) {
	originalDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(deps.Stderr, "%s Could not capture cwd for pre-end hook: %v\n", coloredWarning(), err)
		return
	}
	if err := os.Chdir(worktreePath); err != nil {
		fmt.Fprintf(deps.Stderr, "%s Could not enter %s to run pre-end hook: %v\n", coloredWarning(), worktreePath, err)
		return
	}
	defer func() { _ = os.Chdir(originalDir) }()

	absWorktreePath, _ := filepath.Abs(worktreePath)
	hookEnv := hook.Env{
		WorktreePath: absWorktreePath,
		BranchName:   branchName,
		RepoName:     repoName,
		Command:      commandLabel,
	}
	if err := hook.Execute(hookCmd, hookEnv, deps.Stdout, deps.Stderr); err != nil {
		fmt.Fprintf(deps.Stderr, "%s Pre-end hook failed for %s: %v\n", coloredWarning(), filepath.Base(worktreePath), err)
	}
}

// fetchIfConfigured runs git fetch --all --prune if configured and not skipped
func fetchIfConfigured(deps *Dependencies, noFetch bool) {
	if noFetch || !deps.Config.FetchBeforeCommand {
		return
	}
	sp := spinner.New("Fetching from remotes...", deps.Stdout)
	sp.Start()
	err := deps.Git.FetchAll()
	sp.Stop()
	if err != nil {
		fmt.Fprintf(deps.Stderr, "%s Could not fetch from remotes: %v\n", coloredWarning(), err)
	}
}

// handleEnvFiles is a common function for handling environment files
// Priority order:
// 1. If --copy-envs flag is set, always copy
// 2. If config.CopyEnvs is set (true/false), use that value (unless flag overrides)
// 3. If neither is set, prompt user (interactive mode)
func handleEnvFiles(deps *Dependencies, copyEnvsFlag bool, originalDir, worktreePath string) error {
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
	} else if deps.Config.CopyEnvs != nil {
		// Priority 2: Config is set, use config value
		shouldCopy = *deps.Config.CopyEnvs
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
