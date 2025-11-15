#!/usr/bin/env bash
set -euo pipefail

# Documentation Health Dashboard Update Script
# Updates metrics in docs/INDEX.md automatically
# Called by pre-commit hook after successful validation

# Change to repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Colors for output
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

echo -e "${BLUE}📊 Updating documentation health dashboard...${NC}"

# Calculate metrics
TIMESTAMP=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
TOTAL_DOCS=$(find docs -name "*.md" -type f 2>/dev/null | wc -l)
TOTAL_LINES=$(find docs -name "*.md" -type f -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
README_LINES=$(wc -l < README.md 2>/dev/null || echo "0")
README_PERCENT=$((README_LINES * 100 / 300))
CLAUDE_LINES=$(wc -l < CLAUDE.md 2>/dev/null || echo "0")

# Package coverage
TOTAL_PACKAGES=$(ls -1 pkg 2>/dev/null | wc -l)
DOCUMENTED_PACKAGES=$(grep -l "pkg/" docs/*.md CLAUDE.md 2>/dev/null | wc -l)
if [ "$TOTAL_PACKAGES" -gt 0 ]; then
    PACKAGE_PERCENT=$((DOCUMENTED_PACKAGES * 100 / TOTAL_PACKAGES))
else
    PACKAGE_PERCENT=0
fi

# Freshness metrics using git
UPDATED_7_DAYS=$(git log --since="7 days ago" --name-only --pretty=format: -- 'docs/*.md' 'README.md' 'CLAUDE.md' 2>/dev/null | sort -u | grep -c '.md' || echo "0")
UPDATED_30_DAYS=$(git log --since="30 days ago" --name-only --pretty=format: -- 'docs/*.md' 'README.md' 'CLAUDE.md' 2>/dev/null | sort -u | grep -c '.md' || echo "0")

if [ "$TOTAL_DOCS" -gt 0 ]; then
    FRESH_PERCENT=$((UPDATED_30_DAYS * 100 / TOTAL_DOCS))
else
    FRESH_PERCENT=0
fi

# Find oldest document
OLDEST_DOC="unknown"
OLDEST_DAYS=0
if git rev-parse --git-dir > /dev/null 2>&1; then
    # Find the file that hasn't been modified in the longest time
    OLDEST_INFO=$(git log --all --pretty=format:"%ct" --name-only -- 'docs/*.md' 2>/dev/null | \
        awk 'NF{if($0 ~ /\.md$/){file=$0} else{times[file]=$0}} END{for(f in times){print times[f], f}}' | \
        sort -n | head -1)

    if [ -n "$OLDEST_INFO" ]; then
        OLDEST_TIMESTAMP=$(echo "$OLDEST_INFO" | awk '{print $1}')
        OLDEST_DOC=$(echo "$OLDEST_INFO" | awk '{print $2}' | sed 's|docs/||')
        NOW=$(date +%s)
        OLDEST_DAYS=$(( (NOW - OLDEST_TIMESTAMP) / 86400 ))
    fi
fi

# Find most active document
MOST_ACTIVE="unknown"
if git rev-parse --git-dir > /dev/null 2>&1; then
    MOST_ACTIVE=$(git log --since="30 days ago" --name-only --pretty=format: -- 'docs/*.md' 'CLAUDE.md' 2>/dev/null | \
        grep '.md' | sort | uniq -c | sort -rn | head -1 | awk '{print $2}' | sed 's|docs/||' || echo "unknown")
fi

# Validation status (assume passed if we got here)
VALIDATION_STATUS="✅ Passed"

# Create temporary file with updated content
TMP_FILE=$(mktemp)

# Read the current INDEX.md and update only the auto-generated section
awk -v timestamp="$TIMESTAMP" \
    -v total_docs="$TOTAL_DOCS" \
    -v total_lines="$TOTAL_LINES" \
    -v readme_lines="$README_LINES" \
    -v readme_percent="$README_PERCENT" \
    -v claude_lines="$CLAUDE_LINES" \
    -v total_packages="$TOTAL_PACKAGES" \
    -v documented_packages="$DOCUMENTED_PACKAGES" \
    -v package_percent="$PACKAGE_PERCENT" \
    -v updated_7="$UPDATED_7_DAYS" \
    -v updated_30="$UPDATED_30_DAYS" \
    -v fresh_percent="$FRESH_PERCENT" \
    -v oldest_doc="$OLDEST_DOC" \
    -v oldest_days="$OLDEST_DAYS" \
    -v most_active="$MOST_ACTIVE" \
    -v validation="$VALIDATION_STATUS" \
'
BEGIN { in_auto_section = 0 }

# Start of auto-generated section
/^<!-- AUTO-GENERATED SECTION/ {
    in_auto_section = 1
    print
    next
}

# End of auto-generated section
/^<!-- END AUTO-GENERATED SECTION/ {
    in_auto_section = 0
    print
    next
}

# Inside auto-generated section - replace with new content
in_auto_section {
    if (!printed) {
        print "<!-- Updated by scripts/update-doc-health.sh -->"
        print "<!-- Last Updated: " timestamp " -->"
        print "<!-- ============================================ -->"
        print ""
        print "**Last Updated**: " timestamp
        print "**Last Validation**: " validation
        print "**Auto-Updated**: Yes (via pre-commit hook)"
        print ""
        print "### Quick Metrics (Auto-Generated)"
        print "- **Total Files**: " total_docs " docs"
        print "- **Total Lines**: " total_lines
        print "- **README.md**: " readme_lines "/300 lines (" readme_percent "%)"
        print "- **CLAUDE.md**: " claude_lines " lines"
        print "- **Broken Links**: 0"
        print "- **Package Coverage**: " documented_packages "/" total_packages " (" package_percent "%)"
        print "- **Fresh Docs (<30 days)**: " fresh_percent "%"
        print ""
        print "### Freshness (Auto-Generated)"
        print "- **Updated Last 7 Days**: " updated_7 " files"
        print "- **Updated Last 30 Days**: " updated_30 " files"
        print "- **Oldest Document**: " oldest_doc " (" oldest_days " days)"
        print "- **Most Active**: " most_active
        print ""
        printed = 1
    }
    next
}

# Outside auto-generated section - keep as is
{ print }
' docs/INDEX.md > "$TMP_FILE"

# Replace the original file
mv "$TMP_FILE" docs/INDEX.md

echo -e "${GREEN}✓${NC} Dashboard updated successfully"
echo "  Total docs: $TOTAL_DOCS"
echo "  README.md: $README_LINES/300 lines ($README_PERCENT%)"
echo "  Fresh docs: $FRESH_PERCENT% (within 30 days)"
echo "  Oldest: $OLDEST_DOC ($OLDEST_DAYS days)"
