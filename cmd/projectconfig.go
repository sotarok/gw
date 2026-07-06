package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/term"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/trust"
)

// projectConfigGit is the subset of git operations ResolveProjectConfig uses.
type projectConfigGit interface {
	IsGitRepository() bool
	GetMainRepositoryRoot() (string, error)
}

// projectConfigFileName is the project-local config file name, read from the
// main worktree root — the same name as the global config, mirroring the
// precedent of same-format local config files like .npmrc.
const projectConfigFileName = ".gwrc"

// isTerminalStdin reports whether stdin is attached to a terminal. It is a
// variable rather than a direct term.IsTerminal call so tests can simulate a
// TTY session without needing a real pty.
var isTerminalStdin = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// projectOverlay is a successfully-located and parsed project-local .gwrc.
// It is the shared unit between ResolveProjectConfig (which additionally
// prompts/approves and mutates deps.Config) and `gw config --list`'s
// dedicated read-only display path.
type projectOverlay struct {
	path        string
	cfg         *config.Config
	content     []byte
	presentKeys map[string]bool
}

// locateProjectOverlay resolves and loads the project-local .gwrc, if any.
// ok is false with a nil error when there's simply nothing to apply (not a
// git repo, no main root, or no project file) — every one of those is a
// normal "fall back to global only" case, not a failure. A non-nil error
// means a project file was found but could not be read or parsed.
func locateProjectOverlay(g projectConfigGit) (overlay projectOverlay, ok bool, err error) {
	if !g.IsGitRepository() {
		return projectOverlay{}, false, nil
	}

	mainRoot, err := g.GetMainRepositoryRoot()
	if err != nil {
		return projectOverlay{}, false, err
	}

	path := filepath.Join(mainRoot, projectConfigFileName)
	if _, statErr := os.Stat(path); statErr != nil {
		return projectOverlay{}, false, nil
	}

	cfg, content, presentKeys, loadErr := config.LoadWithPresence(path)
	if loadErr != nil {
		return projectOverlay{}, false, loadErr
	}

	return projectOverlay{path: path, cfg: cfg, content: content, presentKeys: presentKeys}, true, nil
}

// ResolveProjectConfig resolves a project-local .gwrc (if any), evaluates
// trust for any non-empty hook overrides it declares, and — once approved —
// replaces deps.Config's three hook keys with the project's values.
//
// noProjectHooks (true for --no-project-hooks, and for the --force/--dry-run
// paths of end/clean) skips all project hook application, including the
// trust-free empty-value disablement: the project file may still be read
// for the non-hook-key stderr note, but none of its hook values take
// effect.
//
// Project config resolution never fails the calling command: parsing or
// trust errors are reported as warnings on stderr and the global
// configuration is kept (warn-and-continue), matching the policy already
// used for the global config load in DefaultDependencies.
func ResolveProjectConfig(deps *Dependencies, noProjectHooks bool) error {
	g, ok := deps.Git.(projectConfigGit)
	if !ok {
		return nil
	}

	overlay, found, err := locateProjectOverlay(g)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "%s Could not resolve project configuration; using global configuration: %v\n",
			coloredWarning(), err)
		return nil
	}
	if !found {
		return nil
	}

	warnIgnoredNonHookKeys(deps, overlay.presentKeys)

	if noProjectHooks {
		return nil
	}

	trusted := resolveTrust(deps, overlay.path, overlay.content, overlay.cfg, overlay.presentKeys)

	statuses := config.ResolveHookKeyStatuses(deps.Config, overlay.cfg, overlay.presentKeys, trusted)
	for _, status := range statuses {
		_ = deps.Config.SetHookValue(status.Key, status.EffectiveValue)
	}
	return nil
}

// warnIgnoredNonHookKeys prints a one-line stderr note for every known,
// non-hook key the project file declares (parsed but never applied in
// v1.1). Keys are sorted for deterministic output.
func warnIgnoredNonHookKeys(deps *Dependencies, presentKeys map[string]bool) {
	var ignored []string
	for key := range presentKeys {
		if config.IsKnownKey(key) && !config.IsHookKey(key) {
			ignored = append(ignored, key)
		}
	}
	sort.Strings(ignored)
	for _, key := range ignored {
		fmt.Fprintf(deps.Stderr, "note: project .gwrc key '%s' is ignored in v1.1 (hooks-only)\n", key)
	}
}

// resolveTrust determines whether the project file's non-empty hook
// overrides should be honored for this invocation. It returns false without
// prompting when there are no non-empty overrides to evaluate — empty
// overrides need no trust — and otherwise checks the existing trust store,
// falling back to an interactive prompt (fail closed on any error, decline,
// or non-TTY session).
func resolveTrust(deps *Dependencies, projectPath string, content []byte, projectCfg *config.Config, presentKeys map[string]bool) bool {
	hookLines := nonEmptyProjectHookLines(projectCfg, presentKeys)
	if len(hookLines) == 0 {
		return false
	}

	hash := trust.Compute(projectPath, content)
	if trust.IsApproved(hash) {
		return true
	}

	if !isTerminalStdin() {
		fmt.Fprintf(deps.Stderr, "%s Untrusted project hook(s) at %s (non-interactive session);"+
			" using global configuration for those keys.\n", coloredWarning(), projectPath)
		return false
	}

	approved, err := deps.UI.TrustPrompt(projectPath, hookLines)
	if err != nil || !approved {
		fmt.Fprintf(deps.Stderr, "%s Untrusted project hook(s) at %s; using global configuration for those keys.\n",
			coloredWarning(), projectPath)
		return false
	}

	if err := trust.Approve(hash); err != nil {
		fmt.Fprintf(deps.Stderr, "%s Could not record trust approval; using global configuration for those keys: %v\n",
			coloredWarning(), err)
		return false
	}
	return true
}

// nonEmptyProjectHookLines returns "key = value" lines for every hook key
// the project file declares with a non-empty value — the set that requires
// trust approval and is shown to the user in the trust prompt.
func nonEmptyProjectHookLines(projectCfg *config.Config, presentKeys map[string]bool) []string {
	var lines []string
	for _, status := range config.ResolveHookKeyStatuses(config.New(), projectCfg, presentKeys, true) {
		if status.ProjectDeclared && status.ProjectValue != "" {
			lines = append(lines, fmt.Sprintf("%s = %s", status.Key, status.ProjectValue))
		}
	}
	return lines
}
