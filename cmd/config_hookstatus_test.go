package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/trust"
)

// findDisplayStatus returns the post_start_hook status, the key every test
// in this file inspects.
func findDisplayStatus(t *testing.T, statuses []config.HookKeyStatus) config.HookKeyStatus {
	t.Helper()
	for _, s := range statuses {
		if s.Key == "post_start_hook" {
			return s
		}
	}
	t.Fatalf("no status found for post_start_hook")
	return config.HookKeyStatus{}
}

func TestResolveHookStatusesForDisplay_NoGitRepo(t *testing.T) {
	global := config.New()
	global.PostStartHook = "global-start"
	g := &mockGit{isGitRepo: false}

	statuses := resolveHookStatusesForDisplay(global, g)

	start := findDisplayStatus(t, statuses)
	if start.EffectiveValue != "global-start" || start.Origin != config.OriginGlobal {
		t.Errorf("expected global-only display outside a git repo, got %+v", start)
	}
}

func TestResolveHookStatusesForDisplay_NoProjectFile(t *testing.T) {
	mainRoot := t.TempDir()
	global := config.New()
	global.PostStartHook = "global-start"
	g := &mockGit{
		isGitRepo:               true,
		GetMainRepositoryRootFn: func() (string, error) { return mainRoot, nil },
	}

	statuses := resolveHookStatusesForDisplay(global, g)

	start := findDisplayStatus(t, statuses)
	if start.EffectiveValue != "global-start" || start.Origin != config.OriginGlobal || start.ProjectDeclared {
		t.Errorf("expected global-only display with no project file, got %+v", start)
	}
}

func TestResolveHookStatusesForDisplay_TrustedProjectOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook = project-start\n")

	content, err := os.ReadFile(filepath.Join(mainRoot, ".gwrc"))
	if err != nil {
		t.Fatalf("failed to read project config: %v", err)
	}
	hash := trust.Compute(filepath.Join(mainRoot, ".gwrc"), content)
	if err := trust.Approve(hash); err != nil {
		t.Fatalf("failed to pre-approve: %v", err)
	}

	global := config.New()
	global.PostStartHook = "global-start"
	g := &mockGit{
		isGitRepo:               true,
		GetMainRepositoryRootFn: func() (string, error) { return mainRoot, nil },
	}

	statuses := resolveHookStatusesForDisplay(global, g)

	start := findDisplayStatus(t, statuses)
	if start.EffectiveValue != "project-start" || start.Origin != config.OriginProject || !start.ProjectTrusted {
		t.Errorf("expected trusted project override to be shown as effective, got %+v", start)
	}
}

func TestResolveHookStatusesForDisplay_UntrustedProjectOverrideShowsGlobalFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	mainRoot := t.TempDir()
	writeProjectConfig(t, mainRoot, "post_start_hook = evil-command\n")

	global := config.New()
	global.PostStartHook = "global-start"
	g := &mockGit{
		isGitRepo:               true,
		GetMainRepositoryRootFn: func() (string, error) { return mainRoot, nil },
	}

	statuses := resolveHookStatusesForDisplay(global, g)

	start := findDisplayStatus(t, statuses)
	if start.EffectiveValue != "global-start" {
		t.Errorf("expected effective value to fall back to global, got %q", start.EffectiveValue)
	}
	if !start.ProjectDeclared || start.ProjectValue != "evil-command" || start.ProjectTrusted {
		t.Errorf("expected the untrusted project declaration to still be reported, got %+v", start)
	}
}

func TestResolveHookStatusesForDisplay_MainRootErrorFallsBackToGlobalOnly(t *testing.T) {
	global := config.New()
	global.PostStartHook = "global-start"
	g := &mockGit{
		isGitRepo:               true,
		GetMainRepositoryRootFn: func() (string, error) { return "", errBoom },
	}

	statuses := resolveHookStatusesForDisplay(global, g)

	start := findDisplayStatus(t, statuses)
	if start.EffectiveValue != "global-start" || start.ProjectDeclared {
		t.Errorf("expected global-only fallback when main root can't be resolved, got %+v", start)
	}
}

