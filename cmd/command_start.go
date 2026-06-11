package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/hook"
	"github.com/sotarok/gw/internal/iterm2"
	"github.com/sotarok/gw/internal/spinner"
)

// startGit is the subset of git operations StartCommand actually uses.
type startGit interface {
	git.RepositoryReader // IsGitRepository, GetOriginalRepositoryName, GetRepositoryRoot, FetchAll
	git.WorktreeManager  // GetWorktreeForIssue, CreateWorktree
	git.EnvFileHandler   // FindUntrackedEnvFiles, CopyEnvFiles (via handleEnvFiles)
}

// StartCommand handles the start command logic
type StartCommand struct {
	deps     *Dependencies
	copyEnvs bool
	noFetch  bool
}

// NewStartCommand creates a new start command handler
func NewStartCommand(deps *Dependencies, copyEnvs, noFetch bool) *StartCommand {
	return &StartCommand{
		deps:     deps,
		copyEnvs: copyEnvs,
		noFetch:  noFetch,
	}
}

// git returns the command's git dependency narrowed to the operations it uses.
func (c *StartCommand) git() startGit { return c.deps.Git }

// Execute runs the start command
func (c *StartCommand) Execute(issueNumber, baseBranch string) error {
	repoName, envSourceRoot, err := c.resolveTarget(issueNumber)
	if err != nil {
		return err
	}

	worktreePath, err := c.createWorktree(issueNumber, baseBranch)
	if err != nil {
		return err
	}

	c.postCreate(issueNumber, worktreePath, repoName, envSourceRoot)
	return nil
}

// resolveTarget validates the repository, ensures no worktree already exists for
// the issue, updates the iTerm2 tab, and resolves the repository root used as the
// env-file source. It returns the original repository name and that env source root.
func (c *StartCommand) resolveTarget(issueNumber string) (repoName, envSourceRoot string, err error) {
	g := c.git()

	// Check if we're in a git repository
	if !g.IsGitRepository() {
		return "", "", fmt.Errorf("not in a git repository")
	}

	// Fetch from remotes if configured
	fetchIfConfigured(c.deps, c.noFetch)

	// Check if worktree already exists
	if wt, _ := g.GetWorktreeForIssue(issueNumber); wt != nil {
		return "", "", fmt.Errorf("worktree for issue %s already exists at %s", issueNumber, wt.Path)
	}

	// Get the original repository name for the iTerm2 tab so that, when run from
	// inside a worktree, the tab shows the repo name rather than the worktree dir.
	repoName, _ = g.GetOriginalRepositoryName()

	// Update iTerm2 tab if configured
	if iterm2.ShouldUpdateTab(c.deps.Config.UpdateITerm2Tab) {
		_ = iterm2.UpdateTabName(c.deps.Stdout, repoName, issueNumber)
	}

	// Anchor env file lookup/copy to the repository root so that running start
	// from a sub directory still scans the whole repo (rather than just the
	// sub directory) and preserves the relative paths of copied files.
	envSourceRoot, err = g.GetRepositoryRoot()
	if err != nil {
		return "", "", fmt.Errorf("failed to get repository root: %w", err)
	}

	return repoName, envSourceRoot, nil
}

// createWorktree creates the worktree for the issue and reports the resulting path.
func (c *StartCommand) createWorktree(issueNumber, baseBranch string) (string, error) {
	sp := spinner.New(fmt.Sprintf("Creating worktree for issue #%s based on %s...", issueNumber, baseBranch), c.deps.Stdout)
	sp.Start()
	worktreePath, err := c.git().CreateWorktree(issueNumber, baseBranch)
	sp.Stop()
	if err != nil {
		return "", err
	}

	if c.deps.Stdout != nil {
		fmt.Fprintf(c.deps.Stdout, "%s Created worktree at %s\n", coloredSuccess(), worktreePath)
	}
	return worktreePath, nil
}

// postCreate performs the post-creation steps: optional auto-cd, env file copy,
// package manager setup, the post-start hook, and the completion message.
func (c *StartCommand) postCreate(issueNumber, worktreePath, repoName, envSourceRoot string) {
	// Change to the new worktree directory for setup operations
	// Note: This only affects the current process, not the parent shell
	if c.deps.Config.AutoCD {
		if err := os.Chdir(worktreePath); err != nil {
			// Don't fail the command, just log the error
			if c.deps.Stderr != nil {
				fmt.Fprintf(c.deps.Stderr, "%s Could not change to worktree directory: %v\n", coloredWarning(), err)
			}
		}
	}

	// Handle environment files
	if err := c.handleEnvFiles(envSourceRoot, worktreePath); err != nil {
		// Don't fail the command, just warn
		if c.deps.Stderr != nil {
			fmt.Fprintf(c.deps.Stderr, "%s Failed to handle env files: %v\n", coloredWarning(), err)
		}
	}

	// Run setup if a package manager is detected
	if err := c.deps.Detect.RunSetup(worktreePath); err != nil {
		// Don't fail if setup fails, just warn
		if c.deps.Stderr != nil {
			fmt.Fprintf(c.deps.Stderr, "%s Setup failed: %v\n", coloredWarning(), err)
		}
	}

	// Execute post-start hook if configured
	if c.deps.Config.PostStartHook != "" {
		branchName := issueNumber + "/impl"
		absWorktreePath, _ := filepath.Abs(worktreePath)
		hookEnv := hook.Env{
			WorktreePath: absWorktreePath,
			BranchName:   branchName,
			RepoName:     repoName,
			Command:      "start",
		}
		if err := hook.Execute(c.deps.Config.PostStartHook, hookEnv, c.deps.Stdout, c.deps.Stderr); err != nil {
			fmt.Fprintf(c.deps.Stderr, "%s Post-start hook failed: %v\n", coloredWarning(), err)
		}
	}

	if c.deps.Stdout != nil {
		fmt.Fprintf(c.deps.Stdout, "\n✨ Worktree ready at:\n   %s\n", worktreePath)
		if c.deps.Config.AutoCD {
			fmt.Fprintf(c.deps.Stdout, "\n💡 Shell integration will change to this directory after the command completes.\n")
		}
	}
}

func (c *StartCommand) handleEnvFiles(originalDir, worktreePath string) error {
	return handleEnvFiles(c.deps, c.copyEnvs, originalDir, worktreePath)
}
