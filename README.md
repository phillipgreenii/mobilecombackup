# MobileComBackup

[![Built with Devbox](https://www.jetify.com/img/devbox/shield_galaxy.svg)](https://www.jetify.com/devbox/docs/contributor-quickstart/)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=phillipgreenii_mobilecombackup&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=phillipgreenii_mobilecombackup)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=phillipgreenii_mobilecombackup&metric=coverage)](https://sonarcloud.io/summary/new_code?id=phillipgreenii_mobilecombackup)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=phillipgreenii_mobilecombackup&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=phillipgreenii_mobilecombackup)

A command-line tool for processing mobile phone backup files (Call and SMS logs in XML format). It coalesces multiple backup files, removes duplicates, extracts attachments, and organizes data by year.

## Quick Install

```bash
# Run directly with Nix
nix run github:phillipgreenii/mobilecombackup -- --help

# Or install to your system
nix profile install github:phillipgreenii/mobilecombackup
```

📖 **[Complete Installation Guide](docs/INSTALLATION.md)** - All platforms, build from source, and troubleshooting.

## Quick Start

```bash
# Initialize repository
mobilecombackup init

# Import backup files
mobilecombackup import backup.xml

# View repository info
mobilecombackup info

# Validate repository
mobilecombackup validate
```

📖 **[Complete CLI Reference](docs/CLI_REFERENCE.md)** - All commands, flags, examples, and advanced usage.

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

## Contributing

```bash
# Quick setup for contributors
devbox shell          # Enter development environment
devbox run ci         # Run full CI pipeline
```

📖 **[Development Guide](docs/DEVELOPMENT.md)** - Complete setup, testing, and contribution workflow.

## License

This project is open source. See the repository for license details.

---

**Need help?** Check the **[Documentation Index](docs/INDEX.md)** or **[Troubleshooting Guide](docs/TROUBLESHOOTING.md)**.
