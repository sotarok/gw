# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

### Build and Development
```bash
# Build the binary
go build -o gw

# Build and install to $GOPATH/bin
go install

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestDetectPackageManager ./internal/detect

# Check for compile errors without building
go build -o /dev/null ./...

# Update dependencies
go mod tidy

# Format code
make fmt

# Run checks (lint, tests, etc)
make check
```

### Pre-commit Checklist
**IMPORTANT**: Always run these commands before committing:
```bash
# 1. Format the code
make fmt

# 2. Run all checks
make check
```

### Changelog Management
When making changes that affect functionality, bug fixes, or features:

1. **Update CHANGELOG.md** - Add entries under the `[Unreleased]` section
2. **Use conventional commit messages** for easier changelog generation:
   - `fix:` for bug fixes
   - `feat:` for new features
   - `docs:` for documentation changes
   - `chore:` for maintenance tasks
   - `refactor:` for code refactoring
   - `test:` for test additions/changes

3. **Generate changelog entries from commits** (for reference):
   ```bash
   # Show commits since last tag
   git log $(git describe --tags --abbrev=0)..HEAD --oneline
   
   # Show commits with more detail
   git log $(git describe --tags --abbrev=0)..HEAD --pretty=format:"- %s"
   ```

4. **Changelog categories** to use:
   - `Added` for new features
   - `Changed` for changes in existing functionality
   - `Deprecated` for soon-to-be removed features
   - `Removed` for now removed features
   - `Fixed` for any bug fixes
   - `Security` for vulnerability fixes

### Testing the CLI
```bash
# Create a test worktree
./gw start 123 main

# List worktrees
git worktree list

# Remove a worktree
./gw end 123
```

## Architecture Overview

