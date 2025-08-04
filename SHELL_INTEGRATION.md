# Shell Integration for gw

The `gw` command runs as a subprocess and cannot change the parent shell's directory. To enable automatic directory changing after creating a worktree, you need to set up a shell function.

## Bash / Zsh

Add this to your `~/.bashrc` or `~/.zshrc`:

```bash
gw() {
    # Check if we should auto-cd after command
    if [[ "$1" == "start" || "$1" == "checkout" ]] && [[ -f ~/.gwrc ]]; then
        # Check if auto_cd is enabled
        if grep -q "auto_cd = true" ~/.gwrc 2>/dev/null; then
            # Run the actual command (output goes directly to terminal)
            command gw "$@"
            local exit_code=$?
            
            # If command succeeded, get the worktree path and cd to it
            if [[ $exit_code -eq 0 ]]; then
                local identifier="${2:-}"  # Get issue number or branch name
                if [[ -n "$identifier" ]]; then
                    # Get the worktree path using shell-integration command
                    local worktree_path=$(command gw shell-integration --print-path="$identifier" 2>/dev/null)
                    
                    # If we got a path, cd to it
                    if [[ -n "$worktree_path" && -d "$worktree_path" ]]; then
                        cd "$worktree_path"
                        echo "Changed directory to: $worktree_path"
                    fi
                fi
            fi
            
            return $exit_code
        else
            # Auto CD disabled, just run the command normally
            command gw "$@"
        fi
    else
        # Not a start/checkout command, just run normally
        command gw "$@"
    fi
}
```

## Fish

Add this to your `~/.config/fish/functions/gw.fish`:

```fish
function gw
    # Check if we should auto-cd after command
    if test "$argv[1]" = "start" -o "$argv[1]" = "checkout"
        if test -f ~/.gwrc
            # Check if auto_cd is enabled
            if grep -q "auto_cd = true" ~/.gwrc 2>/dev/null
                # Run the actual command (output goes directly to terminal)
                command gw $argv
                set exit_code $status
                
                # If command succeeded, get the worktree path and cd to it
                if test $exit_code -eq 0
                    set identifier "$argv[2]"  # Get issue number or branch name
                    if test -n "$identifier"
                        # Get the worktree path using shell-integration command
                        set worktree_path (command gw shell-integration --print-path="$identifier" 2>/dev/null)
                        
                        # If we got a path, cd to it
                        if test -n "$worktree_path" -a -d "$worktree_path"
                            cd "$worktree_path"
                            echo "Changed directory to: $worktree_path"
                        end
                    end
                end
                
                return $exit_code
            else
                # Auto CD disabled, just run the command normally
                command gw $argv
            end
        else
            # No config file, just run normally
            command gw $argv
        end
    else
        # Not a start/checkout command, just run normally
        command gw $argv
    end
end
```

## Alternative: Output Directory for Manual cd

If you prefer not to use shell functions, you can use the shell-integration command:

```bash
# Create worktree
gw start 123

# Then cd to it
cd $(gw shell-integration --print-path=123)
```

The `shell-integration --print-path` command outputs only the worktree path for the specified issue or branch.