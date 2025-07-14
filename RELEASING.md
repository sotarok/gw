# Releasing Guide

This document describes the release process for the `gw` CLI tool.

## Release Process

### Prerequisites

1. Ensure all tests are passing
2. Update CHANGELOG.md with the new version changes
3. Ensure README.md is up to date

### Semantic Versioning

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version (v1.0.0 → v2.0.0): Incompatible API changes
- **MINOR** version (v1.0.0 → v1.1.0): New functionality in a backwards compatible manner
- **PATCH** version (v1.0.0 → v1.0.1): Backwards compatible bug fixes

### Creating a Release

1. **Ensure main branch is up to date:**
   ```bash
   git checkout main
   git pull origin main
   ```

2. **Run tests and checks:**
   ```bash
   make check
   ```

3. **Update CHANGELOG.md:**
   ```bash
   # Update changelog for the new version
   ./scripts/update-changelog.sh v0.1.0
   
   # Review and commit the changes
   git add CHANGELOG.md
   git commit -m "chore: update CHANGELOG for v0.1.0"
   ```

4. **Create and push a new tag:**
   ```bash
   # For a new patch release
   git tag v0.1.1
   
   # For a new minor release
   git tag v0.2.0
   
   # For a new major release
   git tag v1.0.0
   
   # Push the tag
   git push origin v0.1.1
   ```

5. **GitHub Actions will automatically:**
   - Run all tests
   - Build binaries for multiple platforms
   - Create a GitHub release with:
     - Changelog from commits
     - Binary downloads for all platforms
     - Checksums file
     - Installation instructions

6. **Verify the release:**
   - Check the [GitHub Releases page](https://github.com/yourusername/gw/releases)
   - Download and test a binary
   - Verify checksums

### Manual Release (if needed)

If you need to create a release manually:

```bash
# Dry run to check everything
make release-dry

# Create actual release (requires GITHUB_TOKEN)
export GITHUB_TOKEN="your-token-here"
make release
```

## Post-Release

1. **Announce the release:**
   - Create a discussion or announcement
   - Update any documentation sites
   - Tweet or post about major releases

## Troubleshooting

### Release Failed

1. Check GitHub Actions logs
2. Ensure the tag follows the `v*` pattern
3. Verify GITHUB_TOKEN has proper permissions

### Missing Binaries

1. Check `.goreleaser.yaml` configuration
2. Ensure all target platforms are included
3. Check for build errors in logs

## Best Practices

1. **Always test locally first:**
   ```bash
   make release-dry
   ```

2. **Use conventional commits:**
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation
   - `chore:` for maintenance

3. **Keep a CHANGELOG:**
   - Update CHANGELOG.md before each release
   - Use the same categories as conventional commits

4. **Version in code:**
   - Consider adding version information to the binary:
   ```go
   var (
       version = "dev"
       commit  = "none"
       date    = "unknown"
   )
   ```

5. **Test the release process:**
   - Do a dry run before the actual release
   - Test downloaded binaries on different platforms