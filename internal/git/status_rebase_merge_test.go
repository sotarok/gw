package git

import (
	"os"
	"path/filepath"
	"testing"
)

// setupRebaseMergeRepo builds a repo where "feature" has two commits and main
// has advanced by one unrelated commit since feature branched off. With
// withRemote, main is pushed to a bare origin before feature diverges.
func setupRebaseMergeRepo(t *testing.T, withRemote bool) string {
	t.Helper()

	dir, cleanup := createTestRepo(t)
	t.Cleanup(cleanup)

	commitFile := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
		runGitCommand(t, dir, "add", name)
		runGitCommand(t, dir, "commit", "-m", "add "+name)
	}

	commitFile("base.txt", "base")
	runGitCommand(t, dir, "branch", "-M", "main")

	if withRemote {
		remoteDir := t.TempDir()
		runGitCommand(t, remoteDir, "init", "--bare")
		runGitCommand(t, dir, "remote", "add", "origin", remoteDir)
		runGitCommand(t, dir, "push", "-u", "origin", "main")
	}

	runGitCommand(t, dir, "checkout", "-b", "feature")
	commitFile("a.txt", "a")
	commitFile("b.txt", "b")

	runGitCommand(t, dir, "checkout", "main")
	commitFile("c.txt", "c")

	return dir
}

func TestIsMergedToBaseBranch_RebaseMerged(t *testing.T) {
	t.Run("returns true when all commits were rebased onto origin/main", func(t *testing.T) {
		dir := setupRebaseMergeRepo(t, true)
		runGitCommand(t, dir, "cherry-pick", "feature~1", "feature")
		runGitCommand(t, dir, "push", "origin", "main")
		// Local main must not short-circuit the check via --contains.
		runGitCommand(t, dir, "reset", "--hard", "origin/main~2")

		merged, err := IsMergedToBaseBranch(dir, "feature", "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !merged {
			t.Error("expected branch rebase-merged into origin/main to be considered merged")
		}
	})

	t.Run("returns true when all commits were rebased onto local main without a remote", func(t *testing.T) {
		dir := setupRebaseMergeRepo(t, false)
		runGitCommand(t, dir, "cherry-pick", "feature~1", "feature")

		merged, err := IsMergedToBaseBranch(dir, "feature", "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !merged {
			t.Error("expected branch rebase-merged into local main to be considered merged")
		}
	})

	t.Run("returns false when only some commits were rebased", func(t *testing.T) {
		dir := setupRebaseMergeRepo(t, true)
		runGitCommand(t, dir, "cherry-pick", "feature~1")
		runGitCommand(t, dir, "push", "origin", "main")

		merged, err := IsMergedToBaseBranch(dir, "feature", "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if merged {
			t.Error("expected partially rebased branch to be considered not merged")
		}
	})

	t.Run("unpushed check treats a rebase-merged branch without upstream as pushed", func(t *testing.T) {
		dir := setupRebaseMergeRepo(t, true)
		runGitCommand(t, dir, "cherry-pick", "feature~1", "feature")
		runGitCommand(t, dir, "push", "origin", "main")

		hasUnpushed, err := HasUnpushedCommits(dir, "feature")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hasUnpushed {
			t.Error("expected rebase-merged branch without upstream to have no unpushed commits")
		}
	})
}
