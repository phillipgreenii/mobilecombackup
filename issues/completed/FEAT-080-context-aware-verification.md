# FEAT-080: Context-Aware Verification

## Overview
Implement intelligent verification that adapts test execution strategy based on the type and scope of changes, reducing verification time while maintaining quality assurance before commits.

## Problem Statement
Current verification workflow requires running ALL tests for EVERY change, leading to:
- Slow feedback loops during development (especially on older hardware)
- Unnecessary test execution for documentation-only changes
- Full verification for single-line fixes
- Developer frustration with wait times
- Reduced productivity due to verification overhead

## Requirements

### Functional Requirements
1. **Change Detection and Classification**
   - Analyze git diff to categorize changes
   - Detect file types and change scope
   - Identify affected packages and dependencies
   - Determine appropriate verification level

2. **Change Categories**
   - **Documentation-only** (*.md, *.yaml configs)
     - Skip: tests, build
     - Run: markdown formatter, markdown linter
   
   - **Test-only** (*_test.go files)
     - Run: affected test files first
     - Run: full test suite before commit
     - Skip: build (unless test helpers changed)
   
   - **Single-package** (changes within one package)
     - Run: package tests first
     - Run: dependent package tests
     - Run: full suite before commit
   
   - **Multi-package** (cross-package changes)
     - Always run full verification
     - No shortcuts for complex changes
   
   - **Mixed changes** (code + docs + tests)
     - Default to full verification
     - Provide breakdown of what will run

3. **Progressive Test Execution**
   - Track number of changes since last full test
   - Escalate test scope based on change accumulation:
     - 1st change: affected tests only
     - 2-3 changes: related package tests
     - 4+ changes: full test suite
     - Pre-commit: always full verification

4. **Override Controls**
   - `--full` flag to force complete verification
   - `--quick` flag for minimal verification
   - `--skip-tests` for documentation work (with warnings)
   - Environment variable for default behavior

### Non-Functional Requirements
1. **Accuracy**: Never miss critical tests
2. **Speed**: Reduce verification time by 50-70% during development
3. **Safety**: Full verification always before commit
4. **Transparency**: Clear reporting of what's being run and why

## Design Approach

### Implementation Strategy
```bash
# New command structure
devbox run smart-verify [--full|--quick|--pre-commit]

# Integration with existing workflow
devbox run formatter
devbox run smart-verify  # Context-aware
devbox run linter
devbox run build-cli
```

### Change Analysis Algorithm
```go
type ChangeContext struct {
    Category        ChangeCategory
    AffectedPackages []string
    FileTypes       map[string]int
    ChangeCount     int
    LastFullTest    time.Time
}

func AnalyzeChanges() ChangeContext {
    // 1. Get git diff
    // 2. Classify files by type
    // 3. Identify affected packages
    // 4. Determine verification strategy
    // 5. Return context with recommendations
}
```

### Verification Strategies
```yaml
strategies:
  docs-only:
    formatter: true
    markdown-lint: true
    tests: false
    build: false
    time-saved: "90%"
  
  test-only:
    formatter: true
    tests: affected-only
    build: conditional
    time-saved: "50%"
  
  single-package:
    formatter: true
    tests: package-scoped
    build: true
    time-saved: "60%"
  
  multi-package:
    formatter: true
    tests: full
    build: true
    time-saved: "0%"
```

## Tasks
- [ ] Create change detection and classification system
- [ ] Implement smart-verify command
- [ ] Add progressive test execution logic
- [ ] Create verification strategy engine
- [ ] Add dependency graph for test selection
- [ ] Implement override flags and controls
- [ ] Add metrics tracking for time saved
- [ ] Create clear reporting of verification decisions
- [ ] Integrate with existing verification workflow
- [ ] Add safety checks for pre-commit
- [ ] Create configuration file for customization
- [ ] Add tests for verification strategies
- [ ] Document context-aware verification

## Testing Requirements
1. **Unit Tests**
   - Test change classification accuracy
   - Verify strategy selection logic
   - Test progressive escalation

2. **Integration Tests**
   - Test with various change scenarios
   - Verify correct test selection
   - Ensure pre-commit safety

3. **Performance Tests**
   - Measure actual time savings
   - Compare with full verification
   - Track accuracy of test selection

## Acceptance Criteria
- [ ] Correctly classifies changes into categories
- [ ] Reduces verification time by >50% for typical development
- [ ] Never skips critical tests that would catch bugs
- [ ] Always runs full verification before commits
- [ ] Clear reporting of verification decisions
- [ ] Override controls work as expected
- [ ] Documentation includes strategy flowchart
- [ ] Metrics show significant time savings

## Dependencies
- None directly, but complements FEAT-081 (Test Performance Optimization)

## Priority
MEDIUM-HIGH - Significant developer experience improvement

## Estimated Effort
- Implementation: 8-10 hours
- Testing: 4-5 hours
- Integration: 2-3 hours
- Documentation: 2 hours
- Total: 16-20 hours

## Notes
- Start conservative - prefer over-testing to under-testing
- Consider caching test results between runs
- May need adjustment period to tune strategies
- Could integrate with CI/CD for remote verification
- Future: ML-based test prediction