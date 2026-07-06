package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sotarok/gw/internal/detect"
	"github.com/sotarok/gw/internal/git"
	"github.com/sotarok/gw/internal/ui"
)

// Mock implementations

const testBranch123 = "123/impl"
const testBranchFeature = "feature/test"
const testRepoNameShort = "repo"
const testRepoName = "test-repo"

type mockGit struct {
	isGitRepo           bool
	worktreeExists      bool
	createWorktreeError error
	worktreePath        string
	envFiles            []git.EnvFile
	findEnvError        error
	copyEnvError        error

	// Override functions for custom behavior
	FetchAllFn              func() error
	BranchExistsFn          func(string) (bool, error)
	ListAllBranchesFn       func() ([]string, error)
	GetCurrentBranchFn      func() (string, error)
	GetWorktreeForIssueFn   func(string) (*git.WorktreeInfo, error)
	HasUncommittedChangesFn func() (bool, error)
	HasUnpushedCommitsFn    func() (bool, error)
	IsMergedToBaseBranchFn  func(string) (bool, error)
	// "*AtFn" callbacks receive the same args as the real Git interface
	// methods. Use them when a test needs to vary results by worktree path or
	// branch (the simpler Fn forms above still work for fixed return values).
	HasUncommittedChangesAtFn   func(worktreePath string) (bool, error)
	HasUnpushedCommitsAtFn      func(worktreePath, currentBranch string) (bool, error)
	IsMergedToBaseBranchAtFn    func(worktreePath, currentBranch, targetBranch string) (bool, error)
	DeleteBranchFn              func(string) error
	ListWorktreesFn             func() ([]git.WorktreeInfo, error)
	RemoveWorktreeByPathFn      func(string) error
	GetRepositoryNameFn         func() (string, error)
	GetOriginalRepositoryNameFn func() (string, error)
	GetRepositoryRootFn         func() (string, error)
	GetMainRepositoryRootFn     func() (string, error)
	CreateWorktreeFromBranchFn  func(string, string, string) error
	FindUntrackedEnvFilesFn     func(string) ([]git.EnvFile, error)
	SanitizeBranchNameForDirFn  func(string) string
}

func (m *mockGit) IsGitRepository() bool {
	return m.isGitRepo
}

func (m *mockGit) GetRepositoryName() (string, error) {
	if m.GetRepositoryNameFn != nil {
		return m.GetRepositoryNameFn()
	}
	return testRepoName, nil
}

func (m *mockGit) GetOriginalRepositoryName() (string, error) {
	if m.GetOriginalRepositoryNameFn != nil {
		return m.GetOriginalRepositoryNameFn()
	}
	// Default to the same value as GetRepositoryName: outside a worktree the
	// original repo name and the current repo name are identical, so tests that
	// only set GetRepositoryNameFn keep working.
	return m.GetRepositoryName()
}

func (m *mockGit) GetRepositoryRoot() (string, error) {
	if m.GetRepositoryRootFn != nil {
		return m.GetRepositoryRootFn()
	}
	// Default to cwd so existing tests that rely on relative `../<name>`
	// behavior keep working — they typically chdir into a temp dir first.
	cwd, _ := os.Getwd()
	return cwd, nil
}

func (m *mockGit) GetMainRepositoryRoot() (string, error) {
	if m.GetMainRepositoryRootFn != nil {
		return m.GetMainRepositoryRootFn()
	}
	// Default to the same value as GetRepositoryRoot: tests that don't care
	// about linked-worktree distinctions keep working unchanged.
	return m.GetRepositoryRoot()
}

func (m *mockGit) FetchAll() error {
	if m.FetchAllFn != nil {
		return m.FetchAllFn()
	}
	return nil
}

func (m *mockGit) GetCurrentBranch() (string, error) {
	if m.GetCurrentBranchFn != nil {
		return m.GetCurrentBranchFn()
	}
	return defaultBaseBranch, nil
}

func (m *mockGit) CreateWorktree(issueNumber, baseBranch string) (string, error) {
	if m.createWorktreeError != nil {
		return "", m.createWorktreeError
	}
	return m.worktreePath, nil
}

func (m *mockGit) CreateWorktreeFromBranch(worktreePath, sourceBranch, targetBranch string) error {
	if m.CreateWorktreeFromBranchFn != nil {
		return m.CreateWorktreeFromBranchFn(worktreePath, sourceBranch, targetBranch)
	}
	// Create the worktree directory for the test
	absolutePath, _ := filepath.Abs(worktreePath)
	os.MkdirAll(absolutePath, 0755)
	return nil
}

func (m *mockGit) RemoveWorktree(issueNumber string) error {
	return nil
}

func (m *mockGit) RemoveWorktreeByPath(worktreePath string) error {
	if m.RemoveWorktreeByPathFn != nil {
		return m.RemoveWorktreeByPathFn(worktreePath)
	}
	return nil
}

