# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

## [0.1.0] - TBD

Initial release