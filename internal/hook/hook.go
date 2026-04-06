package hook

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Env holds environment variables passed to hook commands
type Env struct {
	WorktreePath string
	BranchName   string
	RepoName     string
	Command      string // "start" or "checkout"
}

// Execute runs a hook command with the given environment variables.
// If hookCmd is empty, it does nothing and returns nil.
// The hook command is executed via "sh -c" with GW_* environment variables.
func Execute(hookCmd string, env Env, stdout, stderr io.Writer) error {
	if hookCmd == "" {
		return nil
	}

	cmd := exec.Command("sh", "-c", hookCmd)
	cmd.Env = append(os.Environ(), envToSlice(env)...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook command failed: %w", err)
	}

	return nil
}

func envToSlice(env Env) []string {
	return []string{
		"GW_WORKTREE_PATH=" + env.WorktreePath,
		"GW_BRANCH_NAME=" + env.BranchName,
		"GW_REPO_NAME=" + env.RepoName,
		"GW_COMMAND=" + env.Command,
	}
}
