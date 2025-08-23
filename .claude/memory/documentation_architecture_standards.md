# Documentation Architecture Standards

## Core Principles

### README.md Size Limit
- **HARD LIMIT**: 300 lines maximum (currently 544 lines, target <300)
- **WARNING THRESHOLD**: 280 lines (triggers content migration requirement)
- **PURPOSE**: Ensure quick discoverability and mobile-friendly experience

### Single Responsibility Documentation
Each documentation file has a focused purpose:
- **README.md**: Project overview, quick-start, navigation hub
- **docs/INSTALLATION.md**: All installation methods and troubleshooting
- **docs/CLI_REFERENCE.md**: Complete command documentation and examples
- **docs/DEVELOPMENT.md**: Development setup, testing, CI/CD workflows
- **docs/DEPLOYMENT.md**: Production deployment and Docker usage
- **docs/INDEX.md**: Documentation navigation and discovery guide

### Hierarchical Information Architecture
```
Level 1 (README.md): Essential quick-start info only
├── Level 2 (docs/*.md): Comprehensive topic-specific documentation
└── Level 3 (docs/*/): Specialized subdocuments (ADRs, migrations, etc.)
```

## Documentation Placement Rules

### README.md Content Restrictions
ONLY these content types belong in README.md:
1. **Project Overview** (10-15 lines): Title, badges, brief description
2. **Quick Installation** (15-20 lines): Single primary installation method
3. **Basic Usage** (30-40 lines): 2-3 essential commands with minimal examples
4. **Navigation Links** (20-30 lines): Clear paths to detailed documentation
5. **Contributing Basics** (10-15 lines): How to get started contributing
6. **License & Links** (5-10 lines): Essential legal and contact information

### Content Migration Triggers
Move content from README.md when:
- README.md approaches 280 lines
- Content becomes detailed/comprehensive
- Multiple installation methods are documented
- CLI examples exceed basic usage
- Development workflows are explained
- Architecture details are included

## Quality Standards

### Link Management
- Use absolute paths for all documentation references
- Validate all internal links after changes
- Maintain bidirectional navigation where appropriate
- Include "back to top" or "back to index" links in long documents

### Content Consistency
- Use consistent terminology across all documentation
- Maintain unified code example formatting
- Apply consistent heading hierarchy (H1 for titles, H2 for sections, etc.)
- Use standard formatting for CLI commands, file paths, and code blocks

### User Experience Design
- **New User Journey**: Installation → Basic Usage → Advanced Features
- **Developer Journey**: Contributing → Development Setup → Architecture
- **Maintenance Journey**: Troubleshooting → Debugging → Issue Reporting

## Implementation Guidelines

### Content Audit Process
1. Identify content exceeding README.md scope
2. Categorize by topic and target documentation file
3. Preserve all information during migration
4. Create clear navigation between related topics
5. Validate user experience flows

### Migration Strategy
1. **Preserve First**: Never lose information during restructuring
2. **Navigate Second**: Ensure users can find moved content
3. **Validate Third**: Test all links and user experience flows
4. **Measure Success**: Verify README.md under 300 lines

## Success Metrics

### Quantitative Goals
- README.md under 300 lines (hard requirement)
- Zero broken links after restructuring
- All original content preserved in appropriate locations
- Complete navigation coverage between related topics

### Qualitative Goals
- New users find installation in <30 seconds
- Developers locate contribution info in <1 minute
- Documentation flows logically from overview to details
- Mobile-friendly reading experience maintained

## Maintenance Responsibilities

### Agent Requirements
- Check `wc -l README.md` before adding content
- Follow placement decision tree for all new documentation
- Update cross-references when modifying related documents
- Validate navigation flows after documentation changes

### Review Criteria
- Content fits assigned documentation file purpose
- Links between documents work correctly
- User experience flows remain intuitive
- No content duplication between files
- Consistent formatting and style applied