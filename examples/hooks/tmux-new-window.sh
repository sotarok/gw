#!/bin/bash
# Open a new tmux window and cd to the worktree directory
#
# Usage in ~/.gwrc:
#   post_start_hook = ~/.gw/hooks/tmux-new-window.sh
#   post_checkout_hook = ~/.gw/hooks/tmux-new-window.sh

tmux new-window -c "$GW_WORKTREE_PATH" -n "$GW_BRANCH_NAME"
