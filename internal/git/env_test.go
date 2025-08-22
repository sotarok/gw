package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to run git commands in tests
func runGitCommand(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run git %v: %v", args, err)
	}
}

const (
	defaultMainBranch   = "main"
	defaultMasterBranch = "master"
)

// Helper function to get the default branch name (main or master)
func getDefaultBranchName(t *testing.T, dir string) string {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		// If we can't get the branch name, assume main
		return defaultMainBranch
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return defaultMainBranch
	}
	return branch
}

func TestFindUntrackedEnvFiles(t *testing.T) {
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

	// Create directory structure
	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	// Create test files
	testFiles := map[string]bool{
		".env":             false, // untracked
		".env.local":       false, // untracked
		".env.example":     true,  // tracked
		"app/.env":         false, // untracked
		"app/.env.local":   false, // untracked
		"app/.env.example": true,  // tracked
		"app/.envrc":       false, // untracked
	}

	for file, tracked := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}

		if tracked {
			runGitCommand(t, tmpDir, "add", file)
		}
	}

	// Commit tracked files
	runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

	// Find untracked env files
	envFiles, err := FindUntrackedEnvFiles(tmpDir)
	if err != nil {
		t.Fatalf("FindUntrackedEnvFiles failed: %v", err)
	}

	// Expected untracked files
	expectedFiles := map[string]bool{
		".env":           true,
		".env.local":     true,
		"app/.env":       true,
		"app/.env.local": true,
		"app/.envrc":     true,
	}

	// Check if all expected files are found
	foundFiles := make(map[string]bool)
	for _, envFile := range envFiles {
		foundFiles[envFile.Path] = true
	}

	for expectedFile := range expectedFiles {
		if !foundFiles[expectedFile] {
			t.Errorf("Expected to find %s but it was not found", expectedFile)
		}
	}

	// Check if no tracked files are included
	if len(envFiles) != len(expectedFiles) {
		t.Errorf("Expected %d untracked files, got %d", len(expectedFiles), len(envFiles))
	}
}

func TestFindUntrackedEnvFilesWithNodeModules(t *testing.T) {
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

	// Create node_modules directory
	nodeModulesDir := filepath.Join(tmpDir, "node_modules", "some-package")
	if err := os.MkdirAll(nodeModulesDir, 0755); err != nil {
		t.Fatalf("Failed to create node_modules dir: %v", err)
	}

	// Create .env files
	os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(nodeModulesDir, ".env"), []byte("test"), 0644)

	// Find untracked env files
	envFiles, err := FindUntrackedEnvFiles(tmpDir)
	if err != nil {
		t.Fatalf("FindUntrackedEnvFiles failed: %v", err)
	}

	// Should only find the root .env, not the one in node_modules
	if len(envFiles) != 1 {
		t.Errorf("Expected 1 untracked file, got %d", len(envFiles))
	}

	if len(envFiles) > 0 && envFiles[0].Path != ".env" {
		t.Errorf("Expected to find .env, got %s", envFiles[0].Path)
	}
}

func TestFindUntrackedEnvFilesExcludesAllTrackedFiles(t *testing.T) {
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

	// Create directory structure
	dirs := []string{
		"apps/frontend",
		"apps/backend",
		"apps/service",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create test files - mix of tracked and untracked
	testFiles := map[string]bool{
		"apps/frontend/.env.local":        false, // untracked
		"apps/frontend/.env.example":      true,  // tracked
		"apps/backend/.env":               false, // untracked
		"apps/backend/.env.local.example": true,  // tracked
		"apps/service/.env.production":    false, // untracked
		"apps/service/.env.example":       true,  // tracked
	}

	for file, tracked := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}

		if tracked {
			runGitCommand(t, tmpDir, "add", file)
		}
	}

	// Commit tracked files
	runGitCommand(t, tmpDir, "commit", "-m", "Initial commit")

	// Find untracked env files
	envFiles, err := FindUntrackedEnvFiles(tmpDir)
	if err != nil {
		t.Fatalf("FindUntrackedEnvFiles failed: %v", err)
	}

	// Expected untracked files only
	expectedUntracked := []string{
		"apps/frontend/.env.local",
		"apps/backend/.env",
		"apps/service/.env.production",
	}

	// Tracked files that should NOT be found
	trackedFiles := []string{
		"apps/frontend/.env.example",
		"apps/backend/.env.local.example",
		"apps/service/.env.example",
	}

	// Check if all expected untracked files are found
	foundFiles := make(map[string]bool)
	for _, envFile := range envFiles {
		foundFiles[envFile.Path] = true
		t.Logf("Found file: %s", envFile.Path)
	}

	// Verify only untracked files are found
	for _, expectedFile := range expectedUntracked {
		if !foundFiles[expectedFile] {
			t.Errorf("Expected to find untracked file %s but it was not found", expectedFile)
		}
	}

	// Verify tracked files are NOT found
	for _, trackedFile := range trackedFiles {
		if foundFiles[trackedFile] {
			t.Errorf("Tracked file %s should not be found but was included", trackedFile)
		}
	}

	// Verify exact count
	if len(envFiles) != len(expectedUntracked) {
		t.Errorf("Expected exactly %d untracked files, got %d", len(expectedUntracked), len(envFiles))
	}
}

func TestCopyEnvFiles(t *testing.T) {
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

	// Create source structure with env files
	appDir := filepath.Join(srcDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("Failed to create app dir: %v", err)
	}

	// Create test env files
	testFiles := []struct {
		path    string
		content string
	}{
		{".env", "ROOT_ENV=true"},
		{"app/.env", "APP_ENV=true"},
		{"app/.env.local", "APP_LOCAL=true"},
	}

	envFiles := []EnvFile{}
	for _, tf := range testFiles {
		filePath := filepath.Join(srcDir, tf.path)
		if err := os.WriteFile(filePath, []byte(tf.content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", tf.path, err)
		}
		envFiles = append(envFiles, EnvFile{
			Path:         tf.path,
			AbsolutePath: filePath,
		})
	}

	// Copy env files
	err = CopyEnvFiles(envFiles, srcDir, destDir)
	if err != nil {
		t.Fatalf("CopyEnvFiles failed: %v", err)
	}

	// Verify copied files
	for _, tf := range testFiles {
		destPath := filepath.Join(destDir, tf.path)
		content, err := os.ReadFile(destPath)
		if err != nil {
			t.Errorf("Failed to read copied file %s: %v", tf.path, err)
			continue
		}
		if string(content) != tf.content {
			t.Errorf("Content mismatch for %s. Expected: %s, Got: %s", tf.path, tf.content, string(content))
		}
	}
}
