package detect

// Interface defines the package detection operations
type Interface interface {
	DetectPackageManager(path string) (*PackageManager, error)
	RunSetup(path string) error
}

// DefaultDetector implements Interface using actual detection
type DefaultDetector struct{}

// Ensure DefaultDetector implements Interface
var _ Interface = (*DefaultDetector)(nil)

// NewDefaultDetector creates a new default detector
func NewDefaultDetector() *DefaultDetector {
	return &DefaultDetector{}
}

func (d *DefaultDetector) DetectPackageManager(path string) (*PackageManager, error) {
	return DetectPackageManager(path)
}

func (d *DefaultDetector) RunSetup(path string) error {
	return RunSetup(path)
}
