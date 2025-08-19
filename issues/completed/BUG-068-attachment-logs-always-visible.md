# BUG-068: Attachment logs always visible despite --verbose flag

## Status
- **Reported**: 2025-08-19
- **Completed**: 2025-08-19
- **Priority**: medium
- **Severity**: minor

## Overview
The attachment logs, which are the output lines containing "[ATTACHMENT]", appear on the console by default even when the --verbose flag is not specified. These debug logs should only be visible when --verbose is enabled.

## Issue Description
When running import commands, debug messages like "[ATTACHMENT] Processing MMS part: image/jpeg (15234 bytes)" are always displayed, cluttering the output for normal users.

## Root Cause
The SMS attachment extractor in `pkg/sms/extractor.go` was using `log.Printf()` calls directly instead of using the structured logging system that respects the verbose flag.

## Resolution
Fixed by implementing structured logging throughout the SMS extractor:
- Modified AttachmentExtractor to accept a logger parameter
- Replaced all 24 `log.Printf("[ATTACHMENT] ...")` calls with `logger.Debug()` calls
- Updated dependency chain from CLI ’ Importer ’ SMS Importer ’ AttachmentExtractor
- Fixed all test files to use `logging.NewNullLogger()`

## Verification
- All attachment logs now respect the --verbose flag
- Normal operation shows clean output without debug messages
- --verbose flag shows detailed attachment processing information
- All tests pass and linting is clean