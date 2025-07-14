package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPackageManager(t *testing.T) {
	t.Run("detects npm with package-lock.json", func(t *testing.T) {
		// Create temp directory
		tempDir, err := os.MkdirTemp("", "test-npm")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create package.json and package-lock.json
		if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create package.json: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, "package-lock.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create package-lock.json: %v", err)
		}

		pm, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pm.Name != "npm" {
			t.Errorf("expected npm, got %s", pm.Name)
		}
	})

	t.Run("detects yarn with yarn.lock", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-yarn")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create package.json and yarn.lock
		if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create package.json: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, "yarn.lock"), []byte(""), 0644); err != nil {
			t.Fatalf("failed to create yarn.lock: %v", err)
		}

		pm, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pm.Name != "yarn" {
			t.Errorf("expected yarn, got %s", pm.Name)
		}
	})

	t.Run("detects pnpm with pnpm-lock.yaml", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-pnpm")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create package.json and pnpm-lock.yaml
		if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create package.json: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, "pnpm-lock.yaml"), []byte(""), 0644); err != nil {
			t.Fatalf("failed to create pnpm-lock.yaml: %v", err)
		}

		pm, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pm.Name != "pnpm" {
			t.Errorf("expected pnpm, got %s", pm.Name)
		}
	})

	t.Run("defaults to npm with only package.json", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-npm-default")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create only package.json
		if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create package.json: %v", err)
		}

		pm, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pm.Name != "npm" {
			t.Errorf("expected npm as default, got %s", pm.Name)
		}
	})

	t.Run("detects go with go.mod", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-go")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create go.mod
		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
			t.Fatalf("failed to create go.mod: %v", err)
		}

		pm, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pm.Name != "go" {
			t.Errorf("expected go, got %s", pm.Name)
		}
	})

	t.Run("detects cargo with Cargo.toml", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-cargo")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create Cargo.toml
		if err := os.WriteFile(filepath.Join(tempDir, "Cargo.toml"), []byte("[package]\nname = \"test\"\n"), 0644); err != nil {
			t.Fatalf("failed to create Cargo.toml: %v", err)
		}

		pm, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pm.Name != "cargo" {
			t.Errorf("expected cargo, got %s", pm.Name)
		}
	})

	t.Run("returns error when no package manager found", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-none")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		_, err = DetectPackageManager(tempDir)
		if err == nil {
			t.Error("expected error when no package manager found")
		}
	})

	t.Run("prefers Node.js package managers over others", func(t *testing.T) {
		// This test validates that if both package.json and go.mod exist,
		// it should prefer the Node.js package manager
		tempDir, err := os.MkdirTemp("", "test-priority")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create both package.json and go.mod
		if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create package.json: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
			t.Fatalf("failed to create go.mod: %v", err)
		}

		pm, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pm.Name != "npm" {
			t.Errorf("expected npm to take priority, got %s", pm.Name)
		}
	})

	t.Run("returns independent copy of PackageManager", func(t *testing.T) {
		// This test ensures that modifying the returned PackageManager
		// doesn't affect the global packageManagers array
		tempDir, err := os.MkdirTemp("", "test-copy")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create go.mod
		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module test\n"), 0644); err != nil {
			t.Fatalf("failed to create go.mod: %v", err)
		}

		pm1, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Store original values
		originalName := pm1.Name
		originalCmd := make([]string, len(pm1.InstallCmd))
		copy(originalCmd, pm1.InstallCmd)

		// Modify the returned struct
		pm1.Name = "modified"
		pm1.InstallCmd[0] = "modified"

		// Get another instance
		pm2, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// The second instance should have the original values
		if pm2.Name != originalName {
			t.Errorf("expected Name to be %q, got %q - global data was modified!", originalName, pm2.Name)
		}
		if pm2.InstallCmd[0] != originalCmd[0] {
			t.Errorf("expected InstallCmd[0] to be %q, got %q - global data was modified!", originalCmd[0], pm2.InstallCmd[0])
		}
	})
}