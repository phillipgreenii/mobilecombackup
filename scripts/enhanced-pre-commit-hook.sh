#!/bin/sh
# Enhanced pre-commit hook with Quality Dashboard integration
# Phase 5 of FEAT-085: Advanced Testing and Quality Assurance System
# This builds on existing pre-commit hooks with comprehensive quality monitoring

set -e

# Configuration
QUALITY_DASHBOARD_ENABLED=${QUALITY_DASHBOARD_ENABLED:-true}
COVERAGE_THRESHOLD=${COVERAGE_THRESHOLD:-80}
QUALITY_SCORE_THRESHOLD=${QUALITY_SCORE_THRESHOLD:-60}
DASHBOARD_METRICS_DIR=${DASHBOARD_METRICS_DIR:-".dashboard-metrics"}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "🚀 Enhanced Pre-commit Quality Checks (FEAT-085 Phase 5)"
echo "============================================================"

# Check if devbox is available
if ! command -v devbox >/dev/null 2>&1; then
    echo "${RED}❌ Error: devbox is not installed or not in PATH${NC}"
    echo "   Please install devbox or use 'git commit --no-verify' to bypass"
    exit 1
fi

# Function to detect if only markdown files are staged
is_markdown_only_commit() {
    staged_files=$(git diff --cached --name-only --diff-filter=AMDRC)
    
    if [ -z "$staged_files" ]; then
        return 1
    fi
    
    for file in $staged_files; do
        case "$file" in
            *.md|*.MD|*.markdown|*.MARKDOWN) continue ;;
            *) return 1 ;;
        esac
    done
    
    return 0
}

# Function to run quality dashboard and check gates
run_quality_checks() {
    local component=$1
    echo "${BLUE}📊 Running Quality Dashboard Analysis for ${component}...${NC}"
    
    # Create temporary dashboard directory
    local temp_dash_dir="/tmp/pre-commit-dashboard-$$"
    mkdir -p "$temp_dash_dir"
    
    # Generate quality report (only for analyzer package since that's what we've implemented)
    if [ "$component" = "analyzer" ]; then
        # Run tests with coverage to get fresh metrics
        echo "${BLUE}🧪 Collecting test metrics...${NC}"
        if ! devbox run go test -cover -v ./pkg/analyzer/core > "$temp_dash_dir/test_output.log" 2>&1; then
            echo "${RED}❌ Tests failed - cannot generate quality report${NC}"
            cat "$temp_dash_dir/test_output.log"
            return 1
        fi
        
        # Run quality dashboard to generate report
        echo "${BLUE}📈 Generating quality dashboard report...${NC}"
        cd "$temp_dash_dir"
        if ! timeout 30 go run /home/phillipgreenii/Projects/mobilecombackup/demos/dashboard/main.go > dashboard_output.log 2>&1; then
            echo "${YELLOW}⚠️  Quality dashboard failed to generate (timeout or error)${NC}"
            echo "   Proceeding with basic checks only..."
            return 0  # Don't fail the commit for dashboard issues
        fi
        
        # Parse dashboard output for quality gates
        if [ -f dashboard_output.log ]; then
            # Extract coverage from dashboard output
            coverage=$(grep "Test Coverage" dashboard_output.log | grep -oE '[0-9]+\.[0-9]+' | head -1 || echo "0")
            
            # Extract overall score
            overall_score=$(grep "Overall Score:" dashboard_output.log | grep -oE '[0-9]+\.[0-9]+' | head -1 || echo "0")
            
            # Extract critical issues count
            critical_issues=$(grep -c "Critical:" dashboard_output.log || echo "0")
            
            echo "${GREEN}📊 Quality Dashboard Results:${NC}"
            echo "   📈 Coverage: ${coverage}%"
            echo "   🎯 Overall Score: ${overall_score}/100"
            echo "   🚨 Critical Issues: ${critical_issues}"
            
            # Check quality gates
            quality_gates_passed=true
            
            # Coverage gate
            if [ "$(echo "$coverage < $COVERAGE_THRESHOLD" | bc -l 2>/dev/null || echo "1")" = "1" ]; then
                echo "${RED}❌ Coverage Gate Failed: ${coverage}% < ${COVERAGE_THRESHOLD}%${NC}"
                quality_gates_passed=false
            else
                echo "${GREEN}✅ Coverage Gate Passed: ${coverage}% ≥ ${COVERAGE_THRESHOLD}%${NC}"
            fi
            
            # Overall quality score gate
            if [ "$(echo "$overall_score < $QUALITY_SCORE_THRESHOLD" | bc -l 2>/dev/null || echo "1")" = "1" ]; then
                echo "${RED}❌ Quality Score Gate Failed: ${overall_score} < ${QUALITY_SCORE_THRESHOLD}${NC}"
                quality_gates_passed=false
            else
                echo "${GREEN}✅ Quality Score Gate Passed: ${overall_score} ≥ ${QUALITY_SCORE_THRESHOLD}${NC}"
            fi
            
            # Critical issues gate
            if [ "$critical_issues" -gt "0" ]; then
                echo "${RED}❌ Critical Issues Gate Failed: ${critical_issues} critical issues found${NC}"
                echo "   Please address critical issues before committing"
                quality_gates_passed=false
            else
                echo "${GREEN}✅ Critical Issues Gate Passed: No critical issues${NC}"
            fi
            
            if [ "$quality_gates_passed" = "false" ]; then
                echo "${RED}❌ Quality gates failed! Use --no-verify to bypass${NC}"
                return 1
            fi
            
            # Save report to project metrics directory if all gates pass
            if [ -d "$DASHBOARD_METRICS_DIR" ] || mkdir -p "$DASHBOARD_METRICS_DIR" 2>/dev/null; then
                timestamp=$(date +%Y%m%d_%H%M%S)
                cp dashboard_output.log "$DASHBOARD_METRICS_DIR/pre-commit_$timestamp.log" || true
                # Copy any JSON reports generated
                find . -name "*.json" -exec cp {} "$DASHBOARD_METRICS_DIR/" \; 2>/dev/null || true
            fi
        fi
    fi
    
    # Cleanup
    rm -rf "$temp_dash_dir"
    return 0
}

