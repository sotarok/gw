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
- Automatic detection and setup of package managers (npm, yarn, pnpm, cargo, go, pip, bundler, composer)
- Copy untracked environment files (.env, .env.local, etc.) to new worktrees
- Interactive worktree selection for removal
- Interactive branch selection for checkout
- Safety checks before removing worktrees (uncommitted changes, unpushed commits, merge status)
- iTerm2 tab name integration - automatically update tab names when switching worktrees
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

# View and edit configuration interactively
gw config

# List configuration values (non-interactive)
gw config --list
```

The `gw init` command creates a `~/.gwrc` file with your preferences through an interactive setup.
The `gw config` command allows you to view and modify your configuration at any time.

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

The `gw` tool can be configured via `~/.gwrc` file. Use `gw init` for initial setup or `gw config` to modify settings at any time.

### Managing Configuration

```bash
# Interactive configuration editor (TUI)
gw config

# List current configuration
gw config --list
```

The interactive config editor allows you to:
- Navigate settings with arrow keys or j/k
- Toggle boolean values with Enter or Space
- Save changes with 's'
- Quit with 'q'
- View help with '?'

### Configuration Options

- **auto_cd**: Automatically change to the new worktree directory after creation (default: true)
- **update_iterm2_tab**: Update iTerm2 tab name when creating/switching/removing worktrees (default: false)
- **post_start_hook**: Shell command to execute after a successful `gw start` (default: empty)
- **post_checkout_hook**: Shell command to execute after a successful `gw checkout` (default: empty)
- **pre_end_hook**: Shell command to execute before a worktree is removed by `gw end` or `gw clean`, with cwd set to the worktree (default: empty)

Example `~/.gwrc`:
```
# gw configuration file
auto_cd = true
update_iterm2_tab = false
```

#### Hooks

You can configure shell commands to run automatically around worktree lifecycle events. Hook commands are executed via `sh -c` with the following environment variables:

| Variable | Description |
|---|---|
| `GW_WORKTREE_PATH` | Absolute path to the worktree |
| `GW_BRANCH_NAME` | Branch name of the worktree |
| `GW_REPO_NAME` | Repository name |
| `GW_COMMAND` | The command that triggered the hook (`start`, `checkout`, `end`, or `clean`) |

Hook failures are treated as warnings and do not block the overall command.

Available hooks:

| Hook | Fires |
|---|---|
| `post_start_hook` | After `gw start` successfully creates a worktree |
| `post_checkout_hook` | After `gw checkout` successfully creates a worktree |
| `pre_end_hook` | Before `gw end` removes a worktree, and before each worktree `gw clean` removes. Runs with cwd set to the worktree so it can operate on files that are about to disappear |

##### Example: tmux integration

Open a new tmux window for the worktree instead of changing directory in the current shell:

```
auto_cd = false
post_start_hook = tmux new-window -c "$GW_WORKTREE_PATH" -n "$GW_BRANCH_NAME"
post_checkout_hook = tmux new-window -c "$GW_WORKTREE_PATH" -n "$GW_BRANCH_NAME"
```

##### Example: iTerm2 new tab

Open a new iTerm2 tab for the worktree on macOS. First, create a hook script:

```bash
mkdir -p ~/.gw/hooks
cat > ~/.gw/hooks/iterm2-new-tab.sh << 'EOF'
#!/bin/bash
osascript <<APPLESCRIPT
tell application "iTerm"
    tell current window
        create tab with default profile
        tell current session of current tab
            write text "cd '${GW_WORKTREE_PATH}' && clear"
            set name to "${GW_BRANCH_NAME}"
        end tell
    end tell
