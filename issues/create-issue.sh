#!/bin/bash

set -euo pipefail

# FEAT-075: Issue Creation Automation Script
# Automates the process of creating new issues with proper numbering and formatting

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ISSUES_DIR="${SCRIPT_DIR}"
BACKLOG_DIR="${ISSUES_DIR}/backlog"
FEATURE_TEMPLATE="${ISSUES_DIR}/feature_template.md"
BUG_TEMPLATE="${ISSUES_DIR}/bug_template.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Error handling
error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

warn() {
    echo -e "${YELLOW}Warning: $1${NC}" >&2
}

success() {
    echo -e "${GREEN}$1${NC}"
}

# Usage information
usage() {
    cat << EOF
Usage: $0 TYPE TITLE

Create a new issue with automatic numbering and formatting.

Arguments:
  TYPE    Issue type (FEATURE or BUG)
  TITLE   Human-readable issue title

Examples:
  $0 FEATURE "implement user authentication"
  $0 BUG "validation fails on empty input"

Output:
  Created: issues/backlog/FEAT-076-implement-user-authentication.md
  Created: issues/backlog/BUG-077-validation-fails-on-empty-input.md

Templates:
  FEATURE -> issues/feature_template.md
  BUG     -> issues/bug_template.md

EOF
}

# Validate input parameters
validate_inputs() {
    if [[ $# -ne 2 ]]; then
        error "Invalid number of arguments. Expected 2, got $#."
    fi

    local type="$1"
    local title="$2"

    # Validate type
    if [[ "$type" != "FEATURE" && "$type" != "BUG" ]]; then
        error "Invalid issue type '$type'. Must be 'FEATURE' or 'BUG'."
    fi

    # Validate title
    if [[ -z "$title" || "$title" =~ ^[[:space:]]*$ ]]; then
        error "Issue title cannot be empty or whitespace only."
    fi

    # Check title length (reasonable limit)
    if [[ ${#title} -gt 100 ]]; then
        warn "Title is very long (${#title} chars). Consider shortening for readability."
    fi
}

# Validate required files and directories exist
validate_environment() {
    if [[ ! -d "$BACKLOG_DIR" ]]; then
        error "Backlog directory not found: $BACKLOG_DIR"
    fi

    if [[ ! -f "$FEATURE_TEMPLATE" ]]; then
        error "Feature template not found: $FEATURE_TEMPLATE"
    fi

    if [[ ! -f "$BUG_TEMPLATE" ]]; then
        error "Bug template not found: $BUG_TEMPLATE"
    fi
}

# Find the next sequential issue number
get_next_issue_number() {
    local max_num=0
    local current_num

    # Search all issue directories for FEAT-XXX and BUG-XXX patterns
    while IFS= read -r file; do
        if [[ -n "$file" ]]; then
            current_num=$(echo "$file" | sed -n 's/^.*-\([0-9]\{3\}\)-.*/\1/p')
            if [[ -n "$current_num" ]] && [[ "$current_num" =~ ^[0-9]+$ ]]; then
                # Convert to base 10 to handle leading zeros
                current_num=$((10#$current_num))
                if (( current_num > max_num )); then
                    max_num=$current_num
                fi
            fi
        fi
    done < <(find "$ISSUES_DIR" -name "*.md" -type f -exec basename {} \; | grep -E '^(FEAT|BUG)-[0-9]{3}')

    echo "$max_num"
}

# Convert title to kebab-case format
to_kebab_case() {
    local title="$1"
    
    # Convert to lowercase
    title=$(echo "$title" | tr '[:upper:]' '[:lower:]')
    
    # Replace spaces with hyphens
    title=$(echo "$title" | sed 's/[[:space:]]\+/-/g')
    
    # Remove special characters except hyphens and alphanumeric
    title=$(echo "$title" | sed 's/[^a-z0-9-]//g')
    
    # Collapse multiple hyphens to single
    title=$(echo "$title" | sed 's/-\+/-/g')
    
    # Trim leading and trailing hyphens
    title=$(echo "$title" | sed 's/^-\+//;s/-\+$//')
    
    echo "$title"
}

# Create the new issue file
create_issue() {
    local type="$1"
    local title="$2"
    
    # Get next issue number
    local max_num
    max_num=$(get_next_issue_number)
    
    local next_num
    next_num=$((max_num + 1))
    
    # Format number with leading zeros
    local formatted_num
    formatted_num=$(printf "%03d" "$next_num")
    
    # Convert title to kebab-case
    local kebab_title
    kebab_title=$(to_kebab_case "$title")
    
    if [[ -z "$kebab_title" ]]; then
        error "Title conversion resulted in empty string. Please use a title with alphanumeric characters."
    fi
    
    # Determine prefix and template
    local prefix template_file
    if [[ "$type" == "FEATURE" ]]; then
        prefix="FEAT"
        template_file="$FEATURE_TEMPLATE"
    else
        prefix="BUG"
        template_file="$BUG_TEMPLATE"
    fi
    
    # Create filename
    local filename="${prefix}-${formatted_num}-${kebab_title}.md"
    local target_file="${BACKLOG_DIR}/${filename}"
    
    # Check if file already exists (unlikely but possible)
    if [[ -f "$target_file" ]]; then
        error "Target file already exists: $target_file"
    fi
    
    # Copy template and update title
    if ! cp "$template_file" "$target_file"; then
        error "Failed to copy template to target location"
    fi
    
    # Replace template placeholders
    local template_title_pattern template_new_title
    if [[ "$type" == "FEATURE" ]]; then
        template_title_pattern="# FEAT-XXX: Feature Name"
        template_new_title="# ${prefix}-${formatted_num}: ${title}"
    else
        template_title_pattern="# BUG-XXX: Bug Title"
        template_new_title="# ${prefix}-${formatted_num}: ${title}"
    fi
    
    # Use sed to replace the title line
    if ! sed -i "s|${template_title_pattern}|${template_new_title}|" "$target_file"; then
        error "Failed to update title in new issue file"
    fi
    
    # Verify the file was created successfully
    if [[ ! -f "$target_file" ]]; then
        error "Issue file was not created successfully"
    fi
    
    # Return the path for caller reference
    echo "$target_file"
}

# Main execution
main() {
    # Show usage if no arguments
    if [[ $# -eq 0 ]]; then
        usage
        exit 0
    fi
    
    # Handle help flags
    if [[ "$1" == "-h" || "$1" == "--help" ]]; then
        usage
        exit 0
    fi
    
    # Validate inputs
    validate_inputs "$@"
    
    # Validate environment
    validate_environment
    
    # Create the issue
    local created_file
    created_file=$(create_issue "$1" "$2")
    
    # Report success
    success "Created: ${created_file#$SCRIPT_DIR/}"
}

# Execute main function with all arguments
main "$@"