### Command Execution Flow
The application uses a layered architecture where commands flow through:
1. **main.go** → Initializes Cobra CLI framework
2. **cmd/** → Command handlers that orchestrate business logic
3. **internal/** → Core functionality split into focused packages

### Key Design Decisions

**Git Operations via runner + Client**

All git operations go through `internal/git.Client`, which holds a `runner` interface.
The `runner` abstracts subprocess execution (four methods: `run`, `runCombined`,
`runStreaming`, `runShell`) so that tests can substitute a fake without forking real
git processes. The production implementation (`execRunner`) calls `exec.Command("git", ...)`
and uses `-C <dir>` to set the working directory inside the git invocation (rather than
`cmd.Dir`) so that missing directories produce an exit-128 error with a clear message
instead of a silent `fs.PathError`.

There is no go-git dependency; all git I/O goes through the CLI. This ensures
compatibility with the user's git configuration and SSH keys.

**Role-split git interfaces**

`internal/git/interface.go` exposes five narrow interfaces instead of one 38-method
god-interface:

| Interface | Responsibility |
|---|---|
| `RepositoryReader` | Read-only repository introspection and remote sync |
| `WorktreeManager` | Worktree lifecycle (create / remove / list) |
| `BranchManager` | Branch inspection and deletion |
| `StatusChecker` | Safety checks before destructive ops (uncommitted / unpushed / merged) |
| `EnvFileHandler` | Untracked env file discovery and copying |

`git.Interface` composes all five (plus two utility methods) and is still used by
`cmd.Dependencies.Git`. Each command struct declares its own narrower alias (e.g.
`endGit`, `startGit`, `checkoutGit`, `cleanGit`) that lists only the role interfaces
it actually needs, so the compiler enforces the minimal dependency boundary.

`git.Client` implements `git.Interface` and is the only concrete type. The mechanical
delegation layer that existed before phases 3-4 has been removed.

**Config dependency-injected via Dependencies (non-nil guaranteed)**

`cmd.Dependencies` is the single struct passed to every command constructor. It holds:
- `Git git.Interface`
- `UI ui.Interface`
- `Detect detect.Interface`
- `Config *config.Config` — **always non-nil**
- `Stdout io.Writer`, `Stderr io.Writer`

`DefaultDependencies()` loads `~/.gwrc` once. On load failure it prints a warning to
stderr and falls back to `config.New()`, so callers never need a nil guard and the old
two-constructor pattern (with/without config) has been removed.

**Worktree Naming Convention**
- Worktrees are created as `../{repository-name}-{issue-number}`
- Branches follow pattern `{issue-number}/impl`
- This keeps worktrees organized in sibling directories

**Safety-First Design — single implementation in cmd/safety.go**

Both `end` and `clean` previously had separate copies of the three pre-removal checks.
They now both call `runSafetyChecks` in `cmd/safety.go`, which runs the three checks
in parallel goroutines via a shared `sync.WaitGroup`:
  1. Uncommitted changes check
  2. Unpushed commits check (assumes unpushed if no upstream)
  3. Merge status with origin/main (fetches latest first)

`runSafetyChecks` takes a `git.StatusChecker` and returns a `safetyResult` value. Each
command then formats the raw result into its own wording. The `--force` flag on `end`
and `--force` / `--dry-run` flags on `clean` bypass the checks as before.

A git exit-128 on the uncommitted check (broken/missing worktree) sets `InvalidRepo` on
the result so callers can emit a single clear reason instead of three misleading ones.

**Package Manager Detection**
- Checks for Node.js projects first (looks for package.json)
- Determines specific Node.js package manager by lock file
- Falls back to npm if package.json exists without lock file
- Gracefully skips setup if no package manager detected

**stdout / stderr policy**

Output destinations follow a single policy so machine-parseable output and human
diagnostics never get mixed:

| Kind | Destination |
|---|---|
| Command results, progress, success messages | stdout |
| Interactive prompts (confirmation / selection) | stdout |
| Warnings (an anomaly the command continues past) | stderr |
| Errors (an anomaly that stops the command) | stderr |

Notes:
- `gw start` / `gw checkout` write their primary path/result output to stdout;
  shell integration consumes paths via the dedicated `gw shell-integration --print-path`
  command (which prints only the worktree path to stdout), not by parsing other output.
- `gw clean` treats the removable / non-removable listing as the command's result, so
  the table (including per-worktree reasons) goes to stdout; actual removal/deletion
  failures go to stderr.
- Config load failure warning goes to stderr; the command continues with defaults.

### Important Implementation Details

**Interactive UI Behavior**
- Only shows worktrees with "/impl" in the branch name
- Uses vim-style navigation (j/k keys)
- Filters out the main repository worktree

**Directory Changes**
- `start` command optionally changes to the new worktree directory after creation (controlled by `auto_cd` config)
- `end` command runs the pre-end hook with cwd set to the worktree, then restores the original directory

**Error Handling Pattern**
- Errors bubble up through return values, not panics
- User-friendly error messages with context
- Non-critical failures (like package setup or hook failures) show warnings on stderr but continue

### Known Limitations and TODOs

1. **No input validation** - Issue numbers are not validated to be numeric
2. **Hardcoded base branch** - Base branch defaults to `"main"` (`defaultBaseBranch` in `cmd/command.go`) and can only be overridden per-invocation as a positional argument to `gw start`; `internal/config` has no base-branch option

### Security Considerations
- Issue numbers are used directly in shell commands without sanitization
- Package manager commands are executed without validation
- No checks for command injection in user inputs

## Testing Approach

The codebase uses Test-Driven Development (TDD) following these principles:
1. Write tests first to define expected behavior
2. See tests fail (Red phase)
3. Write minimal code to make tests pass (Green phase)
4. Refactor while keeping tests green (Refactor phase)

Example improvements made through TDD:
- Fixed `DetectPackageManager` to return deep copies, preventing global state mutation
- Tests use temporary directories and git repositories for isolation
- Each test is self-contained with proper setup and cleanup