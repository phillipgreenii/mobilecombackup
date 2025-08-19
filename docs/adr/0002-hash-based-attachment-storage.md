# ADR-0002: Hash-based Attachment Storage

**Status:** Accepted
**Date:** 2024-01-15
**Author:** Development Team
**Deciders:** Core development team

## Context

Mobile backup files contain media attachments (images, audio, video) that need efficient storage with the following requirements:

1. **Deduplication**: Same attachment in multiple messages should be stored once
2. **Integrity**: Verify attachment content hasn't been corrupted
3. **Organization**: Scalable directory structure for thousands of files
4. **Content addressing**: Locate attachments by content, not arbitrary names

We needed to choose between several storage approaches for managing these attachments.

## Decision

We chose **SHA-256 hash-based directory structure** with content addressing: `attachments/{hash[0:2]}/{hash}/filename`.

## Rationale

### Content Deduplication
- SHA-256 hash uniquely identifies identical content
- Multiple messages referencing same attachment store only one copy
- Automatic deduplication without complex tracking mechanisms
- Significant storage savings for shared media (profile photos, forwarded content)

### Integrity Verification
- Hash serves as cryptographic checksum for corruption detection
- File integrity verifiable without additional metadata
- Failed verification indicates storage corruption or tampering
- Supports data recovery and repair operations

### Scalable Organization
- Two-level directory structure prevents filesystem performance issues
- First level: 256 directories (00-ff) for hash distribution
- Second level: Full hash directory containing actual files
- Supports millions of attachments without directory traversal slowdown

### Content Addressing Benefits
- Location independent of original filename or source
- Enables content-based operations (find all instances of specific attachment)
- Simplifies backup and synchronization (hash-based comparison)
- Natural fit for distributed or replicated storage systems

### Alternatives Considered

1. **Sequential naming**: `attachments/001/002/003.jpg`
   - Rejected: No deduplication, no integrity verification
   - Vulnerable to filename collisions and organization complexity

2. **Original filename preservation**: `attachments/2023/messages/image.jpg`
   - Rejected: Filename collisions, no deduplication
   - Path length limitations with complex directory structures

3. **Database-based storage**: Store attachments as BLOBs in database
   - Rejected: Database size explosion, backup complexity
   - Poor performance for large binary data

4. **Flat hash directory**: `attachments/{hash}`
   - Rejected: Filesystem performance degradation with many files
   - Directory listing becomes expensive with thousands of files

## Consequences

### Positive Consequences
- **Automatic deduplication**: Identical attachments stored once
- **Integrity verification**: Built-in corruption detection
- **Scalable performance**: Directory structure scales to millions of files
- **Content addressing**: Location by content hash, not arbitrary paths
- **Storage efficiency**: Significant space savings through deduplication
- **Backup/sync friendly**: Hash-based comparison for efficient replication

### Negative Consequences
- **Hash calculation overhead**: SHA-256 computation for every attachment
- **Directory structure complexity**: Less intuitive than filename-based organization
- **Recovery complexity**: Manual browsing requires hash-to-content mapping
- **Metadata storage**: Original filenames must be preserved separately

## Implementation

### Directory Structure
```
attachments/
├── ab/
│   └── abc123.../
│       ├── image.jpg
│       └── document.pdf
├── cd/
│   └── cdef45.../
│       └── audio.mp3
```

### Key Components
- `AttachmentManager`: High-level interface for attachment operations
- `DirectoryAttachmentStorage`: Implements hash-based file organization
- Hash calculation during streaming import process
- Metadata preservation for original filenames and MIME types

### Storage Process
1. Calculate SHA-256 hash during attachment extraction
2. Check if attachment already exists using hash
3. If new, create directory structure and store file
4. Record hash-to-metadata mapping for original context

### Retrieval Process
1. Lookup attachment by hash
2. Construct file path using hash-based directory structure
3. Return file path or content with preserved metadata

## Related Decisions

- **ADR-0001**: Streaming Processing - Enables efficient hash calculation during import
- **ADR-0004**: Repository Structure - Attachments directory is part of overall repository design