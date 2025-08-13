# FEAT-037: Fix Import Yearly Summary Reporting

## Status
- **Completed**: YYYY-MM-DD
- **Priority**: high

## Overview
Fix the import yearly summary that incorrectly shows 0 new and 0 duplicates even when totals > 0.

## Background
During import, the yearly summary displays incorrect statistics showing 0 new and 0 duplicates for all years, even when the total counts are greater than 0.

Root cause: The finalizeSummary function (pkg/importer/importer.go:303) only sets YearStats[year].Final counts but never sets YearStats[year].Added or YearStats[year].Duplicates. Per-file statistics are collected but not properly aggregated by year.

## Requirements
### Functional Requirements
- [ ] Import yearly summary correctly reports new entry counts per year
- [ ] Import yearly summary correctly reports duplicate entry counts per year
- [ ] Yearly statistics match actual processing results
- [ ] Summary displays accurate totals that add up correctly

### Non-Functional Requirements
- [ ] Statistics calculation should not impact import performance
- [ ] Memory usage for tracking statistics should be minimal

## Design
### Approach
Track Added and Duplicates counts during import processing and aggregate them by year in finalizeSummary.

### Root Cause
The current finalizeSummary function only sets Final counts:
```go
// Current broken code in finalizeSummary
for year, entries := range yearlyEntries {
    i.summary.YearStats[year] = &YearStat{
        Final: len(entries), // Only sets Final
        // Added and Duplicates remain 0
    }
}
```

### Solution
Track statistics during entry processing and aggregate by year:
```go
// Track per-year statistics during processing
type YearTracker struct {
    added      map[int]int
    duplicates map[int]int
}

// Update finalizeSummary to set all fields
for year, entries := range yearlyEntries {
    i.summary.YearStats[year] = &YearStat{
        Final:      len(entries),
        Added:      yearTracker.added[year],
        Duplicates: yearTracker.duplicates[year],
    }
}
```

### Implementation Notes
- Track statistics as entries are processed, not just at the end
- Ensure Added + Duplicates + Initial = Final for each year
- Update both per-file and per-year tracking

## Tasks
- [ ] Add year-based statistics tracking during entry processing
- [ ] Update finalizeSummary to set Added and Duplicates counts
- [ ] Ensure statistics are tracked per-year during coalescing
- [ ] Verify yearly totals add up correctly (Added + Duplicates + Initial = Final)
- [ ] Write tests for correct yearly summary reporting
- [ ] Test with multi-year import data
- [ ] Verify display output shows correct counts

## Testing
### Unit Tests
- Test yearly statistics calculation with known data
- Test finalizeSummary sets all YearStat fields correctly
- Test year tracking during entry processing
- Test statistics aggregation math (Added + Duplicates + Initial = Final)

### Integration Tests
- Test import with multi-year data shows correct per-year counts
- Test import summary display shows non-zero Added/Duplicates
- Test import with duplicate entries across multiple years

### Edge Cases
- Import with all new entries (Duplicates = 0)
- Import with all duplicates (Added = 0)
- Import spanning many years
- Import with empty yearly data

## Risks and Mitigations
- **Risk**: Description
  - **Mitigation**: How to handle

## References
- Related features: FEAT-010 (Import subcommand implementation)
- Code locations: pkg/importer/importer.go:303 (finalizeSummary function)
- Code locations: pkg/importer/importer.go:192-196 (summary display code)

## Notes
Additional thoughts, questions, or considerations that arise during planning/implementation.
