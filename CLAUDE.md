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

**Git Operations via CLI Commands**
- All git operations use `exec.Command()` to run git CLI commands rather than the go-git library
- This ensures compatibility with user's git configuration and SSH keys
- Trade-off: Less programmatic control but more reliable

**Worktree Naming Convention**
- Worktrees are created as `../{repository-name}-{issue-number}`
- Branches follow pattern `{issue-number}/impl`
- This keeps worktrees organized in sibling directories

**Safety-First Design**
- The `end` command performs three safety checks before removal:
  1. Uncommitted changes check
  2. Unpushed commits check (assumes unpushed if no upstream)
  3. Merge status with origin/main (fetches latest first)
- Users can bypass with `--force` flag

**Package Manager Detection**
- Checks for Node.js projects first (looks for package.json)
- Determines specific Node.js package manager by lock file
- Falls back to npm if package.json exists without lock file
- Gracefully skips setup if no package manager detected

### Important Implementation Details

**Interactive UI Behavior**
- Only shows worktrees with "/impl" in the branch name
- Uses vim-style navigation (j/k keys)
- Filters out the main repository worktree

**Directory Changes**
- `start` command changes to the new worktree directory after creation
- `end` command temporarily changes directory to check git status
- Original directory is restored if operations fail

**Error Handling Pattern**
- Errors bubble up through return values, not panics
- User-friendly error messages with context
- Non-critical failures (like package setup) show warnings but continue

### Known Limitations and TODOs

1. **Configuration not implemented** - The `internal/config` package exists but is empty
2. **Go version issue** - go.mod specifies Go 1.24.5 which doesn't exist (likely meant 1.21.5)
3. **Unused dependencies** - go-git is included but not used, could be removed
4. **No input validation** - Issue numbers aren't validated to be numeric
5. **Hardcoded defaults** - Base branch defaults to "main" with no config option

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