func TestFormatHookStatusLine_TrustedProject(t *testing.T) {
	line := formatHookStatusLine(config.HookKeyStatus{
		Key: "post_start_hook", EffectiveValue: "pnpm dev", Origin: config.OriginProject,
		ProjectDeclared: true, ProjectValue: "pnpm dev", ProjectTrusted: true,
	})
	if !strings.Contains(line, "post_start_hook") || !strings.Contains(line, "pnpm dev") {
		t.Errorf("expected key and value in line, got %q", line)
	}
	if !strings.Contains(line, "[project]") || !strings.Contains(line, "[trusted]") {
		t.Errorf("expected [project] [trusted] annotation, got %q", line)
	}
}

func TestFormatHookStatusLine_Default(t *testing.T) {
	line := formatHookStatusLine(config.HookKeyStatus{Key: "post_checkout_hook", Origin: config.OriginDefault})
	if !strings.Contains(line, "(not set)") || !strings.Contains(line, "[default]") {
		t.Errorf("expected '(not set)' and '[default]', got %q", line)
	}
}

func TestFormatHookStatusLine_Global(t *testing.T) {
	line := formatHookStatusLine(config.HookKeyStatus{Key: "pre_end_hook", EffectiveValue: "docker compose down", Origin: config.OriginGlobal})
	if !strings.Contains(line, "docker compose down") || !strings.Contains(line, "[global]") {
		t.Errorf("expected value and [global], got %q", line)
	}
}

func TestFormatHookStatusLine_UntrustedProjectShowsGlobalValueUsedNote(t *testing.T) {
	line := formatHookStatusLine(config.HookKeyStatus{
		Key: "post_start_hook", EffectiveValue: "global-start", Origin: config.OriginGlobal,
		ProjectDeclared: true, ProjectValue: "evil-command", ProjectTrusted: false,
	})
	if !strings.Contains(line, "global-start") {
		t.Errorf("expected the effective (global) value to be shown, got %q", line)
	}
	if !strings.Contains(line, "project") || !strings.Contains(line, "untrusted") || !strings.Contains(line, "global value used") {
		t.Errorf("expected an untrusted-project annotation, got %q", line)
	}
}

func TestRunConfig_ListShowsProjectHookOrigins(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	repoDir := t.TempDir()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	globalConfigPath := filepath.Join(home, ".gwrc")
	globalCfg := config.New()
	globalCfg.PreEndHook = "global-pre-end"
	if err := globalCfg.Save(globalConfigPath); err != nil {
		t.Fatalf("failed to save global config: %v", err)
	}

	client := git.NewClient()
	mainRoot, err := client.GetMainRepositoryRoot()
	if err != nil {
		t.Fatalf("failed to resolve main root: %v", err)
	}
	writeProjectConfig(t, mainRoot, "post_start_hook = pnpm dev\n")

	content, err := os.ReadFile(filepath.Join(mainRoot, ".gwrc"))
	if err != nil {
		t.Fatalf("failed to read project config: %v", err)
	}
	hash := trust.Compute(filepath.Join(mainRoot, ".gwrc"), content)
	if err := trust.Approve(hash); err != nil {
		t.Fatalf("failed to approve trust: %v", err)
	}

	configList = true
	defer func() { configList = false }()

	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runErr := runConfig(nil, []string{})

	w.Close()
	os.Stdout = originalStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if runErr != nil {
		t.Fatalf("runConfig failed: %v", runErr)
	}
	if !strings.Contains(output, "post_start_hook") || !strings.Contains(output, "pnpm dev") {
		t.Errorf("expected post_start_hook and its trusted project value in output, got:\n%s", output)
	}
	if !strings.Contains(output, "[project]") || !strings.Contains(output, "[trusted]") {
		t.Errorf("expected [project] [trusted] annotation in output, got:\n%s", output)
	}
	if !strings.Contains(output, "pre_end_hook") || !strings.Contains(output, "global-pre-end") || !strings.Contains(output, "[global]") {
		t.Errorf("expected pre_end_hook to show the global value with [global], got:\n%s", output)
	}
}

func TestFormatHookStatusLine_EmptyProjectOverrideDisablingGlobal(t *testing.T) {
	line := formatHookStatusLine(config.HookKeyStatus{
		Key: "post_start_hook", EffectiveValue: "", Origin: config.OriginProject,
		ProjectDeclared: true, ProjectValue: "", ProjectTrusted: false,
	})
	if !strings.Contains(line, "(not set)") {
		t.Errorf("expected '(not set)' for an empty effective value, got %q", line)
	}
	if !strings.Contains(line, "[project]") {
		t.Errorf("expected a [project] annotation for the disabling override, got %q", line)
	}
}
