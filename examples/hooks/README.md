# Hook Examples

Example hook scripts for use with `gw`'s hook configuration:

- `post_start_hook` — runs after `gw start` creates a worktree
- `post_checkout_hook` — runs after `gw checkout` creates a worktree
- `pre_end_hook` — runs before `gw end` (and for each worktree `gw clean` removes), **with cwd set to the worktree** so you can clean up resources tied to the worktree before the directory is deleted

## Setup

Copy the desired script to `~/.gw/hooks/` and make it executable:

```bash
mkdir -p ~/.gw/hooks
cp examples/hooks/tmux-new-window.sh ~/.gw/hooks/
chmod +x ~/.gw/hooks/tmux-new-window.sh
```

Then configure `~/.gwrc`:

```
auto_cd = false
post_start_hook = ~/.gw/hooks/tmux-new-window.sh
post_checkout_hook = ~/.gw/hooks/tmux-new-window.sh
pre_end_hook = ~/.gw/hooks/docker-compose-down.sh
```

## Available Scripts

| Script | Hook | Description |
|---|---|---|
| `tmux-new-window.sh` | post_start / post_checkout | Open a new tmux window at the worktree directory |
| `iterm2-new-tab.sh` | post_start / post_checkout | Open a new iTerm2 tab at the worktree directory (macOS only) |
| `docker-compose-down.sh` | pre_end | Tear down docker compose containers/volumes for the worktree before it is removed |

## Environment Variables

Hook scripts receive these environment variables from `gw`:

| Variable | Description |
|---|---|
| `GW_WORKTREE_PATH` | Absolute path to the worktree |
| `GW_BRANCH_NAME` | Branch name of the worktree |
| `GW_REPO_NAME` | Repository name |
| `GW_COMMAND` | The command that triggered the hook (`start`, `checkout`, `end`, or `clean`) |

## Writing Your Own

Any executable script or command can be used as a hook. Just make sure it's executable and reference it in `~/.gwrc`.

`pre_end_hook` runs with the worktree as its working directory, so relative paths like `./docker-compose.yml` work naturally.

## Repository-specific hooks (project-local `.gwrc`)

The examples above go in `~/.gwrc` and apply to every repository. If only *one* repository needs a special hook — say, `crowi` needs `pnpm dev` split into its own tmux pane while every other repo just uses `auto_cd` — put a `.gwrc` at that repository's main worktree root (next to its `.git` directory) declaring just the hook keys you want to override:

```
# <crowi-repo>/.gwrc
post_start_hook = /Users/you/.gw/hooks/tmux-new-window.sh
```

**Prefer an absolute path** for the referenced script, as shown above, rather than `./hooks/start.sh`. `gw` changes into the worktree directory before running `post_start_hook`/`post_checkout_hook`, so a relative path resolves against whatever is checked out in that worktree — which can silently differ from what you reviewed when you approved the `.gwrc`. An absolute path always points at the same file regardless of which branch is checked out.

The first time a project `.gwrc` declares a non-empty hook value, `gw` will prompt for approval (default: **No**) before running it — see the "Project-Local Configuration" section of the main [README](../../README.md#project-local-configuration) for the full trust model, `--no-project-hooks`, and other details. Only the three hook keys are project-overridable in v1.1; every other `~/.gwrc` key (`auto_cd`, `copy_envs`, etc.) still comes from the global config only.
