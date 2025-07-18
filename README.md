# gw - Git Worktree CLI Tool

[![CI](https://github.com/sotarok/gw/actions/workflows/ci.yml/badge.svg)](https://github.com/sotarok/gw/actions/workflows/ci.yml)
[![Release](https://github.com/sotarok/gw/actions/workflows/release.yml/badge.svg)](https://github.com/sotarok/gw/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/sotarok/gw)](https://goreportcard.com/report/github.com/sotarok/gw)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A convenient CLI tool for managing Git worktrees with automatic package manager setup.

## Features

- Create worktrees with simple commands
- Automatic detection and setup of package managers (npm, yarn, pnpm, cargo, go, pip, bundler)
- Interactive worktree selection for removal
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

### Create a new worktree

```bash
# Create worktree for issue #123 based on main branch
gw start 123

# Create worktree based on specific branch
gw start 456 develop
```

This will:
1. Create a new worktree at `../{repository-name}-{issue-number}`
2. Create a new branch `{issue-number}/impl`
3. Change to the new worktree directory
4. Automatically run package manager setup if detected

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

Future versions will support configuration via `.gwconfig` file for:
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
go test ./...
```

## License

MIT
