package cmd

import (
	"fmt"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/trust"
)

// resolveHookStatusesForDisplay computes the effective value and origin of
// each hook key for `gw config --list`. It is a dedicated read-only path:
// unlike ResolveProjectConfig, it never prompts for trust and never mutates
// global — trust is determined purely by reading the existing trust store
// (trust.IsApproved), so a project override that has never been approved is
// shown as untrusted rather than triggering a prompt.
func resolveHookStatusesForDisplay(global *config.Config, g projectConfigGit) []config.HookKeyStatus {
	overlay, found, err := locateProjectOverlay(g)
	if err != nil || !found {
		return config.ResolveHookKeyStatuses(global, config.New(), map[string]bool{}, false)
	}

	trusted := trust.IsApproved(trust.Compute(overlay.path, overlay.content))
	return config.ResolveHookKeyStatuses(global, overlay.cfg, overlay.presentKeys, trusted)
}

// formatHookStatusLine renders one hook key's status for `gw config --list`,
// e.g.:
//
//	post_start_hook     : pnpm dev  [project] [trusted]
//	post_checkout_hook  : (not set) [default]
//	pre_end_hook        : (not set) [global]
//
// An untrusted non-empty project override is shown with the effective
// (global fallback) value plus a note that the project value exists but
// isn't the one actually in effect.
func formatHookStatusLine(status config.HookKeyStatus) string {
	value := status.EffectiveValue
	if value == "" {
		value = "(not set)"
	}

	var annotation string
	switch {
	case status.ProjectDeclared && status.ProjectValue != "" && !status.ProjectTrusted:
		annotation = "[project, untrusted — global value used]"
	case status.Origin == config.OriginProject:
		trustLabel := "untrusted"
		if status.ProjectTrusted {
			trustLabel = "trusted"
		}
		annotation = fmt.Sprintf("[project] [%s]", trustLabel)
	case status.Origin == config.OriginGlobal:
		annotation = "[global]"
	default:
		annotation = "[default]"
	}

	return fmt.Sprintf("%-20s: %-9s %s", status.Key, value, annotation)
}
