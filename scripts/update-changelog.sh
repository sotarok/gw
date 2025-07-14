#!/bin/bash
# Script to update CHANGELOG.md before release

VERSION=$1
DATE=$(date +%Y-%m-%d)

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.1.0"
    exit 1
fi

# Remove 'v' prefix if present
VERSION_NUM=${VERSION#v}

# Update CHANGELOG.md
sed -i.bak "s/## \[Unreleased\]/## [Unreleased]\n\n## [$VERSION_NUM] - $DATE/" CHANGELOG.md

echo "Updated CHANGELOG.md for version $VERSION_NUM"
echo "Please review the changes and commit them before creating the tag"