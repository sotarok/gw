# Hook Examples

Example hook scripts for use with `gw`'s `post_start_hook` and `post_checkout_hook` configuration.

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
```

## Available Scripts

| Script | Description |
|---|---|
| `tmux-new-window.sh` | Open a new tmux window at the worktree directory |
| `iterm2-new-tab.sh` | Open a new iTerm2 tab at the worktree directory (macOS only) |

## Environment Variables

Hook scripts receive these environment variables from `gw`:

| Variable | Description |
|---|---|
| `GW_WORKTREE_PATH` | Absolute path to the created worktree |
| `GW_BRANCH_NAME` | Branch name of the worktree |
| `GW_REPO_NAME` | Repository name |
| `GW_COMMAND` | The command that triggered the hook (`start` or `checkout`) |

## Writing Your Own

Any executable script or command can be used as a hook. Just make sure it's executable and reference it in `~/.gwrc`.
