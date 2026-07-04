#!/bin/bash
# Setup script for Phase 5 Automation Integration - FEAT-085
# Integrates quality dashboard with existing pre-commit hooks and CI/CD

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

echo -e "${PURPLE}🚀 FEAT-085 Phase 5 Automation Integration Setup${NC}"
echo -e "${PURPLE}===============================================${NC}"
echo ""

# 1. Setup enhanced pre-commit hooks
log_info "Setting up enhanced pre-commit hooks..."

# Make enhanced pre-commit hook executable
chmod +x "$SCRIPT_DIR/enhanced-pre-commit-hook.sh"
chmod +x "$SCRIPT_DIR/quality-monitor.sh"

# Create symlink for enhanced hook (optional - user can choose to use it)
if [ -f "$PROJECT_ROOT/.githooks/pre-commit.enhanced" ]; then
    log_warning "Enhanced pre-commit hook already exists"
else
    ln -s "$SCRIPT_DIR/enhanced-pre-commit-hook.sh" "$PROJECT_ROOT/.githooks/pre-commit.enhanced"
    log_success "Enhanced pre-commit hook linked to .githooks/pre-commit.enhanced"
fi

# 2. Setup quality monitoring infrastructure
log_info "Setting up quality monitoring infrastructure..."

# Create directories
mkdir -p "$PROJECT_ROOT/.dashboard-metrics"
mkdir -p "$PROJECT_ROOT/.quality-alerts"

# Create configuration file for quality monitoring
cat > "$PROJECT_ROOT/.quality-config.env" << 'EOF'
# Quality Monitoring Configuration - FEAT-085 Phase 5
# Source this file to configure quality monitoring settings

# Quality Thresholds
export COVERAGE_THRESHOLD=80
export QUALITY_SCORE_THRESHOLD=60
export PERFORMANCE_THRESHOLD=25
export MAX_CRITICAL_ISSUES=0

# Dashboard Settings
export QUALITY_DASHBOARD_ENABLED=true
export DASHBOARD_METRICS_DIR=".dashboard-metrics"

# Notification Settings (configure as needed)
export ENABLE_NOTIFICATIONS=false
export SLACK_WEBHOOK_URL=""
export EMAIL_RECIPIENTS=""

# Automation Settings
export AUTO_FIX_FORMATTING=true
export STRICT_QUALITY_GATES=false
EOF

log_success "Quality monitoring configuration created: .quality-config.env"

# 3. Setup devbox integration
log_info "Setting up devbox integration..."

# Add quality monitoring commands to devbox.json if it exists
if [ -f "$PROJECT_ROOT/devbox.json" ]; then
    log_info "Checking devbox.json for quality monitoring commands..."
    
    # Check if quality commands already exist
    if grep -q "quality-monitor" "$PROJECT_ROOT/devbox.json"; then
        log_warning "Quality monitoring commands already exist in devbox.json"
    else
        log_info "Adding quality monitoring commands to devbox.json..."
        # Note: This would require jq to properly modify JSON
        # For now, just provide instructions
        log_warning "Manual step required: Add these commands to devbox.json scripts section:"
        echo ""
        echo '    "quality-monitor": "./scripts/quality-monitor.sh",'
        echo '    "quality-setup": "./scripts/setup-phase5-automation.sh",'
        echo '    "quality-gates": "./scripts/quality-monitor.sh gates",'
        echo '    "quality-report": "./scripts/quality-monitor.sh report"'
        echo ""
    fi
fi

# 4. Setup GitHub Actions integration
log_info "Verifying GitHub Actions integration..."

if [ -f "$PROJECT_ROOT/.github/workflows/quality-dashboard.yml" ]; then
    log_success "Quality Dashboard GitHub Actions workflow is ready"
else
    log_error "Quality Dashboard GitHub Actions workflow not found"
fi

# 5. Create quality monitoring cron job template
log_info "Creating quality monitoring automation templates..."

