# FEAT-038: Import Summary Section Years Ordering

## Status
- **Reported**: 2025-08-13
- **Priority**: medium

## Overview
Import summary sections should display years in ascending chronological order for better readability and consistency. Currently, years may be displayed in an inconsistent or non-chronological order.

## Background
When generating import summaries, users expect to see data organized chronologically with years progressing from oldest to newest. This provides a natural reading flow and makes it easier to understand the timeline of imported data.

## Requirements
### Functional Requirements
- [ ] Import summary sections must display years in ascending chronological order
- [ ] Year ordering should be consistent across all summary output formats
- [ ] Ordering should handle edge cases like missing years gracefully

### Non-Functional Requirements
- [ ] Performance should not be significantly impacted by sorting
- [ ] Implementation should maintain backward compatibility

## Design
### Approach
Modify the import summary generation logic to sort years in ascending order before displaying them in summary sections.

### API/Interface
```go
// Ensure summary generation methods return years in sorted order
type SummaryGenerator interface {
    GenerateYearlySummary(data []Record) map[int]Summary
}
```

### Implementation Notes
- Identify where summary sections are generated
- Add year sorting logic before output generation
- Ensure consistency across different summary types (calls, SMS, etc.)

## Tasks
- [ ] Locate import summary generation code
- [ ] Identify where years are collected and displayed
- [ ] Implement year sorting in ascending order
- [ ] Update any related summary generation functions
- [ ] Write tests to verify year ordering
- [ ] Verify sorting works across different data sets

## Testing
### Unit Tests
- Test year sorting with mixed order input data
- Test edge cases with single year, missing years
- Test empty data scenarios

### Integration Tests
- End-to-end test with multi-year import data
- Verify summary output shows years in ascending order

### Edge Cases
- Handle datasets with gaps in years (e.g., 2020, 2022, 2024)
- Handle single year datasets
- Handle empty datasets

## Risks and Mitigations
- **Risk**: Performance impact on large datasets with many years
  - **Mitigation**: Use efficient sorting algorithms, consider caching if needed

## References
- Code locations: Import summary generation functions
- Related features: Import functionality (FEAT-010)

## Notes
This is a user experience improvement that will make import summaries more readable and professional.