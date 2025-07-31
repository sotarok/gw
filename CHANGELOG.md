# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