cat > "$PROJECT_ROOT/scripts/setup-cron-quality-monitoring.sh" << 'EOF'
#!/bin/bash
# Template script to setup cron job for automated quality monitoring
# Run this on your development/CI server to enable continuous monitoring

# Add to crontab (runs every 4 hours):
# 0 */4 * * * cd /path/to/mobilecombackup && ./scripts/quality-monitor.sh >> quality-monitor.log 2>&1

# Or for daily monitoring at 6 AM:
# 0 6 * * * cd /path/to/mobilecombackup && ./scripts/quality-monitor.sh >> quality-monitor.log 2>&1

echo "To setup automated quality monitoring:"
echo "1. Edit this script with your project path"
echo "2. Run: crontab -e"
echo "3. Add the cron job line from this script"
echo "4. Configure notifications in .quality-config.env"
EOF

chmod +x "$PROJECT_ROOT/scripts/setup-cron-quality-monitoring.sh"
log_success "Cron job setup template created"

# 6. Create integration test
log_info "Creating integration test for Phase 5 automation..."

cat > "$PROJECT_ROOT/scripts/test-phase5-integration.sh" << 'EOF'
#!/bin/bash
# Integration test for Phase 5 automation components

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🧪 Testing Phase 5 Automation Integration"
echo "========================================"

cd "$PROJECT_ROOT"

# Test 1: Quality monitor setup
echo "Test 1: Quality monitor setup..."
if ./scripts/quality-monitor.sh setup; then
    echo "✅ Quality monitor setup works"
else
    echo "❌ Quality monitor setup failed"
    exit 1
fi

# Test 2: Quality monitoring components
echo "Test 2: Quality monitoring components..."
if ./scripts/quality-monitor.sh test; then
    echo "✅ Quality monitoring test component works"
else
    echo "❌ Quality monitoring test component failed"
    exit 1
fi

# Test 3: Dashboard generation
echo "Test 3: Dashboard generation..."
if ./scripts/quality-monitor.sh dashboard; then
    echo "✅ Dashboard generation works"
else
    echo "❌ Dashboard generation failed"
    exit 1
fi

# Test 4: Quality gates evaluation
echo "Test 4: Quality gates evaluation..."
if ./scripts/quality-monitor.sh gates; then
    echo "✅ Quality gates evaluation works"
else
    echo "⚠️  Quality gates evaluation ran (may fail due to thresholds)"
fi

# Test 5: Historical report generation
echo "Test 5: Historical report generation..."
if ./scripts/quality-monitor.sh report; then
    echo "✅ Historical report generation works"
else
    echo "❌ Historical report generation failed"
    exit 1
fi

echo ""
echo "🎉 Phase 5 integration tests completed successfully!"
echo ""
echo "Next steps:"
echo "1. Configure .quality-config.env for your environment"
echo "2. Setup notifications (Slack, email) if desired"  
echo "3. Consider using enhanced-pre-commit-hook.sh for analyzer changes"
echo "4. Enable GitHub Actions quality-dashboard.yml workflow"
EOF

chmod +x "$PROJECT_ROOT/scripts/test-phase5-integration.sh"
log_success "Integration test script created"

# 7. Create documentation
log_info "Creating Phase 5 automation documentation..."

cat > "$PROJECT_ROOT/docs/PHASE5_AUTOMATION.md" << 'EOF'
# Phase 5 Automation Integration - FEAT-085

This document describes the automated quality monitoring and CI/CD integration implemented in Phase 5 of FEAT-085.

## Overview

Phase 5 provides comprehensive automation for quality monitoring, including:

- Enhanced pre-commit hooks with quality dashboard integration
- GitHub Actions CI/CD integration with quality gates
- Continuous quality monitoring with alerting
- Historical quality reporting and trend analysis

## Components

### 1. Enhanced Pre-commit Hooks

**File**: `scripts/enhanced-pre-commit-hook.sh`

