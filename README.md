# gw - Git Worktree CLI Tool

[![CI](https://github.com/sotarok/gw/actions/workflows/ci.yml/badge.svg)](https://github.com/sotarok/gw/actions/workflows/ci.yml)
[![Release](https://github.com/sotarok/gw/actions/workflows/release.yml/badge.svg)](https://github.com/sotarok/gw/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/sotarok/gw/branch/main/graph/badge.svg)](https://codecov.io/gh/sotarok/gw)
[![Go Report Card](https://goreportcard.com/badge/github.com/sotarok/gw)](https://goreportcard.com/report/github.com/sotarok/gw)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A convenient CLI tool for managing Git worktrees with automatic package manager setup.

## Features

- Create worktrees with simple commands
- Checkout existing branches as worktrees
- Automatic detection and setup of package managers (npm, yarn, pnpm, cargo, go, pip, bundler)
- Copy untracked environment files (.env, .env.local, etc.) to new worktrees
- Interactive worktree selection for removal
- Interactive branch selection for checkout
- Safety checks before removing worktrees (uncommitted changes, unpushed commits, merge status)
- Cross-platform support (macOS, Linux)

## Installation

### Using go install

```bash
go install github.com/sotarok/gw@latest
```

### Download Binary

Download the latest binary for your platform from the [Releases page](https://github.com/sotarok/gw/releases).

#### Linux

```bash
# AMD64
curl -L https://github.com/sotarok/gw/releases/latest/download/gw_Linux_x86_64.tar.gz | tar xz
sudo mv gw /usr/local/bin/

# ARM64
curl -L https://github.com/sotarok/gw/releases/latest/download/gw_Linux_arm64.tar.gz | tar xz
sudo mv gw /usr/local/bin/
```

#### macOS

```bash
# Intel Mac
curl -L https://github.com/sotarok/gw/releases/latest/download/gw_Darwin_x86_64.tar.gz | tar xz
sudo mv gw /usr/local/bin/

# Apple Silicon
curl -L https://github.com/sotarok/gw/releases/latest/download/gw_Darwin_arm64.tar.gz | tar xz
sudo mv gw /usr/local/bin/
```


### From source

```bash
git clone https://github.com/sotarok/gw.git
cd gw
make install
```

## Usage

### Initialize configuration

```bash
# Run interactive configuration setup
gw init
```

This will create a `~/.gwrc` file with your preferences.

### Create a new worktree

```bash
# Create worktree for issue #123 (creates branch "123/impl")
gw start 123

# Create worktree with custom branch name
gw start 476/impl-migration-script

# Create worktree for feature branch
gw start feature/new-feature

# Create worktree based on specific base branch
gw start 456 develop

# Create worktree and copy environment files
gw start 789 --copy-envs
```

This will:
1. Create a new worktree at `../{repository-name}-{identifier}`
2. Create a new branch (either `{issue-number}/impl` for numbers, or the exact branch name provided)
3. Change to the new worktree directory
4. Optionally copy untracked .env files from the original repository
5. Automatically run package manager setup if detected

### Checkout an existing branch

```bash
# Checkout specific branch as worktree
gw checkout feature/auth

# Checkout remote branch
gw checkout origin/feature/api

# Interactive mode - select from list of branches
gw checkout

# Checkout and copy environment files
gw checkout feature/auth --copy-envs
```

This will:
1. Create a new worktree at `../{repository-name}-{branch-name}`
2. Checkout the specified branch (or create tracking branch for remote)
3. Change to the new worktree directory
4. Optionally copy untracked .env files from the original repository
5. Automatically run package manager setup if detected

### Remove a worktree

```bash
# Remove specific worktree
gw end 123

# Interactive mode - select from list
gw end

# Force removal without safety checks
gw end 123 --force
```

Safety checks include:
- Uncommitted changes
- Unpushed commits
- Merge status with origin/main

## Configuration

The `gw` tool can be configured via `~/.gwrc` file. Run `gw init` to create the configuration interactively.

### Configuration Options

- **auto_cd**: Automatically change to the new worktree directory after creation (default: true)

Example `~/.gwrc`:
```
# gw configuration file
auto_cd = true
```

### Shell Integration

To enable automatic directory changing after creating worktrees, you need to set up shell integration:

#### Quick Setup

Add one of these lines to your shell configuration file:

```bash
# For Bash (~/.bashrc)
eval "$(gw shell-integration --show-script --shell=bash)"

# For Zsh (~/.zshrc)
eval "$(gw shell-integration --show-script --shell=zsh)"

# For Fish (~/.config/fish/config.fish)
gw shell-integration --show-script --shell=fish | source
```

This method ensures you always have the latest shell integration code. See [SHELL_INTEGRATION.md](SHELL_INTEGRATION.md) for more details.

### Future Configuration Options

Future versions will support additional configuration:
- Default base branch
- Custom worktree location
- Package manager preferences

## Development

### Project Structure

```
gw/
├── cmd/               # Command implementations
├── internal/
│   ├── git/          # Git operations
│   ├── detect/       # Package manager detection
│   ├── ui/           # Interactive UI components
│   └── config/       # Configuration management
├── main.go
└── go.mod
```

### Building

```bash
go build -o gw
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
make test

# Generate coverage report
make coverage

# View coverage in terminal
make coverage-report
```

## License

MIT
