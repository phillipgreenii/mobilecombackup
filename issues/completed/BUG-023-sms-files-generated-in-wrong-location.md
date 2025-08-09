# BUG-023: SMS files generated in root instead of sms directory

## Status
- **Reported**: 2025-08-08
- **Fixed**: 2025-08-09 (fixed as part of BUG-016)
- **Priority**: high
- **Severity**: major

## Overview
During import, SMS XML files (sms-YYYY.xml) are being generated in the repository root directory instead of in the `sms/` subdirectory where they belong according to the repository structure.

## Reproduction Steps
1. Initialize a new repository with `mobilecombackup init`
2. Import SMS data with `mobilecombackup import path/to/sms-data.xml`
3. Check the repository structure
4. Observe that sms-2024.xml files are in the root directory instead of sms/

## Expected Behavior
SMS files should be created in the `sms/` directory:
```
repository/
├── .mobilecombackup.yaml
├── sms/
│   ├── sms-2024.xml
│   └── sms-2025.xml
└── ...
```

## Actual Behavior
SMS files are created in the repository root:
```
repository/
├── .mobilecombackup.yaml
├── sms-2024.xml    # Wrong location!
├── sms-2025.xml    # Wrong location!
├── sms/            # Empty directory
└── ...
```

## Environment
- Version: Current main branch
- OS: All platforms affected
- Go version: 1.24

## Root Cause Analysis
### Investigation Notes
This bug was discovered to be the same root cause as BUG-016. The SMS writer was being initialized with the repository root path instead of the SMS subdirectory path.

### Root Cause
In `pkg/importer/sms.go` line 265, the SMS writer was being created with:
```go
writer, err := sms.NewXMLSMSWriter(si.options.RepoRoot)
```

This caused files to be written to the repository root instead of the sms/ subdirectory.

## Fix Approach
Fixed as part of BUG-016 by updating the SMS writer initialization to:
```go
smsDir := filepath.Join(si.options.RepoRoot, "sms")
writer, err := sms.NewXMLSMSWriter(smsDir)
```

## Tasks
- [x] Identify SMS file path construction code (found in pkg/importer/sms.go)
- [x] Fix path to include sms/ subdirectory (fixed in BUG-016)
- [x] Ensure sms/ directory exists before writing (writer creates it)
- [x] Add tests for correct file placement (TestSMSImporter_BUG023_FilesInCorrectDirectory)
- [x] Test with multiple year files (covered by existing tests)
- [x] Update integration tests (TestSMSImporter_BUG016_MessagesNotWritten)

## Testing
### Regression Tests
- Test SMS import creates files in sms/ directory
- Test with multiple years of data
- Verify calls still go to calls/ directory
- Test mixed import (calls + SMS)

### Verification Steps
1. Import SMS test data
2. Verify all sms-YYYY.xml files are in sms/ directory
3. Verify no XML files in repository root
4. Run validation to ensure files are found

## Workaround
Users can manually move the files from root to sms/ directory after import.

## Related Issues
- BUG-016: Imported SMS missing inner elements (fixed both issues with same change)
- FEAT-009: Import SMS functionality (where the bug existed)
- FEAT-010: Add Import subcommand

## Notes
This bug was fixed as part of BUG-016. The same code change that fixed SMS messages not being written also fixed the incorrect file location issue. Both bugs had the same root cause - incorrect SMS writer initialization.