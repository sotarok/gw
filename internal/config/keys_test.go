package config

import (
	"reflect"
	"sort"
	"testing"
)

func TestHookKeys_ReturnsExactlyTheThreeHookKeys(t *testing.T) {
	got := HookKeys()
	sort.Strings(got)
	want := []string{"post_checkout_hook", "post_start_hook", "pre_end_hook"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestIsHookKey(t *testing.T) {
	if !IsHookKey("post_start_hook") {
		t.Error("expected post_start_hook to be a hook key")
	}
	if IsHookKey("auto_cd") {
		t.Error("expected auto_cd to not be a hook key")
	}
	if IsHookKey("totally_unknown_key") {
		t.Error("expected an unknown key to not be a hook key")
	}
}

func TestIsKnownKey(t *testing.T) {
	if !IsKnownKey("auto_cd") {
		t.Error("expected auto_cd to be a known key")
	}
	if !IsKnownKey("post_start_hook") {
		t.Error("expected post_start_hook to be a known key")
	}
	if IsKnownKey("totally_unknown_key") {
		t.Error("expected an unrecognized key to not be known")
	}
}

func TestSetHookValue(t *testing.T) {
	c := New()

	if err := c.SetHookValue("post_start_hook", "pnpm dev"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.PostStartHook != "pnpm dev" {
		t.Errorf("expected PostStartHook to be set, got %q", c.PostStartHook)
	}

	if err := c.SetHookValue("pre_end_hook", ""); err != nil {
		t.Fatalf("unexpected error setting empty value: %v", err)
	}
	if c.PreEndHook != "" {
		t.Errorf("expected PreEndHook to be empty, got %q", c.PreEndHook)
	}

	if err := c.SetHookValue("auto_cd", "true"); err == nil {
		t.Error("expected an error when setting a non-hook key via SetHookValue")
	}

	if err := c.SetHookValue("unknown_key", "value"); err == nil {
		t.Error("expected an error when setting an unknown key via SetHookValue")
	}
}
