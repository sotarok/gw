# Shell Integration for gw

The `gw` command runs as a subprocess and cannot change the parent shell's directory. To enable automatic directory changing after creating a worktree, you need to set up a shell function.

## Bash / Zsh

Add this to your `~/.bashrc` or `~/.zshrc`:

```bash
gw() {
    # Run the actual gw command
    command gw "$@"
    
    # Check if we should cd to the new directory
    if [[ "$1" == "start" || "$1" == "checkout" ]] && [[ -f ~/.gwrc ]]; then
        # Check if auto_cd is enabled
        if grep -q "auto_cd = true" ~/.gwrc 2>/dev/null; then
            # Extract the worktree path from the output
            local output=$(command gw "$@" 2>&1)
            local worktree_path=$(echo "$output" | grep "✓ Created worktree at" | sed 's/.*✓ Created worktree at //')
            
            # If we found a path, cd to it
            if [[ -n "$worktree_path" && -d "$worktree_path" ]]; then
                cd "$worktree_path"
                echo "Changed directory to: $worktree_path"
            fi
        fi
    fi
}
```

## Fish

Add this to your `~/.config/fish/functions/gw.fish`:

```fish
function gw
    # Run the actual gw command
    command gw $argv
    
    # Check if we should cd to the new directory
    if test "$argv[1]" = "start" -o "$argv[1]" = "checkout"
        if test -f ~/.gwrc
            # Check if auto_cd is enabled
            if grep -q "auto_cd = true" ~/.gwrc 2>/dev/null
                # Extract the worktree path from the output
                set output (command gw $argv 2>&1)
                set worktree_path (echo "$output" | grep "✓ Created worktree at" | sed 's/.*✓ Created worktree at //')
                
                # If we found a path, cd to it
                if test -n "$worktree_path" -a -d "$worktree_path"
                    cd "$worktree_path"
                    echo "Changed directory to: $worktree_path"
                end
            end
        end
    end
end
```

## Alternative: Output Directory for Manual cd

If you prefer not to use shell functions, you can use command substitution:

```bash
# Create worktree and cd to it
cd $(gw start 123 --print-path)
```

For this to work, we would need to add a `--print-path` flag that outputs only the worktree path.