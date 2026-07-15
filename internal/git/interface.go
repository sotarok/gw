package git

// RepositoryReader exposes read-only repository introspection and remote sync.
type RepositoryReader interface {
	IsGitRepository() bool
	GetRepositoryName() (string, error)
	GetOriginalRepositoryName() (string, error)
	GetRepositoryRoot() (string, error)
	GetMainRepositoryRoot() (string, error)
	GetCurrentBranch() (string, error)
	FetchAll() error
}

// WorktreeManager exposes worktree lifecycle operations.
type WorktreeManager interface {
	CreateWorktree(issueNumber, baseBranch string) (string, error)
	CreateWorktreeFromBranch(worktreePath, sourceBranch, targetBranch string) error
	RemoveWorktree(issueNumber string) error
	RemoveWorktreeByPath(worktreePath string) error
	ListWorktrees() ([]WorktreeInfo, error)
	GetWorktreeForIssue(issueNumber string) (*WorktreeInfo, error)
}

// BranchManager exposes branch inspection and deletion.
type BranchManager interface {
	BranchExists(branch string) (bool, error)
	ListAllBranches() ([]string, error)
	DeleteBranch(branch string) error
}

// StatusChecker exposes the safety checks performed before destructive ops.
type StatusChecker interface {
	HasUncommittedChanges(worktreePath string) (bool, error)
	HasUnpushedCommits(worktreePath, currentBranch string) (bool, error)
	IsMergedToBaseBranch(worktreePath, currentBranch, targetBranch string) (bool, error)
}

// EnvFileHandler exposes untracked env file discovery and copying.
type EnvFileHandler interface {
	FindUntrackedEnvFiles(repoPath string) ([]EnvFile, error)
	CopyEnvFiles(envFiles []EnvFile, sourceRoot, destRoot string) error
}

// Interface is the composed surface used by cmd.Dependencies. It aggregates the
// role interfaces above plus the remaining utility operations. Phase 4 will move
// individual commands onto the narrower role interfaces.
type Interface interface {
	RepositoryReader
	WorktreeManager
	BranchManager
	StatusChecker
	EnvFileHandler

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
