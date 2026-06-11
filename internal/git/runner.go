package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runner abstracts git subprocess execution so Client methods can be tested
// without forking real git processes.
type runner interface {
	// run executes git with args in dir (empty dir = current directory) and
	// returns trimmed stdout. A non-zero exit returns *GitError.
	run(dir string, args ...string) (string, error)

	// runCombined executes git with args in dir and returns trimmed combined
	// stdout+stderr. It is used by callers that historically built their error
	// message from CombinedOutput(). A non-zero exit returns *GitError whose
	// Stderr holds the combined output.
	runCombined(dir string, args ...string) (string, error)

	// runStreaming executes git with args in dir, streaming stdout/stderr to
	// the process's own stdout/stderr (e.g. worktree add/remove progress). It
	// returns only an error.
	runStreaming(dir string, args ...string) error

	// runShell executes command via `sh -c`, streaming stdout/stderr. It backs
	// the RunCommand utility.
	runShell(command string) error
}

// GitError carries the exit code and stderr of a failed git invocation, so
// callers can branch on failure kind without string matching. Its Error()
// preserves the classic "exit status N" suffix produced by exec for
// backwards compatibility with callers that still match on that text.
type GitError struct {
	Args     []string
	ExitCode int
	Stderr   string
}

// Error returns a message that includes the trimmed stderr (when present) and
// the classic "exit status N" text so existing string-based checks keep
// working.
func (e *GitError) Error() string {
	stderr := strings.TrimSpace(e.Stderr)
	if stderr != "" {
		return fmt.Sprintf("git %s: %s: exit status %d", strings.Join(e.Args, " "), stderr, e.ExitCode)
	}
	return fmt.Sprintf("git %s: exit status %d", strings.Join(e.Args, " "), e.ExitCode)
}

// execRunner runs git via os/exec.
type execRunner struct{}

// gitArgs prepends `-C <dir>` to args when dir is non-empty so git itself
// changes into the working directory. This is preferred over setting cmd.Dir:
// when dir does not exist, git reports a normal exit 128 with
// "fatal: cannot change to '<dir>'" on stderr, whereas cmd.Dir makes
// exec.Command fail at chdir with an *fs.PathError (exit code -1, empty
// stderr), swallowing the cause and breaking callers that branch on exit 128.
func gitArgs(dir string, args []string) []string {
	if dir == "" {
		return args
	}
	return append([]string{"-C", dir}, args...)
}

func (execRunner) run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", gitArgs(dir, args)...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", &GitError{
			Args:     args,
			ExitCode: exitCodeFromErr(err),
			Stderr:   stderr.String(),
		}
	}

	return strings.TrimSpace(string(out)), nil
}

func (execRunner) runCombined(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", gitArgs(dir, args)...)

	out, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(out))
	if err != nil {
		return trimmed, &GitError{
			Args:     args,
			ExitCode: exitCodeFromErr(err),
			Stderr:   trimmed,
		}
	}

	return trimmed, nil
}

// exitCodeFromErr extracts the process exit code from an exec error, returning
// -1 when the error is not an *exec.ExitError.
func exitCodeFromErr(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func (execRunner) runStreaming(dir string, args ...string) error {
	cmd := exec.Command("git", gitArgs(dir, args)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (execRunner) runShell(command string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
