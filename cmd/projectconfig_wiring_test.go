package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/git"
)

// These tests confirm that EndCommand/CleanCommand pass the right
// noProjectHooks value into ResolveProjectConfig — specifically that --force
// (and, for clean, --dry-run) skip project hook resolution even when the
// --no-project-hooks flag itself was not passed. ResolveProjectConfig's own
// behavior is covered exhaustively in projectconfig_test.go; these just
// verify the OR-wiring at the command layer.

func TestEndCommand_Execute_ForceSkipsProjectHooks(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "pre_end_hook = untrusted-project-hook\n")
	home := t.TempDir()
	t.Setenv("HOME", home)

	worktreeDir := t.TempDir()
	mockGitInstance := &mockGit{
		isGitRepo:               true,
		GetMainRepositoryRootFn: func() (string, error) { return mainRoot, nil },
		GetWorktreeForIssueFn: func(string) (*git.WorktreeInfo, error) {
			return &git.WorktreeInfo{Path: worktreeDir, Branch: testBranch123}, nil
		},
	}

	global := config.New()
	global.PreEndHook = "global-pre-end"
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{},
		Detect: &mockDetect{},
		Config: global,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	// force=true, noProjectHooks ctor flag left false — force alone must
	// still skip the project override.
	cmd := NewEndCommand(deps, true, true, false)
	if err := cmd.Execute("123"); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if deps.Config.PreEndHook != "global-pre-end" {
		t.Errorf("expected --force to skip the untrusted project hook override, got %q", deps.Config.PreEndHook)
	}
}

func TestCleanCommand_Execute_ForceSkipsProjectHooks(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "pre_end_hook = untrusted-project-hook\n")
	home := t.TempDir()
	t.Setenv("HOME", home)

	mockGitInstance := &mockGit{
		isGitRepo:               true,
		GetMainRepositoryRootFn: func() (string, error) { return mainRoot, nil },
		ListWorktreesFn:         func() ([]git.WorktreeInfo, error) { return nil, nil },
	}

	global := config.New()
	global.PreEndHook = "global-pre-end"
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{},
		Config: global,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	cmd := NewCleanCommand(deps, true, false, true, false) // force=true, noFetch=true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if deps.Config.PreEndHook != "global-pre-end" {
		t.Errorf("expected --force to skip the untrusted project hook override, got %q", deps.Config.PreEndHook)
	}
}

func TestCleanCommand_Execute_DryRunSkipsProjectHooks(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "pre_end_hook = untrusted-project-hook\n")
	home := t.TempDir()
	t.Setenv("HOME", home)

	mockGitInstance := &mockGit{
		isGitRepo:               true,
		GetMainRepositoryRootFn: func() (string, error) { return mainRoot, nil },
		ListWorktreesFn:         func() ([]git.WorktreeInfo, error) { return nil, nil },
	}

	global := config.New()
	global.PreEndHook = "global-pre-end"
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{},
		Config: global,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	cmd := NewCleanCommand(deps, false, true, true, false) // dryRun=true, noFetch=true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if deps.Config.PreEndHook != "global-pre-end" {
		t.Errorf("expected --dry-run to skip the untrusted project hook override, got %q", deps.Config.PreEndHook)
	}
}

func TestShellIntegration_DoesNotCallResolveProjectConfig(t *testing.T) {
	// gw shell-integration --print-path must never trigger project config
	// resolution (and thus never a trust prompt): its stdout is parsed
	// mechanically by shell integration and must contain only the path.
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook = untrusted-project-hook\n")
	home := t.TempDir()
	t.Setenv("HOME", home)

	worktreeDir := filepath.Join(mainRoot, "..", filepath.Base(mainRoot)+"-123")
	if err := os.MkdirAll(worktreeDir, 0755); err != nil {
		t.Fatalf("failed to create worktree dir: %v", err)
	}

	mockGitInstance := &mockGit{
		isGitRepo:               true,
		GetMainRepositoryRootFn: func() (string, error) { return mainRoot, nil },
		GetOriginalRepositoryNameFn: func() (string, error) {
			return filepath.Base(mainRoot), nil
		},
		GetRepositoryRootFn: func() (string, error) { return mainRoot, nil },
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	shellCmd := NewShellIntegrationCommand(mockGitInstance, stdout, stderr)
	shellCmd.printPath = "123"

	// Not checking the error: what matters is that stdout stays clean and no
	// trust-related output ever appears, regardless of whether the worktree
	// lookup itself succeeds.
	_ = shellCmd.Execute()

	if contains(stdout.String(), "Untrusted") || contains(stderr.String(), "Untrusted") {
		t.Error("expected shell-integration to never evaluate project config trust")
	}
}
