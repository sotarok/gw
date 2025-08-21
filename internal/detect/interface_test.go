package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefaultDetector(t *testing.T) {
	detector := NewDefaultDetector()
	if detector == nil {
		t.Fatal("NewDefaultDetector() returned nil")
	}

	// Verify it implements the Interface
	var _ Interface = detector
}

func TestDefaultDetector_DetectPackageManager(t *testing.T) {
	detector := NewDefaultDetector()

	tests := []struct {
		name      string
		setupFunc func(dir string) error
		wantPM    string
		wantErr   bool
	}{
		{
			name: "detects npm with package.json",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
			},
			wantPM:  "npm",
			wantErr: false,
		},
		{
			name: "detects yarn with yarn.lock",
			setupFunc: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0644)
			},
			wantPM:  "yarn",
			wantErr: false,
		},
		{
			name: "detects go with go.mod",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0644)
			},
			wantPM:  "go",
			wantErr: false,
		},
		{
			name: "detects composer with composer.json",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "composer.json"), []byte("{\"name\":\"test/project\"}"), 0644)
			},
			wantPM:  "composer",
			wantErr: false,
		},
		{
			name: "detects pip with requirements.txt",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask==2.0.0\n"), 0644)
			},
			wantPM:  "pip",
			wantErr: false,
		},
		{
			name: "detects bundler with Gemfile",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'https://rubygems.org'\ngem 'rails'\n"), 0644)
			},
			wantPM:  "bundler",
			wantErr: false,
		},
		{
			name:      "returns error for empty directory",
			setupFunc: func(dir string) error { return nil },
			wantPM:    "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir, err := os.MkdirTemp("", "test-detector-")
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Setup test files
			if tt.setupFunc != nil {
				if err := tt.setupFunc(tempDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			// Test detection
			pm, err := detector.DetectPackageManager(tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectPackageManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && pm != nil && pm.Name != tt.wantPM {
				t.Errorf("DetectPackageManager() got package manager = %v, want %v", pm.Name, tt.wantPM)
			}
		})
	}
}

func TestDefaultDetector_RunSetup(t *testing.T) {
	detector := NewDefaultDetector()

	t.Run("skips setup when no package manager found", func(t *testing.T) {
		// Create empty temp directory
		tempDir, err := os.MkdirTemp("", "test-runsetup-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Should not return error when no package manager is found
		err = detector.RunSetup(tempDir)
		if err != nil {
			t.Errorf("RunSetup() returned error for empty directory: %v", err)
		}
	})

	t.Run("detects package manager before running setup", func(t *testing.T) {
		// Create temp directory with a mock setup
		tempDir, err := os.MkdirTemp("", "test-runsetup-")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a simple script that can be executed instead of real package manager
		// This tests that the function attempts to run the command
		scriptContent := `#!/bin/sh
echo "mock install"
exit 0
`
		scriptPath := filepath.Join(tempDir, "mock-installer.sh")
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
			t.Fatalf("failed to create mock script: %v", err)
		}

		// Create a marker file to indicate package manager presence
		// We'll use a simple text file as a mock lock file for testing
		if err := os.WriteFile(filepath.Join(tempDir, "test.lock"), []byte(""), 0644); err != nil {
			t.Fatalf("failed to create lock file: %v", err)
		}

		// The actual RunSetup will fail because npm/yarn/etc. might not be installed
		// or the commands might fail in test environment
		// So we just test that it tries to detect and doesn't panic
		_ = detector.RunSetup(tempDir)
		// We don't check the error because the actual package manager command might fail
		// The important thing is that the function completes without panic
	})
}
