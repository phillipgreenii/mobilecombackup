# ADR-0004: Repository Structure Design

**Status:** Accepted
**Date:** 2024-01-15
**Author:** Development Team
**Deciders:** Core development team

## Context

We needed to design a repository structure for storing processed mobile backup data that supports:

1. **Time-based organization**: Efficient querying by date ranges
2. **Scalable storage**: Handle years of communication data
3. **Clear structure**: Intuitive organization for users and tools
4. **Type separation**: Different data types organized appropriately
5. **Metadata management**: Manifest files, checksums, and repository metadata

The repository serves as the canonical storage format for processed backup data.

## Decision

We chose **year-based partitioning with typed directories** using UTC-based year determination:

```
repository/
├── .mobilecombackup.yaml  # Repository marker and metadata
├── calls/                 # Call records by year (UTC)
│   ├── calls-2021.xml
│   ├── calls-2022.xml
│   └── calls-2023.xml
├── sms/                   # SMS/MMS records by year (UTC)
│   ├── sms-2021.xml
│   ├── sms-2022.xml
│   └── sms-2023.xml
├── attachments/           # Hash-based attachment storage
│   ├── ab/abc123.../
│   └── cd/cdef45.../
├── contacts.yaml          # Contact information
├── summary.yaml           # Repository statistics
├── files.yaml             # File manifest
└── checksum               # Repository integrity checksum
```

## Rationale

### Year-Based Partitioning Benefits
- **Time-range queries**: Efficient access to specific year's data
- **Scalable growth**: New years create new files, old files unchanged
- **Backup efficiency**: Incremental backups only need current year
- **Archive capability**: Old years can be archived or compressed independently

### UTC-Based Year Determination
- **Consistent partitioning**: No timezone-dependent boundary issues
- **Predictable behavior**: Same input always produces same output
- **Global compatibility**: Works consistently across timezones
- **Archival stability**: Year assignment never changes with timezone config

### Typed Directory Organization
- **Clear separation**: Different data types in dedicated directories
- **Tool compatibility**: Easy for scripts and tools to locate specific data
- **Permission management**: Different access controls per data type
- **Processing efficiency**: Can process one data type independently

### Repository Metadata
- **Marker file**: `.mobilecombackup.yaml` identifies valid repositories
- **Integrity checking**: Checksums and manifests detect corruption
- **Statistics tracking**: Summary information for repository health
- **Version compatibility**: Repository format version tracking

### Alternatives Considered

1. **Flat file structure**: All data in single directory
   - Rejected: Becomes unwieldy with thousands of files
   - Difficult to organize and maintain over time

2. **Date-based hierarchy**: `YYYY/MM/DD/` directory structure
   - Rejected: Too granular, creates excessive directory depth
   - Most queries are year-based, not day-based

3. **Single file per type**: One large file for all calls, all SMS
   - Rejected: Files become extremely large and hard to process
   - No time-based partitioning for efficient queries

4. **Database storage**: Store data in SQLite or other database
   - Rejected: Backup and portability complexity
   - Vendor lock-in and dependency management

5. **Month-based partitioning**: Files per month instead of year
   - Rejected: Too granular for typical usage patterns
   - 12x more files with minimal query benefit

## Consequences

### Positive Consequences
- **Efficient time queries**: Year-based access patterns optimized
- **Scalable growth**: Repository size grows predictably over time
- **Clear organization**: Users and tools easily understand structure
- **Backup friendly**: Incremental backups work naturally
- **Archive capability**: Old data easily archived or compressed
- **Tool compatibility**: Standard directory structure easy to process

### Negative Consequences
- **Year boundary complexity**: Records at year boundaries require careful handling
- **Migration requirements**: Changing partitioning scheme requires data migration
- **Cross-year queries**: Queries spanning multiple years require multiple file access
- **Empty year handling**: Years with no data still create directory entries

## Implementation

### Repository Creation
1. Create directory structure with proper permissions
2. Generate `.mobilecombackup.yaml` marker file with metadata
3. Initialize empty data files for current year
4. Create integrity manifest and checksums

### Data Organization Process
1. Determine year from record timestamp (UTC-based)
2. Route record to appropriate year file
3. Update manifests and checksums incrementally
4. Generate summary statistics per processing session

### Year Boundary Handling
- Timestamps converted to UTC before year determination
- Consistent year assignment regardless of local timezone
- Clear documentation of UTC-based partitioning
- Tools handle timezone conversion for user display

### Repository Validation
- Marker file presence and format validation
- File manifest consistency checking
- Checksum verification for integrity
- Structure validation for expected directories

## Related Decisions

- **ADR-0002**: Hash-based Storage - Attachments directory structure
- **ADR-0001**: Streaming Processing - Repository written during streaming import