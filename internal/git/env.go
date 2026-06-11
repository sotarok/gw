package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	permEnvDir  = 0o755 // directories: rwxr-xr-x
	permEnvFile = 0o600 // env files: rw------- (owner-only, contains secrets)
)

// EnvFile represents an environment file found in the repository
type EnvFile struct {
	Path         string // Relative path from repository root
	AbsolutePath string // Absolute path
}

// skipDirs is the set of directory names that FindUntrackedEnvFiles skips when
// walking the repository tree.
var skipDirs = map[string]bool{
	gitDir:         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
}

// collectEnvFiles returns the relative paths of all .env* files found while
// walking repoPath, excluding directories in skipDirs.
func collectEnvFiles(repoPath string) ([]string, error) {
	var paths []string
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}
		if info.IsDir() && skipDirs[info.Name()] {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), ".env") {
			rel, relErr := filepath.Rel(repoPath, path)
			if relErr != nil {
				return nil
			}
			paths = append(paths, rel)
		}
		return nil
	})
	return paths, err
}

// trackedFileSet returns the set of files currently tracked by git in repoPath.
func (c *Client) trackedFileSet(repoPath string) (map[string]bool, error) {
	out, err := c.r.run(repoPath, "ls-files", "--cached")
	if err != nil {
		return nil, fmt.Errorf("failed to get tracked files: %w", err)
	}
	tracked := make(map[string]bool)
	for _, f := range strings.Split(out, "\n") {
		if f != "" {
			tracked[f] = true
		}
	}
	return tracked, nil
}

// FindUntrackedEnvFiles finds all untracked .env* files in the repository
func (c *Client) FindUntrackedEnvFiles(repoPath string) ([]EnvFile, error) {
	allEnvFiles, err := collectEnvFiles(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	trackedFiles, err := c.trackedFileSet(repoPath)
	if err != nil {
		return nil, err
	}

	var untrackedEnvFiles []EnvFile
	for _, envFile := range allEnvFiles {
		if !trackedFiles[envFile] {
			untrackedEnvFiles = append(untrackedEnvFiles, EnvFile{
				Path:         envFile,
				AbsolutePath: filepath.Join(repoPath, envFile),
			})
		}
	}

	return untrackedEnvFiles, nil
}

// CopyEnvFiles copies environment files from source to destination
func (c *Client) CopyEnvFiles(envFiles []EnvFile, sourceRoot, destRoot string) error {
	for _, envFile := range envFiles {
		srcPath := filepath.Join(sourceRoot, envFile.Path)
		destPath := filepath.Join(destRoot, envFile.Path)

		// Create destination directory if it doesn't exist
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, permEnvDir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		// Read source file
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", srcPath, err)
		}

		// Write to destination with secure permissions
		if err := os.WriteFile(destPath, data, permEnvFile); err != nil {
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}

		fmt.Printf("Copied: %s\n", envFile.Path)
	}

	return nil
}
