package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWorktree(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		baseBranch         string
		expectedBranchName string
		expectedDirSuffix  string
	}{
		{
			name:               "creates worktree with issue number",
			input:              "123",
			baseBranch:         "main",
			expectedBranchName: "123/impl",
			expectedDirSuffix:  "123",
		},
		{
			name:               "creates worktree with full branch name",
			input:              "476/impl-migration-script",
			baseBranch:         "main",
			expectedBranchName: "476/impl-migration-script",
			expectedDirSuffix:  "476-impl-migration-script",
		},
		{
			name:               "creates worktree with feature branch",
			input:              "feature/new-feature",
			baseBranch:         "main",
			expectedBranchName: "feature/new-feature",
			expectedDirSuffix:  "feature-new-feature",
		},
		{
			name:               "handles branch name without slash",
			input:              "hotfix",
			baseBranch:         "main",
			expectedBranchName: "hotfix/impl",
			expectedDirSuffix:  "hotfix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test repository
			tempDir := t.TempDir()
			repoDir := filepath.Join(tempDir, "test-repo")
			if err := os.MkdirAll(repoDir, 0755); err != nil {
				t.Fatalf("Failed to create repo dir: %v", err)
			}

			// Initialize git repo
			if err := os.Chdir(repoDir); err != nil {
				t.Fatalf("Failed to change to repo dir: %v", err)
			}

			// Initialize repository with main branch
			cmd := exec.Command("git", "init", "-b", "main")
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to init git repo: %v", err)
			}

			// Configure git
			exec.Command("git", "config", "user.email", "test@example.com").Run()
			exec.Command("git", "config", "user.name", "Test User").Run()

			// Create initial commit
			if err := os.WriteFile("README.md", []byte("# Test"), 0644); err != nil {
				t.Fatalf("Failed to write README: %v", err)
			}
			exec.Command("git", "add", ".").Run()
			exec.Command("git", "commit", "-m", "initial commit").Run()

			// Call CreateWorktree
			worktreePath, err := CreateWorktree(tt.input, tt.baseBranch)
			if err != nil {
				t.Errorf("CreateWorktree failed: %v", err)
				return
			}

			// Check if worktree was created with correct path
			if !strings.HasSuffix(worktreePath, tt.expectedDirSuffix) {
				t.Errorf("Expected worktree path to end with %s, got %s", tt.expectedDirSuffix, worktreePath)
			}

			// Check if branch was created correctly
			cmd = exec.Command("git", "worktree", "list", "--porcelain")
			output, err := cmd.Output()
			if err != nil {
				t.Errorf("Failed to list worktrees: %v", err)
				return
			}

			outputStr := string(output)
			if !strings.Contains(outputStr, tt.expectedBranchName) {
				t.Errorf("Expected branch name %s not found in worktree list:\n%s", tt.expectedBranchName, outputStr)
			}

			// Cleanup
			exec.Command("git", "worktree", "remove", worktreePath).Run()
		})
	}
}

func TestDetermineWorktreeNames(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedBranchName string
		expectedDirSuffix  string
	}{
		{
			name:               "issue number only",
			input:              "123",
			expectedBranchName: "123/impl",
			expectedDirSuffix:  "123",
		},
		{
			name:               "branch name with slash",
			input:              "476/impl-migration-script",
			expectedBranchName: "476/impl-migration-script",
			expectedDirSuffix:  "476-impl-migration-script",
		},
		{
			name:               "feature branch",
			input:              "feature/new-feature",
			expectedBranchName: "feature/new-feature",
			expectedDirSuffix:  "feature-new-feature",
		},
		{
			name:               "branch without slash",
			input:              "hotfix",
			expectedBranchName: "hotfix/impl",
			expectedDirSuffix:  "hotfix",
		},
		{
			name:               "complex branch name",
			input:              "bugfix/issue-789/fix-bug",
			expectedBranchName: "bugfix/issue-789/fix-bug",
			expectedDirSuffix:  "bugfix-issue-789-fix-bug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branchName, dirSuffix := DetermineWorktreeNames(tt.input)

			if branchName != tt.expectedBranchName {
				t.Errorf("Expected branch name %s, got %s", tt.expectedBranchName, branchName)
			}

			if dirSuffix != tt.expectedDirSuffix {
				t.Errorf("Expected dir suffix %s, got %s", tt.expectedDirSuffix, dirSuffix)
			}
		})
	}
}