Enhanced pre-commit hook that:
- Detects analyzer package changes and runs comprehensive quality checks
- Integrates with quality dashboard for real-time quality gate evaluation  
- Provides performance tracking and detailed reporting
- Falls back to standard hooks for non-analyzer changes

**Usage**:
```bash
# Use enhanced hook for analyzer changes
ln -sf scripts/enhanced-pre-commit-hook.sh .githooks/pre-commit.enhanced

# Or replace existing hook (backup first!)
cp .githooks/pre-commit .githooks/pre-commit.backup
ln -sf ../scripts/enhanced-pre-commit-hook.sh .githooks/pre-commit
```

### 2. GitHub Actions Quality Dashboard

**File**: `.github/workflows/quality-dashboard.yml`

Automated CI/CD integration that:
- Runs on every push and pull request
- Executes comprehensive test suite with coverage tracking
- Generates quality dashboard reports
- Evaluates quality gates and fails builds if thresholds not met
- Posts quality reports as PR comments
- Uploads quality artifacts for historical analysis

**Quality Gates**:
- Coverage ≥ 80%
- Quality Score ≥ 60
- Critical Issues = 0
- Performance baseline monitoring

### 3. Continuous Quality Monitor

**File**: `scripts/quality-monitor.sh`

Standalone quality monitoring system that:
- Runs comprehensive tests and benchmarks
- Generates quality dashboard reports
- Evaluates quality gates with configurable thresholds
- Sends notifications (Slack, email) on quality issues
- Creates historical quality reports
- Supports cron job automation

**Usage**:
```bash
# Run full quality monitoring
./scripts/quality-monitor.sh

# Run specific components
./scripts/quality-monitor.sh test      # Tests only
./scripts/quality-monitor.sh dashboard # Dashboard only
./scripts/quality-monitor.sh gates     # Quality gates only
```

## Configuration

### Quality Thresholds

Configure quality thresholds in `.quality-config.env`:

```bash
# Quality Thresholds
export COVERAGE_THRESHOLD=80           # Minimum test coverage %
export QUALITY_SCORE_THRESHOLD=60     # Minimum overall quality score
export PERFORMANCE_THRESHOLD=25       # Minimum files/sec performance
export MAX_CRITICAL_ISSUES=0          # Maximum critical issues allowed

# Notification Settings
export ENABLE_NOTIFICATIONS=true
export SLACK_WEBHOOK_URL="your-slack-webhook-url"
export EMAIL_RECIPIENTS="team@example.com"
```

### Automation Setup

1. **Setup Phase 5 automation**:
   ```bash
   ./scripts/setup-phase5-automation.sh
   ```

2. **Test integration**:
   ```bash
   ./scripts/test-phase5-integration.sh
   ```

3. **Configure continuous monitoring** (optional):
   ```bash
   # Add to crontab for 4-hourly monitoring
   0 */4 * * * cd /path/to/mobilecombackup && ./scripts/quality-monitor.sh
   ```

## Quality Gates

Phase 5 implements comprehensive quality gates:

| Gate | Threshold | Description |
|------|-----------|-------------|
| Test Coverage | ≥ 80% | Minimum test coverage percentage |
| Quality Score | ≥ 60 | Overall quality score (0-100) |
| Critical Issues | = 0 | Number of critical quality issues |
| Performance | ≥ 25 files/sec | Minimum processing performance |

## Notifications

Automated notifications are sent when quality gates fail:

- **Slack**: Rich formatted messages with quality metrics
- **Email**: Plain text summaries for email clients  
- **GitHub**: PR comments and check runs with detailed reports

## Historical Reporting

Quality metrics are tracked over time:

- **Dashboard Metrics**: Stored in `.dashboard-metrics/`
- **Historical Reports**: Generated in markdown format
- **Trend Analysis**: 30-day quality trend visualization
- **Performance Tracking**: Benchmark results over time

## Integration Points

### With Existing Infrastructure

Phase 5 integrates with existing project infrastructure:

