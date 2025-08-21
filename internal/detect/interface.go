package detect

// Interface defines the package detection operations
type Interface interface {
	DetectPackageManager(path string) (*PackageManager, error)
	RunSetup(path string) error
}

// DefaultDetector implements Interface using actual detection
type DefaultDetector struct {
	executor CommandExecutor
}

// Ensure DefaultDetector implements Interface
var _ Interface = (*DefaultDetector)(nil)

// NewDefaultDetector creates a new default detector
func NewDefaultDetector() *DefaultDetector {
	return &DefaultDetector{
		executor: NewDefaultExecutor(),
	}
}

// NewDefaultDetectorWithExecutor creates a detector with a custom executor (for testing)
func NewDefaultDetectorWithExecutor(executor CommandExecutor) *DefaultDetector {
	return &DefaultDetector{
		executor: executor,
	}
}

func (d *DefaultDetector) DetectPackageManager(path string) (*PackageManager, error) {
	return DetectPackageManager(path)
}

func (d *DefaultDetector) RunSetup(path string) error {
	return RunSetupWithExecutor(path, d.executor)
}
