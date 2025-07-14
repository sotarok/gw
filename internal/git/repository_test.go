package git

import (
	"os"
	"testing"
)

func TestGetRepositoryName(t *testing.T) {
	t.Run("returns repository name from git repository", func(t *testing.T) {
		// This test will fail initially because we're testing against the actual git command
		// In TDD style, we write the test first, see it fail, then make it pass
		
		name, err := GetRepositoryName()
		
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		// We expect "gw" as the repository name since we're in the gw directory
		expected := "gw"
		if name != expected {
			t.Errorf("expected repository name %q, got %q", expected, name)
		}
	})
	
	t.Run("returns error when not in git repository", func(t *testing.T) {
		// Create a temporary directory that's not a git repo
		tempDir, err := os.MkdirTemp("", "test-non-git")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		// Change to temp directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer os.Chdir(originalDir)
		
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}
		
		// Now test should return an error
		_, err = GetRepositoryName()
		if err == nil {
			t.Error("expected error when not in git repository, got nil")
		}
	})
}

func TestIsGitRepository(t *testing.T) {
	t.Run("returns true in git repository", func(t *testing.T) {
		// We're running this test in a git repository
		if !IsGitRepository() {
			t.Error("expected IsGitRepository to return true in git repository")
		}
	})
	
	t.Run("returns false outside git repository", func(t *testing.T) {
		// Create a temporary directory that's not a git repo
		tempDir, err := os.MkdirTemp("", "test-non-git")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		// Change to temp directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer os.Chdir(originalDir)
		
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}
		
		if IsGitRepository() {
			t.Error("expected IsGitRepository to return false outside git repository")
		}
	})
}