func TestListWorktrees(t *testing.T) {
	t.Run("lists worktrees correctly", func(t *testing.T) {
		// Create a temporary git repository
		tempDir, err := os.MkdirTemp("", "test-list-worktrees")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save and restore working directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer os.Chdir(originalDir)

		// Initialize git repo
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		if err := RunCommand("git init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		// Configure git
		if err := RunCommand("git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
			t.Fatalf("failed to configure git: %v", err)
		}

		// Create initial commit
		if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'initial commit'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// List worktrees (should have main worktree)
		worktrees, err := ListWorktrees()
		if err != nil {
			t.Fatalf("failed to list worktrees: %v", err)
		}

		if len(worktrees) != 1 {
			t.Errorf("expected 1 worktree, got %d", len(worktrees))
		}

		// Check that main worktree is marked as current
		if len(worktrees) > 0 && !worktrees[0].IsCurrent {
			t.Error("main worktree not marked as current")
		}
	})
}

func TestRemoveWorktree(t *testing.T) {
	t.Run("removes worktree successfully", func(t *testing.T) {
		// Create a temporary git repository
		tempDir, err := os.MkdirTemp("", "test-remove-worktree")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save and restore working directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer os.Chdir(originalDir)

		// Initialize git repo
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		if err := RunCommand("git init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		// Configure git
		if err := RunCommand("git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
			t.Fatalf("failed to configure git: %v", err)
		}

		// Create initial commit
		if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'initial commit'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Create main branch
		if err := RunCommand("git checkout -b main"); err != nil {
			// Might already be on main
			_ = RunCommand("git branch -m main")
		}

		// Create worktree
		worktreePath, err := CreateWorktree("456", "main")
		if err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		// Verify worktree exists
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Fatalf("worktree was not created at %s", worktreePath)
		}

		// Remove worktree
		if err := RemoveWorktree("456"); err != nil {
			t.Fatalf("failed to remove worktree: %v", err)
		}

		// Verify worktree no longer exists
		if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
			t.Error("worktree directory still exists after removal")
		}
	})

	t.Run("fails when not in git repository", func(t *testing.T) {
		// Create a non-git directory
		tempDir, err := os.MkdirTemp("", "test-non-git")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save and restore working directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer os.Chdir(originalDir)

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		// Try to remove worktree
		err = RemoveWorktree("123")
		if err == nil {
			t.Error("expected error when not in git repository")
		}
		if !strings.Contains(err.Error(), "not in a git repository") {
			t.Errorf("expected 'not in a git repository' error, got: %v", err)
		}
	})
}

func TestRemoveWorktreeByPath(t *testing.T) {
	t.Run("removes worktree by path successfully", func(t *testing.T) {
		// Create a temporary git repository
		tempDir, err := os.MkdirTemp("", "test-remove-by-path")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save and restore working directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer os.Chdir(originalDir)

		// Initialize git repo
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		if err := RunCommand("git init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		// Configure git
		if err := RunCommand("git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
			t.Fatalf("failed to configure git: %v", err)
		}

		// Create initial commit
		if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'initial commit'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Create worktree with specific path
		worktreePath := filepath.Join(tempDir, "..", "test-worktree-path")
		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "test-branch")
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		// Verify worktree exists
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Fatalf("worktree was not created at %s", worktreePath)
		}

		// Remove worktree by path
		if err := RemoveWorktreeByPath(worktreePath); err != nil {
			t.Fatalf("failed to remove worktree by path: %v", err)
		}

		// Verify worktree no longer exists
		if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
			t.Error("worktree directory still exists after removal")
		}
	})
}

