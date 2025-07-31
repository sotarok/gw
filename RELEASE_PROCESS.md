# Release Process

This document describes the release process for the gw project.

## Prerequisites

1. Ensure all changes are merged to main
2. Update CHANGELOG.md with all changes under [Unreleased]
3. Ensure all tests pass: `make check`
4. Ensure dependencies are up to date: `go mod tidy`
5. Ensure GoReleaser is installed: `go install github.com/goreleaser/goreleaser@latest`

## Release Steps

### 1. Update CHANGELOG.md

Move all entries from `[Unreleased]` to a new version section:

```markdown
## [Unreleased]

## [0.4.0] - 2025-01-31

### Added
- (move items here)

### Changed
- (move items here)

### Fixed
- (move items here)
```

### 2. Commit the changelog

```bash
git add CHANGELOG.md
git commit -m "chore: prepare for v0.4.0 release"
git push origin main
```

### 3. Create and push the tag

```bash
git tag -a v0.4.0 -m "Release v0.4.0"
git push origin v0.4.0
```

### 4. Verify the release

The GitHub Actions workflow will automatically:
1. Run tests
2. Build binaries for all platforms
3. Create a GitHub release with:
   - Automatically generated changelog from commit messages
   - Binary artifacts
   - Installation instructions

### 5. Post-release

After successful release:
1. Check the [releases page](https://github.com/sotarok/gw/releases)
2. Verify all artifacts are uploaded
3. Test installation: `go install github.com/sotarok/gw@latest`

## Versioning

We follow [Semantic Versioning](https://semver.org/):
- MAJOR version for incompatible API changes
- MINOR version for new functionality in a backwards compatible manner
- PATCH version for backwards compatible bug fixes

## Commit Message Format

Use [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` for new features (triggers MINOR version bump)
- `fix:` for bug fixes (triggers PATCH version bump)
- `feat!:` or `fix!:` for breaking changes (triggers MAJOR version bump)
- `refactor:` for code refactoring
- `perf:` for performance improvements
- `docs:` for documentation only
- `test:` for test additions/changes
- `chore:` for maintenance tasks
- `ci:` for CI/CD changes

## GoReleaser Changelog

GoReleaser automatically generates a changelog based on commit messages since the last tag.
The changelog groups commits by type:
- Features (feat)
- Bug Fixes (fix)
- Performance (perf)
- Refactors (refactor)

Commits with `docs:`, `test:`, `chore:`, and `ci:` prefixes are excluded from the release notes.

## Testing the Release (Dry Run)

Before creating an actual release, you can test the process:

```bash
# Test the release process without pushing
make release-dry
```

This will:
- Build all binaries
- Generate the changelog
- Create release artifacts locally
- Show what would be released

## Troubleshooting

### Release Failed
1. Check GitHub Actions logs
2. Ensure the tag follows the `v*` pattern
3. Verify GitHub Actions has proper permissions

### Missing Binaries
1. Check `.goreleaser.yaml` configuration
2. Ensure all target platforms are included
3. Check for build errors in logs

### go install Not Working
1. Wait a few minutes for proxy.golang.org to update
2. Check availability: `curl https://proxy.golang.org/github.com/sotarok/gw/@v/list`
3. Clear local module cache: `go clean -modcache`