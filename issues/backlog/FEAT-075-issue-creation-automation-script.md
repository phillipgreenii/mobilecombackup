# FEAT-075: Issue Creation Automation Script

## Status
- **Priority**: high

## Overview
Create a shell script that automates the issue creation process for agents, reducing manual overhead while maintaining quality standards. The script will handle issue numbering, template selection, file placement, and title conversion automatically.

## Background
Currently, agents need to manually:
1. Find the next sequential issue number by examining all existing issues
2. Convert titles to kebab-case format
3. Copy appropriate templates
4. Place files in correct locations
5. Update titles within files

This manual process is error-prone, time-consuming, and inconsistent across agents. An automation script will standardize the workflow and reduce cognitive overhead.

## Requirements

### Functional Requirements
- [ ] Accept TYPE parameter (BUG or FEATURE) and TITLE parameter
- [ ] Automatically determine next sequential issue number across all directories
- [ ] Support both FEAT-XXX and BUG-XXX numbering sequences
- [ ] Convert titles to proper kebab-case format
- [ ] Copy appropriate template (feature_template.md or bug_template.md) 
- [ ] Place new files in issues/backlog/ directory
- [ ] Update title field in copied template automatically
- [ ] Preserve all other template content unchanged
- [ ] Return new file path for agent reference

### Non-Functional Requirements
- [ ] Robust error handling for invalid inputs
- [ ] Validation of template file existence
- [ ] Proper file permissions (executable)
- [ ] Clean, readable bash code with comments
- [ ] Fast execution (< 2 seconds)

## Design

### Approach
Create `issues/create-issue.sh` as a standalone bash script that:
1. Validates input parameters and templates
2. Scans all issue directories to find highest number
3. Increments number and formats filename
4. Copies template and updates title
5. Reports success with new file path

### API/Interface
```bash
# Script usage
./issues/create-issue.sh TYPE TITLE

# Examples
./issues/create-issue.sh FEATURE "implement user authentication"
./issues/create-issue.sh BUG "validation fails on empty input"

# Output
Created: issues/backlog/FEAT-076-implement-user-authentication.md
Created: issues/backlog/BUG-077-validation-fails-on-empty-input.md
```

### Data Structures
```bash
# Key variables
TYPE="FEATURE|BUG"           # Issue type parameter
TITLE="descriptive title"     # Human-readable title
KEBAB_TITLE="kebab-case"     # Converted title
NEXT_NUM=075                 # Next sequential number
TEMPLATE_FILE="path/to/template" # Source template
TARGET_FILE="path/to/new/issue"  # Destination file
```

### Implementation Notes

**Number Detection Algorithm:**
- Scan issues/**/*.md files for FEAT-XXX and BUG-XXX patterns
- Extract numeric portions and find maximum
- Handle both completed and active issues
- Default to 001 if no existing issues found

**Title Conversion:**
- Convert to lowercase
- Replace spaces with hyphens
- Remove special characters except hyphens
- Collapse multiple hyphens to single
- Trim leading/trailing hyphens

**Template Processing:**
- Validate template exists before processing
- Use sed to replace placeholder title
- Preserve all formatting and structure
- Maintain proper line endings

## Tasks
- [ ] Design and implement number detection algorithm
- [ ] Create kebab-case title conversion function
- [ ] Implement template copying and title replacement
- [ ] Add comprehensive error handling and validation
- [ ] Create executable script with proper permissions
- [ ] Write shell script tests and validation
- [ ] Update documentation and agent instructions
- [ ] Update CLAUDE.md workflow references
- [ ] Create usage examples and test cases

## Testing

### Unit Tests
- Number detection with various existing issue patterns
- Kebab-case conversion with edge cases (spaces, special chars, numbers)
- Template copying and title replacement accuracy
- Error handling for missing templates or invalid parameters

### Integration Tests
- End-to-end issue creation with both FEATURE and BUG types
- Validation that created files have correct format and content
- Testing with various title formats and lengths
- Verification of file placement and permissions

### Edge Cases
- Empty title handling
- Very long titles (truncation)
- Special characters in titles
- Missing template files
- No existing issues (starting from 001)
- Mixed numbering gaps in existing issues

## Risks and Mitigations
- **Risk**: Number collision if script runs concurrently
  - **Mitigation**: Atomic file creation and validation
- **Risk**: Invalid bash syntax causing failures
  - **Mitigation**: Thorough testing and shell linting
- **Risk**: Template format changes breaking title replacement
  - **Mitigation**: Flexible sed patterns and validation
- **Risk**: Permission issues preventing script execution
  - **Mitigation**: Proper chmod and documentation

## References
- Template files: issues/feature_template.md, issues/bug_template.md
- Issue directories: issues/backlog/, issues/active/, issues/completed/
- Documentation: docs/ISSUE_WORKFLOW.md, CLAUDE.md
- Related workflow: Issue creation commands in docs/SLASH_COMMANDS.md

## Notes

**Documentation Updates Required:**
1. CLAUDE.md - Reference script in issue creation workflow
2. docs/ISSUE_WORKFLOW.md - Update manual process to use script
3. Agent memory updates for issue creation instructions
4. Examples and usage documentation

**Agent Integration:**
- Script should be the preferred method for issue creation
- Agents should validate script output before proceeding
- Maintain ability to manually create issues as fallback
- Update agent prompts to reference script usage

**Future Enhancements:**
- Could extend to support additional issue types
- Integration with git commit automation
- Template validation and formatting
- Integration with project management tools