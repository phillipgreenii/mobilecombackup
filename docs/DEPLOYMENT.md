# Deployment Guide

Production deployment strategies, containerization, and operational considerations for MobileComBackup.

## Deployment Strategies

### Binary Distribution (Recommended)

The simplest and most reliable deployment method using pre-built static binaries.

#### Direct Binary Deployment

```bash
# Download latest release
curl -L -o mobilecombackup https://github.com/phillipgreenii/mobilecombackup/releases/latest/download/mobilecombackup-linux-amd64

# Make executable
chmod +x mobilecombackup

# Install system-wide
sudo mv mobilecombackup /usr/local/bin/

# Verify installation
mobilecombackup --version
```

#### Automated Deployment Script

```bash
#!/bin/bash
# deploy-mobilecombackup.sh

set -e

PLATFORM="linux-amd64"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="mobilecombackup"

echo "Deploying MobileComBackup..."

# Download latest release
echo "Downloading latest release..."
curl -L -o "${BINARY_NAME}" \
  "https://github.com/phillipgreenii/mobilecombackup/releases/latest/download/mobilecombackup-${PLATFORM}"

# Make executable
chmod +x "${BINARY_NAME}"

# Install
echo "Installing to ${INSTALL_DIR}..."
sudo mv "${BINARY_NAME}" "${INSTALL_DIR}/"

# Verify
echo "Verifying installation..."
"${INSTALL_DIR}/${BINARY_NAME}" --version

echo "✅ MobileComBackup deployed successfully!"
```

### Package Manager Distribution

#### Nix Flakes (Production)

Create a system configuration with Nix:

```nix
# configuration.nix or flake.nix
{
  environment.systemPackages = [
    inputs.mobilecombackup.packages.${system}.default
  ];
  
  # Optional: Create system service
  systemd.services.mobilecombackup = {
    description = "Mobile backup processing service";
    serviceConfig = {
      Type = "oneshot";
      User = "backup-user";
      ExecStart = "${inputs.mobilecombackup.packages.${system}.default}/bin/mobilecombackup import /data/backups/";
    };
  };
}
```

#### Future Package Managers

- **Homebrew**: Planned for macOS distribution
- **APT/YUM**: Consider for major Linux distributions
- **Snap/Flatpak**: Potential universal Linux packaging

## Containerization

### Docker Deployment

#### Dockerfile

```dockerfile
FROM scratch

# Copy the static binary
COPY mobilecombackup /usr/local/bin/mobilecombackup

# Set working directory
WORKDIR /data

# Expose volume for data
VOLUME ["/data"]

# Default command
ENTRYPOINT ["/usr/local/bin/mobilecombackup"]
CMD ["--help"]
```

#### Building Container

```bash
# Build the binary (static)
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
  -ldflags "-X main.Version=$(git describe --tags --always --dirty)" \
  -o mobilecombackup ./cmd/mobilecombackup

# Build Docker image
docker build -t mobilecombackup:latest .

# Run container
docker run --rm -v "$(pwd)/data:/data" mobilecombackup:latest init
```

#### Multi-stage Build

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

RUN apk add --no-cache git
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.Version=$(git describe --tags --always --dirty)" \
    -o mobilecombackup ./cmd/mobilecombackup

# Runtime stage
FROM scratch

COPY --from=builder /app/mobilecombackup /usr/local/bin/mobilecombackup

