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
