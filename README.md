# gw - Git Worktree CLI Tool

A convenient CLI tool for managing Git worktrees with automatic package manager setup.

## Features

- Create worktrees with simple commands
- Automatic detection and setup of package managers (npm, yarn, pnpm, cargo, go, pip, bundler)
- Interactive worktree selection for removal
- Safety checks before removing worktrees (uncommitted changes, unpushed commits, merge status)
- Cross-platform support (Windows, macOS, Linux)

## Installation

### From source

```bash
git clone https://github.com/yourusername/gw.git
cd gw
go build -o gw
# Move to your PATH
mv gw /usr/local/bin/
```

### Using go install

```bash
go install github.com/yourusername/gw@latest
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