package git

// Interface defines the git operations used by the application
type Interface interface {
	// Repository operations
	IsGitRepository() bool
	GetRepositoryName() (string, error)
	GetOriginalRepositoryName() (string, error)
	GetRepositoryRoot() (string, error)
	GetCurrentBranch() (string, error)
	FetchAll() error

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
	DeleteBranch(branch string) error

	// Status operations
	HasUncommittedChanges(worktreePath string) (bool, error)
	HasUnpushedCommits(worktreePath, currentBranch string) (bool, error)
	IsMergedToBaseBranch(worktreePath, currentBranch, targetBranch string) (bool, error)

	// Environment file operations
	FindUntrackedEnvFiles(repoPath string) ([]EnvFile, error)
	CopyEnvFiles(envFiles []EnvFile, sourceRoot, destRoot string) error

	// Utility operations
	RunCommand(command string) error
	SanitizeBranchNameForDirectory(branch string) string
}

// Client implements git operations by invoking the git CLI through a runner.
type Client struct {
	r runner
}

// Ensure Client implements Interface
var _ Interface = (*Client)(nil)

// NewClient creates a new git client backed by the real git executable.
func NewClient() *Client {
	return &Client{r: execRunner{}}
}

// SanitizeBranchNameForDirectory is a thin method wrapper over the package-level
// pure function so Client satisfies Interface. Callers with a concrete
// dependency may call the package function directly.
func (c *Client) SanitizeBranchNameForDirectory(branch string) string {
	return SanitizeBranchNameForDirectory(branch)
}
