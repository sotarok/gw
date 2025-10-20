# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Add `copy_envs` configuration option for automatic .env file copying behavior
  - Configure default behavior in `~/.gwrc` to avoid interactive prompts
  - Three-tier priority system: `--copy-envs` flag > config setting > interactive prompt
  - When unset, maintains existing interactive prompt behavior
  - Can be toggled in `gw config` interactive UI or `gw init` setup wizard

## [0.5.1] - 2025-09-05

### Fixed
- Fix `gw end` incorrectly warning about unpushed commits for merged branches with deleted remotes
  - The command now correctly detects when a branch has been merged to main even if the remote branch was deleted
  - This commonly occurs after PR merges with automatic branch deletion on GitHub

### Changed
- Improved test coverage:
  - internal/git package coverage increased to 94.6%
  - internal/detect package coverage increased to 100%

### Documentation
- Updated README to include Composer in the list of supported package managers

## [0.5.0] - 2025-08-21

### Added
- New `gw config` command for viewing and editing configuration interactively
  - Interactive TUI with arrow/j/k navigation for browsing settings
  - Toggle boolean values with Enter/Space keys
  - Save changes with 's' key
  - Non-interactive mode with `--list` flag for CI/scripting
  - Shows configuration descriptions and default values
  - Visual indicators (✅/❌) for boolean settings
- New `auto_remove_branch` configuration option for automatic branch deletion
  - When enabled, automatically deletes the local branch after successful worktree removal
  - Works similarly to GitHub's "Automatically delete head branches" feature
  - Default value is false to avoid unexpected behavior
  - Branch deletion errors are non-fatal and only show warnings
- Composer support for automatic dependency installation (Thanks [@zonuexe](https://github.com/zonuexe)! #10)
  - Detects PHP projects with composer.json
  - Automatically runs `composer install` when setting up worktrees

### Changed
- Improved test coverage significantly:
  - Added comprehensive tests for DefaultClient in interface.go
  - Increased internal/git package coverage from 42.9% to 79.2%
  - Added tests for config Update method
  - Added tests for gw config command

## [0.4.0] - 2025-08-14

### Added
- Configuration file support via `~/.gwrc`
- New `gw init` command for interactive configuration setup
- Auto-cd configuration option to control directory change behavior after worktree creation
- New `gw shell-integration` command with multiple features:
  - `--show-script` flag to dynamically generate shell integration code
  - `--shell` parameter to specify shell type (bash/zsh/fish)
  - `--print-path` flag to get worktree paths for specific issues/branches
  - Supports eval-based setup for always up-to-date shell integration
- Test coverage reporting with Codecov integration
- GitHub Actions workflow for automated test coverage
- Makefile commands for local coverage reporting (`make coverage`, `make coverage-report`)
- iTerm2 tab name update feature:
  - New `update_iterm2_tab` configuration option (default: false)
  - Automatically updates tab name to "{repo name} {issue num/branch}" when creating/switching worktrees
  - Resets tab name when removing worktrees
  - Works with `gw start`, `gw checkout`, and `gw end` commands
  - Only active when running in iTerm2 terminal
- Support for custom branch names in `gw start` command:
  - Accepts full branch names (e.g., `gw start 476/impl-migration-script`)
  - Only appends `/impl` to pure numbers (e.g., `gw start 123` → branch `123/impl`)
  - Properly sanitizes branch names for directory creation

### Changed
- Refactored cmd package to use dependency injection pattern for better testability
- Test coverage increased from ~20% to 71.2%

### Fixed
- Worktrees are now created relative to repository root directory, not current working directory

## [0.3.1] - 2025-07-31

### Fixed
- Fix `gw start` command failing to find environment files (exit status 128)
- Use original repository root for env file search in `gw start` command
- Ensure only untracked env files are copied (exclude .env.example and similar tracked files)
- Show environment files list before confirmation prompt in both `start` and `checkout` commands

## [0.3.0] - 2025-07-30

### Added
- Add `--copy-envs` flag to `start` and `checkout` commands to copy untracked .env files
- Interactive prompt to copy environment files when `--copy-envs` flag is not specified
- Automatic detection of untracked .env files (excluding .git, node_modules, vendor, dist, build directories)
- Display list of environment files to be copied before copying
- Add `BranchExists` function to validate branch existence before checkout

### Fixed
- Show environment files list before confirmation prompt for better UX
- Use original repository root for finding env files in checkout command
- Include non-numeric branches in `gw end` interactive mode (now shows all branches with "/impl" pattern)
- Improve error message for non-existent branches in checkout command
- Handle CI environments without remote branches in tests

## [0.2.0] - 2025-07-28

### Added
- New `gw checkout` command to create worktrees from existing branches
- Interactive branch selection when no branch is specified for checkout
- Support for checking out both local and remote branches
- Automatic creation of tracking branches for remote branches
- Proper sanitization of branch names for directory creation (handles special characters like /, \, *, ?, <, >, |, :, ")

## [0.1.1] - 2025-07-16

### Fixed
- Update module path from `gw` to `github.com/sotarok/gw` for go install compatibility
- Fix import paths throughout the codebase
- Fix Go version in go.mod (was incorrectly set to 1.24.5)

### Changed
- Add coverage.txt to .gitignore

### Documentation
- Add pre-commit checklist to CLAUDE.md (run `make fmt` and `make check` before committing)

## [0.1.0] - 2025-07-14

### Added
- Initial implementation of `gw start` command for creating worktrees
- Initial implementation of `gw end` command for removing worktrees
- Interactive worktree selection when no issue number is provided
- Automatic package manager detection and setup (npm, yarn, pnpm, cargo, go, pip, bundler)
- Safety checks before removing worktrees
- Cross-platform support (Linux, macOS, Windows)
- Comprehensive test suite
- GitHub Actions CI/CD pipeline
- GoReleaser configuration for automated releases

### Security
- Deep copy of PackageManager structs to prevent global state mutation

