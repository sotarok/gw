# Shell Integration

The `gw` command runs as a subprocess and cannot change the parent shell's directory. To enable automatic directory changing after creating a worktree, you need to set up shell integration.

## Quick Setup

The easiest way to set up shell integration is to use the `eval` method, which automatically loads the latest shell integration code every time you start your shell:

### Bash

Add this line to your `~/.bashrc`:

```bash
eval "$(gw shell-integration --show-script --shell=bash)"
```

### Zsh

Add this line to your `~/.zshrc`:

```bash
eval "$(gw shell-integration --show-script --shell=zsh)"
```

### Fish

Add this line to your `~/.config/fish/config.fish`:

```fish
gw shell-integration --show-script --shell=fish | source
```

## Benefits

Using the `eval` method has several advantages:

- **Always up-to-date**: The shell function is dynamically generated, so you always get the latest version when `gw` is updated
- **No manual updates**: You don't need to edit your shell configuration when the integration code changes
- **Clean configuration**: Your shell configuration file stays minimal and readable

## How it Works

The shell integration creates a `gw` function that:

1. Checks if you're running `gw start` or `gw checkout`
2. Verifies if `auto_cd = true` in your `~/.gwrc` file
3. Runs the actual `gw` command
4. If successful, automatically changes to the new worktree directory

## Manual Installation

If you prefer not to use `eval`, you can see the shell function code by running:

```bash
gw shell-integration --show-script --shell=bash
```

Then copy and paste the output into your shell configuration file.

## Alternative: Output Directory for Manual cd

If you prefer not to use shell functions, you can use the shell-integration command:

```bash
# Create worktree
gw start 123

# Then cd to it
cd $(gw shell-integration --print-path=123)
```

The `shell-integration --print-path` command outputs only the worktree path for the specified issue or branch.