package detect

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunSetup(t *testing.T) {
	t.Run("handles empty directory gracefully", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-runsetup-empty-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Should not return error for empty directory
		err = RunSetup(tempDir)
		if err != nil {
			t.Errorf("RunSetup() returned error for empty directory: %v", err)
		}
	})

	t.Run("attempts to run package manager command", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-runsetup-mock-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a mock package manager setup
		// We'll create a simple executable script that simulates a package manager
		mockCmd := "mock-pm"
		var scriptContent string
		var scriptExt string

		if runtime.GOOS == "windows" {
			scriptExt = ".bat"
			scriptContent = `@echo off
echo Mock package manager install
exit 0
`
		} else {
			scriptExt = ".sh"
			scriptContent = `#!/bin/sh
echo "Mock package manager install"
exit 0
`
		}

		// Create the mock executable
		mockPath := filepath.Join(tempDir, mockCmd+scriptExt)
		if err := os.WriteFile(mockPath, []byte(scriptContent), 0755); err != nil {
			t.Fatalf("failed to create mock script: %v", err)
		}

		// Create a go.mod file to trigger go package manager detection
		// But we'll modify PATH to use our mock instead
		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
			t.Fatalf("failed to create go.mod: %v", err)
		}

		// The actual command will likely fail since 'go mod download' is a real command
		// We're mainly testing that the function doesn't panic and handles errors gracefully
		_ = RunSetup(tempDir)
	})

	t.Run("handles non-existent directory", func(t *testing.T) {
		nonExistentDir := "/tmp/non-existent-directory-for-testing-12345"

		// Make sure it doesn't exist
		os.RemoveAll(nonExistentDir)

		// Should handle gracefully (no package manager will be detected)
		err := RunSetup(nonExistentDir)
		if err != nil {
			t.Errorf("RunSetup() returned error for non-existent directory: %v", err)
		}
	})

	t.Run("detects and reports package manager type", func(t *testing.T) {
		// Test that the function correctly identifies different package managers
		tests := []struct {
			name   string
			files  map[string]string
			wantPM string
		}{
			{
				name: "npm project",
				files: map[string]string{
					"package.json": "{}",
				},
				wantPM: "npm",
			},
			{
				name: "yarn project",
				files: map[string]string{
					"package.json": "{}",
					"yarn.lock":    "",
				},
				wantPM: "yarn",
			},
			{
				name: "pnpm project",
				files: map[string]string{
					"package.json":   "{}",
					"pnpm-lock.yaml": "",
				},
				wantPM: "pnpm",
			},
			{
				name: "go project",
				files: map[string]string{
					"go.mod": "module test\n\ngo 1.21\n",
				},
				wantPM: "go",
			},
			{
				name: "cargo project",
				files: map[string]string{
					"Cargo.toml": "[package]\nname = \"test\"\nversion = \"0.1.0\"\n",
				},
				wantPM: "cargo",
			},
			{
				name: "composer project",
				files: map[string]string{
					"composer.json": `{"name": "test/project", "require": {}}`,
				},
				wantPM: "composer",
			},
			{
				name: "pip project",
				files: map[string]string{
					"requirements.txt": "flask==2.0.0\nrequests==2.28.0\n",
				},
				wantPM: "pip",
			},
			{
				name: "bundler project",
				files: map[string]string{
					"Gemfile": "source 'https://rubygems.org'\ngem 'rails', '~> 7.0'\n",
				},
				wantPM: "bundler",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tempDir, err := os.MkdirTemp("", "test-runsetup-detect-")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				defer os.RemoveAll(tempDir)

				// Create test files
				for filename, content := range tt.files {
					filepath := filepath.Join(tempDir, filename)
					if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
						t.Fatalf("failed to create %s: %v", filename, err)
					}
				}

				// Detect package manager (we'll check this separately)
				pm, err := DetectPackageManager(tempDir)
				if err != nil {
					t.Fatalf("failed to detect package manager: %v", err)
				}

				if pm.Name != tt.wantPM {
					t.Errorf("detected package manager = %v, want %v", pm.Name, tt.wantPM)
				}

				// RunSetup will try to execute the actual command which might fail
				// We're testing detection logic, not actual execution
				_ = RunSetup(tempDir)
			})
		}
	})
}

func TestRunSetup_CommandExecution(t *testing.T) {
	t.Run("executes echo command successfully", func(t *testing.T) {
		// This test verifies that command execution works when we have a valid command
		// We'll create a custom test that uses 'echo' which should be available everywhere
		tempDir, err := os.MkdirTemp("", "test-runsetup-echo-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save original packageManagers and restore after test
		originalPMs := packageManagers
		defer func() { packageManagers = originalPMs }()

		// Create a mock package manager that uses echo command
		packageManagers = []PackageManager{
			{
				Name:       "echo-test",
				LockFile:   "echo.lock",
				InstallCmd: []string{"echo", "test-install-success"},
			},
		}

		// Create the lock file to trigger our mock package manager
		if err := os.WriteFile(filepath.Join(tempDir, "echo.lock"), []byte(""), 0644); err != nil {
			t.Fatalf("failed to create lock file: %v", err)
		}

		// Run setup - echo should succeed
		err = RunSetup(tempDir)
		if err != nil {
			t.Errorf("RunSetup() with echo command failed: %v", err)
		}
	})

	t.Run("handles command execution failure", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-runsetup-fail-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save original packageManagers and restore after test
		originalPMs := packageManagers
		defer func() { packageManagers = originalPMs }()

		// Create a mock package manager with a command that will fail
		packageManagers = []PackageManager{
			{
				Name:       "fail-test",
				LockFile:   "fail.lock",
				InstallCmd: []string{"false"}, // 'false' command always exits with error
			},
		}

		// Create the lock file to trigger our mock package manager
		if err := os.WriteFile(filepath.Join(tempDir, "fail.lock"), []byte(""), 0644); err != nil {
			t.Fatalf("failed to create lock file: %v", err)
		}

		// Run setup - should return error
		err = RunSetup(tempDir)
		if err == nil {
			t.Error("RunSetup() should have returned error for failing command")
		}
		if err != nil && !strings.Contains(err.Error(), "failed to run fail-test") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("handles non-existent command", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-runsetup-notfound-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Save original packageManagers and restore after test
		originalPMs := packageManagers
		defer func() { packageManagers = originalPMs }()

		// Create a mock package manager with non-existent command
		packageManagers = []PackageManager{
			{
				Name:       "notfound-test",
				LockFile:   "notfound.lock",
				InstallCmd: []string{"this-command-definitely-does-not-exist-12345"},
			},
		}

		// Create the lock file to trigger our mock package manager
		if err := os.WriteFile(filepath.Join(tempDir, "notfound.lock"), []byte(""), 0644); err != nil {
			t.Fatalf("failed to create lock file: %v", err)
		}

		// Run setup - should return error
		err = RunSetup(tempDir)
		if err == nil {
			t.Error("RunSetup() should have returned error for non-existent command")
		}

		// Check if it's exec.Error type (command not found)
		if err != nil {
			if _, ok := err.(*exec.Error); !ok {
				// It should be wrapped, so check the error message
				if !strings.Contains(err.Error(), "failed to run notfound-test") {
					t.Errorf("unexpected error type/message: %v", err)
				}
			}
		}
	})
}
