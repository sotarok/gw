# gw — Git Worktree CLI

[![CI](https://github.com/sotarok/gw/actions/workflows/ci.yml/badge.svg)](https://github.com/sotarok/gw/actions/workflows/ci.yml)
[![Release](https://github.com/sotarok/gw/actions/workflows/release.yml/badge.svg)](https://github.com/sotarok/gw/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/sotarok/gw/branch/main/graph/badge.svg)](https://codecov.io/gh/sotarok/gw)
[![Go Report Card](https://goreportcard.com/badge/github.com/sotarok/gw)](https://goreportcard.com/report/github.com/sotarok/gw)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> Stop juggling `git worktree add ../repo-123 -b 123/impl && cd && npm install && cp .env`.
> `gw start 123` does all of it — and `gw end 123` cleans up safely.

```
$ gw start 123
✓ Created worktree at ../myrepo-123
Detected npm, running setup...
✓ npm setup completed

✨ Worktree ready at:
   ../myrepo-123

💡 Shell integration will change to this directory after the command completes.

$ gw end 123
✓ Successfully removed worktree for issue #123
```

## Why gw?

| plain git worktree | gw |
|---|---|
| `git worktree add ../repo-123 -b 123/impl && cd ../repo-123 && npm install && cp ../.env .` | `gw start 123` |
| `git worktree remove` refuses on a dirty tree — or discards it with `--force` | `gw end` checks uncommitted changes, unpushed commits, and merge status, then asks before removing |
| Worktrees pile up across old issues | `gw clean` bulk-removes every merged, clean worktree in one command |

## Quick Start

1. **Install**

   ```bash
   go install github.com/sotarok/gw@latest
   ```

2. **Set up shell integration** (required for auto-cd)

   ```bash
   # Add to ~/.zshrc
   eval "$(gw shell-integration --show-script --shell=zsh)"
   ```

   Bash and Fish variants are in the [Shell Integration](#shell-integration) section.

3. **Create a worktree**

   ```bash
   gw start 123
   ```

4. **Remove it when you're done**

   ```bash
   gw end 123
   ```

That's it. Everything below is reference.

## Features

**Core**
- One command to create a worktree, check out a branch, install dependencies, and optionally copy `.env` files (`gw start` / `gw checkout`)
- Automatic package-manager detection and setup: npm, yarn, pnpm, cargo, go, pip, bundler, composer
- Auto-cd into the new worktree directory via shell integration
- Interactive branch/worktree selection when no argument is given

**Safety**
- Three pre-removal checks run in parallel before `gw end` or `gw clean`: uncommitted changes, unpushed commits, and merge status against the base branch
- `gw clean --dry-run` previews what would be removed before touching anything
- direnv-style trust model for project-local hook files (`.gwrc`)

**Integrations**
- Lifecycle hooks: `post_start_hook`, `post_checkout_hook`, `pre_end_hook`
- Project-local `.gwrc` at the repository root overrides hook keys per-repo (new in v1.1)
- iTerm2 tab name updated automatically when creating, switching, or removing worktrees
- Zsh completion via shell integration (`gw end` completes worktree branch names)

## Installation

### Using go install

```bash
go install github.com/sotarok/gw@latest
```

### Download Binary

Download the latest binary for your platform from the [Releases page](https://github.com/sotarok/gw/releases).

<details>
<summary>Linux</summary>

```bash
# AMD64
curl -L https://github.com/sotarok/gw/releases/latest/download/gw_Linux_x86_64.tar.gz | tar xz
sudo mv gw /usr/local/bin/

# ARM64
curl -L https://github.com/sotarok/gw/releases/latest/download/gw_Linux_arm64.tar.gz | tar xz
sudo mv gw /usr/local/bin/
```

</details>

<details>
<summary>macOS</summary>

```bash
# Intel Mac
curl -L https://github.com/sotarok/gw/releases/latest/download/gw_Darwin_x86_64.tar.gz | tar xz
sudo mv gw /usr/local/bin/

# Apple Silicon
curl -L https://github.com/sotarok/gw/releases/latest/download/gw_Darwin_arm64.tar.gz | tar xz
sudo mv gw /usr/local/bin/
```

</details>

### From Source

```bash
git clone https://github.com/sotarok/gw.git
cd gw
make install
```

## Commands

### gw start

Create a new worktree for an issue number or branch name.

```bash
# Create worktree for issue #123 — creates branch "123/impl"
gw start 123

# Branch name with "/" — used exactly as given
gw start 476/impl-migration-script

# Named feature branch
gw start feature/new-feature

# Different base branch
gw start 456 develop

# Also copy .env files from the main worktree
gw start 789 --copy-envs
```

This will:
1. Create a new worktree at `../{repository-name}-{identifier}`
2. Create a new branch (`{issue-number}/impl` for plain numbers, or the exact name provided)
3. Optionally copy untracked `.env` files from the original repository
4. Run package-manager setup if a package manager is detected
5. Change to the new worktree directory (requires shell integration)

| Flag | Description |
|---|---|
| `--copy-envs` | Copy untracked `.env` files to the new worktree |
| `--no-fetch` | Skip `git fetch` before running the command |
| `--no-project-hooks` | Skip project-local `.gwrc` hook overrides for this run |

### gw checkout

Checkout an existing branch as a new worktree. If no branch is given, an interactive selector is shown.

```bash
# Checkout a specific branch
gw checkout feature/auth

# Checkout a remote branch
gw checkout origin/feature/api

# Interactive mode — select from list
gw checkout

# Checkout and copy .env files
gw checkout feature/auth --copy-envs
```

This will:
1. Create a new worktree at `../{repository-name}-{branch-name}`
2. Checkout the specified branch (or create a local tracking branch for a remote)
3. Optionally copy untracked `.env` files from the original repository
4. Run package-manager setup if a package manager is detected
5. Change to the new worktree directory (requires shell integration)

| Flag | Description |
|---|---|
| `--copy-envs` | Copy untracked `.env` files to the new worktree |
| `--no-fetch` | Skip `git fetch` before running the command |
| `--no-project-hooks` | Skip project-local `.gwrc` hook overrides for this run |

### gw end

Remove a worktree. If no issue number is given, an interactive selector is shown.

```bash
# Remove the worktree for issue #123
gw end 123

# Interactive mode — select from list
gw end

# Skip safety checks and remove immediately
gw end 123 --force
```

Before removing, `gw end` runs three safety checks in parallel:
- Uncommitted changes in the worktree
- Unpushed commits on the branch
- Whether the branch is merged into the base branch

If any check trips, `gw end` prints the warnings and prompts for confirmation. Use `--force` to skip all checks.

| Flag | Short | Description |
|---|---|---|
| `--force` | `-f` | Force removal without safety checks |
| `--no-fetch` | | Skip `git fetch` before running the command |
| `--no-project-hooks` | | Skip project-local `.gwrc` hook overrides for this run |

### gw clean

Bulk-remove all worktrees that are safe to delete. Useful for clearing out merged work after a sprint.

```bash
# Preview what would be removed
gw clean --dry-run

# Confirm and remove all merged/clean worktrees
gw clean

# Remove without the confirmation prompt
gw clean --force
```

`gw clean` evaluates each worktree against the same three safety checks as `gw end`, then displays a table showing which worktrees are removable and which are not (with per-worktree reasons). It asks for confirmation before removing anything, unless `--force` is given.

`--dry-run` shows the table but skips the confirmation and removal entirely.

The `pre_end_hook` runs for each worktree that is about to be removed, with cwd set to that worktree.

| Flag | Short | Description |
|---|---|---|
| `--force` | `-f` | Remove without confirmation prompt |
| `--dry-run` | | Show what would be removed without removing |
| `--no-fetch` | | Skip `git fetch` before running the command |
| `--no-project-hooks` | | Skip project-local `.gwrc` hook overrides for this run |

### gw config

View and edit configuration interactively, or list current values.

```bash
# Open the interactive TUI editor
gw config

# Print all configuration values (non-interactive)
gw config --list
```

The interactive editor supports:
- Arrow keys or `j`/`k` to navigate
- `Enter` or `Space` to toggle boolean values
- `s` to save
- `q` to quit
- `?` to view help

### gw init

Run an interactive setup to create `~/.gwrc` with your preferences.

```bash
gw init
```

### gw shell-integration

Print the shell integration script. Normally consumed via `eval` in your shell config — see [Shell Integration](#shell-integration).

```bash
gw shell-integration --show-script --shell=zsh
```

## Configuration

All configuration lives in `~/.gwrc`. Use `gw init` for first-time setup or `gw config` to edit at any time.

### Key Reference

| Key | Default | Description |
|---|---|---|
| `auto_cd` | `true` | Automatically change directory to the new worktree after creation (requires shell integration) |
| `update_iterm2_tab` | `false` | Update iTerm2 tab title with worktree information (macOS only) |
| `auto_remove_branch` | `false` | Automatically delete the local branch after successful worktree removal |
| `copy_envs` | *(unset)* | Copy `.env` files to new worktrees. When unset (nil), `gw` prompts each time; set to `true` or `false` to fix the behavior |
| `fetch_before_command` | `true` | Run `git fetch --all --prune` before commands to sync remote branch info |
| `post_start_hook` | *(empty)* | Shell command to execute after a successful `gw start` |
| `post_checkout_hook` | *(empty)* | Shell command to execute after a successful `gw checkout` |
| `pre_end_hook` | *(empty)* | Shell command to execute before a worktree is removed by `gw end` or `gw clean`, with cwd set to the worktree |

### Example `~/.gwrc`

```
# gw configuration file
auto_cd = true
update_iterm2_tab = false
auto_remove_branch = false
# copy_envs = false  # Uncomment to set default behavior
fetch_before_command = true

# Hook commands executed after successful worktree operations
# Available env vars: GW_WORKTREE_PATH, GW_BRANCH_NAME, GW_REPO_NAME, GW_COMMAND
# post_start_hook =
# post_checkout_hook =

# Hook commands executed before a worktree is removed (from end/clean)
# Runs with cwd set to the worktree. Same env vars as above; GW_COMMAND is "end" or "clean"
# pre_end_hook =
```

### Hooks

Hook commands are executed via `sh -c` with the following environment variables:

| Variable | Description |
|---|---|
| `GW_WORKTREE_PATH` | Absolute path to the worktree |
| `GW_BRANCH_NAME` | Branch name of the worktree |
| `GW_REPO_NAME` | Repository name |
| `GW_COMMAND` | The command that triggered the hook (`start`, `checkout`, `end`, or `clean`) |

Hook failures are treated as warnings and do not block the overall command.

| Hook | Fires |
|---|---|
| `post_start_hook` | After `gw start` successfully creates a worktree |
| `post_checkout_hook` | After `gw checkout` successfully creates a worktree |
| `pre_end_hook` | Before `gw end` removes a worktree, and before each worktree `gw clean` removes. Runs with cwd set to the worktree so it can operate on files that are about to disappear |

#### Example: tmux integration

Open a new tmux window for the worktree instead of changing directory in the current shell:

```
auto_cd = false
post_start_hook = tmux new-window -c "$GW_WORKTREE_PATH" -n "$GW_BRANCH_NAME"
post_checkout_hook = tmux new-window -c "$GW_WORKTREE_PATH" -n "$GW_BRANCH_NAME"
```

#### Example: iTerm2 new tab

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

#### Example: Custom notification

Send a desktop notification after worktree creation:

```
post_start_hook = osascript -e 'display notification "Worktree ready at '"$GW_WORKTREE_PATH"'" with title "gw"'
```

#### Example: docker compose cleanup on worktree removal

If each worktree runs its own `docker compose` stack, tear it down before the worktree is deleted. `pre_end_hook` runs with the worktree as its working directory, so relative paths work naturally:

```
pre_end_hook = docker compose down -v --remove-orphans
```

Or use the bundled example script (`examples/hooks/docker-compose-down.sh`) which skips worktrees with no compose file:

```
pre_end_hook = ~/.gw/hooks/docker-compose-down.sh
```

### iTerm2 Tab Integration

When `update_iterm2_tab` is enabled and you're using iTerm2:
- Tab name is updated to `{repository-name} {issue-number/branch-name}` when creating or switching worktrees
- Tab name is reset when removing worktrees
- Only activates when running inside iTerm2 (automatically detected)

## Project-Local Configuration & Trust

For repository-specific hook commands (e.g. a `post_start_hook` that only makes sense for one project), place a `.gwrc` at the **main worktree root** — the same directory as the repository's `.git` — using the same `key = value` format as `~/.gwrc`.

```
# <repo>/.gwrc
post_start_hook = pnpm dev
```

**Scope (v1.1): hooks only.** Only the three hook keys — `post_start_hook`, `post_checkout_hook`, `pre_end_hook` — can be overridden per project. Any other key (`auto_cd`, `update_iterm2_tab`, `auto_remove_branch`, `copy_envs`, `fetch_before_command`) is parsed but never applied from a project `.gwrc`; `gw` prints a one-line note to stderr (`note: project .gwrc key 'auto_cd' is ignored in v1.1 (hooks-only)`) and keeps using the global value.

**Merge rule.** The global `~/.gwrc` is the base. Only the hook keys the project file actually writes are overridden — a hook key the project file doesn't mention keeps its global value. Writing a hook key with an empty value (`post_start_hook =`) disables that global hook for the project, without needing trust approval (an empty value can't execute code).

**Where the project file is found.** For a linked worktree (created by `gw start`/`gw checkout`), `gw` resolves `git rev-parse --git-common-dir`'s parent directory as the main worktree root and reads `.gwrc` there, so every linked worktree reads the *same* project `.gwrc` as the main repository; a `.gwrc` accidentally checked out inside a linked worktree itself is never read. For every other layout — including a `--separate-git-dir` checkout or a git submodule, where `--git-common-dir`'s parent is not the worktree root — `gw` instead resolves the root via `git rev-parse --show-toplevel`.

### Trust

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

### Accepted Limitations

- **Trust covers the `.gwrc` file only**, not any script it references. `post_start_hook = ./scripts/setup.sh` is approved by the content of `.gwrc`, not the content of `setup.sh` — if `setup.sh` changes later, no re-approval is triggered (the same boundary direnv draws around `.envrc` and the files it sources).
- **Relative-path hook values are resolved against the worktree's cwd at hook-execution time**, not the main root where `.gwrc` was approved. Since `gw` changes into the new/target worktree before running `post_start_hook`/`post_checkout_hook` (and into the worktree being removed for `pre_end_hook`), a relative path actually runs whatever file exists at that path *on the checked-out branch* — which can differ from what you approved if you check out a different branch afterward. **Prefer absolute paths** for hook scripts (`post_start_hook = /abs/path/to/scripts/setup.sh`) to avoid this entirely.
- **Hook values must fit on one physical line** — the config parser does not support line continuations. Wrap multi-command hooks in a script file and reference it by path.

See `examples/hooks/README.md` for a worked example.

## Shell Integration

Shell integration is what makes `auto_cd` work: after `gw start` or `gw checkout` prints the worktree path to stdout, the shell wrapper reads it and runs `cd` for you in the current shell process.

Add one of these lines to your shell configuration file:

```bash
# For Bash (~/.bashrc)
eval "$(gw shell-integration --show-script --shell=bash)"

# For Zsh (~/.zshrc)
eval "$(gw shell-integration --show-script --shell=zsh)"

# For Fish (~/.config/fish/config.fish)
gw shell-integration --show-script --shell=fish | source
```

This method ensures you always have the latest shell integration code. See [SHELL_INTEGRATION.md](SHELL_INTEGRATION.md) for full details.

## Troubleshooting / FAQ

**`gw start` doesn't change my directory**

Shell integration is not set up. Add the `eval` line for your shell to your shell config file and reload it — see [Shell Integration](#shell-integration) above.

**`gw` asks me to trust a `.gwrc`**

The repository you cloned ships a project-local hook file. `gw` shows the hook values before running them. Approve to run the hooks, decline to fall back to your global config, or pass `--no-project-hooks` to skip project hooks for that invocation without being prompted.

**`gw end` refuses to remove my worktree**

The safety checks found uncommitted changes, unpushed commits, or a branch not yet merged into the base branch. `gw end` prints the specific reason(s). Resolve them first, or use `gw end --force` to override all checks.

**How do I skip the automatic fetch?**

Pass `--no-fetch` to any command for a one-off skip, or set `fetch_before_command = false` in `~/.gwrc` to disable it permanently.

## Development

### Project Structure

```
gw/
├── cmd/               # Command implementations (start, checkout, end, clean, config, …)
├── examples/hooks/    # Example hook scripts (tmux, iTerm2, docker-compose, etc.)
├── internal/
│   ├── config/       # Configuration loading, saving, and fieldSpecs table
│   ├── detect/       # Package-manager detection and setup
│   ├── git/          # Git operations via CLI subprocess (no go-git)
│   ├── hook/         # Lifecycle hook execution
│   ├── iterm2/       # iTerm2 tab-name integration
│   ├── spinner/      # Terminal spinner for long-running operations
│   ├── trust/        # Trust store for project-local hook approval
│   └── ui/           # Interactive TUI components (worktree/branch selector)
├── main.go
└── go.mod
```

### Building

```bash
go build -o gw
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
make test

# Generate coverage report
make coverage

# View coverage in terminal
make coverage-report
```

Run `make fmt && make check` as the pre-commit gate — format first, then lint + tests.

## License

MIT
