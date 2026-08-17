package cmd

import "fmt"

// envSource is the resolved origin of the untracked env files copied into a new
// worktree.
type envSource struct {
	Root string
	// FallbackFrom names the base branch whose worktree could not be found. It
	// is empty when Root came from that worktree, and drives the warning in
	// handleEnvFiles.
	FallbackFrom string
}

// envSourceGit is the subset of git operations env source resolution needs.
type envSourceGit interface {
	GetMainRepositoryRoot() (string, error)
	GetWorktreeRootForBranch(branch string) (string, error)
}

// resolveEnvSource picks the directory whose untracked env files are copied into
// a new worktree: the worktree that currently has baseBranch checked out, so a
// stacked branch (main → A → B) inherits the env edits made in A rather than
// only what main has. It falls back to the main repository root when no worktree
// holds baseBranch — which is the whole story for `gw checkout`, where the new
// branch's parent is unknown and baseBranch is empty.
func resolveEnvSource(g envSourceGit, baseBranch string) (envSource, error) {
	if baseBranch != "" {
		if root, err := g.GetWorktreeRootForBranch(baseBranch); err == nil {
			return envSource{Root: root}, nil
		}
	}

	root, err := g.GetMainRepositoryRoot()
	if err != nil {
		return envSource{}, fmt.Errorf("failed to get repository root: %w", err)
	}
	return envSource{Root: root, FallbackFrom: baseBranch}, nil
}
