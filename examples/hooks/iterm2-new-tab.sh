#!/bin/bash
# Open a new iTerm2 tab and cd to the worktree directory (macOS only)
#
# Usage in ~/.gwrc:
#   post_start_hook = ~/.gw/hooks/iterm2-new-tab.sh
#   post_checkout_hook = ~/.gw/hooks/iterm2-new-tab.sh

osascript <<EOF
tell application "iTerm"
    tell current window
        create tab with default profile
        tell current session of current tab
            write text "cd '${GW_WORKTREE_PATH}' && clear"
            set name to "${GW_BRANCH_NAME}"
        end tell
    end tell
end tell
EOF
