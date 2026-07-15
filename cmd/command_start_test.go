package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sotarok/gw/internal/config"
	"github.com/sotarok/gw/internal/git"
)

func TestStartCommand_Execute(t *testing.T) {
	// Save and restore working directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tests := []struct {
		name            string
		issueNumber     string
		baseBranch      string
		copyEnvs        bool
		updateITerm2Tab bool
		mockSetup       func() (*mockGit, *mockUI, *mockDetect, func())
		expectedError   string
		checkOutput     func(t *testing.T, stdout, stderr string)
	}{
		{
			name:        "not in git repository",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				return &mockGit{isGitRepo: false}, &mockUI{}, &mockDetect{}, func() {}
			},
			expectedError: "not in a git repository",
		},
		{
			name:        "worktree already exists",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				return &mockGit{
					isGitRepo:      true,
					worktreeExists: true,
				}, &mockUI{}, &mockDetect{}, func() {}
			},
			expectedError: "worktree for issue 123 already exists at /existing/path",
		},
		{
			name:        "successful creation without env files",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				// Create a real temporary directory for worktree path
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				return &mockGit{
					isGitRepo:    true,
					worktreePath: tempDir,
					envFiles:     []git.EnvFile{},
				}, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Creating worktree for issue #123") {
					t.Error("Expected creation message in stdout")
				}
				if !contains(stdout, "✓ Created worktree at") {
					t.Error("Expected success message in stdout")
				}
			},
		},
		{
			name:        "successful creation with env files - user confirms",
			issueNumber: "123",
			baseBranch:  "main",
			copyEnvs:    false,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				// Create a real env file in the current directory for testing
				envFile := ".env.test"
				os.WriteFile(envFile, []byte("TEST=value"), 0600)
				return &mockGit{
						isGitRepo:    true,
						worktreePath: tempDir,
						envFiles: []git.EnvFile{
							{Path: ".env.test", AbsolutePath: envFile},
						},
					}, &mockUI{confirmResult: true}, &mockDetect{}, func() {
						os.RemoveAll(tempDir)
						os.Remove(envFile)
					}
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Found 1 untracked environment file(s):") {
					t.Error("Expected env files message in stdout")
				}
				if !contains(stdout, "✓ Environment files copied successfully") {
					t.Error("Expected copy success message in stdout")
				}
			},
		},
		{
			name:        "successful creation with env files - copyEnvs flag set",
			issueNumber: "123",
			baseBranch:  "main",
			copyEnvs:    true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				// Create a real env file in the current directory for testing
				envFile := ".env.test2"
				os.WriteFile(envFile, []byte("TEST=value"), 0600)
				return &mockGit{
						isGitRepo:    true,
						worktreePath: tempDir,
						envFiles: []git.EnvFile{
							{Path: ".env.test2", AbsolutePath: envFile},
						},
					}, &mockUI{}, &mockDetect{}, func() {
						os.RemoveAll(tempDir)
						os.Remove(envFile)
					}
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stdout, "Copying environment files:") {
					t.Errorf("Expected copying message in stdout, got:\n%s", stdout)
				}
				if !contains(stdout, "✓ Environment files copied successfully") {
					t.Errorf("Expected copy success message in stdout, got:\n%s", stdout)
				}
			},
		},
		{
			name:        "setup failure warns but doesn't fail",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				return &mockGit{
					isGitRepo:    true,
					worktreePath: tempDir,
				}, &mockUI{}, &mockDetect{setupError: fmt.Errorf("npm install failed")}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stderr, "⚠ Setup failed: npm install failed") {
					t.Error("Expected setup warning in stderr")
				}
				if !contains(stdout, "✨ Worktree ready at:") {
					t.Error("Expected success message despite setup failure")
				}
			},
		},
		{
			name:        "error creating worktree",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				return &mockGit{
					isGitRepo:           true,
					createWorktreeError: fmt.Errorf("permission denied"),
				}, &mockUI{}, &mockDetect{}, func() {}
			},
			expectedError: "permission denied",
		},
		{
			name:        "error finding env files",
			issueNumber: "123",
			baseBranch:  "main",
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				return &mockGit{
					isGitRepo:    true,
					worktreePath: tempDir,
					findEnvError: fmt.Errorf("failed to find env files"),
				}, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stderr, "⚠ Failed to handle env files") {
					t.Error("Expected env files warning in stderr")
				}
				// Should still complete successfully
				if !contains(stdout, "✨ Worktree ready at:") {
					t.Error("Expected success message despite env files error")
				}
			},
		},
		{
			name:        "error copying env files",
			issueNumber: "123",
			baseBranch:  "main",
			copyEnvs:    true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				return &mockGit{
					isGitRepo:    true,
					worktreePath: tempDir,
					envFiles: []git.EnvFile{
						{Path: ".env", AbsolutePath: "/repo/.env"},
					},
					copyEnvError: fmt.Errorf("permission denied"),
				}, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				if !contains(stderr, "⚠ Failed to handle env files") {
					t.Error("Expected env files warning in stderr")
				}
				// Should still complete successfully
				if !contains(stdout, "✨ Worktree ready at:") {
					t.Error("Expected success message despite copy error")
				}
			},
		},
		{
			name:            "updates iTerm2 tab when enabled",
			issueNumber:     "789",
			baseBranch:      "main",
			updateITerm2Tab: true,
			mockSetup: func() (*mockGit, *mockUI, *mockDetect, func()) {
				tempDir, _ := os.MkdirTemp("", "gw-worktree-*")
				return &mockGit{
					isGitRepo:    true,
					worktreePath: tempDir,
				}, &mockUI{}, &mockDetect{}, func() { os.RemoveAll(tempDir) }
			},
			checkOutput: func(t *testing.T, stdout, stderr string) {
				// Check for iTerm2 escape sequence in output
				// Format: "\033]0;test-repo 789\007"
				// Note: We can't test the actual escape sequence output without mocking
				// but we can verify the command completes successfully
				if !contains(stdout, "✨ Worktree ready at:") {
					t.Error("Expected success message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for testing
			tempDir, err := os.MkdirTemp("", "gw-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp dir: %v", err)
			}

			// Setup mocks
			mockGit, mockUI, mockDetect, cleanup := tt.mockSetup()
			defer cleanup()
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			deps := &Dependencies{
				Git:    mockGit,
				UI:     mockUI,
				Detect: mockDetect,
				Config: config.New(),
				Stdout: stdout,
				Stderr: stderr,
			}

			// Create config for test
			cfg := &config.Config{
				AutoCD:          false,
				UpdateITerm2Tab: tt.updateITerm2Tab,
			}
			deps.Config = cfg
			cmd := NewStartCommand(deps, tt.copyEnvs, true, false)
			err = cmd.Execute(tt.issueNumber, tt.baseBranch)

			// Check error
			if tt.expectedError != "" {
				if err == nil || err.Error() != tt.expectedError {
					t.Errorf("Expected error %q, got %v", tt.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output
			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout.String(), stderr.String())
			}
		})
	}
}

