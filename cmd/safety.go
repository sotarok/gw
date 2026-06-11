package cmd

import (
	"errors"
	"sync"

	"github.com/sotarok/gw/internal/git"
)

// safetyCheck is the outcome of a single pre-removal check for one worktree.
type safetyCheck struct {
	// Tripped is true when the check's condition fired: uncommitted changes
	// present, unpushed commits present, or the branch is not merged.
	Tripped bool
	// Err is non-nil when the check itself could not be evaluated.
	Err error
}

// safetyResult is the outcome of the three pre-removal checks for one worktree.
// Each field is reported verbatim; callers format the warnings in their own
// wording (end and clean intentionally differ).
type safetyResult struct {
	Uncommitted safetyCheck
	Unpushed    safetyCheck
	Merged      safetyCheck // Tripped means "not merged to baseBranch"
	// InvalidRepo is true when the worktree is broken or missing (git exits
	// 128). When set, the remaining checks were not meaningful and callers
	// should surface a single "invalid git repository" reason.
	InvalidRepo bool
}

// numSafetyChecks is the number of parallel goroutines started by runSafetyChecks
// (uncommitted, unpushed, merged). Centralized so wg.Add stays in sync with
// the actual goroutine count.
const numSafetyChecks = 3

// runSafetyChecks runs the three pre-removal checks in parallel against the
// worktree at worktreePath using the StatusChecker.
//
// If the uncommitted-changes check fails with a git exit code 128 (a broken or
// missing worktree), InvalidRepo is set; the other checks may still run but
// their results are not meaningful and callers ignore them.
func runSafetyChecks(g git.StatusChecker, worktreePath, branch, baseBranch string) safetyResult {
	var result safetyResult

	var wg sync.WaitGroup
	wg.Add(numSafetyChecks)
	go func() {
		defer wg.Done()
		hasChanges, err := g.HasUncommittedChanges(worktreePath)
		if err != nil {
			result.Uncommitted.Err = err
			return
		}
		result.Uncommitted.Tripped = hasChanges
	}()
	go func() {
		defer wg.Done()
		hasUnpushed, err := g.HasUnpushedCommits(worktreePath, branch)
		if err != nil {
			result.Unpushed.Err = err
			return
		}
		result.Unpushed.Tripped = hasUnpushed
	}()
	go func() {
		defer wg.Done()
		isMerged, err := g.IsMergedToBaseBranch(worktreePath, branch, baseBranch)
		if err != nil {
			result.Merged.Err = err
			return
		}
		result.Merged.Tripped = !isMerged
	}()
	wg.Wait()

	// A broken or missing worktree surfaces as git exit 128 on the first
	// check. Flag it so callers can report a single clear reason instead of
	// three meaningless ones.
	if isInvalidRepoErr(result.Uncommitted.Err) {
		result.InvalidRepo = true
	}

	return result
}

// isInvalidRepoErr reports whether err is a git failure indicating the worktree
// is not a usable git repository (exit code 128), e.g. its directory was
// deleted out from under it.
func isInvalidRepoErr(err error) bool {
	var gitErr *git.GitError
	return errors.As(err, &gitErr) && gitErr.ExitCode == 128
}
