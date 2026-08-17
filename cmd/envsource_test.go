package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/git"
)

// envSourceFixture lays out a main repo, the worktree the command is invoked
// from, and the worktree being created, and wires a mockGit that records which
// root FindUntrackedEnvFiles was called with.
type envSourceFixture struct {
	mainRoot        string
	currentWorktree string
	newWorktree     string
	git             *mockGit
	stdout          *bytes.Buffer
	stderr          *bytes.Buffer
	deps            *Dependencies
	scanRoot        string
}

func newEnvSourceFixture(t *testing.T, envFileNames ...string) *envSourceFixture {
	t.Helper()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	tempDir := t.TempDir()
	f := &envSourceFixture{
		mainRoot:        filepath.Join(tempDir, "repo"),
		currentWorktree: filepath.Join(tempDir, "repo-999"),
		newWorktree:     filepath.Join(tempDir, "repo-123"),
	}
	for _, dir := range []string{f.mainRoot, f.currentWorktree, f.newWorktree} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}
	if err := os.Chdir(f.currentWorktree); err != nil {
		t.Fatalf("failed to chdir into worktree: %v", err)
	}

	envFiles := make([]git.EnvFile, 0, len(envFileNames))
	for _, name := range envFileNames {
		if err := os.WriteFile(filepath.Join(f.mainRoot, name), []byte("K=V\n"), 0600); err != nil {
			t.Fatalf("failed to write env file: %v", err)
		}
		envFiles = append(envFiles, git.EnvFile{Path: name})
	}

	f.git = &mockGit{isGitRepo: true, worktreePath: f.newWorktree}
	f.git.GetOriginalRepositoryNameFn = func() (string, error) { return testRepoNameShort, nil }
	f.git.GetRepositoryRootFn = func() (string, error) { return f.currentWorktree, nil }
	f.git.GetMainRepositoryRootFn = func() (string, error) { return f.mainRoot, nil }
	f.git.BranchExistsFn = func(string) (bool, error) { return true, nil }
	f.git.FindUntrackedEnvFilesFn = func(root string) ([]git.EnvFile, error) {
		f.scanRoot = root
		// Env files live in whichever root is scanned, so mirror them there.
		for _, e := range envFiles {
			if err := os.WriteFile(filepath.Join(root, e.Path), []byte("K=V\n"), 0600); err != nil {
				return nil, err
			}
		}
		return envFiles, nil
	}

	f.stdout = &bytes.Buffer{}
	f.stderr = &bytes.Buffer{}
	f.deps = &Dependencies{
		Git:    f.git,
		UI:     &mockUI{},
		Detect: &mockDetect{},
		Config: &config.Config{},
		Stdout: f.stdout,
		Stderr: f.stderr,
	}
	return f
}

func TestStartCommand_EnvSource_UsesBaseBranchWorktree(t *testing.T) {
	f := newEnvSourceFixture(t, ".env")

	baseWorktree := filepath.Join(filepath.Dir(f.mainRoot), "repo-a")
	if err := os.MkdirAll(baseWorktree, 0755); err != nil {
		t.Fatalf("failed to create base worktree: %v", err)
	}
	var requestedBranch string
	f.git.GetWorktreeRootForBranchFn = func(branch string) (string, error) {
		requestedBranch = branch
		return baseWorktree, nil
	}

	cmd := NewStartCommand(f.deps, true, true, false)
	if err := cmd.Execute("b", "a/impl"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestedBranch != "a/impl" {
		t.Errorf("expected the base branch to be looked up, got %q", requestedBranch)
	}
	if f.scanRoot != baseWorktree {
		t.Errorf("env scan root\n  got:  %s\n  want: %s", f.scanRoot, baseWorktree)
	}
}

func TestStartCommand_EnvSource_FallsBackToMainRootWithWarning(t *testing.T) {
	f := newEnvSourceFixture(t, ".env")
	// Default mockGit behavior: no worktree has the base branch checked out.

	cmd := NewStartCommand(f.deps, true, true, false)
	if err := cmd.Execute("123", "main"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.scanRoot != f.mainRoot {
		t.Errorf("env scan root\n  got:  %s\n  want: %s", f.scanRoot, f.mainRoot)
	}
	warning := f.stderr.String()
	if !strings.Contains(warning, "main") || !strings.Contains(warning, f.mainRoot) {
		t.Errorf("expected a fallback warning naming the base branch and the main repo root, got: %q", warning)
	}
}

func TestStartCommand_EnvSource_StaysQuietWhenFallbackFindsNoEnvFiles(t *testing.T) {
	f := newEnvSourceFixture(t)

	cmd := NewStartCommand(f.deps, true, true, false)
	if err := cmd.Execute("123", "main"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if warning := f.stderr.String(); warning != "" {
		t.Errorf("there is nothing to copy, so the fallback must not warn, got: %q", warning)
	}
}

func TestCheckoutCommand_EnvSource_UsesMainRootFromInsideWorktree(t *testing.T) {
	f := newEnvSourceFixture(t, ".env")
	f.git.CreateWorktreeFromBranchFn = func(worktreePath, _, _ string) error {
		return os.MkdirAll(worktreePath, 0755)
	}

	cmd := NewCheckoutCommand(f.deps, true, true, false)
	if err := cmd.Execute(testBranchFeature); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.scanRoot != f.mainRoot {
		t.Errorf("env scan root\n  got:  %s\n  want: %s", f.scanRoot, f.mainRoot)
	}
	if warning := f.stderr.String(); warning != "" {
		t.Errorf("checkout has no base branch, so it must not warn about a fallback, got: %q", warning)
	}
}
