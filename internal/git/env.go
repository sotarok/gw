package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnvFile represents an environment file found in the repository
type EnvFile struct {
	Path         string // Relative path from repository root
	AbsolutePath string // Absolute path
}

// FindUntrackedEnvFiles finds all untracked .env* files in the repository
func FindUntrackedEnvFiles(repoPath string) ([]EnvFile, error) {
	// Get all .env* files
	var allEnvFiles []string
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories we can't read
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Skip node_modules and similar directories
		if info.IsDir() && (info.Name() == "node_modules" || info.Name() == "vendor" || info.Name() == "dist" || info.Name() == "build") {
			return filepath.SkipDir
		}

		// Check if it's an .env file
		if !info.IsDir() && strings.HasPrefix(info.Name(), ".env") {
			relPath, err := filepath.Rel(repoPath, path)
			if err != nil {
				return nil
			}
			allEnvFiles = append(allEnvFiles, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Get tracked files from git
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get tracked files: %w", err)
	}

	trackedFiles := make(map[string]bool)
	for _, file := range strings.Split(string(output), "\n") {
		if file != "" {
			trackedFiles[file] = true
		}
	}

	// Filter out tracked files
	var untrackedEnvFiles []EnvFile
	for _, envFile := range allEnvFiles {
		if !trackedFiles[envFile] {
			absPath := filepath.Join(repoPath, envFile)
			untrackedEnvFiles = append(untrackedEnvFiles, EnvFile{
				Path:         envFile,
				AbsolutePath: absPath,
			})
		}
	}

	return untrackedEnvFiles, nil
}

// CopyEnvFiles copies environment files from source to destination
func CopyEnvFiles(envFiles []EnvFile, sourceRoot, destRoot string) error {
	for _, envFile := range envFiles {
		srcPath := filepath.Join(sourceRoot, envFile.Path)
		destPath := filepath.Join(destRoot, envFile.Path)

		// Create destination directory if it doesn't exist
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		// Read source file
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", srcPath, err)
		}

		// Write to destination with secure permissions
		if err := os.WriteFile(destPath, data, 0600); err != nil {
			return fmt.Errorf("failed to write file %s: %w", destPath, err)
		}

		fmt.Printf("Copied: %s\n", envFile.Path)
	}

	return nil
}