func TestStartCommand_Execute_FetchBeforeCommand(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	fetchCalled := false
	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}
	mockGitInstance.FetchAllFn = func() error {
		fetchCalled = true
		return nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{},
		Detect: &mockDetect{},
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	// noFetch=false with FetchBeforeCommand=true should call fetch
	deps.Config = &config.Config{FetchBeforeCommand: true}
	cmd := NewStartCommand(deps, false, false, false)
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !fetchCalled {
		t.Error("Expected FetchAll to be called when FetchBeforeCommand is true")
	}
}

func TestStartCommand_Execute_FetchErrorWarnsButDoesNotFail(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}
	mockGitInstance.FetchAllFn = func() error {
		return fmt.Errorf("network error")
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{},
		Detect: &mockDetect{},
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{FetchBeforeCommand: true}
	cmd := NewStartCommand(deps, false, false, false)
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Expected no error (fetch failure should warn), got: %v", err)
	}
	if !contains(stderr.String(), "Could not fetch from remotes") {
		t.Error("Expected fetch warning in stderr")
	}
	if !contains(stdout.String(), "Worktree ready at:") {
		t.Error("Expected success message despite fetch failure")
	}
}

func TestStartCommand_Execute_NoFetchSkipsFetch(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	fetchCalled := false
	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}
	mockGitInstance.FetchAllFn = func() error {
		fetchCalled = true
		return nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{},
		Detect: &mockDetect{},
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	// noFetch=true should skip fetch even with FetchBeforeCommand=true
	deps.Config = &config.Config{FetchBeforeCommand: true}
	cmd := NewStartCommand(deps, false, true, false)
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if fetchCalled {
		t.Error("Expected FetchAll NOT to be called when noFetch is true")
	}
}

func TestStartCommand_Execute_AutoCDEnabled(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{},
		Detect: &mockDetect{},
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{AutoCD: true}
	cmd := NewStartCommand(deps, false, true, false)
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !contains(stdout.String(), "Shell integration will change to this directory") {
		t.Error("Expected shell integration message when AutoCD is enabled")
	}
}

