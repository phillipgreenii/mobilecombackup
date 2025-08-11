#!/bin/bash
set -e

# Extract base version from VERSION file (remove -dev suffix)
BASE_VERSION=$(cat VERSION 2>/dev/null | sed 's/-dev$//' || echo "dev")

# Check if we're on an exact git tag (release builds)
GIT_TAG=$(git describe --tags --exact-match 2>/dev/null || echo "")

if [ -n "$GIT_TAG" ]; then
    # Release build: use git tag version (remove v prefix)
    VERSION=${GIT_TAG#v}
else
    # Development build: append git hash
    GIT_HASH=$(git rev-parse --short=7 HEAD 2>/dev/null || echo "unknown")
    if [ "$GIT_HASH" != "unknown" ]; then
        VERSION="${BASE_VERSION}-dev-g${GIT_HASH}"
    else
        VERSION="${BASE_VERSION}-dev"
    fi
fi

echo "$VERSION"