func (m *mockGit) ListWorktrees() ([]git.WorktreeInfo, error) {
	if m.ListWorktreesFn != nil {
		return m.ListWorktreesFn()
	}
	return nil, nil
}

func (m *mockGit) GetWorktreeForIssue(issueNumber string) (*git.WorktreeInfo, error) {
	if m.GetWorktreeForIssueFn != nil {
		return m.GetWorktreeForIssueFn(issueNumber)
	}
	if m.worktreeExists {
		return &git.WorktreeInfo{Path: "/existing/path"}, nil
	}
	return nil, nil
}

func (m *mockGit) BranchExists(branch string) (bool, error) {
	if m.BranchExistsFn != nil {
		return m.BranchExistsFn(branch)
	}
	return false, nil
}

func (m *mockGit) ListAllBranches() ([]string, error) {
	if m.ListAllBranchesFn != nil {
		return m.ListAllBranchesFn()
	}
	return []string{defaultBaseBranch, "feature"}, nil
}

func (m *mockGit) HasUncommittedChanges(worktreePath string) (bool, error) {
	if m.HasUncommittedChangesAtFn != nil {
		return m.HasUncommittedChangesAtFn(worktreePath)
	}
	if m.HasUncommittedChangesFn != nil {
		return m.HasUncommittedChangesFn()
	}
	return false, nil
}

func (m *mockGit) HasUnpushedCommits(worktreePath, currentBranch string) (bool, error) {
	if m.HasUnpushedCommitsAtFn != nil {
		return m.HasUnpushedCommitsAtFn(worktreePath, currentBranch)
	}
	if m.HasUnpushedCommitsFn != nil {
		return m.HasUnpushedCommitsFn()
	}
	return false, nil
}

func (m *mockGit) IsMergedToBaseBranch(worktreePath, currentBranch, targetBranch string) (bool, error) {
	if m.IsMergedToBaseBranchAtFn != nil {
		return m.IsMergedToBaseBranchAtFn(worktreePath, currentBranch, targetBranch)
	}
	if m.IsMergedToBaseBranchFn != nil {
		return m.IsMergedToBaseBranchFn(targetBranch)
	}
	// Default to merged unless overridden
	return true, nil
}

func (m *mockGit) FindUntrackedEnvFiles(repoPath string) ([]git.EnvFile, error) {
	if m.FindUntrackedEnvFilesFn != nil {
		return m.FindUntrackedEnvFilesFn(repoPath)
	}
	if m.findEnvError != nil {
		return nil, m.findEnvError
	}
	return m.envFiles, nil
}

func (m *mockGit) CopyEnvFiles(envFiles []git.EnvFile, sourceRoot, destRoot string) error {
	if m.copyEnvError != nil {
		return m.copyEnvError
	}
	// Actually copy files for testing
	return git.NewClient().CopyEnvFiles(envFiles, sourceRoot, destRoot)
}

func (m *mockGit) RunCommand(command string) error {
	return nil
}

func (m *mockGit) SanitizeBranchNameForDirectory(branch string) string {
	if m.SanitizeBranchNameForDirFn != nil {
		return m.SanitizeBranchNameForDirFn(branch)
	}
	// Simple sanitization for testing
	return strings.ReplaceAll(branch, "/", "_")
}

func (m *mockGit) DeleteBranch(branch string) error {
	if m.DeleteBranchFn != nil {
		return m.DeleteBranchFn(branch)
	}
	return nil
}

type mockUI struct {
	confirmResult bool
	confirmError  error
	confirmCalled bool

	// Override functions for custom behavior
	ShowSelectorFn   func(string, []ui.SelectorItem) (*ui.SelectorItem, error)
	SelectWorktreeFn func() (*git.WorktreeInfo, error)
}

func (m *mockUI) SelectWorktree() (*git.WorktreeInfo, error) {
	if m.SelectWorktreeFn != nil {
		return m.SelectWorktreeFn()
	}
	return nil, nil
}

func (m *mockUI) ShowSelector(title string, items []ui.SelectorItem) (*ui.SelectorItem, error) {
	if m.ShowSelectorFn != nil {
		return m.ShowSelectorFn(title, items)
	}
	return nil, nil
}

func (m *mockUI) ConfirmPrompt(message string) (bool, error) {
	m.confirmCalled = true
	return m.confirmResult, m.confirmError
}

func (m *mockUI) ShowEnvFilesList(files []string) {
	// Mock implementation - do nothing
}

type mockDetect struct {
	setupError error
}

func (m *mockDetect) DetectPackageManager(path string) (*detect.PackageManager, error) {
	return nil, nil
}

func (m *mockDetect) RunSetup(path string) error {
	return m.setupError
}
