package detect

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

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
	// First check for Node.js projects
	if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
		// Check for specific lock files to determine the package manager
		for _, pm := range packageManagers {
			if pm.Name == npmName || pm.Name == yarnName || pm.Name == pnpmName {
				if _, err := os.Stat(filepath.Join(path, pm.LockFile)); err == nil {
					// Return a deep copy to prevent modifications to the global array
					return copyPackageManager(pm), nil
				}
			}
		}
		// Default to npm if no specific lock file is found
		return copyPackageManager(packageManagers[0]), nil
	}

	// Check for other package managers
	for _, pm := range packageManagers {
		if pm.Name != npmName && pm.Name != yarnName && pm.Name != pnpmName {
			if _, err := os.Stat(filepath.Join(path, pm.LockFile)); err == nil {
				// Return a deep copy to prevent modifications to the global array
				return copyPackageManager(pm), nil
			}
		}
	}

	return nil, fmt.Errorf("no supported package manager found")
}

func (d *DefaultDetector) RunSetup(path string) error {
	pm, err := d.DetectPackageManager(path)
	if err != nil {
		// No package manager found, but that's okay
		fmt.Println("No package manager detected, skipping setup")
		return nil
	}

	fmt.Printf("Detected %s, running setup...\n", pm.Name)

	if err := d.executor.Execute(path, pm.InstallCmd[0], pm.InstallCmd[1:]); err != nil {
		return fmt.Errorf("failed to run %s: %w", pm.Name, err)
	}

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	fmt.Printf("%s %s setup completed\n", successStyle.Render("✓"), pm.Name)
	return nil
}