func TestStartCommand_Execute_ITerm2Tab(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{},
		Detect: &mockDetect{},
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{UpdateITerm2Tab: true}
	cmd := NewStartCommand(deps, false, true, false)
	err = cmd.Execute("456", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Verify command completes successfully with iTerm2 tab update enabled
	if !contains(stdout.String(), "Worktree ready at:") {
		t.Error("Expected success message")
	}
}

func TestStartCommand_Execute_EnvFilesUserDeclines(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	// Create a real env file
	envFile := ".env.decline-test"
	os.WriteFile(envFile, []byte("KEY=value"), 0600)
	defer os.Remove(envFile)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
		envFiles: []git.EnvFile{
			{Path: envFile, AbsolutePath: filepath.Join(tempDir, envFile)},
		},
	}

	mockUIInstance := &mockUI{confirmResult: false}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     mockUIInstance,
		Detect: &mockDetect{},
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewStartCommand(deps, false, true, false)
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !mockUIInstance.confirmCalled {
		t.Error("Expected user to be prompted")
	}
	// Files should NOT be copied
	if contains(stdout.String(), "Environment files copied successfully") {
		t.Error("Files should not have been copied when user declines")
	}
}

func TestStartCommand_Execute_ConfigCopyEnvsTrue(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	envFile := ".env.configtest"
	os.WriteFile(envFile, []byte("KEY=value"), 0600)
	defer os.Remove(envFile)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
		envFiles: []git.EnvFile{
			{Path: envFile, AbsolutePath: filepath.Join(tempDir, envFile)},
		},
	}

	mockUIInstance := &mockUI{}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     mockUIInstance,
		Detect: &mockDetect{},
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	// copyEnvs flag=false but config.CopyEnvs=true should copy without prompting
	deps.Config = &config.Config{CopyEnvs: boolPtr(true)}
	cmd := NewStartCommand(deps, false, true, false)
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if mockUIInstance.confirmCalled {
		t.Error("Should not prompt user when config.CopyEnvs is true")
	}
	if !contains(stdout.String(), "Copying environment files:") {
		t.Error("Expected auto-copy message")
	}
	if !contains(stdout.String(), "Environment files copied successfully") {
		t.Error("Expected copy success message")
	}
}

func TestStartCommand_Execute_EnvSourceAnchoredToRepoRoot(t *testing.T) {
	// Regression: env file scan used cwd, so running `gw start` from a sub
	// directory missed env files at the repo root and copied sub-dir env files
	// to the worktree root with wrong relative paths.
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoRoot := filepath.Join(tempDir, "repo")
	subDir := filepath.Join(repoRoot, "apps", "admin")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create sub directory: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to chdir into sub directory: %v", err)
	}

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	var capturedScanRoot string
	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}
	mockGitInstance.GetRepositoryRootFn = func() (string, error) { return repoRoot, nil }
	mockGitInstance.FindUntrackedEnvFilesFn = func(p string) ([]git.EnvFile, error) {
		capturedScanRoot = p
		return nil, nil
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{},
		Detect: &mockDetect{},
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{}
	cmd := NewStartCommand(deps, true, true, false)
	if err := cmd.Execute("123", "main"); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if capturedScanRoot != repoRoot {
		t.Errorf("FindUntrackedEnvFiles called with wrong root.\n  got:  %s\n  want: %s", capturedScanRoot, repoRoot)
	}
}

func TestStartCommand_Execute_PostHook(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{},
		Detect: &mockDetect{},
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{
		PostStartHook: `echo "HOOK_OUTPUT:$GW_WORKTREE_PATH:$GW_COMMAND"`,
	}
	cmd := NewStartCommand(deps, false, true, false)
	err = cmd.Execute("123", "main")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := stdout.String()
	if !contains(output, "HOOK_OUTPUT:"+worktreeDir+":start") {
		t.Errorf("Expected hook output with worktree path and command, got:\n%s", output)
	}
}

func TestStartCommand_Execute_PostHookFailure(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir, err := os.MkdirTemp("", "gw-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	os.Chdir(tempDir)

	worktreeDir, _ := os.MkdirTemp("", "gw-worktree-*")
	defer os.RemoveAll(worktreeDir)

	mockGitInstance := &mockGit{
		isGitRepo:    true,
		worktreePath: worktreeDir,
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	deps := &Dependencies{
		Git:    mockGitInstance,
		UI:     &mockUI{},
		Detect: &mockDetect{},
		Config: config.New(),
		Stdout: stdout,
		Stderr: stderr,
	}

	deps.Config = &config.Config{
		PostStartHook: "exit 1",
	}
	cmd := NewStartCommand(deps, false, true, false)
	err = cmd.Execute("123", "main")

	// Command should succeed even if hook fails
	if err != nil {
		t.Fatalf("Expected no error when hook fails, got: %v", err)
	}
	if !contains(stderr.String(), "Post-start hook failed") {
		t.Errorf("Expected warning about hook failure in stderr, got:\n%s", stderr.String())
	}
	// Should still show completion message
	if !contains(stdout.String(), "Worktree ready at:") {
		t.Error("Expected success message even when hook fails")
	}
}
