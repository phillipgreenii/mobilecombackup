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
- [ ] Track per-year statistics at entry level during import processing
- [ ] Count Initial entries per year when loading existing repository
- [ ] Yearly statistics match actual processing results (Added + Duplicates + Initial = Final)
- [ ] Summary displays accurate totals that add up correctly

### Non-Functional Requirements
- [ ] Per-entry statistics tracking should not significantly impact import performance
- [ ] Memory usage for year-based tracking should be minimal (map of ints per year)
- [ ] Repository loading for Initial counts should be efficient

## Design
### Approach
Track Added, Duplicates, and Initial counts at the entry level during import processing, maintaining per-year statistics in the importer.

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

### Entry-Level Tracking Solution

#### 1. Add Year Tracker to Importer
```go
type Importer struct {
    // ... existing fields
    yearTracker *YearTracker
}

type YearTracker struct {
    initial    map[int]int  // Entries loaded from existing repository
    added      map[int]int  // New entries added during import
    duplicates map[int]int  // Duplicate entries found during import
}
```

#### 2. Track Initial Counts During Repository Loading
```go
// During repository loading, count existing entries per year
func (yt *YearTracker) trackInitialEntry(entry BackupEntry) {
    year := entry.Year()
    yt.initial[year]++
}
```

#### 3. Track Added/Duplicates During Import Processing
```go
// During import, track each entry based on coalescer result
func (yt *YearTracker) trackImportEntry(entry BackupEntry, wasAdded bool) {
    year := entry.Year()
    if wasAdded {
        yt.added[year]++
    } else {
        yt.duplicates[year]++
    }
}
```

#### 4. Update finalizeSummary to Use Tracked Statistics
```go
// Updated finalizeSummary using tracked per-year statistics
for year, entries := range yearlyEntries {
    i.summary.YearStats[year] = &YearStat{
        Initial:    i.yearTracker.initial[year],
        Added:      i.yearTracker.added[year],
        Duplicates: i.yearTracker.duplicates[year],
        Final:      len(entries), // Should equal Initial + Added
    }
}
```

### Integration Points

#### Repository Loading Phase
- Load existing calls/SMS from repository
- For each entry, call `yearTracker.trackInitialEntry(entry)`
- Populate Initial counts per year before import begins

#### Import Processing Phase
- Process each import file entry-by-entry
- Use coalescer.Add() return value to determine if entry was added (true) or duplicate (false)
- Call `yearTracker.trackImportEntry(entry, wasAdded)` for each entry
- Accumulate Added/Duplicates counts per year during processing

### Implementation Notes
- Entry-level tracking provides accurate per-year statistics
- Coalescer.Add() already returns true/false for new/duplicate detection
- Repository loading must happen before import to establish Initial baselines
- Math verification: Initial + Added = Final (Duplicates are not added to repository)
- Use efficient map[int]int for year-based counters (minimal memory overhead)

## Tasks
- [ ] Add YearTracker struct to Importer with initial, added, duplicates maps
- [ ] Implement trackInitialEntry method for repository loading phase
- [ ] Implement trackImportEntry method for import processing phase
- [ ] Update repository loading to count existing entries per year (Initial)
- [ ] Update import processing to track Added/Duplicates per entry using coalescer results
- [ ] Update finalizeSummary to use YearTracker statistics instead of only Final counts
- [ ] Verify mathematics: Initial + Added = Final for each year
- [ ] Add validation that Duplicates are tracked but not added to Final
- [ ] Write tests for YearTracker accuracy with known multi-year data
- [ ] Test integration with existing import workflow

## Testing
### Unit Tests
- Test YearTracker.trackInitialEntry increments correct year counter
- Test YearTracker.trackImportEntry handles Added vs Duplicates correctly
- Test finalizeSummary sets all YearStat fields from YearTracker data
- Test entry-level year extraction (entry.Year()) for various dates
- Test statistics math validation (Initial + Added = Final per year)

### Integration Tests
- Test repository loading populates Initial counts correctly per year
- Test import with multi-year data tracks Added/Duplicates per year
- Test import summary display shows accurate non-zero counts
- Test import with mixed new/duplicate entries across multiple years
- Test end-to-end: repository load → import → finalization → display

### Edge Cases
- Import into empty repository (Initial = 0, only Added counts)
- Import with all new entries (Duplicates = 0 per year)
- Import with all duplicates (Added = 0 per year)
- Import spanning many years (2010-2025)
- Repository with existing multi-year data + new multi-year import
- Single-year imports into multi-year repository

## Risks and Mitigations
- **Risk**: Performance impact from per-entry year tracking during large imports
  - **Mitigation**: YearTracker uses simple map lookups/increments (O(1) operations); minimal overhead
- **Risk**: Memory usage increase from tracking statistics per year
  - **Mitigation**: Uses map[int]int (small integers); even 50 years × 3 stats = 150 ints (~600 bytes)
- **Risk**: Repository loading phase becomes slower due to Initial counting
  - **Mitigation**: Counting happens during existing entry enumeration; no additional I/O
- **Risk**: Complexity increase in import workflow
  - **Mitigation**: YearTracker encapsulates all complexity; clean integration points

## References
- Related features: FEAT-010 (Import subcommand implementation)
- Code locations: pkg/importer/importer.go:303 (finalizeSummary function)
- Code locations: pkg/importer/importer.go:192-196 (summary display code)

## Notes
Additional thoughts, questions, or considerations that arise during planning/implementation.