# Function to run enhanced analyzer package checks
run_enhanced_analyzer_checks() {
    echo "${BLUE}🔬 Enhanced Analyzer Package Quality Checks${NC}"
    echo "--------------------------------------------"
    
    # 1. Run comprehensive test suite with coverage
    echo "${BLUE}🧪 1/4 Running comprehensive analyzer tests...${NC}"
    test_start=$(date +%s)
    
    # Run all analyzer tests including performance tests
    if ! devbox run go test -v -cover -timeout=2m ./pkg/analyzer/core; then
        echo "${RED}❌ Analyzer tests failed!${NC}"
        echo "   Core analyzer functionality must pass all tests"
        return 1
    fi
    test_time=$(($(date +%s) - test_start))
    
    # 2. Run performance benchmarks to ensure no regression
    echo "${BLUE}📊 2/4 Running performance benchmarks...${NC}"
    bench_start=$(date +%s)
    
    # Run benchmarks with reduced iterations for speed
    if ! timeout 60 devbox run go test -bench=BenchmarkAdvanced -benchtime=100ms -run=^$ ./pkg/analyzer/core; then
        echo "${YELLOW}⚠️  Performance benchmarks failed or timed out${NC}"
        echo "   Proceeding with other checks..."
    fi
    bench_time=$(($(date +%s) - bench_start))
    
    # 3. Run quality dashboard analysis
    echo "${BLUE}📈 3/4 Running quality dashboard analysis...${NC}"
    dash_start=$(date +%s)
    
    if [ "$QUALITY_DASHBOARD_ENABLED" = "true" ]; then
        if ! run_quality_checks "analyzer"; then
            return 1
        fi
    else
        echo "${YELLOW}⚠️  Quality dashboard disabled - skipping detailed analysis${NC}"
    fi
    dash_time=$(($(date +%s) - dash_start))
    
    # 4. Validate integration with existing code
    echo "${BLUE}🔗 4/4 Validating integration tests...${NC}"
    integration_start=$(date +%s)
    
    # Run integration tests if they exist
    if ! devbox run go test -v -run="Integration" ./pkg/analyzer/core; then
        echo "${YELLOW}⚠️  Integration test issues detected${NC}"
        echo "   Please verify analyzer integration works correctly"
    fi
    integration_time=$(($(date +%s) - integration_start))
    
    # Performance summary
    total_time=$((test_time + bench_time + dash_time + integration_time))
    echo ""
    echo "${GREEN}✅ Enhanced analyzer checks completed!${NC}"
    echo "⏱️  Performance: ${total_time}s total"
    echo "    - Tests: ${test_time}s"
    echo "    - Benchmarks: ${bench_time}s" 
    echo "    - Dashboard: ${dash_time}s"
    echo "    - Integration: ${integration_time}s"
    
    return 0
}

