package trust

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompute_Deterministic(t *testing.T) {
	h1 := Compute("/repo/.gwrc", []byte("post_start_hook = pnpm dev\n"))
	h2 := Compute("/repo/.gwrc", []byte("post_start_hook = pnpm dev\n"))
	if h1 != h2 {
		t.Errorf("expected identical (path, content) to produce identical hash, got %q vs %q", h1, h2)
	}
}

func TestCompute_DiffersByContent(t *testing.T) {
	h1 := Compute("/repo/.gwrc", []byte("post_start_hook = pnpm dev\n"))
	h2 := Compute("/repo/.gwrc", []byte("post_start_hook = pnpm run dev\n"))
	if h1 == h2 {
		t.Error("expected different content to produce a different hash")
	}
}

func TestCompute_DiffersByPath(t *testing.T) {
	h1 := Compute("/repo-a/.gwrc", []byte("post_start_hook = pnpm dev\n"))
	h2 := Compute("/repo-b/.gwrc", []byte("post_start_hook = pnpm dev\n"))
	if h1 == h2 {
		t.Error("expected different absolute paths to produce a different hash, even with identical content")
	}
}

func TestIsApproved_FalseWhenNeverApproved(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if IsApproved("deadbeef") {
		t.Error("expected IsApproved to be false for a hash that was never approved")
	}
}

func TestApprove_ThenIsApprovedTrue(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	hash := Compute("/repo/.gwrc", []byte("post_start_hook = pnpm dev\n"))

	if err := Approve(hash); err != nil {
		t.Fatalf("unexpected error approving hash: %v", err)
	}
	if !IsApproved(hash) {
		t.Error("expected IsApproved to be true after Approve")
	}
}

func TestApprove_CreatesFileWithOwnerOnlyPermissions(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	hash := Compute("/repo/.gwrc", []byte("pre_end_hook = docker compose down\n"))
	if err := Approve(hash); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	trustFile := filepath.Join(home, ".gw", "trust", hash)
	info, err := os.Stat(trustFile)
	if err != nil {
		t.Fatalf("expected trust file to exist at %s: %v", trustFile, err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected trust file permissions 0600, got %v", info.Mode().Perm())
	}
}

func TestApprove_IsIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	hash := Compute("/repo/.gwrc", []byte("post_checkout_hook = pnpm dev\n"))

	if err := Approve(hash); err != nil {
		t.Fatalf("unexpected error on first approve: %v", err)
	}
	if err := Approve(hash); err != nil {
		t.Errorf("expected second Approve of the same hash to be a no-op, got error: %v", err)
	}
}

func TestApprove_FailsClosedWhenTrustDirUnusable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create a regular file where the ".gw" directory should be, so
	// MkdirAll(".gw/trust") fails — this must surface as an error rather than
	// silently approving (fail-closed).
	if err := os.WriteFile(filepath.Join(home, ".gw"), []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("failed to set up blocking file: %v", err)
	}

	hash := Compute("/repo/.gwrc", []byte("post_start_hook = pnpm dev\n"))
	if err := Approve(hash); err == nil {
		t.Error("expected Approve to fail when the trust directory cannot be created")
	}
	if IsApproved(hash) {
		t.Error("expected IsApproved to remain false after a failed Approve")
	}
}

func TestDifferentContentRequiresReapproval(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	oldHash := Compute("/repo/.gwrc", []byte("post_start_hook = pnpm dev\n"))
	if err := Approve(oldHash); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newHash := Compute("/repo/.gwrc", []byte("post_start_hook = pnpm run dev\n"))
	if IsApproved(newHash) {
		t.Error("expected a changed file's new hash to not be approved even though the old hash was")
	}
}
