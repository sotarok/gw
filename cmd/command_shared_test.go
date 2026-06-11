package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/git"
)

func TestDefaultDependencies(t *testing.T) {
	deps := DefaultDependencies()

	if deps == nil {
		t.Fatal("Expected non-nil dependencies")
	}

	if deps.Git == nil {
		t.Error("Expected Git dependency to be initialized")
	}

	if deps.UI == nil {
		t.Error("Expected UI dependency to be initialized")
	}

	if deps.Detect == nil {
		t.Error("Expected Detect dependency to be initialized")
	}

	if deps.Stdout != os.Stdout {
		t.Error("Expected Stdout to be os.Stdout")
	}

	if deps.Stderr != os.Stderr {
		t.Error("Expected Stderr to be os.Stderr")
	}
}

// Test copy_envs configuration priority
func TestHandleEnvFiles_ConfigPriority(t *testing.T) {
	tests := []struct {
		name           string
		configCopyEnvs *bool
		copyEnvsFlag   bool
		expectedPrompt bool
		expectedCopy   bool
		userResponse   bool
	}{
		{
			name:           "flag set to true - always copy",
			configCopyEnvs: nil,
			copyEnvsFlag:   true,
			expectedPrompt: false,
			expectedCopy:   true,
		},
		{
			name:           "flag set, config is false - flag overrides config",
			configCopyEnvs: boolPtr(false),
			copyEnvsFlag:   true,
			expectedPrompt: false,
			expectedCopy:   true,
		},
		{
			name:           "config is true, flag not set - use config",
			configCopyEnvs: boolPtr(true),
			copyEnvsFlag:   false,
			expectedPrompt: false,
			expectedCopy:   true,
		},
		{
			name:           "config is false, flag not set - use config (don't copy)",
			configCopyEnvs: boolPtr(false),
			copyEnvsFlag:   false,
			expectedPrompt: false,
			expectedCopy:   false,
		},
		{
			name:           "neither config nor flag set - prompt user (yes)",
			configCopyEnvs: nil,
			copyEnvsFlag:   false,
			expectedPrompt: true,
			expectedCopy:   true,
			userResponse:   true,
		},
		{
			name:           "neither config nor flag set - prompt user (no)",
			configCopyEnvs: nil,
			copyEnvsFlag:   false,
			expectedPrompt: true,
			expectedCopy:   false,
			userResponse:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temporary directories
			tempDir := t.TempDir()
			originalDir := filepath.Join(tempDir, "original")
			worktreeDir := filepath.Join(tempDir, "worktree")
			os.MkdirAll(originalDir, 0755)
			os.MkdirAll(worktreeDir, 0755)

			// Create a test env file
			envFile := filepath.Join(originalDir, ".env.local")
			os.WriteFile(envFile, []byte("TEST=value"), 0600)

			// Setup mocks
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			mockGit := &mockGit{
				isGitRepo: true,
				envFiles: []git.EnvFile{
					{Path: ".env.local", AbsolutePath: envFile},
				},
			}

			mockUI := &mockUI{
				confirmResult: tt.userResponse,
			}

			deps := &Dependencies{
				Git:    mockGit,
				UI:     mockUI,
				Detect: &mockDetect{},
				Config: &config.Config{
					CopyEnvs: tt.configCopyEnvs,
				},
				Stdout: stdout,
				Stderr: stderr,
			}

			// Execute handleEnvFiles
			err := handleEnvFiles(deps, tt.copyEnvsFlag, originalDir, worktreeDir)
			if err != nil {
				t.Fatalf("handleEnvFiles failed: %v", err)
			}

			// Verify prompt behavior
			if tt.expectedPrompt && !mockUI.confirmCalled {
				t.Error("Expected user to be prompted, but wasn't")
			}
			if !tt.expectedPrompt && mockUI.confirmCalled {
				t.Error("Expected no prompt, but user was prompted")
			}

			// Verify copy behavior
			copiedFile := filepath.Join(worktreeDir, ".env.local")
			fileExists := false
			if _, err := os.Stat(copiedFile); err == nil {
				fileExists = true
			}

			if tt.expectedCopy && !fileExists {
				t.Errorf("Expected file to be copied, but it wasn't. Output:\n%s", stdout.String())
			}
			if !tt.expectedCopy && fileExists {
				t.Error("Expected file not to be copied, but it was")
			}
		})
	}
}
