# FEAT-072: Pre-commit Hook Optimization - Skip Tests for Markdown-Only Changes

## Status
- **Priority**: medium

## Overview
Optimize the pre-commit hook to skip expensive test operations when changes only affect markdown files, improving developer productivity by reducing commit time for documentation updates.

## Background
Currently, the pre-commit hook runs the full test suite (formatter, tests, linter) for every commit, even when changes only affect documentation files (*.md, *.markdown). This is inefficient because:

1. Documentation changes don't affect code functionality and don't require test validation
2. Test execution takes ~20 seconds on average, making documentation commits unnecessarily slow
3. Developers may be tempted to use `--no-verify` for doc-only changes, bypassing all quality checks
4. The current hook performance target of <30s is often exceeded for simple doc updates

## Requirements
### Functional Requirements
- [ ] Detect when staged changes only affect markdown files (.md, .markdown extensions)
- [ ] Skip test execution for markdown-only changes while preserving other quality checks
- [ ] Continue running all checks (formatter, tests, linter) for commits involving code changes
- [ ] Handle mixed commits (code + markdown) by running all checks
- [ ] Maintain existing bypass mechanism (`git commit --no-verify`)
- [ ] Provide clear feedback about which checks are being skipped and why

### Non-Functional Requirements
- [ ] Markdown-only commits should complete in <10 seconds (vs current ~30s)
- [ ] Detection logic should be reliable and handle edge cases
- [ ] Must not introduce security vulnerabilities or bypass important checks
- [ ] Should maintain compatibility with existing devbox workflow

## Design
### Approach
Enhance the existing pre-commit hook (.githooks/pre-commit) to analyze staged files and conditionally execute checks based on file types.

### Detection Logic
```bash
# Check if only markdown files are staged
is_markdown_only_commit() {
    # Get list of staged files (added, modified, deleted)
    staged_files=$(git diff --cached --name-only --diff-filter=AMDRC)
    
    # Return false if no staged files
    if [ -z "$staged_files" ]; then
        return 1
    fi
    
    # Check each staged file
    for file in $staged_files; do
        case "$file" in
            *.md|*.markdown) continue ;;
            *) return 1 ;;  # Non-markdown file found
        esac
    done
    
    return 0  # All staged files are markdown
}
```

### Hook Flow Enhancement
```bash
#!/bin/sh
# Enhanced pre-commit hook with markdown optimization

echo "🔍 Analyzing staged changes..."

# Check if this is a markdown-only commit
if is_markdown_only_commit; then
    echo "📝 Detected markdown-only changes - optimizing checks..."
    run_markdown_optimized_checks
else
    echo "🔧 Detected code changes - running full quality checks..."
    run_full_quality_checks
fi
```

### Check Strategies
1. **Markdown-only commits**: Run formatter and linter only (skip tests)
2. **Mixed/code commits**: Run all checks (formatter, tests, linter)
3. **Empty commits**: Run all checks (safety default)

### Implementation Notes
- Use `git diff --cached --name-only` to analyze staged files only
- Handle edge cases: renamed files, deleted files, new files
- Preserve existing error handling and performance tracking
- Maintain clear progress indicators and timing information
- Consider case-insensitive file extension matching

## Tasks
### Phase 1: Core Detection Logic
- [ ] Implement `is_markdown_only_commit()` function
- [ ] Add comprehensive test cases for file detection logic
- [ ] Handle edge cases (renames, deletions, case sensitivity)
- [ ] Test detection accuracy with various commit scenarios

### Phase 2: Hook Integration
- [ ] Integrate detection logic into existing pre-commit hook
- [ ] Implement conditional check execution paths
- [ ] Preserve existing error handling and bypass mechanisms
- [ ] Update performance tracking and reporting

### Phase 3: User Experience
- [ ] Add clear messaging about optimization decisions
- [ ] Update performance targets and warnings
- [ ] Enhance progress indicators for different check modes
- [ ] Ensure compatibility with existing devbox scripts

### Phase 4: Testing and Documentation
- [ ] Create comprehensive test suite for hook behavior
- [ ] Test with real-world commit scenarios
- [ ] Update installation script documentation
- [ ] Add optimization details to CLAUDE.md

## Testing
### Unit Tests
- File detection accuracy with various extensions (.md, .MD, .markdown, .MARKDOWN)
- Mixed commit detection (markdown + code files)
- Edge cases: empty commits, renames, deletions, new files
- Case sensitivity handling across different file systems

### Integration Tests
- End-to-end testing with actual git commits
- Performance measurement: markdown-only vs full checks
- Devbox integration testing
- Hook behavior with different git configurations

### Edge Cases
- Files with multiple extensions (README.md.backup)
- Symbolic links to markdown files
- Binary files with .md names (unlikely but possible)
- Very large markdown files
- Commits with only deleted files
- Submodule changes mixed with markdown

## Performance Targets
- **Markdown-only commits**: < 10 seconds total (60% improvement)
- **Mixed/code commits**: < 30 seconds total (unchanged)
- **Detection overhead**: < 1 second

## Risks and Mitigations
- **Risk**: Incorrect file type detection bypassing necessary checks
  - **Mitigation**: Comprehensive test suite, conservative defaults, thorough edge case handling
- **Risk**: Security implications of skipping checks
  - **Mitigation**: Only skip tests, maintain formatter and linter for all commits
- **Risk**: Developers becoming over-reliant on optimization
  - **Mitigation**: Clear documentation about when full checks still run
- **Risk**: Complexity making hook maintenance difficult
  - **Mitigation**: Well-documented code, modular functions, comprehensive tests

## References
- Related features: FEAT-063 (git hooks configuration)
- Code locations: .githooks/pre-commit, scripts/install-hooks.sh
- Performance baseline: Current hook averages 20-30 seconds for all commits

## Notes
This optimization specifically targets the common developer workflow of making documentation updates, which currently triggers unnecessary test execution. The enhancement maintains all safety guarantees while significantly improving the developer experience for documentation-focused commits.

Key considerations:
- Must err on the side of caution - when in doubt, run all checks
- Detection logic should be simple and reliable
- Performance gains should be substantial to justify the complexity
- Should integrate seamlessly with existing developer workflows