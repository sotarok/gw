package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithPresence_FileNotExist(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	cfg, content, presentKeys, err := LoadWithPresence(configPath)
	if err != nil {
		t.Fatalf("expected no error when file doesn't exist, got: %v", err)
	}
	if !cfg.AutoCD {
		t.Error("expected default config (AutoCD=true) when file doesn't exist")
	}
	if content != nil {
		t.Errorf("expected nil content when file doesn't exist, got: %q", content)
	}
	if len(presentKeys) != 0 {
		t.Errorf("expected empty presentKeys when file doesn't exist, got: %v", presentKeys)
	}
}

func TestLoadWithPresence_TracksPresentKeys(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")
	fileContent := "auto_cd = false\npost_start_hook = pnpm dev\n"

	if err := os.WriteFile(configPath, []byte(fileContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, content, presentKeys, err := LoadWithPresence(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AutoCD {
		t.Error("expected AutoCD to be false based on file content")
	}
	if cfg.PostStartHook != "pnpm dev" {
		t.Errorf("expected PostStartHook %q, got %q", "pnpm dev", cfg.PostStartHook)
	}
	if string(content) != fileContent {
		t.Errorf("expected returned content to match file bytes exactly.\nwant: %q\ngot:  %q", fileContent, content)
	}
	if !presentKeys["auto_cd"] {
		t.Error("expected presentKeys[\"auto_cd\"] to be true")
	}
	if !presentKeys["post_start_hook"] {
		t.Error("expected presentKeys[\"post_start_hook\"] to be true")
	}
	if presentKeys["pre_end_hook"] {
		t.Error("expected presentKeys[\"pre_end_hook\"] to be false (not declared)")
	}
}

func TestLoadWithPresence_EmptyHookValue(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")

	if err := os.WriteFile(configPath, []byte("post_start_hook =\n"), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, _, presentKeys, err := LoadWithPresence(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !presentKeys["post_start_hook"] {
		t.Error("expected presentKeys[\"post_start_hook\"] to be true for an explicit empty value")
	}
	if cfg.PostStartHook != "" {
		t.Errorf("expected PostStartHook to be empty, got %q", cfg.PostStartHook)
	}
}

func TestLoadWithPresence_UnreadableFile(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".gwrc")
	if err := os.WriteFile(configPath, []byte("auto_cd = true"), 0000); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	_, content, presentKeys, loadErr := LoadWithPresence(configPath)
	if loadErr == nil {
		t.Errorf("expected error when reading a file without permissions, got content=%q presentKeys=%v", content, presentKeys)
	}
}

func TestMergeHooks_NoProjectConfig(t *testing.T) {
	base := New()
	base.PostStartHook = "global-start"
	base.PreEndHook = "global-pre-end"

	overlay := New()
	presentKeys := map[string]bool{}

	merged := MergeHooks(base, overlay, presentKeys)

	if merged.PostStartHook != "global-start" {
		t.Errorf("expected merged result to equal base when no keys present, got PostStartHook=%q", merged.PostStartHook)
	}
	if merged.PreEndHook != "global-pre-end" {
		t.Errorf("expected merged result to equal base when no keys present, got PreEndHook=%q", merged.PreEndHook)
	}
	if merged.AutoCD != base.AutoCD {
		t.Error("expected non-hook fields to be preserved from base")
	}
}

func TestMergeHooks_OverwritesOnlyPresentHookKeys(t *testing.T) {
	base := New()
	base.PostStartHook = "global-start"
	base.PostCheckoutHook = "global-checkout"

	overlay := New()
	overlay.PostStartHook = "project-start"
	overlay.PostCheckoutHook = "project-checkout" // present in overlay's file but NOT in presentKeys below

	presentKeys := map[string]bool{"post_start_hook": true}

	merged := MergeHooks(base, overlay, presentKeys)

	if merged.PostStartHook != "project-start" {
		t.Errorf("expected PostStartHook to come from overlay, got %q", merged.PostStartHook)
	}
	if merged.PostCheckoutHook != "global-checkout" {
		t.Errorf("expected PostCheckoutHook to stay at base value since not in presentKeys, got %q", merged.PostCheckoutHook)
	}
}

func TestMergeHooks_EmptyOverlayValueDisablesGlobalHook(t *testing.T) {
	base := New()
	base.PostStartHook = "global-start"

	overlay := New()
	overlay.PostStartHook = "" // explicit empty override

	presentKeys := map[string]bool{"post_start_hook": true}

	merged := MergeHooks(base, overlay, presentKeys)

	if merged.PostStartHook != "" {
		t.Errorf("expected empty overlay value to disable the global hook, got %q", merged.PostStartHook)
	}
}

func TestMergeHooks_IgnoresNonHookKeysEvenIfPresent(t *testing.T) {
	base := New()
	base.AutoCD = true

	overlay := New()
	overlay.AutoCD = false

	// auto_cd is a non-hook key; even if a caller mistakenly marks it present,
	// MergeHooks must never apply it.
	presentKeys := map[string]bool{"auto_cd": true}

	merged := MergeHooks(base, overlay, presentKeys)

	if merged.AutoCD != true {
		t.Errorf("expected non-hook key auto_cd to be ignored by MergeHooks, got %v", merged.AutoCD)
	}
}

func TestMergeHooks_DoesNotMutateBase(t *testing.T) {
	base := New()
	base.PostStartHook = "global-start"

	overlay := New()
	overlay.PostStartHook = "project-start"

	presentKeys := map[string]bool{"post_start_hook": true}

	_ = MergeHooks(base, overlay, presentKeys)

	if base.PostStartHook != "global-start" {
		t.Errorf("expected base to remain unmutated, got PostStartHook=%q", base.PostStartHook)
	}
}

func TestOriginSource_Values(t *testing.T) {
	if OriginDefault == OriginGlobal || OriginGlobal == OriginProject || OriginDefault == OriginProject {
		t.Error("expected OriginDefault, OriginGlobal, OriginProject to be distinct values")
	}
}
