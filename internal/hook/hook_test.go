package hook

import (
	"bytes"
	"os"
	"runtime"
	"testing"
)

func TestExecute_EmptyCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute("", Env{}, &stdout, &stderr)
	if err != nil {
		t.Errorf("Expected no error for empty command, got: %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("Expected no stdout output, got: %s", stdout.String())
	}
}

const osWindows = "windows"

func TestExecute_SimpleCommand(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("Skipping on Windows")
	}

	var stdout, stderr bytes.Buffer
	env := Env{
		WorktreePath: "/tmp/test-worktree",
		BranchName:   "123/impl",
		RepoName:     "my-repo",
		Command:      "start",
	}

	err := Execute("echo $GW_WORKTREE_PATH", env, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "/tmp/test-worktree\n"
	if stdout.String() != expected {
		t.Errorf("Expected stdout %q, got %q", expected, stdout.String())
	}
}

func TestExecute_AllEnvVars(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("Skipping on Windows")
	}

	var stdout, stderr bytes.Buffer
	env := Env{
		WorktreePath: "/path/to/worktree",
		BranchName:   "feature/test",
		RepoName:     "test-repo",
		Command:      "checkout",
	}

	err := Execute("echo $GW_WORKTREE_PATH:$GW_BRANCH_NAME:$GW_REPO_NAME:$GW_COMMAND", env, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "/path/to/worktree:feature/test:test-repo:checkout\n"
	if stdout.String() != expected {
		t.Errorf("Expected stdout %q, got %q", expected, stdout.String())
	}
}

func TestExecute_FailingCommand(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("Skipping on Windows")
	}

	var stdout, stderr bytes.Buffer
	err := Execute("exit 1", Env{}, &stdout, &stderr)
	if err == nil {
		t.Error("Expected error for failing command, got nil")
	}
}

func TestExecute_StderrOutput(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("Skipping on Windows")
	}

	var stdout, stderr bytes.Buffer
	err := Execute("echo error >&2", Env{}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if stderr.String() != "error\n" {
		t.Errorf("Expected stderr %q, got %q", "error\n", stderr.String())
	}
}

func TestExecute_InheritsParentEnv(t *testing.T) {
	if runtime.GOOS == osWindows {
		t.Skip("Skipping on Windows")
	}

	os.Setenv("GW_TEST_PARENT_VAR", "hello")
	defer os.Unsetenv("GW_TEST_PARENT_VAR")

	var stdout, stderr bytes.Buffer
	err := Execute("echo $GW_TEST_PARENT_VAR", Env{}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if stdout.String() != "hello\n" {
		t.Errorf("Expected stdout %q, got %q", "hello\n", stdout.String())
	}
}

func TestEnvToSlice(t *testing.T) {
	env := Env{
		WorktreePath: "/path/to/worktree",
		BranchName:   "123/impl",
		RepoName:     "my-repo",
		Command:      "start",
	}

	slice := envToSlice(env)

	expected := []string{
		"GW_WORKTREE_PATH=/path/to/worktree",
		"GW_BRANCH_NAME=123/impl",
		"GW_REPO_NAME=my-repo",
		"GW_COMMAND=start",
	}

	if len(slice) != len(expected) {
		t.Fatalf("Expected %d env vars, got %d", len(expected), len(slice))
	}

	for i, exp := range expected {
		if slice[i] != exp {
			t.Errorf("Expected env[%d] = %q, got %q", i, exp, slice[i])
		}
	}
}
