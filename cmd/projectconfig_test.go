package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/trust"
)

var errBoom = errors.New("boom")

// newProjectConfigTestDeps builds Dependencies wired to a real temp directory
// acting as the main repository root (so ResolveProjectConfig can read an
// actual .gwrc file from disk) and an isolated $HOME (so the trust store
// doesn't leak between tests or touch the real developer's ~/.gw/trust).
// It returns the Dependencies plus the stderr buffer for assertions.
func newProjectConfigTestDeps(t *testing.T, mainRoot string, globalCfg *config.Config, ui *mockUI) (*Dependencies, *bytes.Buffer) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git: &mockGit{
			isGitRepo:               true,
			GetMainRepositoryRootFn: func() (string, error) { return mainRoot, nil },
		},
		UI:     ui,
		Config: globalCfg,
		Stdout: stdout,
		Stderr: stderr,
	}
	return deps, stderr
}

func writeProjectConfig(t *testing.T, mainRoot, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(mainRoot, ".gwrc"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write project .gwrc: %v", err)
	}
}

func TestResolveProjectConfig_NoProjectFile(t *testing.T) {
	mainRoot := t.TempDir()
	global := config.New()
	global.PostStartHook = "global-start"
	deps, _ := newProjectConfigTestDeps(t, mainRoot, global, &mockUI{})

	if err := ResolveProjectConfig(deps, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.Config.PostStartHook != "global-start" {
		t.Errorf("expected global hook to remain untouched, got %q", deps.Config.PostStartHook)
	}
}

func TestResolveProjectConfig_NotInGitRepository(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	global := config.New()
	global.PostStartHook = "global-start"

	deps := &Dependencies{
		Git:    &mockGit{isGitRepo: false},
		UI:     &mockUI{},
		Config: global,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
	}

	if err := ResolveProjectConfig(deps, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.Config.PostStartHook != "global-start" {
		t.Error("expected global config untouched outside a git repository")
	}
}

func TestResolveProjectConfig_MainRootResolutionFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	global := config.New()
	global.PostStartHook = "global-start"
	stderr := &bytes.Buffer{}

	deps := &Dependencies{
		Git: &mockGit{
			isGitRepo:               true,
			GetMainRepositoryRootFn: func() (string, error) { return "", errBoom },
		},
		UI:     &mockUI{},
		Config: global,
		Stdout: &bytes.Buffer{},
		Stderr: stderr,
	}

	if err := ResolveProjectConfig(deps, false); err != nil {
		t.Fatalf("expected warn-and-continue, got error: %v", err)
	}
	if deps.Config.PostStartHook != "global-start" {
		t.Error("expected global config untouched when main root can't be resolved")
	}
	if stderr.String() == "" {
		t.Error("expected a warning on stderr when main root resolution fails")
	}
}

func TestResolveProjectConfig_TrustedNonEmptyOverrideApplies(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook = project-start\n")
	global := config.New()
	global.PostStartHook = "global-start"
	deps, _ := newProjectConfigTestDeps(t, mainRoot, global, &mockUI{})

	// Pre-approve the hash so no prompt is needed.
	content, err := os.ReadFile(filepath.Join(mainRoot, ".gwrc"))
	if err != nil {
		t.Fatalf("failed to read project config: %v", err)
	}
	absPath := filepath.Join(mainRoot, ".gwrc")
	hash := trust.Compute(absPath, content)
	if err := trust.Approve(hash); err != nil {
		t.Fatalf("failed to pre-approve: %v", err)
	}

	if err := ResolveProjectConfig(deps, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.Config.PostStartHook != "project-start" {
		t.Errorf("expected trusted project value to apply, got %q", deps.Config.PostStartHook)
	}
}

func TestResolveProjectConfig_EmptyOverrideDisablesGlobalWithoutPrompt(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook =\n")
	global := config.New()
	global.PostStartHook = "global-start"
	ui := &mockUI{}
	deps, _ := newProjectConfigTestDeps(t, mainRoot, global, ui)

	if err := ResolveProjectConfig(deps, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.Config.PostStartHook != "" {
		t.Errorf("expected empty override to disable the global hook, got %q", deps.Config.PostStartHook)
	}
	if ui.trustPromptCalled {
		t.Error("expected no trust prompt for an empty (code-free) override")
	}
}

func TestResolveProjectConfig_NoProjectHooksSkipsEmptyOverrideToo(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook =\n")
	global := config.New()
	global.PostStartHook = "global-start"
	ui := &mockUI{}
	deps, _ := newProjectConfigTestDeps(t, mainRoot, global, ui)

	if err := ResolveProjectConfig(deps, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.Config.PostStartHook != "global-start" {
		t.Errorf("expected --no-project-hooks to skip even an empty override, got %q", deps.Config.PostStartHook)
	}
	if ui.trustPromptCalled {
		t.Error("expected no trust prompt under --no-project-hooks")
	}
}

func TestResolveProjectConfig_NonHookKeyIgnoredWithStderrNote(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "auto_cd = false\n")
	global := config.New()
	deps, stderr := newProjectConfigTestDeps(t, mainRoot, global, &mockUI{})

	if err := ResolveProjectConfig(deps, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.Config.AutoCD != global.AutoCD {
		t.Error("expected non-hook key to not be applied")
	}
	if !contains(stderr.String(), "auto_cd") || !contains(stderr.String(), "ignored") {
		t.Errorf("expected stderr note about ignored non-hook key, got %q", stderr.String())
	}
}

func TestResolveProjectConfig_UntrustedNonEmptyOverrideFallsBackNonInteractive(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook = evil-command\n")
	global := config.New()
	global.PostStartHook = "global-start"
	ui := &mockUI{}
	deps, _ := newProjectConfigTestDeps(t, mainRoot, global, ui)

	if err := ResolveProjectConfig(deps, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.Config.PostStartHook != "global-start" {
		t.Errorf("expected fallback to global value for an unapproved hook in a non-interactive test process, got %q", deps.Config.PostStartHook)
	}
	if ui.trustPromptCalled {
		t.Error("expected no trust prompt when stdin is not a terminal (test process)")
	}
}

func TestResolveProjectConfig_MixedTrustedFallbackAndEmptyOverride(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook = evil-command\npre_end_hook =\n")
	global := config.New()
	global.PostStartHook = "global-start"
	global.PreEndHook = "global-pre-end"
	deps, _ := newProjectConfigTestDeps(t, mainRoot, global, &mockUI{})

	if err := ResolveProjectConfig(deps, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.Config.PostStartHook != "global-start" {
		t.Errorf("expected untrusted non-empty key to fall back to global, got %q", deps.Config.PostStartHook)
	}
	if deps.Config.PreEndHook != "" {
		t.Errorf("expected empty override to still disable pre_end_hook independent of the other key's trust state, got %q",
			deps.Config.PreEndHook)
	}
}

func TestResolveProjectConfig_LoadErrorWarnsAndContinues(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook = pnpm dev\n")
	if err := os.Chmod(filepath.Join(mainRoot, ".gwrc"), 0000); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	defer os.Chmod(filepath.Join(mainRoot, ".gwrc"), 0644)

	global := config.New()
	global.PostStartHook = "global-start"
	deps, _ := newProjectConfigTestDeps(t, mainRoot, global, &mockUI{})

	if err := ResolveProjectConfig(deps, false); err != nil {
		t.Fatalf("expected warn-and-continue (no error) on load failure, got: %v", err)
	}
	if deps.Config.PostStartHook != "global-start" {
		t.Error("expected global config untouched when project .gwrc can't be read")
	}
}

// withSimulatedTTY overrides isTerminalStdin for the duration of fn, so
// ResolveProjectConfig's TTY-gated prompt branch can be exercised without a
// real pty.
func withSimulatedTTY(t *testing.T, isTTY bool, fn func()) {
	t.Helper()
	orig := isTerminalStdin
	isTerminalStdin = func() bool { return isTTY }
	defer func() { isTerminalStdin = orig }()
	fn()
}

func TestResolveProjectConfig_TTYPromptApprovedAppliesAndPersistsTrust(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook = project-start\n")
	global := config.New()
	global.PostStartHook = "global-start"
	ui := &mockUI{trustPromptResult: true}
	deps, _ := newProjectConfigTestDeps(t, mainRoot, global, ui)

	withSimulatedTTY(t, true, func() {
		if err := ResolveProjectConfig(deps, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if deps.Config.PostStartHook != "project-start" {
		t.Errorf("expected project value to apply after prompt approval, got %q", deps.Config.PostStartHook)
	}
	if !ui.trustPromptCalled {
		t.Error("expected TrustPrompt to be called under a simulated TTY")
	}
	if ui.trustPromptPath != filepath.Join(mainRoot, ".gwrc") {
		t.Errorf("expected TrustPrompt to receive the project config path, got %q", ui.trustPromptPath)
	}

	content, err := os.ReadFile(filepath.Join(mainRoot, ".gwrc"))
	if err != nil {
		t.Fatalf("failed to read project config: %v", err)
	}
	hash := trust.Compute(filepath.Join(mainRoot, ".gwrc"), content)
	if !trust.IsApproved(hash) {
		t.Error("expected approval via the prompt to persist to the trust store")
	}
}

func TestResolveProjectConfig_TTYPromptDeclinedFallsBackAndDoesNotPersist(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook = project-start\n")
	global := config.New()
	global.PostStartHook = "global-start"
	ui := &mockUI{trustPromptResult: false}
	deps, stderr := newProjectConfigTestDeps(t, mainRoot, global, ui)

	withSimulatedTTY(t, true, func() {
		if err := ResolveProjectConfig(deps, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if deps.Config.PostStartHook != "global-start" {
		t.Errorf("expected fallback to global value after decline, got %q", deps.Config.PostStartHook)
	}
	if stderr.String() == "" {
		t.Error("expected a warning on stderr after a declined trust prompt")
	}

	content, err := os.ReadFile(filepath.Join(mainRoot, ".gwrc"))
	if err != nil {
		t.Fatalf("failed to read project config: %v", err)
	}
	hash := trust.Compute(filepath.Join(mainRoot, ".gwrc"), content)
	if trust.IsApproved(hash) {
		t.Error("expected a decline to not persist any trust approval")
	}
}

func TestResolveProjectConfig_ApproveFailureFailsClosed(t *testing.T) {
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook = evil-command\n")
	global := config.New()
	global.PostStartHook = "global-start"
	ui := &mockUI{trustPromptResult: true} // user says yes, but Approve() will fail below
	deps, _ := newProjectConfigTestDeps(t, mainRoot, global, ui)

	// Make the trust directory unusable so Approve() fails (fail-closed).
	home := os.Getenv("HOME")
	if err := os.WriteFile(filepath.Join(home, ".gw"), []byte("blocking"), 0o600); err != nil {
		t.Fatalf("failed to set up blocking file: %v", err)
	}

	withSimulatedTTY(t, true, func() {
		if err := ResolveProjectConfig(deps, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	if deps.Config.PostStartHook != "global-start" {
		t.Errorf("expected fail-closed fallback to global value, got %q", deps.Config.PostStartHook)
	}
}
