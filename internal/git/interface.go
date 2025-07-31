package git

// Interface defines the git operations used by the application
type Interface interface {
	// Repository operations
	IsGitRepository() bool
	GetRepositoryName() (string, error)
	GetCurrentBranch() (string, error)

	// Worktree operations
	CreateWorktree(issueNumber, baseBranch string) (string, error)
	CreateWorktreeFromBranch(worktreePath, sourceBranch, targetBranch string) error
	RemoveWorktree(issueNumber string) error
	RemoveWorktreeByPath(worktreePath string) error
	ListWorktrees() ([]WorktreeInfo, error)
	GetWorktreeForIssue(issueNumber string) (*WorktreeInfo, error)

	// Branch operations
	BranchExists(branch string) (bool, error)
	ListAllBranches() ([]string, error)

	// Status operations
	HasUncommittedChanges() (bool, error)
	HasUnpushedCommits() (bool, error)

	// Environment file operations
	FindUntrackedEnvFiles(repoPath string) ([]EnvFile, error)
	CopyEnvFiles(envFiles []EnvFile, sourceRoot, destRoot string) error

	// Utility operations
	RunCommand(command string) error
	SanitizeBranchNameForDirectory(branch string) string
}

// DefaultClient implements Interface using actual git commands
type DefaultClient struct{}

// Ensure DefaultClient implements Interface
var _ Interface = (*DefaultClient)(nil)

// NewDefaultClient creates a new default git client
func NewDefaultClient() *DefaultClient {
	return &DefaultClient{}
}

// Repository operations
func (c *DefaultClient) IsGitRepository() bool {
	return IsGitRepository()
}

func (c *DefaultClient) GetRepositoryName() (string, error) {
	return GetRepositoryName()
}

func (c *DefaultClient) GetCurrentBranch() (string, error) {
	return GetCurrentBranch()
}

// Worktree operations
func (c *DefaultClient) CreateWorktree(issueNumber, baseBranch string) (string, error) {
	return CreateWorktree(issueNumber, baseBranch)
}

func (c *DefaultClient) CreateWorktreeFromBranch(worktreePath, sourceBranch, targetBranch string) error {
	return CreateWorktreeFromBranch(worktreePath, sourceBranch, targetBranch)
}

func (c *DefaultClient) RemoveWorktree(issueNumber string) error {
	return RemoveWorktree(issueNumber)
}

func (c *DefaultClient) RemoveWorktreeByPath(worktreePath string) error {
	return RemoveWorktreeByPath(worktreePath)
}

func (c *DefaultClient) ListWorktrees() ([]WorktreeInfo, error) {
	return ListWorktrees()
}

func (c *DefaultClient) GetWorktreeForIssue(issueNumber string) (*WorktreeInfo, error) {
	return GetWorktreeForIssue(issueNumber)
}

// Branch operations
func (c *DefaultClient) BranchExists(branch string) (bool, error) {
	return BranchExists(branch)
}

func (c *DefaultClient) ListAllBranches() ([]string, error) {
	return ListAllBranches()
}

// Status operations
func (c *DefaultClient) HasUncommittedChanges() (bool, error) {
	return HasUncommittedChanges()
}

func (c *DefaultClient) HasUnpushedCommits() (bool, error) {
	return HasUnpushedCommits()
}

// Environment file operations
func (c *DefaultClient) FindUntrackedEnvFiles(repoPath string) ([]EnvFile, error) {
	return FindUntrackedEnvFiles(repoPath)
}

func (c *DefaultClient) CopyEnvFiles(envFiles []EnvFile, sourceRoot, destRoot string) error {
	return CopyEnvFiles(envFiles, sourceRoot, destRoot)
}

// Utility operations
func (c *DefaultClient) RunCommand(command string) error {
	return RunCommand(command)
}

func (c *DefaultClient) SanitizeBranchNameForDirectory(branch string) string {
	return SanitizeBranchNameForDirectory(branch)
}