end tell
APPLESCRIPT
EOF
chmod +x ~/.gw/hooks/iterm2-new-tab.sh
```

Then configure `.gwrc`:

```
auto_cd = false
post_start_hook = ~/.gw/hooks/iterm2-new-tab.sh
post_checkout_hook = ~/.gw/hooks/iterm2-new-tab.sh
```

##### Example: Custom notification

Send a desktop notification after worktree creation:

```
post_start_hook = osascript -e 'display notification "Worktree ready at '"$GW_WORKTREE_PATH"'" with title "gw"'
```

##### Example: docker compose cleanup on worktree removal

If each worktree runs its own `docker compose` stack, tear it down before the worktree is deleted. `pre_end_hook` runs with the worktree as its working directory, so relative paths work naturally:

```
pre_end_hook = docker compose down -v --remove-orphans
```

Or use the bundled example script (`examples/hooks/docker-compose-down.sh`) which skips worktrees with no compose file:

```
pre_end_hook = ~/.gw/hooks/docker-compose-down.sh
```

#### iTerm2 Tab Integration

When `update_iterm2_tab` is enabled and you're using iTerm2:
- Tab name is updated to "{repository-name} {issue-number/branch-name}" when creating or switching worktrees
- Tab name is reset when removing worktrees
- Only works when running in iTerm2 terminal (automatically detected)

### Project-Local Configuration

For repository-specific hook commands (e.g. a `post_start_hook` that only makes sense for one project), place a `.gwrc` at the **main worktree root** — the same directory as the repository's `.git` — using the same `key = value` format as `~/.gwrc`.

```
# <repo>/.gwrc
post_start_hook = pnpm dev
```

**Scope (v1.1): hooks only.** Only the three hook keys — `post_start_hook`, `post_checkout_hook`, `pre_end_hook` — can be overridden per project. Any other key (`auto_cd`, `update_iterm2_tab`, `auto_remove_branch`, `copy_envs`, `fetch_before_command`) is parsed but never applied from a project `.gwrc`; `gw` prints a one-line note to stderr (`note: project .gwrc key 'auto_cd' is ignored in v1.1 (hooks-only)`) and keeps using the global value.

**Merge rule.** The global `~/.gwrc` is the base. Only the hook keys the project file actually writes are overridden — a hook key the project file doesn't mention keeps its global value. Writing a hook key with an empty value (`post_start_hook =`) disables that global hook for the project, without needing trust approval (an empty value can't execute code).

**Where the project file is found.** `gw` resolves `git rev-parse --git-common-dir`'s parent directory as the main worktree root and reads `.gwrc` there — so a linked worktree (created by `gw start`/`gw checkout`) reads the *same* project `.gwrc` as the main repository; a `.gwrc` accidentally checked out inside a linked worktree itself is never read. This resolution does not support `--separate-git-dir` layouts or standard git submodules (where `--git-common-dir`'s parent isn't the worktree root); in those layouts `gw` silently falls back to the global config only.

#### Trust

Because a project `.gwrc` ships with the repository, its hook values could run arbitrary code as soon as someone runs a `gw` command in a clone they don't fully trust. `gw` uses a direnv-style trust model:

- The first time a project `.gwrc` declares a **non-empty** hook value, `gw` prompts (default: **No**) before running it, showing the file path and the hook value(s) awaiting approval.
- Approval is keyed by a hash of the file's absolute path *and* content. Editing the file — even by one character — invalidates the old approval and re-prompts. A different clone (different absolute path) of the same content also re-prompts.
- Approval is stored in `~/.gw/trust/<hash>` and applies repo-wide: once one worktree approves a project `.gwrc`, every other worktree of that same repository (which all read the same main-root file) uses it without re-prompting, as long as the content hasn't changed.
- The prompt appears on stderr / your terminal, never on stdout — `gw start`/`gw checkout`'s stdout is consumed by shell integration and must stay machine-parseable.
- If stdin isn't a terminal (e.g. running in CI or a script), or the prompt is declined, or the trust store can't be written, `gw` **fails closed**: the untrusted project value is not used, the command falls back to the global value for that key, and a warning is printed to stderr. The command itself still completes.
- `gw shell-integration` and `gw config --list` never trigger this prompt: the former's stdout must stay clean, and the latter only reads the existing trust state to display it.

Use `--no-project-hooks` to skip project hook overrides entirely for one invocation (no prompt, global values used):

```bash
gw start 123 --no-project-hooks
```

`gw end --force` and `gw clean --force`/`--dry-run` also skip project hooks automatically (no prompt) — these are typically scripted/non-interactive invocations where blocking on a prompt would be unwelcome. Note that under any of these skip paths, even an *empty*-value override (which normally needs no trust) does not apply — global values are used unconditionally.

Check what's currently in effect, including origin and trust state, with:

```bash
gw config --list
```

```
post_start_hook     : pnpm dev  [project] [trusted]
post_checkout_hook  : (not set) [default]
pre_end_hook        : (not set) [global]
```

An untrusted project override is shown with the effective (global) value plus a note, e.g. `[project, untrusted — global value used]`, rather than being hidden.

#### Accepted limitations

- **Trust covers the `.gwrc` file only**, not any script it references. `post_start_hook = ./scripts/setup.sh` is approved by the content of `.gwrc`, not the content of `setup.sh` — if `setup.sh` changes later, no re-approval is triggered (the same boundary direnv draws around `.envrc` and the files it sources).
- **Relative-path hook values are resolved against the worktree's cwd at hook-execution time**, not the main root where `.gwrc` was approved. Since `gw` changes into the new/target worktree before running `post_start_hook`/`post_checkout_hook` (and into the worktree being removed for `pre_end_hook`), a relative path actually runs whatever file exists at that path *on the checked-out branch* — which can differ from what you approved if you check out a different branch afterward. **Prefer absolute paths** for hook scripts (`post_start_hook = /abs/path/to/scripts/setup.sh`) to avoid this entirely.
- **Hook values must fit on one physical line** — the config parser does not support line continuations. Wrap multi-command hooks in a script file and reference it by path.

See `examples/hooks/README.md` for a worked example.

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
├── examples/hooks/    # Example hook scripts (tmux, iTerm2, etc.)
├── internal/
│   ├── git/          # Git operations
│   ├── detect/       # Package manager detection
│   ├── ui/           # Interactive UI components
│   ├── config/       # Configuration management
│   ├── hook/         # Post-command hook execution
│   └── iterm2/       # iTerm2 integration
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