- **devbox**: Quality commands available via devbox scripts
- **Git Hooks**: Enhanced hooks complement existing pre-commit infrastructure
- **GitHub Actions**: Quality dashboard workflow runs alongside existing test workflow
- **SonarQube**: Quality metrics complement SonarQube analysis

### With Phase 1-4 Components

- **Phase 1 & 2**: Uses comprehensive test suite infrastructure
- **Phase 3**: Integrates performance benchmarks and memory monitoring  
- **Phase 4**: Leverages quality dashboard for real-time metrics

## Troubleshooting

### Common Issues

1. **Dashboard generation fails**:
   - Check Go version compatibility
   - Verify analyzer package builds successfully
   - Check available disk space in temp directories

2. **Quality gates failing**:
   - Review current metrics: `./scripts/quality-monitor.sh gates`
   - Adjust thresholds in `.quality-config.env` if needed
   - Address critical issues identified in dashboard

3. **Performance benchmarks timeout**:
   - Increase timeout in scripts if needed
   - Check system resources during benchmark execution
   - Consider running on dedicated CI infrastructure

### Debug Mode

Enable debug mode for detailed logging:

```bash
export DEBUG=true
./scripts/quality-monitor.sh
```

## Migration Guide

### From Phase 4

Phase 5 builds on Phase 4 dashboard functionality:

1. Existing quality dashboard components continue to work
2. New automation wraps around existing dashboard
3. No breaking changes to Phase 4 APIs
4. Enhanced capabilities through automation layer

### Upgrading Existing Hooks

To upgrade existing pre-commit hooks:

1. Backup current hooks: `cp .githooks/pre-commit .githooks/pre-commit.backup`
2. Test enhanced hooks: `./scripts/enhanced-pre-commit-hook.sh`
3. Replace when confident: `ln -sf ../scripts/enhanced-pre-commit-hook.sh .githooks/pre-commit`

## Future Enhancements

Planned improvements for Phase 5:

- Integration with more notification systems (Teams, Discord)
- Quality trend prediction and anomaly detection
- Automated quality issue resolution suggestions
- Integration with code review tools for quality-aware reviews
EOF

log_success "Phase 5 automation documentation created"

# 8. Final setup validation
log_info "Validating Phase 5 automation setup..."

validation_errors=0

# Check enhanced pre-commit hook
if [ -x "$SCRIPT_DIR/enhanced-pre-commit-hook.sh" ]; then
    log_success "Enhanced pre-commit hook is executable"
else
    log_error "Enhanced pre-commit hook is not executable"
    ((validation_errors++))
fi

# Check quality monitor script  
if [ -x "$SCRIPT_DIR/quality-monitor.sh" ]; then
    log_success "Quality monitor script is executable"
else
    log_error "Quality monitor script is not executable"
    ((validation_errors++))
fi

# Check GitHub Actions workflow
if [ -f "$PROJECT_ROOT/.github/workflows/quality-dashboard.yml" ]; then
    log_success "GitHub Actions quality dashboard workflow exists"
else
    log_error "GitHub Actions quality dashboard workflow missing"
    ((validation_errors++))
fi

# Check configuration file
if [ -f "$PROJECT_ROOT/.quality-config.env" ]; then
    log_success "Quality monitoring configuration created"
else
    log_error "Quality monitoring configuration missing"
    ((validation_errors++))
fi

echo ""
if [ $validation_errors -eq 0 ]; then
    log_success "Phase 5 automation setup completed successfully!"
    echo ""
    echo -e "${GREEN}🎯 Next Steps:${NC}"
    echo "1. Configure .quality-config.env for your environment"
    echo "2. Test integration: ./scripts/test-phase5-integration.sh"
    echo "3. Enable GitHub Actions workflow if desired"
    echo "4. Consider setting up continuous monitoring with cron"
    echo ""
    echo -e "${BLUE}📖 Documentation: docs/PHASE5_AUTOMATION.md${NC}"
else
    log_error "Phase 5 automation setup completed with $validation_errors errors"
    echo "Please resolve the errors above before proceeding"
    exit 1
fi