func TestGetWorktreeForIssue(t *testing.T) {
	t.Run("finds existing worktree", func(t *testing.T) {
		// Create a temporary git repository
		tempDir, err := os.MkdirTemp("", "test-get-worktree")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save and restore working directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer os.Chdir(originalDir)

		// Initialize git repo
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		if err := RunCommand("git init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		// Configure git
		if err := RunCommand("git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
			t.Fatalf("failed to configure git: %v", err)
		}

		// Create initial commit
		if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'initial commit'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Create main branch
		if err := RunCommand("git checkout -b main"); err != nil {
			// Might already be on main
			_ = RunCommand("git branch -m main")
		}

		// Create worktree
		worktreePath, err := CreateWorktree("999", "main")
		if err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}
		defer RemoveWorktreeByPath(worktreePath)

		// Find worktree
		wt, err := GetWorktreeForIssue("999")
		if err != nil {
			t.Fatalf("failed to get worktree: %v", err)
		}

		if wt.Branch != "999/impl" {
			t.Errorf("expected branch '999/impl', got %q", wt.Branch)
		}

		if !strings.Contains(wt.Path, "999") {
			t.Errorf("expected path to contain '999', got %q", wt.Path)
		}
	})

	t.Run("returns error for non-existent worktree", func(t *testing.T) {
		// Create a temporary git repository
		tempDir, err := os.MkdirTemp("", "test-nonexistent-worktree")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save and restore working directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer os.Chdir(originalDir)

		// Initialize git repo
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		if err := RunCommand("git init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		// Configure git
		if err := RunCommand("git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
			t.Fatalf("failed to configure git: %v", err)
		}

		// Create initial commit
		if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'initial commit'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Try to find non-existent worktree
		_, err = GetWorktreeForIssue("nonexistent-12345")
		if err == nil {
			t.Error("expected error for non-existent worktree")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})
}

func TestCreateWorktreeFromBranch(t *testing.T) {
	t.Run("creates worktree from local branch", func(t *testing.T) {
		// Create a temporary git repository
		tempDir, err := os.MkdirTemp("", "test-create-from-branch")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save and restore working directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer os.Chdir(originalDir)

		// Initialize git repo
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		if err := RunCommand("git init"); err != nil {
			t.Fatalf("failed to init git repo: %v", err)
		}

		// Configure git
		if err := RunCommand("git config user.email 'test@example.com' && git config user.name 'Test User'"); err != nil {
			t.Fatalf("failed to configure git: %v", err)
		}

		// Create initial commit
		if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'initial commit'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Create a feature branch
		if err := RunCommand("git checkout -b feature-branch"); err != nil {
			t.Fatalf("failed to create feature branch: %v", err)
		}

		// Add a commit to feature branch
		if err := os.WriteFile("feature.txt", []byte("feature"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		if err := RunCommand("git add . && git commit -m 'feature commit'"); err != nil {
			t.Fatalf("failed to create commit: %v", err)
		}

		// Switch back to main
		if err := RunCommand("git checkout -"); err != nil {
			t.Fatalf("failed to switch back: %v", err)
		}

		// Create worktree from feature branch
		worktreePath := filepath.Join(tempDir, "..", "test-worktree")
		err = CreateWorktreeFromBranch(worktreePath, "feature-branch", "")
		if err != nil {
			t.Fatalf("failed to create worktree from branch: %v", err)
		}
		defer func() {
			cmd := exec.Command("git", "worktree", "remove", worktreePath)
			_ = cmd.Run()
		}()

		// Verify worktree exists
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Error("worktree directory was not created")
		}

		// Verify feature.txt exists in worktree
		featurePath := filepath.Join(worktreePath, "feature.txt")
		if _, err := os.Stat(featurePath); os.IsNotExist(err) {
			t.Error("feature.txt not found in worktree")
		}
	})

	t.Run("fails when not in git repository", func(t *testing.T) {
		// Create a non-git directory
		tempDir, err := os.MkdirTemp("", "test-non-git")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save and restore working directory
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get current dir: %v", err)
		}
		defer os.Chdir(originalDir)

		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change dir: %v", err)
		}

		// Try to create worktree
		err = CreateWorktreeFromBranch("/tmp/test", "main", "new-branch")
		if err == nil {
			t.Error("expected error when not in git repository")
		}
		if !strings.Contains(err.Error(), "not in a git repository") {
			t.Errorf("expected 'not in a git repository' error, got: %v", err)
		}
	})
}
