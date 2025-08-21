package detect

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	goName    = "go"
	cargoName = "cargo"
)

func TestDetectPackageManager_EdgeCases(t *testing.T) {
	t.Run("handles symlinked lock files", func(t *testing.T) {
		// Create source and target directories
		sourceDir, err := os.MkdirTemp("", "test-symlink-source-")
		if err != nil {
			t.Fatalf("failed to create source dir: %v", err)
		}
		defer os.RemoveAll(sourceDir)

		targetDir, err := os.MkdirTemp("", "test-symlink-target-")
		if err != nil {
			t.Fatalf("failed to create target dir: %v", err)
		}
		defer os.RemoveAll(targetDir)

		// Create actual lock file in source
		if err := os.WriteFile(filepath.Join(sourceDir, "package-lock.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create package-lock.json: %v", err)
		}

		// Create package.json in target
		if err := os.WriteFile(filepath.Join(targetDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create package.json: %v", err)
		}

		// Create symlink in target pointing to source lock file
		symlinkPath := filepath.Join(targetDir, "package-lock.json")
		sourcePath := filepath.Join(sourceDir, "package-lock.json")
		if err := os.Symlink(sourcePath, symlinkPath); err != nil {
			// Skip test if symlinks are not supported (e.g., Windows without privileges)
			t.Skip("symlinks not supported on this system")
		}

		// Should detect npm
		pm, err := DetectPackageManager(targetDir)
		if err != nil {
			t.Fatalf("failed to detect package manager with symlinked lock file: %v", err)
		}

		if pm.Name != npmName {
			t.Errorf("expected npm, got %s", pm.Name)
		}
	})

	t.Run("handles directories with spaces in path", func(t *testing.T) {
		// Create directory with spaces
		baseDir, err := os.MkdirTemp("", "test spaces in path-")
		if err != nil {
			t.Fatalf("failed to create base dir: %v", err)
		}
		defer os.RemoveAll(baseDir)

		dirWithSpaces := filepath.Join(baseDir, "my project", "with spaces")
		if err := os.MkdirAll(dirWithSpaces, 0755); err != nil {
			t.Fatalf("failed to create dir with spaces: %v", err)
		}

		// Create go.mod in directory with spaces
		if err := os.WriteFile(filepath.Join(dirWithSpaces, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
			t.Fatalf("failed to create go.mod: %v", err)
		}

		pm, err := DetectPackageManager(dirWithSpaces)
		if err != nil {
			t.Fatalf("failed to detect package manager in path with spaces: %v", err)
		}

		if pm.Name != goName {
			t.Errorf("expected go, got %s", pm.Name)
		}
	})

	t.Run("handles read-only directories", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-readonly-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer func() {
			// Restore permissions before cleanup
			os.Chmod(tempDir, 0755)
			os.RemoveAll(tempDir)
		}()

		// Create package.json before making read-only
		if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create package.json: %v", err)
		}

		// Make directory read-only
		if err := os.Chmod(tempDir, 0555); err != nil {
			t.Fatalf("failed to make directory read-only: %v", err)
		}

		// Detection should still work on read-only directory
		pm, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("failed to detect package manager in read-only directory: %v", err)
		}

		if pm.Name != npmName {
			t.Errorf("expected npm, got %s", pm.Name)
		}
	})

	t.Run("handles very deep directory structures", func(t *testing.T) {
		baseDir, err := os.MkdirTemp("", "test-deep-")
		if err != nil {
			t.Fatalf("failed to create base dir: %v", err)
		}
		defer os.RemoveAll(baseDir)

		// Create a very deep directory structure
		deepPath := baseDir
		for i := 0; i < 20; i++ {
			deepPath = filepath.Join(deepPath, "level")
		}

		if err := os.MkdirAll(deepPath, 0755); err != nil {
			t.Fatalf("failed to create deep directory: %v", err)
		}

		// Create Cargo.toml in deep directory
		if err := os.WriteFile(filepath.Join(deepPath, "Cargo.toml"), []byte("[package]\nname = \"test\"\n"), 0644); err != nil {
			t.Fatalf("failed to create Cargo.toml: %v", err)
		}

		pm, err := DetectPackageManager(deepPath)
		if err != nil {
			t.Fatalf("failed to detect package manager in deep directory: %v", err)
		}

		if pm.Name != cargoName {
			t.Errorf("expected cargo, got %s", pm.Name)
		}
	})

	t.Run("handles multiple package managers with correct priority", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-multi-pm-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create files for multiple package managers
		files := map[string]string{
			"package.json":     "{}",
			"go.mod":           "module test\n\ngo 1.21\n",
			"Cargo.toml":       "[package]\nname = \"test\"\n",
			"requirements.txt": "flask==2.0.0\n",
			"Gemfile":          "source 'https://rubygems.org'\n",
			"composer.json":    `{"name": "test/project"}`,
		}

		for filename, content := range files {
			if err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644); err != nil {
				t.Fatalf("failed to create %s: %v", filename, err)
			}
		}

		// Node.js should take priority
		pm, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("failed to detect package manager: %v", err)
		}

		if pm.Name != npmName {
			t.Errorf("expected npm to have priority, got %s", pm.Name)
		}
	})

	t.Run("handles corrupted package files gracefully", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-corrupted-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a package.json with invalid content (but should still be detected)
		if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte("not valid json{{{"), 0644); err != nil {
			t.Fatalf("failed to create package.json: %v", err)
		}

		// Detection should still work based on file presence
		pm, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("failed to detect package manager with corrupted file: %v", err)
		}

		if pm.Name != npmName {
			t.Errorf("expected npm, got %s", pm.Name)
		}
	})

	t.Run("handles empty files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-empty-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create empty lock files
		emptyFiles := []string{
			"package.json",
			"yarn.lock",
		}

		for _, filename := range emptyFiles {
			if err := os.WriteFile(filepath.Join(tempDir, filename), []byte(""), 0644); err != nil {
				t.Fatalf("failed to create %s: %v", filename, err)
			}
		}

		// Should detect yarn (yarn.lock takes precedence)
		pm, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("failed to detect package manager with empty files: %v", err)
		}

		if pm.Name != yarnName {
			t.Errorf("expected yarn, got %s", pm.Name)
		}
	})

	t.Run("returns fresh copy on each call", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-fresh-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create package.json
		if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create package.json: %v", err)
		}

		// Get two instances
		pm1, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("first detection failed: %v", err)
		}

		pm2, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("second detection failed: %v", err)
		}

		// Modify first instance
		pm1.Name = "modified"
		pm1.InstallCmd[0] = "changed"

		// Second instance should be unaffected
		if pm2.Name != npmName {
			t.Errorf("second instance was affected by first instance modification")
		}
		if pm2.InstallCmd[0] != npmName {
			t.Errorf("second instance InstallCmd was affected by first instance modification")
		}

		// Get third instance to verify global state is intact
		pm3, err := DetectPackageManager(tempDir)
		if err != nil {
			t.Fatalf("third detection failed: %v", err)
		}

		if pm3.Name != npmName {
			t.Errorf("global state was modified, expected npm, got %s", pm3.Name)
		}
	})
}
