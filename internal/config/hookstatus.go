package config

// HookKeyStatus is the effective value and provenance for one hook key,
// combining a global config, an optional project overlay, and whether the
// project file is trust-approved.
type HookKeyStatus struct {
	Key            string
	EffectiveValue string
	Origin         OriginSource

	// ProjectDeclared, ProjectValue and ProjectTrusted describe the project
	// overlay's raw declaration for this key, independent of whether it won
	// out as the effective value. ProjectValue/ProjectTrusted are only
	// meaningful when ProjectDeclared is true.
	ProjectDeclared bool
	ProjectValue    string
	ProjectTrusted  bool
}

// ResolveHookKeyStatuses computes, for each of the three hook keys, the
// effective value and its origin from a global config, an optional project
// overlay (presentKeys marks which keys the project file declares), and
// whether the project file is trust-approved (trusted).
//
// Both ResolveProjectConfig (which applies EffectiveValue) and
// `gw config --list` (which displays Origin plus the project/trust
// annotations) are built on this single function so the two can never
// diverge.
//
// The per-key fallback rule matches the security model: an explicit empty
// project override (presentKeys[key] true, project value "") always applies
// — it can only disable a hook, never execute code, so it needs no trust.
// A non-empty override applies only when trusted; otherwise the global
// value (or the built-in default, if global is also empty) is used.
func ResolveHookKeyStatuses(global, project *Config, presentKeys map[string]bool, trusted bool) []HookKeyStatus {
	// filteredPresentKeys keeps only the overlay keys that should actually
	// take effect, so MergeHooks can be reused verbatim for the merge
	// mechanics instead of duplicating the fieldSpecs walk.
	filteredPresentKeys := map[string]bool{}
	for i := range fieldSpecs {
		spec := &fieldSpecs[i]
		if spec.kind != kindString || !presentKeys[spec.key] {
			continue
		}
		if spec.getString(project) == "" || trusted {
			filteredPresentKeys[spec.key] = true
		}
	}
	merged := MergeHooks(global, project, filteredPresentKeys)

	statuses := make([]HookKeyStatus, 0, numHookKeys)
	for i := range fieldSpecs {
		spec := &fieldSpecs[i]
		if spec.kind != kindString {
			continue
		}

		var origin OriginSource
		switch {
		case filteredPresentKeys[spec.key]:
			origin = OriginProject
		case spec.getString(global) != "":
			origin = OriginGlobal
		default:
			origin = OriginDefault
		}

		statuses = append(statuses, HookKeyStatus{
			Key:             spec.key,
			EffectiveValue:  spec.getString(merged),
			Origin:          origin,
			ProjectDeclared: presentKeys[spec.key],
			ProjectValue:    spec.getString(project),
			ProjectTrusted:  trusted,
		})
	}
	return statuses
}

// numHookKeys is the number of project-overridable hook keys (post_start_hook,
// post_checkout_hook, pre_end_hook), used to size the statuses slice.
const numHookKeys = 3