WORKDIR /data
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/mobilecombackup"]
CMD ["--help"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  mobilecombackup:
    image: mobilecombackup:latest
    volumes:
      - ./data:/data
      - ./backups:/backups:ro
    command: import /backups
    
  # Optional: Web interface (future)
  mobilecombackup-web:
    image: mobilecombackup-web:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data:ro
    depends_on:
      - mobilecombackup
```

### Kubernetes Deployment

#### Job for One-time Processing

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: mobilecombackup-import
spec:
  template:
    spec:
      containers:
      - name: mobilecombackup
        image: mobilecombackup:latest
        command: ["mobilecombackup"]
        args: ["import", "/backups"]
        volumeMounts:
        - name: data-volume
          mountPath: /data
        - name: backup-volume
          mountPath: /backups
          readOnly: true
      volumes:
      - name: data-volume
        persistentVolumeClaim:
          claimName: mobilecombackup-data
      - name: backup-volume
        configMap:
          name: backup-files
      restartPolicy: OnFailure
```

#### CronJob for Scheduled Processing

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: mobilecombackup-scheduled
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: mobilecombackup
            image: mobilecombackup:latest
            command: ["mobilecombackup"]
            args: ["import", "/backups", "--quiet"]
            volumeMounts:
            - name: data-volume
              mountPath: /data
            - name: backup-volume
              mountPath: /backups
              readOnly: true
          volumes:
          - name: data-volume
            persistentVolumeClaim:
              claimName: mobilecombackup-data
          - name: backup-volume
            nfs:
              server: backup-server.example.com
              path: /exports/mobile-backups
          restartPolicy: OnFailure
```

## Configuration Management

### Environment Variables

```bash
# Repository configuration
export MB_REPO_ROOT="/data/mobile-backups"

# Operational settings
export MB_LOG_LEVEL="info"        # Future: logging levels
export MB_MAX_WORKERS="4"         # Future: parallel processing
export MB_TEMP_DIR="/tmp/mb"      # Future: temporary file location
```

### Configuration File (Future)

```yaml
# mobilecombackup.yaml
repository:
  root: "/data/mobile-backups"
  
processing:
  max_workers: 4
  temp_dir: "/tmp/mb"
  
logging:
  level: "info"
  format: "json"
  
monitoring:
  metrics_enabled: true
  health_check_port: 8081
```

## Monitoring and Observability

### Health Checks

```bash
# Basic health check script
#!/bin/bash
# health-check.sh

set -e

REPO_ROOT="${MB_REPO_ROOT:-/data/mobile-backups}"

# Check if repository is valid
mobilecombackup validate --repo-root "$REPO_ROOT" --quiet

# Check repository info
mobilecombackup info --repo-root "$REPO_ROOT" --json > /tmp/mb-status.json

echo "✅ MobileComBackup is healthy"
```

### Metrics Collection (Future)

```bash
# Prometheus metrics endpoint (planned)
curl http://localhost:8081/metrics

# Example metrics:
# mobilecombackup_repository_calls_total{year="2024"} 1234
# mobilecombackup_repository_sms_total{year="2024"} 5678
# mobilecombackup_repository_size_bytes 1073741824
# mobilecombackup_import_duration_seconds 12.5
```

### Log Management

```bash
# Structured logging output (JSON format)
mobilecombackup import /backups --json > import-log.json

# Log rotation with logrotate
cat > /etc/logrotate.d/mobilecombackup << EOF
/var/log/mobilecombackup/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 backup-user backup-user
}
EOF
```

## Performance Optimization

### Hardware Requirements

#### Minimum Requirements
- **CPU**: 1 core, 1GHz
- **Memory**: 256MB RAM
- **Storage**: 100MB for binary + data space
- **Network**: Not required for local processing

#### Recommended Production
- **CPU**: 4+ cores for parallel processing (future)
- **Memory**: 2GB+ RAM for large datasets
- **Storage**: SSD for better I/O performance
- **Network**: High-speed for remote backup sources

### Performance Tuning

```bash
# Process large files efficiently
mobilecombackup import --verbose /large-backups/ 2>&1 | \
  tee import-$(date +%Y%m%d).log

