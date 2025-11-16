#!/usr/bin/env bash
set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Track overall status
EXIT_CODE=0

# Helper functions
print_header() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
    EXIT_CODE=1
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Change to repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo -e "${BLUE}Documentation Validation${NC}"
echo ""

# Check 1: README.md line count
print_header "Checking README.md line count"
README_LINES=$(wc -l < README.md)
if [ "$README_LINES" -lt 300 ]; then
    print_success "README.md has $README_LINES lines (< 300 line limit)"
else
    print_error "README.md has $README_LINES lines (exceeds 300 line limit)"
fi
echo ""

# Check 2: Validate internal documentation links
print_header "Validating documentation links"

# Files to check for links
DOC_FILES=(
    "README.md"
    "CLAUDE.md"
    "docs/INDEX.md"
)

validate_links_in_file() {
    local file=$1
    local broken_links=0

    if [ ! -f "$file" ]; then
        print_warning "File not found: $file"
        return
    fi

    # Get directory of the file for resolving relative paths
    local file_dir=$(dirname "$file")

    # Extract markdown links: [text](path) and also plain file paths in backticks
    # Look for patterns like docs/FILE.md, issues/FILE.md, scripts/FILE.sh
    while IFS= read -r link; do
        # Skip external links (http/https)
        if [[ "$link" =~ ^https?:// ]]; then
            continue
        fi

        # Skip anchors
        if [[ "$link" =~ ^# ]]; then
            continue
        fi

        # Remove anchor if present
        link_without_anchor="${link%#*}"

        # Resolve relative path from the file's directory
        local resolved_path
        if [[ "$link_without_anchor" = /* ]]; then
            # Absolute path from repo root
            resolved_path="$link_without_anchor"
        else
            # Relative path - resolve from file's directory
            resolved_path="$file_dir/$link_without_anchor"
        fi

        # Normalize the path (remove ./ and ../)
        resolved_path=$(realpath -m "$resolved_path" 2>/dev/null || echo "$resolved_path")

        # Check if file exists
        if [ ! -f "$resolved_path" ] && [ ! -d "$resolved_path" ]; then
            print_error "Broken link in $file: $link_without_anchor (resolved to: $resolved_path)"
            broken_links=$((broken_links + 1))
        fi
    done < <(grep -oP '\[.*?\]\(\K[^)]+' "$file" 2>/dev/null || true)

    if [ "$broken_links" -eq 0 ]; then
        print_success "All links in $file are valid"
    fi
}

for doc_file in "${DOC_FILES[@]}"; do
    validate_links_in_file "$doc_file"
done
echo ""

# Check 3: Validate mentioned scripts exist and are executable
print_header "Validating scripts"

SCRIPTS=(
    "scripts/install-hooks.sh"
    "scripts/build-version.sh"
    "scripts/validate-version.sh"
    "issues/create-issue.sh"
)

for script in "${SCRIPTS[@]}"; do
    if [ ! -f "$script" ]; then
        print_error "Script not found: $script"
    elif [ ! -x "$script" ]; then
        print_error "Script not executable: $script"
    else
        print_success "Script exists and is executable: $script"
    fi
done
echo ""

# Check 4: Validate packages mentioned in CLAUDE.md exist
print_header "Validating packages"

# Extract package names from CLAUDE.md
# Look for lines mentioning pkg/ directories
PACKAGES=(
    "pkg/calls"
    "pkg/sms"
    "pkg/contacts"
    "pkg/attachments"
    "pkg/manifest"
    "pkg/importer"
    "pkg/coalescer"
)

for pkg in "${PACKAGES[@]}"; do
    if [ ! -d "$pkg" ]; then
        print_error "Package directory not found: $pkg"
    else
        print_success "Package directory exists: $pkg"
    fi
done
echo ""

# Check 5: Validate symlink
print_header "Validating .cursorrules symlink"
if [ -L ".cursorrules" ]; then
    target=$(readlink .cursorrules)
    if [ "$target" = "CLAUDE.md" ]; then
        print_success ".cursorrules symlink points to CLAUDE.md"
    else
        print_error ".cursorrules symlink points to $target (expected CLAUDE.md)"
    fi
elif [ -f ".cursorrules" ]; then
    print_error ".cursorrules exists but is not a symlink"
else
    print_error ".cursorrules symlink does not exist"
fi
echo ""

# Summary
if [ "$EXIT_CODE" -eq 0 ]; then
    echo -e "${GREEN}All documentation validation checks passed!${NC}"
else
    echo -e "${RED}Documentation validation failed. Please fix the errors above.${NC}"
fi

exit "$EXIT_CODE"
