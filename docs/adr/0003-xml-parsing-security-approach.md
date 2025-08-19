# ADR-0003: XML Parsing Security Approach

**Status:** Accepted
**Date:** 2024-01-15
**Author:** Development Team
**Deciders:** Core development team

## Context

Mobile backup files are XML documents that contain user data from potentially untrusted sources. XML parsing has several well-known security vulnerabilities:

1. **XML External Entity (XXE) attacks**: Malicious XML can reference external resources
2. **XML Bomb attacks**: Exponentially expanding XML structures causing DoS
3. **Billion Laughs attack**: Nested entity expansion consuming excessive memory
4. **Path traversal**: XML content containing filesystem references

We needed a secure XML parsing approach that prevents these attacks while maintaining functionality for legitimate backup files.

## Decision

We implemented a **custom secure XML decoder wrapper** that enforces security constraints by default.

## Rationale

### Security by Default
- All XML parsing goes through security-hardened wrapper
- No possibility of accidentally using unsafe parsing in new code
- Centralized security controls easy to audit and update
- Conservative defaults with explicit opt-in for advanced features

### XXE Prevention
- External entity resolution completely disabled
- No network access during XML parsing
- Local file system access blocked
- Prevents data exfiltration through malicious XML

### Resource Limit Enforcement
- Maximum XML document size limits
- Entity expansion depth restrictions
- Processing time limits to prevent DoS
- Memory usage bounds for parser operations

### Compatibility Preservation
- Standard XML parsing for legitimate backup files works unchanged
- No impact on normal parsing performance
- Maintains compatibility with all known backup file formats
- Graceful error handling for malformed inputs

### Alternatives Considered

1. **Standard library XML parser with manual security checks**
   - Rejected: Easy to forget security checks in new code
   - Inconsistent application of security measures
   - Maintenance burden of repeated security code

2. **Input sanitization before parsing**
   - Rejected: Complex to implement correctly
   - Risk of breaking legitimate XML constructs
   - Still vulnerable to parser-level attacks

3. **Sandboxed parsing environment**
   - Rejected: Implementation complexity
   - Platform-specific sandboxing mechanisms
   - Performance overhead of process isolation

4. **Third-party security-focused XML library**
   - Rejected: Additional dependency management
   - Security properties depend on external maintenance
   - Integration complexity with existing streaming architecture

## Consequences

### Positive Consequences
- **Security by default**: All XML parsing is automatically secured
- **Centralized protection**: Single point of security control
- **XXE prevention**: Complete protection against external entity attacks
- **DoS resistance**: Resource limits prevent XML bomb attacks
- **Audit simplicity**: Security configuration in one location
- **Compatibility**: No changes required for legitimate backup files

### Negative Consequences
- **Additional abstraction layer**: Wrapper adds complexity to XML parsing
- **Limited flexibility**: Security restrictions may block legitimate edge cases
- **Custom implementation**: Need to maintain security wrapper code
- **Performance considerations**: Security checks add minimal processing overhead

## Implementation

### Core Security Features
```go
// Secure XML decoder configuration
type SecureXMLConfig struct {
    MaxDocumentSize    int64  // 100MB default limit
    MaxEntityDepth     int    // 10 levels maximum
    DisableExternalRef bool   // Always true
    ProcessingTimeout  time.Duration
}
```

### Security Controls
- **External entities disabled**: No file or network access
- **Entity expansion limits**: Prevent billion laughs attacks
- **Document size limits**: Prevent memory exhaustion
- **Processing timeouts**: Prevent infinite processing loops
- **Strict parsing mode**: Reject malformed or suspicious XML

### Wrapper Implementation
- Transparent replacement for standard XML parsing
- Consistent error handling for security violations
- Detailed logging of security-related parsing failures
- Performance monitoring for security overhead

### Usage Pattern
```go
// All XML parsing uses secure wrapper
decoder := security.NewSecureXMLDecoder(reader, config)
for {
    token, err := decoder.Token()
    // Handle token normally, security is transparent
}
```

## Related Decisions

- **ADR-0001**: Streaming Processing - Security applies to streaming XML parsing
- **ADR-0004**: Repository Structure - Security protects repository integrity