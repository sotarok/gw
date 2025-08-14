package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefaultClient(t *testing.T) {
	client := NewDefaultClient()
	if client == nil {
		t.Error("NewDefaultClient() returned nil")
	}

	// Verify it implements Interface
	var _ Interface = client
}

func TestDefaultClient_IsGitRepository(t *testing.T) {
	// Create a temporary git repository
	tempDir, err := os.MkdirTemp("", "test-default-client")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Test non-git directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	client := NewDefaultClient()
	if client.IsGitRepository() {
		t.Error("IsGitRepository() should return false for non-git directory")
	}

	// Initialize git repo
	if err := RunCommand("git init"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	if !client.IsGitRepository() {
		t.Error("IsGitRepository() should return true for git directory")
	}
}

func TestDefaultClient_GetRepositoryName(t *testing.T) {
	// Create a temporary git repository
	tempDir, err := os.MkdirTemp("", "test-repo-name")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoDir := filepath.Join(tempDir, "my-test-repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	// Initialize git repo
	if err := RunCommand("git init"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	client := NewDefaultClient()
	name, err := client.GetRepositoryName()
	if err != nil {
		t.Fatalf("GetRepositoryName() failed: %v", err)
	}

	if name != "my-test-repo" {
		t.Errorf("expected repository name 'my-test-repo', got %s", name)
	}
}

func TestDefaultClient_GetCurrentBranch(t *testing.T) {
	// Create a temporary git repository
	tempDir, err := os.MkdirTemp("", "test-current-branch")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	// Initialize git repo
	if err := RunCommand("git init"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git
	RunCommand("git config user.email 'test@example.com'")
	RunCommand("git config user.name 'Test User'")

	// Create initial commit
	if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := RunCommand("git add . && git commit -m 'initial'"); err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}

	client := NewDefaultClient()
	branch, err := client.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch() failed: %v", err)
	}

	// Should be on default branch (main or master)
	const mainBranch = "main"
	const masterBranch = "master"
	if branch != mainBranch && branch != masterBranch {
		t.Errorf("expected 'main' or 'master', got %s", branch)
	}
}

func TestDefaultClient_BranchOperations(t *testing.T) {
	// Create a temporary git repository
	tempDir, err := os.MkdirTemp("", "test-branch-ops")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	// Initialize git repo
	if err := RunCommand("git init"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git
	RunCommand("git config user.email 'test@example.com'")
	RunCommand("git config user.name 'Test User'")

	// Create initial commit
	if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := RunCommand("git add . && git commit -m 'initial'"); err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}

	client := NewDefaultClient()

	// Test BranchExists
	const mainBranch = "main"
	const masterBranch = "master"
	exists, err := client.BranchExists(mainBranch)
	if err != nil {
		t.Fatalf("BranchExists() failed: %v", err)
	}
	if !exists {
		exists, _ = client.BranchExists(masterBranch)
		if !exists {
			t.Error("BranchExists() should return true for default branch")
		}
	}

	// Test ListAllBranches
	branches, err := client.ListAllBranches()
	if err != nil {
		t.Fatalf("ListAllBranches() failed: %v", err)
	}
	if len(branches) == 0 {
		t.Error("ListAllBranches() should return at least one branch")
	}

	// Create a test branch
	if err := RunCommand("git checkout -b test-branch"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}
	if err := RunCommand("git checkout -"); err != nil {
		t.Fatalf("failed to switch back: %v", err)
	}

	// Test DeleteBranch
	err = client.DeleteBranch("test-branch")
	if err != nil {
		t.Fatalf("DeleteBranch() failed: %v", err)
	}

	// Verify branch is deleted
	exists, _ = client.BranchExists("test-branch")
	if exists {
		t.Error("Branch should not exist after deletion")
	}
}

func TestDefaultClient_StatusOperations(t *testing.T) {
	// Create a temporary git repository
	tempDir, err := os.MkdirTemp("", "test-status-ops")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	// Initialize git repo
	if err := RunCommand("git init"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git
	RunCommand("git config user.email 'test@example.com'")
	RunCommand("git config user.name 'Test User'")

	// Create initial commit
	if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := RunCommand("git add . && git commit -m 'initial'"); err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}

	client := NewDefaultClient()

	// Test HasUncommittedChanges - should be false for clean repo
	hasChanges, err := client.HasUncommittedChanges()
	if err != nil {
		t.Fatalf("HasUncommittedChanges() failed: %v", err)
	}
	if hasChanges {
		t.Error("HasUncommittedChanges() should return false for clean repo")
	}

	// Make a change
	if err := os.WriteFile("test.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Test HasUncommittedChanges - should be true with untracked file
	hasChanges, err = client.HasUncommittedChanges()
	if err != nil {
		t.Fatalf("HasUncommittedChanges() failed: %v", err)
	}
	if !hasChanges {
		t.Error("HasUncommittedChanges() should return true with untracked file")
	}

	// Clean up
	os.Remove("test.txt")

	// Test HasUnpushedCommits - should return true (no upstream)
	hasUnpushed, err := client.HasUnpushedCommits()
	if err != nil {
		t.Fatalf("HasUnpushedCommits() failed: %v", err)
	}
	if !hasUnpushed {
		t.Error("HasUnpushedCommits() should return true when no upstream")
	}
}

func TestDefaultClient_WorktreeOperations(t *testing.T) {
	// Save original directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create a temporary git repository
	tempDir, err := os.MkdirTemp("", "test-worktree-ops")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	// Initialize git repo
	if err := RunCommand("git init"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git
	RunCommand("git config user.email 'test@example.com'")
	RunCommand("git config user.name 'Test User'")

	// Create initial commit
	if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := RunCommand("git add . && git commit -m 'initial'"); err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}

	// Create main branch
	RunCommand("git checkout -b main")

	client := NewDefaultClient()

	// Test CreateWorktree
	worktreePath, err := client.CreateWorktree("123", "main")
	if err != nil {
		t.Fatalf("CreateWorktree() failed: %v", err)
	}
	defer client.RemoveWorktreeByPath(worktreePath)

	// Test ListWorktrees
	worktrees, err := client.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() failed: %v", err)
	}
	if len(worktrees) < 2 { // main + new worktree
		t.Error("ListWorktrees() should return at least 2 worktrees")
	}

	// Test GetWorktreeForIssue
	wt, err := client.GetWorktreeForIssue("123")
	if err != nil {
		t.Fatalf("GetWorktreeForIssue() failed: %v", err)
	}
	if wt.Branch != "123/impl" {
		t.Errorf("expected branch '123/impl', got %s", wt.Branch)
	}

	// Test RemoveWorktree
	err = client.RemoveWorktree("123")
	if err != nil {
		t.Fatalf("RemoveWorktree() failed: %v", err)
	}

	// Verify worktree is removed
	_, err = client.GetWorktreeForIssue("123")
	if err == nil {
		t.Error("GetWorktreeForIssue() should fail after removal")
	}
}

func TestDefaultClient_EnvFileOperations(t *testing.T) {
	// Create source directory
	srcDir, err := os.MkdirTemp("", "test-env-src")
	if err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	// Create destination directory
	destDir, err := os.MkdirTemp("", "test-env-dest")
	if err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Initialize git repo in source
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(srcDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	if err := RunCommand("git init"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git
	RunCommand("git config user.email 'test@example.com'")
	RunCommand("git config user.name 'Test User'")

	// Create env files
	if err := os.WriteFile(".env", []byte("TEST=true"), 0644); err != nil {
		t.Fatalf("failed to create .env: %v", err)
	}
	if err := os.WriteFile(".env.local", []byte("LOCAL=true"), 0644); err != nil {
		t.Fatalf("failed to create .env.local: %v", err)
	}

	// Track one file
	if err := os.WriteFile(".env.example", []byte("EXAMPLE=true"), 0644); err != nil {
		t.Fatalf("failed to create .env.example: %v", err)
	}
	RunCommand("git add .env.example && git commit -m 'add example'")

	client := NewDefaultClient()

	// Test FindUntrackedEnvFiles
	envFiles, err := client.FindUntrackedEnvFiles(srcDir)
	if err != nil {
		t.Fatalf("FindUntrackedEnvFiles() failed: %v", err)
	}

	// Should find .env and .env.local but not .env.example
	if len(envFiles) != 2 {
		t.Errorf("expected 2 untracked env files, got %d", len(envFiles))
	}

	// Test CopyEnvFiles
	err = client.CopyEnvFiles(envFiles, srcDir, destDir)
	if err != nil {
		t.Fatalf("CopyEnvFiles() failed: %v", err)
	}

	// Verify files were copied
	for _, ef := range envFiles {
		destPath := filepath.Join(destDir, ef.Path)
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			t.Errorf("file %s was not copied to destination", ef.Path)
		}
	}
}

func TestDefaultClient_UtilityOperations(t *testing.T) {
	client := NewDefaultClient()

	// Test SanitizeBranchNameForDirectory
	sanitized := client.SanitizeBranchNameForDirectory("feature/test-branch")
	if sanitized != "feature-test-branch" {
		t.Errorf("expected 'feature-test-branch', got %s", sanitized)
	}

	// Test RunCommand
	err := client.RunCommand("echo test")
	if err != nil {
		t.Errorf("RunCommand() failed: %v", err)
	}
}

func TestDefaultClient_IsMergedToOrigin(t *testing.T) {
	// This is a complex test that requires a remote repository
	// For now, just test that the method exists and can be called
	client := NewDefaultClient()

	// Create a temporary git repository
	tempDir, err := os.MkdirTemp("", "test-merged")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	// Initialize git repo
	if err := RunCommand("git init"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// The method should at least not panic
	_, _ = client.IsMergedToOrigin("main")
}

func TestDefaultClient_CreateWorktreeFromBranch(t *testing.T) {
	// Save original directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	// Create a temporary git repository
	tempDir, err := os.MkdirTemp("", "test-worktree-from-branch")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change dir: %v", err)
	}

	// Initialize git repo
	if err := RunCommand("git init"); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git
	RunCommand("git config user.email 'test@example.com'")
	RunCommand("git config user.name 'Test User'")

	// Create initial commit
	if err := os.WriteFile("README.md", []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := RunCommand("git add . && git commit -m 'initial'"); err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}

	// Create a feature branch
	if err := RunCommand("git checkout -b feature-branch"); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}
	if err := os.WriteFile("feature.txt", []byte("feature"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if err := RunCommand("git add . && git commit -m 'feature'"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	if err := RunCommand("git checkout -"); err != nil {
		t.Fatalf("failed to switch back: %v", err)
	}

	client := NewDefaultClient()

	// Test CreateWorktreeFromBranch
	worktreePath := filepath.Join(tempDir, "..", "test-worktree")
	err = client.CreateWorktreeFromBranch(worktreePath, "feature-branch", "")
	if err != nil {
		t.Fatalf("CreateWorktreeFromBranch() failed: %v", err)
	}

	// Clean up
	RunCommand("git worktree remove " + worktreePath)
}
