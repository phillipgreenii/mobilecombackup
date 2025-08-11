#!/bin/bash
set -e

# Version validation script to verify VERSION file format
# Ensures VERSION file contains valid semantic version with optional -dev suffix

VERSION_FILE="VERSION"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_error() {
    echo -e "${RED}ERROR: $1${NC}" >&2
}

print_success() {
    echo -e "${GREEN}SUCCESS: $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}WARNING: $1${NC}"
}

# Function to validate semantic version format
validate_semver() {
    local version="$1"
    local base_version="$version"
    
    # Remove -dev suffix if present for validation
    if [[ "$version" == *-dev ]]; then
        base_version="${version%-dev}"
    fi
    
    # Check if it matches semantic version pattern (major.minor.patch)
    if [[ ! "$base_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        return 1
    fi
    
    return 0
}

# Check if VERSION file exists
if [[ ! -f "$VERSION_FILE" ]]; then
    print_error "VERSION file not found in current directory"
    exit 1
fi

# Read VERSION file content
VERSION_CONTENT=$(cat "$VERSION_FILE" | tr -d '[:space:]')

# Check if VERSION file is empty
if [[ -z "$VERSION_CONTENT" ]]; then
    print_error "VERSION file is empty"
    exit 1
fi

# Validate version format
if ! validate_semver "$VERSION_CONTENT"; then
    print_error "Invalid version format in VERSION file: '$VERSION_CONTENT'"
    echo "Expected format: major.minor.patch or major.minor.patch-dev"
    echo "Examples: 2.0.0-dev, 1.2.3, 0.1.0-dev"
    exit 1
fi

# Check if version ends with -dev (expected for development)
if [[ "$VERSION_CONTENT" == *-dev ]]; then
    BASE_VERSION="${VERSION_CONTENT%-dev}"
    print_success "Valid development version: $VERSION_CONTENT (base: $BASE_VERSION)"
else
    print_warning "Release version format detected: $VERSION_CONTENT"
    print_warning "Development versions should typically end with '-dev'"
fi

# Additional checks
if [[ "$VERSION_CONTENT" == "0.0.0"* ]]; then
    print_warning "Version 0.0.0 detected - consider using proper semantic version"
fi

# Check for common issues
if [[ "$VERSION_CONTENT" == *" "* ]] || [[ "$VERSION_CONTENT" == *$'\t'* ]]; then
    print_error "VERSION file contains whitespace characters"
    exit 1
fi

if [[ $(wc -l < "$VERSION_FILE") -gt 1 ]]; then
    print_warning "VERSION file contains multiple lines - only first line will be used"
fi

print_success "VERSION file validation passed"
echo "Version: $VERSION_CONTENT"

# Test version extraction script if it exists
if [[ -f "scripts/build-version.sh" ]]; then
    echo ""
    echo "Testing version extraction:"
    EXTRACTED_VERSION=$(bash scripts/build-version.sh 2>/dev/null || echo "FAILED")
    if [[ "$EXTRACTED_VERSION" == "FAILED" ]]; then
        print_error "Version extraction script failed"
        exit 1
    else
        print_success "Extracted version: $EXTRACTED_VERSION"
    fi
fi

exit 0