package git

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	testEnvFileName = ".env"
)

func TestFindUntrackedEnvFiles_EdgeCases(t *testing.T) {
	t.Run("handles directory read errors gracefully", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create a directory with no read permissions
		restrictedDir := filepath.Join(tmpDir, "restricted")
		if err := os.MkdirAll(restrictedDir, 0755); err != nil {
			t.Fatalf("Failed to create restricted dir: %v", err)
		}

		// Add .env file to restricted dir
		if err := os.WriteFile(filepath.Join(restrictedDir, testEnvFileName), []byte("SECRET=value"), 0644); err != nil {
			t.Fatalf("Failed to create .env in restricted dir: %v", err)
		}

		// Remove read permissions
		if err := os.Chmod(restrictedDir, 0000); err != nil {
			t.Fatalf("Failed to change permissions: %v", err)
		}
		defer os.Chmod(restrictedDir, 0755) // Restore permissions for cleanup

		// Create a normal .env file
		if err := os.WriteFile(filepath.Join(tmpDir, testEnvFileName), []byte("TEST=value"), 0644); err != nil {
			t.Fatalf("Failed to create .env: %v", err)
		}

		// Should still find env files outside restricted directory
		envFiles, err := FindUntrackedEnvFiles(tmpDir)
		if err != nil {
			t.Fatalf("FindUntrackedEnvFiles should not fail due to unreadable directory: %v", err)
		}

		// Should find at least the root .env file
		found := false
		for _, envFile := range envFiles {
			if envFile.Path == testEnvFileName {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find root .env file despite restricted directory")
		}
	})

	t.Run("handles vendor directory correctly", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create vendor directory
		vendorDir := filepath.Join(tmpDir, "vendor", "github.com", "example")
		if err := os.MkdirAll(vendorDir, 0755); err != nil {
			t.Fatalf("Failed to create vendor dir: %v", err)
		}

		// Create .env files
		os.WriteFile(filepath.Join(tmpDir, testEnvFileName), []byte("ROOT=true"), 0644)
		os.WriteFile(filepath.Join(vendorDir, testEnvFileName), []byte("VENDOR=true"), 0644)

		// Find untracked env files
		envFiles, err := FindUntrackedEnvFiles(tmpDir)
		if err != nil {
			t.Fatalf("FindUntrackedEnvFiles failed: %v", err)
		}

		// Should only find the root .env, not the one in vendor
		if len(envFiles) != 1 {
			t.Errorf("Expected 1 untracked file, got %d", len(envFiles))
		}

		if len(envFiles) > 0 && envFiles[0].Path != testEnvFileName {
			t.Errorf("Expected to find .env, got %s", envFiles[0].Path)
		}
	})

	t.Run("handles dist and build directories", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create excluded directories
		excludedDirs := []string{"dist", "build"}
		for _, dir := range excludedDirs {
			fullPath := filepath.Join(tmpDir, dir)
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				t.Fatalf("Failed to create %s dir: %v", dir, err)
			}
			// Add .env file to excluded dir
			os.WriteFile(filepath.Join(fullPath, testEnvFileName), []byte("EXCLUDED=true"), 0644)
		}

		// Create .env in root
		os.WriteFile(filepath.Join(tmpDir, testEnvFileName), []byte("ROOT=true"), 0644)

		// Find untracked env files
		envFiles, err := FindUntrackedEnvFiles(tmpDir)
		if err != nil {
			t.Fatalf("FindUntrackedEnvFiles failed: %v", err)
		}

		// Should only find the root .env
		if len(envFiles) != 1 {
			t.Errorf("Expected 1 untracked file, got %d", len(envFiles))
		}

		if len(envFiles) > 0 && envFiles[0].Path != testEnvFileName {
			t.Errorf("Expected to find only root .env, got %s", envFiles[0].Path)
		}
	})

	t.Run("handles git command failure", func(t *testing.T) {
		// Create temporary directory without git initialization
		tmpDir, err := os.MkdirTemp("", "gw-test-nogit-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create .env file
		os.WriteFile(filepath.Join(tmpDir, testEnvFileName), []byte("TEST=value"), 0644)

		// Should fail because it's not a git repository
		_, err = FindUntrackedEnvFiles(tmpDir)
		if err == nil {
			t.Error("Expected error when git ls-files fails")
		}
	})

	t.Run("handles filepath.Rel errors gracefully", func(t *testing.T) {
		// Create temporary directory
		tmpDir, err := os.MkdirTemp("", "gw-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Initialize git repository
		runGitCommand(t, tmpDir, "init")
		runGitCommand(t, tmpDir, "config", "user.email", "test@example.com")
		runGitCommand(t, tmpDir, "config", "user.name", "Test User")

		// Create .env file
		if err := os.WriteFile(filepath.Join(tmpDir, testEnvFileName), []byte("TEST=value"), 0644); err != nil {
			t.Fatalf("Failed to create .env: %v", err)
		}

		// This should work normally
		envFiles, err := FindUntrackedEnvFiles(tmpDir)
		if err != nil {
			t.Fatalf("FindUntrackedEnvFiles failed: %v", err)
		}

		if len(envFiles) != 1 {
			t.Errorf("Expected 1 untracked file, got %d", len(envFiles))
		}
	})
}

func TestCopyEnvFiles_EdgeCases(t *testing.T) {
	t.Run("handles MkdirAll failure", func(t *testing.T) {
		// Create source directory
		srcDir, err := os.MkdirTemp("", "gw-src-*")
		if err != nil {
			t.Fatalf("Failed to create source dir: %v", err)
		}
		defer os.RemoveAll(srcDir)

		// Create a file where we expect a directory
		destDir, err := os.MkdirTemp("", "gw-dest-*")
		if err != nil {
			t.Fatalf("Failed to create dest dir: %v", err)
		}
		defer os.RemoveAll(destDir)

		// Create a file where a directory is expected
		blockerPath := filepath.Join(destDir, "app")
		if err := os.WriteFile(blockerPath, []byte("blocker"), 0644); err != nil {
			t.Fatalf("Failed to create blocker file: %v", err)
		}

		// Create source file
		appDir := filepath.Join(srcDir, "app")
		if err := os.MkdirAll(appDir, 0755); err != nil {
			t.Fatalf("Failed to create app dir: %v", err)
		}

		srcFile := filepath.Join(appDir, testEnvFileName)
		if err := os.WriteFile(srcFile, []byte("TEST=value"), 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		envFiles := []EnvFile{
			{
				Path:         "app/" + testEnvFileName,
				AbsolutePath: srcFile,
			},
		}

		// Should fail because can't create directory
		err = CopyEnvFiles(envFiles, srcDir, destDir)
		if err == nil {
			t.Error("Expected error when MkdirAll fails")
		}
	})

	t.Run("handles ReadFile failure", func(t *testing.T) {
		// Create source directory
		srcDir, err := os.MkdirTemp("", "gw-src-*")
		if err != nil {
			t.Fatalf("Failed to create source dir: %v", err)
		}
		defer os.RemoveAll(srcDir)

		// Create destination directory
		destDir, err := os.MkdirTemp("", "gw-dest-*")
		if err != nil {
			t.Fatalf("Failed to create dest dir: %v", err)
		}
		defer os.RemoveAll(destDir)

		// Create env file list with non-existent file
		envFiles := []EnvFile{
			{
				Path:         testEnvFileName,
				AbsolutePath: filepath.Join(srcDir, testEnvFileName),
			},
		}

		// Should fail because source file doesn't exist
		err = CopyEnvFiles(envFiles, srcDir, destDir)
		if err == nil {
			t.Error("Expected error when reading non-existent file")
		}
	})

	t.Run("handles WriteFile failure", func(t *testing.T) {
		// Create source directory
		srcDir, err := os.MkdirTemp("", "gw-src-*")
		if err != nil {
			t.Fatalf("Failed to create source dir: %v", err)
		}
		defer os.RemoveAll(srcDir)

		// Create source file
		srcFile := filepath.Join(srcDir, testEnvFileName)
		if err := os.WriteFile(srcFile, []byte("TEST=value"), 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// Create destination directory with restricted permissions
		destDir, err := os.MkdirTemp("", "gw-dest-*")
		if err != nil {
			t.Fatalf("Failed to create dest dir: %v", err)
		}
		defer os.RemoveAll(destDir)

		// Remove write permissions
		if err := os.Chmod(destDir, 0555); err != nil {
			t.Fatalf("Failed to change permissions: %v", err)
		}
		defer os.Chmod(destDir, 0755) // Restore for cleanup

		envFiles := []EnvFile{
			{
				Path:         testEnvFileName,
				AbsolutePath: srcFile,
			},
		}

		// Should fail because can't write to destination
		err = CopyEnvFiles(envFiles, srcDir, destDir)
		if err == nil {
			t.Error("Expected error when writing to read-only directory")
		}
	})

	t.Run("empty file list", func(t *testing.T) {
		// Create directories
		srcDir, err := os.MkdirTemp("", "gw-src-*")
		if err != nil {
			t.Fatalf("Failed to create source dir: %v", err)
		}
		defer os.RemoveAll(srcDir)

		destDir, err := os.MkdirTemp("", "gw-dest-*")
		if err != nil {
			t.Fatalf("Failed to create dest dir: %v", err)
		}
		defer os.RemoveAll(destDir)

		// Copy empty file list
		err = CopyEnvFiles([]EnvFile{}, srcDir, destDir)
		if err != nil {
			t.Errorf("CopyEnvFiles should succeed with empty list: %v", err)
		}
	})
}
