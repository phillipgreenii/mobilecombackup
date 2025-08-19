# Architecture Decision Records (ADR) Index

This directory contains Architecture Decision Records (ADRs) that document significant architectural decisions made during the development of the mobilecombackup project.

## What are ADRs?

Architecture Decision Records (ADRs) are documents that capture important architectural decisions along with their context and consequences. They help teams understand why certain technical choices were made and provide guidance for future decisions.

## Current ADRs

| ADR # | Title | Status | Date | Description |
|-------|-------|--------|------|-------------|
| [0001](0001-streaming-vs-batch-processing.md) | Streaming vs Batch Processing | Accepted | 2024-01-15 | Decision to use streaming processing for large XML files |
| [0002](0002-hash-based-attachment-storage.md) | Hash-based Attachment Storage | Accepted | 2024-01-15 | Use SHA-256 content addressing for attachment storage |
| [0003](0003-xml-parsing-security-approach.md) | XML Parsing Security Approach | Accepted | 2024-01-15 | Security measures for XML processing including XXE prevention |
| [0004](0004-repository-structure-design.md) | Repository Structure Design | Accepted | 2024-01-15 | Year-based partitioning with typed directories using UTC |
| [0005](0005-development-tool-choices.md) | Development Tool Ecosystem Choices | Accepted | 2024-01-15 | Devbox-based development environment with comprehensive tooling |

## ADR Lifecycle

- **Proposed**: Under discussion and review
- **Accepted**: Decision has been made and is being implemented
- **Deprecated**: Decision is outdated but still in effect for legacy reasons
- **Superseded**: Decision has been replaced by a newer ADR

## Creating New ADRs

1. Copy the [template.md](template.md) file
2. Name the new file with the next sequential number: `XXXX-descriptive-title.md`
3. Fill in all sections of the template
4. Update this index.md file with the new ADR entry
5. Commit the ADR along with related code changes

## ADR Numbering

ADRs are numbered sequentially starting from 0001. Use leading zeros to maintain consistent sorting (0001, 0002, 0003, etc.).

## Template

Use the [ADR template](template.md) as the starting point for all new ADRs. The template provides a consistent structure for documenting decisions.

## Guidelines

- **Be specific**: Focus on architectural decisions that have long-term impact
- **Include context**: Explain the problem or situation that led to the decision
- **Document alternatives**: Show what options were considered and why they were rejected
- **Update when needed**: If a decision changes, create a new ADR that supersedes the old one
- **Link related decisions**: Reference other ADRs that are related or affected

## Maintenance

**Important**: When adding new ADRs, always update this index.md file to include the new record in the table above. This ensures the index remains current and useful for finding relevant architectural decisions.