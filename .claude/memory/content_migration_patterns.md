# Content Migration Patterns

## README.md Content Guidelines

### What STAYS in README.md (Essential Quick-Start Only)

#### Project Overview (10-15 lines)
```markdown
# Project Title
[![badges]()] 
Brief description of what the tool does (1-2 sentences)
```

#### Quick Installation (15-20 lines)
```markdown
## Installation
### Quick Install (Primary Method Only)
```bash
# Single command for most common installation
nix run github:phillipgreenii/mobilecombackup -- --help
```

📖 **[See complete installation guide](docs/INSTALLATION.md)** for all methods and troubleshooting.
```

#### Basic Usage (30-40 lines)
```markdown
## Usage
### Quick Start
```bash
# Initialize repository
mobilecombackup init

# Import backup files
mobilecombackup import backup.xml
```

📖 **[See complete CLI reference](docs/CLI_REFERENCE.md)** for all commands and options.
```

#### Documentation Navigation (20-30 lines)
```markdown
## Documentation
- 📦 **[Installation Guide](docs/INSTALLATION.md)** - All installation methods and troubleshooting
- 🛠️ **[CLI Reference](docs/CLI_REFERENCE.md)** - Complete command documentation
- 🔧 **[Development Guide](docs/DEVELOPMENT.md)** - Setup, testing, and contribution
- 🚀 **[Deployment Guide](docs/DEPLOYMENT.md)** - Production deployment
- 🗂️ **[Documentation Index](docs/INDEX.md)** - Find everything
```

### What MOVES from README.md

#### Installation Details → docs/INSTALLATION.md
```markdown
MOVE: Detailed Nix flake examples
MOVE: Build from source instructions
MOVE: Platform-specific notes
MOVE: Dependency management
MOVE: Installation troubleshooting
```

#### CLI Examples → docs/CLI_REFERENCE.md
```markdown
MOVE: Detailed command examples (init, validate, info, import)
MOVE: All flag documentation
MOVE: JSON output examples  
MOVE: Exit code tables
MOVE: Advanced usage scenarios
```

#### Development Info → docs/DEVELOPMENT.md
```markdown
MOVE: Development sandbox setup
MOVE: Testing commands and strategies
MOVE: CI/CD pipeline documentation
MOVE: Code quality tool configuration
MOVE: Contribution workflow details
```

## Migration Transformation Examples

### Before (README.md - Too Detailed)
```markdown
### Init Command

Initialize a new mobilecombackup repository with the required directory structure.

```bash
# Initialize in current directory
$ mobilecombackup init

# Initialize in specific directory
$ mobilecombackup init --repo-root /path/to/new/repo

# Preview without creating (dry run)
$ mobilecombackup init --dry-run

# Initialize quietly (suppress output)
$ mobilecombackup init --quiet
```

The init command creates:
- `calls/` - Directory for call log XML files
- `sms/` - Directory for SMS/MMS XML files
- `attachments/` - Directory for extracted attachment files
- `.mobilecombackup.yaml` - Repository marker file with version metadata
- `contacts.yaml` - Empty contacts file for future use
- `summary.yaml` - Initial summary with zero counts

Example output:
```
Initialized mobilecombackup repository in: /path/to/repo

Created structure:
repo
├── calls
├── sms
├── attachments
├── .mobilecombackup.yaml
├── contacts.yaml
└── summary.yaml
```
```

### After (README.md - Essential Only)
```markdown
## Usage

### Quick Start
```bash
# Initialize repository
mobilecombackup init

# Import backup files  
mobilecombackup import backup.xml

# View repository info
mobilecombackup info
```

📖 **[Complete CLI Reference](docs/CLI_REFERENCE.md)** - All commands, flags, and examples.
```

### After (docs/CLI_REFERENCE.md - Detailed)
```markdown
# CLI Reference

## init Command

Initialize a new mobilecombackup repository with the required directory structure.

### Syntax
```bash
mobilecombackup init [flags]
```

### Examples
```bash
# Initialize in current directory
$ mobilecombackup init

# Initialize in specific directory
$ mobilecombackup init --repo-root /path/to/new/repo

# Preview without creating (dry run)
$ mobilecombackup init --dry-run

# Initialize quietly (suppress output)
$ mobilecombackup init --quiet
```

### Created Structure
The init command creates:
- `calls/` - Directory for call log XML files
- `sms/` - Directory for SMS/MMS XML files
- `attachments/` - Directory for extracted attachment files
- `.mobilecombackup.yaml` - Repository marker file with version metadata
- `contacts.yaml` - Empty contacts file for future use
- `summary.yaml` - Initial summary with zero counts

### Example Output
```
Initialized mobilecombackup repository in: /path/to/repo

Created structure:
repo
├── calls
├── sms
├── attachments
├── .mobilecombackup.yaml
├── contacts.yaml
└── summary.yaml
```

[← Back to README](../README.md) | [Next: validate command](#validate-command)
```

## Navigation Pattern Examples

### README.md Navigation Section
```markdown
## Documentation

Find detailed information in our comprehensive documentation:

### 🚀 **Getting Started**
- 📦 **[Installation Guide](docs/INSTALLATION.md)** - All installation methods, prerequisites, and troubleshooting
- ⚡ **[Quick Start Tutorial](docs/CLI_REFERENCE.md#quick-start)** - Get up and running in 5 minutes

### 📖 **Reference Guides**  
- 🛠️ **[Complete CLI Reference](docs/CLI_REFERENCE.md)** - Every command, flag, and example
- 🏗️ **[Architecture Overview](docs/ARCHITECTURE.md)** - System design and technical decisions

### 🔧 **Development**
- 👨‍💻 **[Development Guide](docs/DEVELOPMENT.md)** - Setup, testing, and contribution workflows
- 🚀 **[Deployment Guide](docs/DEPLOYMENT.md)** - Production deployment and Docker usage

### 🔍 **Find Anything**
- 🗂️ **[Documentation Index](docs/INDEX.md)** - Complete documentation directory and search guide
```

### Cross-Reference Pattern
```markdown
<!-- In docs/INSTALLATION.md -->
## Next Steps

After installation, see:
- **[Quick Start Tutorial](CLI_REFERENCE.md#quick-start)** - Your first commands
- **[Development Setup](DEVELOPMENT.md)** - Contributing to the project
- **[← Back to README](../README.md)** - Project overview

---
📖 **[Documentation Index](INDEX.md)** | 🏠 **[Back to README](../README.md)**
```

## Content Quality Standards

### Consistency Requirements
- Use same terminology across all documents
- Maintain unified code block formatting
- Apply consistent emoji usage for navigation
- Use standard heading hierarchy (H1=title, H2=sections, etc.)

### Link Management
- Always use relative paths for internal docs
- Include bidirectional navigation (forward/back)
- Provide "up" navigation to parent topics
- Test all links after migration

### User Experience Optimization
- Lead with most common use cases
- Group related information together
- Use progressive disclosure (summary → details)
- Include clear calls-to-action for next steps

## Validation Checklist

### Content Migration Verification
- [ ] All original information preserved
- [ ] No content duplication between files  
- [ ] Consistent terminology across documents
- [ ] Appropriate level of detail for each file

### Navigation Verification
- [ ] All internal links functional
- [ ] Clear path from README to detailed info
- [ ] Bidirectional navigation where appropriate
- [ ] docs/INDEX.md includes all topics

### User Experience Verification
- [ ] New users can find installation quickly (<30 seconds)
- [ ] Developers can find contribution info quickly (<1 minute)
- [ ] Mobile-friendly reading experience
- [ ] Logical information hierarchy maintained