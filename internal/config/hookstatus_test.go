package config

import "testing"

func findHookStatus(t *testing.T, statuses []HookKeyStatus, key string) HookKeyStatus {
	t.Helper()
	for _, s := range statuses {
		if s.Key == key {
			return s
		}
	}
	t.Fatalf("no status found for key %q", key)
	return HookKeyStatus{}
}

func TestResolveHookKeyStatuses_NoProjectConfig(t *testing.T) {
	global := New()
	global.PostStartHook = "global-start"
	project := New()
	presentKeys := map[string]bool{}

	statuses := ResolveHookKeyStatuses(global, project, presentKeys, false)

	start := findHookStatus(t, statuses, "post_start_hook")
	if start.EffectiveValue != "global-start" {
		t.Errorf("expected effective value %q, got %q", "global-start", start.EffectiveValue)
	}
	if start.Origin != OriginGlobal {
		t.Errorf("expected OriginGlobal, got %v", start.Origin)
	}
	if start.ProjectDeclared {
		t.Error("expected ProjectDeclared to be false")
	}

	checkout := findHookStatus(t, statuses, "post_checkout_hook")
	if checkout.EffectiveValue != "" || checkout.Origin != OriginDefault {
		t.Errorf("expected empty/default for undeclared+unset key, got %q/%v", checkout.EffectiveValue, checkout.Origin)
	}
}

func TestResolveHookKeyStatuses_TrustedProjectOverride(t *testing.T) {
	global := New()
	global.PostStartHook = "global-start"
	project := New()
	project.PostStartHook = "project-start"
	presentKeys := map[string]bool{"post_start_hook": true}

	statuses := ResolveHookKeyStatuses(global, project, presentKeys, true)

	start := findHookStatus(t, statuses, "post_start_hook")
	if start.EffectiveValue != "project-start" {
		t.Errorf("expected project value to win when trusted, got %q", start.EffectiveValue)
	}
	if start.Origin != OriginProject {
		t.Errorf("expected OriginProject, got %v", start.Origin)
	}
	if !start.ProjectDeclared || start.ProjectValue != "project-start" || !start.ProjectTrusted {
		t.Errorf("expected ProjectDeclared/ProjectValue/ProjectTrusted to reflect the trusted override, got %+v", start)
	}
}

func TestResolveHookKeyStatuses_UntrustedProjectOverrideFallsBackToGlobal(t *testing.T) {
	global := New()
	global.PostStartHook = "global-start"
	project := New()
	project.PostStartHook = "project-start"
	presentKeys := map[string]bool{"post_start_hook": true}

	statuses := ResolveHookKeyStatuses(global, project, presentKeys, false)

	start := findHookStatus(t, statuses, "post_start_hook")
	if start.EffectiveValue != "global-start" {
		t.Errorf("expected fallback to global value when untrusted, got %q", start.EffectiveValue)
	}
	if start.Origin != OriginGlobal {
		t.Errorf("expected OriginGlobal for the effective (fallback) value, got %v", start.Origin)
	}
	if !start.ProjectDeclared || start.ProjectValue != "project-start" || start.ProjectTrusted {
		t.Errorf("expected the untrusted project declaration to still be reported, got %+v", start)
	}
}

func TestResolveHookKeyStatuses_UntrustedProjectOverrideNoGlobalFallsBackToDefault(t *testing.T) {
	global := New() // PostStartHook left empty
	project := New()
	project.PostStartHook = "project-start"
	presentKeys := map[string]bool{"post_start_hook": true}

	statuses := ResolveHookKeyStatuses(global, project, presentKeys, false)

	start := findHookStatus(t, statuses, "post_start_hook")
	if start.EffectiveValue != "" {
		t.Errorf("expected empty effective value when neither project (untrusted) nor global is usable, got %q", start.EffectiveValue)
	}
	if start.Origin != OriginDefault {
		t.Errorf("expected OriginDefault, got %v", start.Origin)
	}
}

func TestResolveHookKeyStatuses_EmptyProjectOverrideAppliesRegardlessOfTrust(t *testing.T) {
	global := New()
	global.PostStartHook = "global-start"
	project := New()
	project.PostStartHook = "" // explicit disable
	presentKeys := map[string]bool{"post_start_hook": true}

	statuses := ResolveHookKeyStatuses(global, project, presentKeys, false)

	start := findHookStatus(t, statuses, "post_start_hook")
	if start.EffectiveValue != "" {
		t.Errorf("expected empty override to disable the global hook even when untrusted, got %q", start.EffectiveValue)
	}
	if start.Origin != OriginProject {
		t.Errorf("expected OriginProject for an explicit empty override, got %v", start.Origin)
	}
	if !start.ProjectDeclared {
		t.Error("expected ProjectDeclared to be true")
	}
}

func TestResolveHookKeyStatuses_OnlyDeclaredKeyIsAffected(t *testing.T) {
	global := New()
	global.PostStartHook = "global-start"
	global.PostCheckoutHook = "global-checkout"
	project := New()
	project.PostStartHook = "project-start"
	presentKeys := map[string]bool{"post_start_hook": true} // post_checkout_hook not declared

	statuses := ResolveHookKeyStatuses(global, project, presentKeys, true)

	checkout := findHookStatus(t, statuses, "post_checkout_hook")
	if checkout.EffectiveValue != "global-checkout" || checkout.Origin != OriginGlobal || checkout.ProjectDeclared {
		t.Errorf("expected undeclared key to be untouched, got %+v", checkout)
	}
}
