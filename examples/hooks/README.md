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
