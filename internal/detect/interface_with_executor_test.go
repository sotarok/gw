package detect

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDetectorWithExecutor(t *testing.T) {
	t.Run("uses custom executor for RunSetup", func(t *testing.T) {
		// Create a mock executor
		mockExecutor := &MockExecutor{}

		// Create detector with mock executor
		detector := NewDefaultDetectorWithExecutor(mockExecutor)

		// Create temp directory with package.json
		tempDir, err := os.MkdirTemp("", "test-executor-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte("{}"), 0644); err != nil {
			t.Fatalf("failed to create package.json: %v", err)
		}

		// Run setup
		err = detector.RunSetup(tempDir)
		if err != nil {
			t.Errorf("RunSetup() failed: %v", err)
		}

		// Verify the executor was called
		if len(mockExecutor.ExecuteCalls) != 1 {
			t.Fatalf("expected 1 executor call, got %d", len(mockExecutor.ExecuteCalls))
		}

		call := mockExecutor.ExecuteCalls[0]
		if call.Dir != tempDir {
			t.Errorf("executor called with wrong dir: got %s, want %s", call.Dir, tempDir)
		}
		if call.Command != npmName {
			t.Errorf("executor called with wrong command: got %s, want npm", call.Command)
		}
		if len(call.Args) != 1 || call.Args[0] != "install" {
			t.Errorf("executor called with wrong args: got %v, want [install]", call.Args)
		}
	})

	t.Run("handles executor errors", func(t *testing.T) {
		// Create a mock executor that returns an error
		expectedErr := fmt.Errorf("mock execution failed")
		mockExecutor := &MockExecutor{
			ReturnError: expectedErr,
		}

		// Create detector with mock executor
		detector := NewDefaultDetectorWithExecutor(mockExecutor)

		// Create temp directory with go.mod
		tempDir, err := os.MkdirTemp("", "test-executor-error-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
			t.Fatalf("failed to create go.mod: %v", err)
		}

		// Run setup - should return wrapped error
		err = detector.RunSetup(tempDir)
		if err == nil {
			t.Error("RunSetup() should have returned error from executor")
		}

		// Verify error message
		if err != nil && err.Error() != "failed to run go: mock execution failed" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("skips execution when no package manager found", func(t *testing.T) {
		// Create a mock executor
		mockExecutor := &MockExecutor{}

		// Create detector with mock executor
		detector := NewDefaultDetectorWithExecutor(mockExecutor)

		// Create empty temp directory
		tempDir, err := os.MkdirTemp("", "test-no-pm-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Run setup - should skip without calling executor
		err = detector.RunSetup(tempDir)
		if err != nil {
			t.Errorf("RunSetup() returned error for empty directory: %v", err)
		}

		// Verify executor was NOT called
		if len(mockExecutor.ExecuteCalls) != 0 {
			t.Errorf("executor should not be called when no package manager found, got %d calls", len(mockExecutor.ExecuteCalls))
		}
	})
}

func TestRunSetupWithExecutor(t *testing.T) {
	t.Run("passes correct parameters to executor", func(t *testing.T) {
		tests := []struct {
			name        string
			setupFile   string
			fileContent string
			wantCommand string
			wantArgs    []string
		}{
			{
				name:        "npm project",
				setupFile:   "package.json",
				fileContent: "{}",
				wantCommand: "npm",
				wantArgs:    []string{"install"},
			},
			{
				name:        "yarn project",
				setupFile:   "yarn.lock",
				fileContent: "",
				wantCommand: "yarn",
				wantArgs:    []string{"install"},
			},
			{
				name:        "go project",
				setupFile:   "go.mod",
				fileContent: "module test\n\ngo 1.21\n",
				wantCommand: "go",
				wantArgs:    []string{"mod", "download"},
			},
			{
				name:        "cargo project",
				setupFile:   "Cargo.toml",
				fileContent: "[package]\nname = \"test\"\n",
				wantCommand: "cargo",
				wantArgs:    []string{"build"},
			},
			{
				name:        "pip project",
				setupFile:   "requirements.txt",
				fileContent: "flask==2.0.0\n",
				wantCommand: "pip",
				wantArgs:    []string{"install", "-r", "requirements.txt"},
			},
			{
				name:        "composer project",
				setupFile:   "composer.json",
				fileContent: `{"name": "test/project"}`,
				wantCommand: "composer",
				wantArgs:    []string{"install"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Create mock executor
				mockExecutor := &MockExecutor{}

				// Create temp directory with test file
				tempDir, err := os.MkdirTemp("", "test-params-")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				defer os.RemoveAll(tempDir)

				// Special handling for yarn - needs package.json too
				if tt.setupFile == "yarn.lock" {
					if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte("{}"), 0644); err != nil {
						t.Fatalf("failed to create package.json: %v", err)
					}
				}

				if err := os.WriteFile(filepath.Join(tempDir, tt.setupFile), []byte(tt.fileContent), 0644); err != nil {
					t.Fatalf("failed to create %s: %v", tt.setupFile, err)
				}

				// Run setup with mock executor
				err = RunSetupWithExecutor(tempDir, mockExecutor)
				if err != nil {
					t.Errorf("RunSetupWithExecutor() failed: %v", err)
				}

				// Verify executor was called correctly
				if len(mockExecutor.ExecuteCalls) != 1 {
					t.Fatalf("expected 1 call, got %d", len(mockExecutor.ExecuteCalls))
				}

				call := mockExecutor.ExecuteCalls[0]
				if call.Command != tt.wantCommand {
					t.Errorf("wrong command: got %s, want %s", call.Command, tt.wantCommand)
				}

				if len(call.Args) != len(tt.wantArgs) {
					t.Errorf("wrong number of args: got %d, want %d", len(call.Args), len(tt.wantArgs))
				} else {
					for i, arg := range call.Args {
						if arg != tt.wantArgs[i] {
							t.Errorf("wrong arg[%d]: got %s, want %s", i, arg, tt.wantArgs[i])
						}
					}
				}
			})
		}
	})
}
