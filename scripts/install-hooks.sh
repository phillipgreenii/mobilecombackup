#!/bin/bash
# Git hooks installation script
# This script configures git to use custom hooks from .githooks directory

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in a git repository
if ! git rev-parse --git-dir >/dev/null 2>&1; then
    print_error "Not in a git repository!"
    exit 1
fi

# Get the repository root
REPO_ROOT=$(git rev-parse --show-toplevel)
GITHOOKS_DIR="$REPO_ROOT/.githooks"

print_status "Installing git hooks for mobilecombackup project..."

# Check if .githooks directory exists
if [ ! -d "$GITHOOKS_DIR" ]; then
    print_error ".githooks directory not found at $GITHOOKS_DIR"
    exit 1
fi

# Check if pre-commit hook exists
if [ ! -f "$GITHOOKS_DIR/pre-commit" ]; then
    print_error "pre-commit hook not found at $GITHOOKS_DIR/pre-commit"
    exit 1
fi

# Check if pre-commit hook is executable
if [ ! -x "$GITHOOKS_DIR/pre-commit" ]; then
    print_warning "Making pre-commit hook executable..."
    chmod +x "$GITHOOKS_DIR/pre-commit"
fi

# Configure git to use custom hooks directory
print_status "Configuring git to use .githooks directory..."
git config core.hooksPath .githooks

# Verify configuration
HOOKS_PATH=$(git config core.hooksPath)
if [ "$HOOKS_PATH" = ".githooks" ]; then
    print_success "Git hooks configured successfully!"
    print_success "Hooks directory: $GITHOOKS_DIR"
else
    print_error "Failed to configure git hooks path"
    exit 1
fi

# Test hook availability
print_status "Testing hook installation..."
if [ -x "$GITHOOKS_DIR/pre-commit" ]; then
    print_success "pre-commit hook is ready"
else
    print_error "pre-commit hook is not executable"
    exit 1
fi

# Check for devbox
if command -v devbox >/dev/null 2>&1; then
    print_success "devbox is available"
else
    print_warning "devbox not found in PATH"
    print_warning "Make sure devbox is installed for hooks to work properly"
fi

echo ""
print_success "Git hooks installation complete!"
echo ""
echo "📋 What's configured:"
echo "   • Pre-commit hook: runs formatter, tests, and linter"
echo "   • Performance target: <30 seconds total"
echo "   • Bypass: use 'git commit --no-verify' for emergencies"
echo ""
echo "🔧 To disable hooks later:"
echo "   git config --unset core.hooksPath"
echo ""
echo "🧪 To test hooks without committing:"
echo "   .githooks/pre-commit"