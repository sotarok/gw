package detect

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// PackageManager represents a supported package manager
type PackageManager struct {
	Name        string
	LockFile    string
	InstallCmd  []string
}

var packageManagers = []PackageManager{
	{
		Name:       "npm",
		LockFile:   "package-lock.json",
		InstallCmd: []string{"npm", "install"},
	},
	{
		Name:       "yarn",
		LockFile:   "yarn.lock",
		InstallCmd: []string{"yarn", "install"},
	},
	{
		Name:       "pnpm",
		LockFile:   "pnpm-lock.yaml",
		InstallCmd: []string{"pnpm", "install"},
	},
	{
		Name:       "cargo",
		LockFile:   "Cargo.toml",
		InstallCmd: []string{"cargo", "build"},
	},
	{
		Name:       "go",
		LockFile:   "go.mod",
		InstallCmd: []string{"go", "mod", "download"},
	},
	{
		Name:       "pip",
		LockFile:   "requirements.txt",
		InstallCmd: []string{"pip", "install", "-r", "requirements.txt"},
	},
	{
		Name:       "bundler",
		LockFile:   "Gemfile",
		InstallCmd: []string{"bundle", "install"},
	},
}

// copyPackageManager creates a deep copy of a PackageManager
func copyPackageManager(pm PackageManager) *PackageManager {
	// Create a new slice for InstallCmd to avoid sharing the underlying array
	cmdCopy := make([]string, len(pm.InstallCmd))
	copy(cmdCopy, pm.InstallCmd)
	
	return &PackageManager{
		Name:       pm.Name,
		LockFile:   pm.LockFile,
		InstallCmd: cmdCopy,
	}
}

// DetectPackageManager detects the package manager used in the given directory
func DetectPackageManager(dir string) (*PackageManager, error) {
	// First check for Node.js projects
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		// Check for specific lock files to determine the package manager
		for _, pm := range packageManagers {
			if pm.Name == "npm" || pm.Name == "yarn" || pm.Name == "pnpm" {
				if _, err := os.Stat(filepath.Join(dir, pm.LockFile)); err == nil {
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
		if pm.Name != "npm" && pm.Name != "yarn" && pm.Name != "pnpm" {
			if _, err := os.Stat(filepath.Join(dir, pm.LockFile)); err == nil {
				// Return a deep copy to prevent modifications to the global array
				return copyPackageManager(pm), nil
			}
		}
	}

	return nil, fmt.Errorf("no supported package manager found")
}

// RunSetup runs the setup command for the detected package manager
func RunSetup(dir string) error {
	pm, err := DetectPackageManager(dir)
	if err != nil {
		// No package manager found, but that's okay
		fmt.Println("No package manager detected, skipping setup")
		return nil
	}

	fmt.Printf("Detected %s, running setup...\n", pm.Name)

	cmd := exec.Command(pm.InstallCmd[0], pm.InstallCmd[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s: %w", pm.Name, err)
	}

	fmt.Printf("✓ %s setup completed\n", pm.Name)
	return nil
}