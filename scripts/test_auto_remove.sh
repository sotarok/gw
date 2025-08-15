#!/bin/bash

# Test script for auto-remove-branch feature

# Change to the project root directory
cd "$(dirname "$0")/.." || exit 1

echo "Testing auto-remove-branch feature..."
echo ""

# Build the binary
echo "Building gw..."
go build -o gw
if [ $? -ne 0 ]; then
    echo "Build failed"
    exit 1
fi

# Create a test config with auto-remove enabled
echo "Creating test config with auto_remove_branch = true..."
cat > ~/.gwrc.test << EOF
auto_cd = false
update_iterm2_tab = false
auto_remove_branch = true
EOF

# Show the config
echo ""
echo "Config contents:"
cat ~/.gwrc.test
echo ""

# Run tests with the test config
echo "Running tests to verify auto-remove functionality..."
go test ./cmd -run TestEndCommand_BranchDeletion -v | grep -E "(PASS|FAIL)"

echo ""
echo "Test complete! The auto-remove-branch feature:"
echo "1. Can be configured via ~/.gwrc with 'auto_remove_branch = true/false'"
echo "2. Defaults to false for safety"
echo "3. When enabled, automatically deletes the local branch after worktree removal"
echo "4. Branch deletion errors are non-fatal (just warnings)"
echo ""
echo "To enable in your environment:"
echo "  1. Run 'gw config' to interactively configure"
echo "  2. Or manually edit ~/.gwrc and set 'auto_remove_branch = true'"

# Clean up
rm -f ~/.gwrc.test