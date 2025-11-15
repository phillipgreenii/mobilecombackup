# Installation Guide

Complete guide to installing MobileComBackup on all supported platforms.

---

**Last Updated**: 2025-01-15
**Related Documents**: [CLI Reference](CLI_REFERENCE.md) | [Troubleshooting](TROUBLESHOOTING.md) | [Development Guide](DEVELOPMENT.md)
**Prerequisites**: Nix with flakes (recommended) or Go 1.24+

---

## Quick Install (Recommended)

### Using Nix Flakes

If you have Nix with flakes enabled, this is the fastest method:

```bash
# Run without installing
nix run github:phillipgreenii/mobilecombackup -- --help

# Install to profile  
nix profile install github:phillipgreenii/mobilecombackup
```

The Nix flake provides:
- **Smart version detection**: Automatically detects release tags or development versions
- **Static binary**: No runtime dependencies, works on any Linux system
- **Multi-platform**: Supports x86_64-linux, aarch64-linux, x86_64-darwin, aarch64-darwin
- **Quality checks**: Built-in validation tests ensuring binary functionality

## Advanced Installation Methods

### Using Nix in Your Project

Add to your `flake.nix` inputs for reproducible builds:

```nix
{
  inputs = {
    mobilecombackup.url = "github:phillipgreenii/mobilecombackup";
  };
  
  outputs = { self, nixpkgs, mobilecombackup, ... }: {
    # Use in your development environment
    devShells.default = pkgs.mkShell {
      buildInputs = [
        mobilecombackup.packages.${system}.default
      ];
    };
  };
}
```

### Building from Source

#### Prerequisites

- **Go 1.24+**: Required for building
- **Git**: For cloning the repository
- **Devbox** (optional): For consistent development environment

#### Build Steps

```bash
# Clone the repository
git clone https://github.com/phillipgreenii/mobilecombackup.git
cd mobilecombackup

# Option 1: Build with automatic version injection (recommended)
devbox run build-cli

# Option 2: Manual build with version information
VERSION=$(bash scripts/build-version.sh)
go build -ldflags "-X main.Version=$VERSION" -o mobilecombackup github.com/phillipgreenii/mobilecombackup/cmd/mobilecombackup

# Option 3: Basic build without version (development only)
go build -o mobilecombackup github.com/phillipgreenii/mobilecombackup/cmd/mobilecombackup
```

#### Development Environment Setup

For active development, use the devbox environment:

```bash
# Enter development environment with all dependencies
devbox shell

# Available tools: go 1.24, golangci-lint, gotestsum, and more
```

### Binary Releases

Pre-built binaries are available for all supported platforms:

1. Visit the [Releases page](https://github.com/phillipgreenii/mobilecombackup/releases)
2. Download the binary for your platform:
   - `mobilecombackup-linux-amd64` (Linux x86_64)
   - `mobilecombackup-linux-arm64` (Linux ARM64)
   - `mobilecombackup-darwin-amd64` (macOS Intel)
   - `mobilecombackup-darwin-arm64` (macOS Apple Silicon)
3. Make executable: `chmod +x mobilecombackup-*`
4. Move to PATH: `sudo mv mobilecombackup-* /usr/local/bin/mobilecombackup`

## Platform-Specific Instructions

### Linux (Ubuntu/Debian)

```bash
# Install Nix (if not already installed)
curl -L https://nixos.org/nix/install | sh

# Install with Nix flakes
nix profile install github:phillipgreenii/mobilecombackup

# Verify installation
mobilecombackup --version
```

### macOS

#### Using Nix (Recommended)

```bash
# Install Nix (if not already installed)
curl -L https://nixos.org/nix/install | sh

# Enable flakes (if not already enabled)
mkdir -p ~/.config/nix
echo "experimental-features = nix-command flakes" >> ~/.config/nix/nix.conf

# Install mobilecombackup
nix profile install github:phillipgreenii/mobilecombackup
```

#### Using Homebrew (Future)

Homebrew tap is planned for future releases. For now, use Nix or build from source.

### Windows (WSL2)

MobileComBackup runs on Windows through WSL2:

```bash
# In WSL2 Ubuntu
# Install Nix
curl -L https://nixos.org/nix/install | sh

# Install mobilecombackup
nix profile install github:phillipgreenii/mobilecombackup
```

## Verification

After installation, verify everything is working:

```bash
# Check version
mobilecombackup --version

# View help
mobilecombackup --help

# Test initialization (in empty directory)
mkdir test-repo && cd test-repo
mobilecombackup init
ls -la  # Should show created structure
```

## Version Management

### Checking Your Version

```bash
# Show current version
mobilecombackup --version
```

### Version Formats

- **Development builds**: `2.0.0-dev-g1234567` (base version + git hash)
- **Release builds**: `2.0.0` (clean semantic version from git tags)

### Updating

#### Nix Installation
```bash
# Update to latest version
nix profile upgrade mobilecombackup
```

#### Source Build
```bash
# Pull latest changes
git pull origin main

# Rebuild with latest version
devbox run build-cli
```

## Troubleshooting

### Common Issues

#### "command not found: mobilecombackup"

**Solution**: Add Nix profile to your PATH:
```bash
echo 'export PATH="$HOME/.nix-profile/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

#### "experimental-features" error with Nix

**Solution**: Enable flakes in Nix configuration:
```bash
mkdir -p ~/.config/nix
echo "experimental-features = nix-command flakes" >> ~/.config/nix/nix.conf
```

#### Build fails with "go: version too old"

**Solution**: Use devbox for consistent Go version:
```bash
devbox shell
devbox run build-cli
```

#### Version shows as "dev" instead of actual version

This is normal for development builds. Release builds will show proper semantic versions.

### Getting Help

If you encounter issues not covered here:

1. **Check existing issues**: [GitHub Issues](https://github.com/phillipgreenii/mobilecombackup/issues)
2. **Review troubleshooting**: [docs/TROUBLESHOOTING.md](TROUBLESHOOTING.md)
3. **Create new issue**: Include:
   - Operating system and version
   - Installation method used
   - Complete error message
   - Output of `mobilecombackup --version` (if working)

## Next Steps

After successful installation:

- **[Quick Start Tutorial](CLI_REFERENCE.md#quick-start)** - Get up and running in 5 minutes
- **[Complete CLI Reference](CLI_REFERENCE.md)** - Learn all available commands  
- **[Development Guide](DEVELOPMENT.md)** - Set up development environment
- **[Architecture Overview](ARCHITECTURE.md)** - Understand system design

---

📖 **[Documentation Index](INDEX.md)** | 🏠 **[Back to README](../README.md)**