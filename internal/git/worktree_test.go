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