# Monitor resource usage during import
nohup mobilecombackup import /backups > import.log 2>&1 &
watch -n 5 'ps aux | grep mobilecombackup'
```

### Storage Optimization

```bash
# Repository size analysis
du -sh /data/mobile-backups/*

# Attachment deduplication verification
find /data/mobile-backups/attachments -name "*.jpg" | \
  xargs sha256sum | sort | uniq -d -w 64
```

## Backup and Recovery

### Repository Backup

```bash
#!/bin/bash
# backup-repository.sh

REPO_ROOT="${MB_REPO_ROOT:-/data/mobile-backups}"
BACKUP_DATE=$(date +%Y%m%d)
BACKUP_DIR="/backups/mobilecombackup-${BACKUP_DATE}"

echo "Creating repository backup..."

# Validate repository before backup
mobilecombackup validate --repo-root "$REPO_ROOT" --quiet

# Create backup
mkdir -p "$BACKUP_DIR"
cp -R "$REPO_ROOT"/* "$BACKUP_DIR"/

# Create archive
tar -czf "${BACKUP_DIR}.tar.gz" -C "/backups" "mobilecombackup-${BACKUP_DATE}"

# Verify backup
echo "Backup size: $(du -sh "${BACKUP_DIR}.tar.gz")"
echo "✅ Repository backup completed: ${BACKUP_DIR}.tar.gz"
```

### Disaster Recovery

```bash
#!/bin/bash
# restore-repository.sh

BACKUP_FILE="$1"
RESTORE_DIR="${MB_REPO_ROOT:-/data/mobile-backups}"

if [ -z "$BACKUP_FILE" ]; then
  echo "Usage: $0 <backup-file.tar.gz>"
  exit 1
fi

echo "Restoring repository from: $BACKUP_FILE"

# Extract backup
tar -xzf "$BACKUP_FILE" -C "$(dirname "$RESTORE_DIR")"

# Validate restored repository
mobilecombackup validate --repo-root "$RESTORE_DIR"

echo "✅ Repository restored successfully"
```

## Security Considerations

### Access Control

```bash
# Create dedicated user for backup processing
sudo useradd -r -s /bin/false -d /data/mobile-backups backup-user

# Set appropriate permissions
sudo chown -R backup-user:backup-user /data/mobile-backups
sudo chmod -R 750 /data/mobile-backups

# Limit binary permissions
sudo chown root:backup-user /usr/local/bin/mobilecombackup
sudo chmod 755 /usr/local/bin/mobilecombackup
```

### Data Protection

```bash
# Encrypt repository at rest (using LUKS)
cryptsetup luksFormat /dev/sdb
cryptsetup open /dev/sdb mobile-backups
mkfs.ext4 /dev/mapper/mobile-backups
mount /dev/mapper/mobile-backups /data/mobile-backups

# Network security (if accessing remote backups)
# Use SSH tunnels or VPN for secure data transfer
```

### Audit Logging

```bash
# Log all mobilecombackup operations
cat > /etc/rsyslog.d/mobilecombackup.conf << EOF
# Log mobilecombackup operations
:programname, isequal, "mobilecombackup" /var/log/mobilecombackup.log
& stop
EOF

systemctl restart rsyslog
```

## Operational Procedures

### Regular Maintenance

```bash
#!/bin/bash
# maintenance.sh - Daily maintenance script

REPO_ROOT="${MB_REPO_ROOT:-/data/mobile-backups}"

echo "Starting daily maintenance..."

# Validate repository integrity
echo "Validating repository..."
mobilecombackup validate --repo-root "$REPO_ROOT"

# Generate status report
echo "Generating status report..."
mobilecombackup info --repo-root "$REPO_ROOT" --json > /var/log/daily-status.json

# Clean up temporary files (future)
# find /tmp -name "mb-*" -mtime +7 -delete

echo "✅ Daily maintenance completed"
```

### Deployment Checklist

- [ ] Download and verify binary checksums
- [ ] Test binary on staging environment
- [ ] Backup existing installation
- [ ] Deploy new binary
- [ ] Verify version and functionality
- [ ] Update monitoring/alerting
- [ ] Document deployment in change log
- [ ] Monitor for 24 hours post-deployment

## Future Enhancements

### Planned Features

- **Web Interface**: Browser-based repository management
- **API Server**: REST API for programmatic access
- **Metrics Export**: Prometheus metrics endpoint
- **Configuration File**: YAML-based configuration
- **Parallel Processing**: Multi-threaded import/export
- **Cloud Storage**: S3/GCS backend support

### Scalability Considerations

- **Horizontal Scaling**: Process multiple repositories in parallel
- **Database Backend**: Replace file-based storage for large deployments
- **Distributed Processing**: Split large imports across multiple workers
- **Caching Layer**: Redis for frequently accessed metadata

## Next Steps

After deployment:

- **[CLI Reference](CLI_REFERENCE.md)** - Learn all available commands
- **[Troubleshooting Guide](TROUBLESHOOTING.md)** - Fix operational issues  
- **[Development Guide](DEVELOPMENT.md)** - Contribute improvements
- **[Architecture Overview](ARCHITECTURE.md)** - Understand system design

---

📖 **[Documentation Index](INDEX.md)** | 🏠 **[Back to README](../README.md)**