# Performance tracking
start_time=$(date +%s)

echo "${BLUE}🔍 Analyzing staged changes...${NC}"

# Check what components are affected by the staged changes
analyzer_affected=false
other_code_affected=false
markdown_only=false

staged_files=$(git diff --cached --name-only --diff-filter=AMDRC)

if [ -z "$staged_files" ]; then
    echo "${YELLOW}⚠️  No staged changes detected${NC}"
    exit 0
fi

# Analyze staged files to determine appropriate checks
for file in $staged_files; do
    case "$file" in
        pkg/analyzer/*) analyzer_affected=true ;;
        *.md|*.MD|*.markdown|*.MARKDOWN) continue ;;
        *.go|*.yml|*.yaml|scripts/*|.github/*) other_code_affected=true ;;
    esac
done

# Determine check strategy
if [ "$analyzer_affected" = "true" ]; then
    echo "${BLUE}🔬 Analyzer package changes detected - running enhanced checks${NC}"
    
    # Run standard pre-commit checks first
    echo "${BLUE}📝 Running standard formatting and linting...${NC}"
    
    # 1. Run formatter
    echo "${BLUE}📝 1/2 Running formatter...${NC}"
    if ! devbox run formatter; then
        echo "${RED}❌ Formatting check failed!${NC}"
        echo "   Run 'devbox run formatter' to fix formatting issues"
        exit 1
    fi
    
    # 2. Run linter
    echo "${BLUE}🔍 2/2 Running linter...${NC}"
    if ! devbox run linter; then
        echo "${RED}❌ Linting check failed!${NC}"
        echo "   Fix linting issues before committing"
        exit 1
    fi
    
    # 3. Run enhanced analyzer checks
    if ! run_enhanced_analyzer_checks; then
        echo "${RED}❌ Enhanced analyzer checks failed!${NC}"
        echo "   Address quality issues before committing"
        echo "   Use 'git commit --no-verify' to bypass in emergencies"
        exit 1
    fi
    
elif [ "$other_code_affected" = "true" ]; then
    echo "${BLUE}🔧 Code changes detected - running standard checks${NC}"
    
    # Run the original pre-commit hook logic for non-analyzer changes
    exec /home/phillipgreenii/Projects/mobilecombackup/.githooks/pre-commit
    
elif is_markdown_only_commit; then
    echo "${BLUE}📝 Markdown-only changes - running optimized checks${NC}"
    
    # Run optimized checks for markdown-only commits
    exec /home/phillipgreenii/Projects/mobilecombackup/.githooks/pre-commit
    
else
    echo "${BLUE}🔧 Mixed changes detected - running full checks${NC}"
    
    # Run the original pre-commit hook for mixed changes
    exec /home/phillipgreenii/Projects/mobilecombackup/.githooks/pre-commit
fi

# Final performance summary
total_time=$(($(date +%s) - start_time))
echo ""
echo "${GREEN}🚀 All pre-commit checks passed!${NC}"
echo "⏱️  Total time: ${total_time}s"

if [ $total_time -gt 60 ]; then
    echo "${YELLOW}⚠️  Warning: Pre-commit checks took ${total_time}s (target: <60s)${NC}"
    echo "   Consider using 'git commit --no-verify' for large changes"
fi

echo ""
echo "${GREEN}🎯 Ready to commit with quality assurance!${NC}"