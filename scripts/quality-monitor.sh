#!/bin/bash
# Automated Quality Monitoring System - Phase 5 of FEAT-085
# Continuous quality monitoring with alerting and reporting

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DASHBOARD_METRICS_DIR="${PROJECT_ROOT}/.dashboard-metrics"
ALERTS_DIR="${PROJECT_ROOT}/.quality-alerts"
LOG_FILE="${PROJECT_ROOT}/quality-monitor.log"

# Quality thresholds
COVERAGE_THRESHOLD=${COVERAGE_THRESHOLD:-80}
QUALITY_SCORE_THRESHOLD=${QUALITY_SCORE_THRESHOLD:-60}
PERFORMANCE_THRESHOLD=${PERFORMANCE_THRESHOLD:-25}  # files/sec minimum
MAX_CRITICAL_ISSUES=${MAX_CRITICAL_ISSUES:-0}

# Notification settings
ENABLE_NOTIFICATIONS=${ENABLE_NOTIFICATIONS:-false}
SLACK_WEBHOOK_URL=${SLACK_WEBHOOK_URL:-""}
EMAIL_RECIPIENTS=${EMAIL_RECIPIENTS:-""}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

log_message() {
    local level=$1
    local message=$2
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] [$level] $message" | tee -a "$LOG_FILE"
}

log_info() {
    log_message "INFO" "$1"
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    log_message "SUCCESS" "$1"
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    log_message "WARNING" "$1"
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    log_message "ERROR" "$1"
    echo -e "${RED}❌ $1${NC}"
}

setup_monitoring() {
    log_info "Setting up quality monitoring infrastructure..."
    
    # Create necessary directories
    mkdir -p "$DASHBOARD_METRICS_DIR"
    mkdir -p "$ALERTS_DIR"
    
    # Initialize log file
    if [ ! -f "$LOG_FILE" ]; then
        touch "$LOG_FILE"
        log_info "Created quality monitoring log file: $LOG_FILE"
    fi
    
    # Verify dependencies
    if ! command -v devbox >/dev/null 2>&1; then
        log_error "devbox is not installed or not in PATH"
        return 1
    fi
    
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed or not in PATH"
        return 1
    fi
    
    log_success "Quality monitoring setup completed"
    return 0
}

run_comprehensive_tests() {
    log_info "Running comprehensive test suite..."
    
    cd "$PROJECT_ROOT"
    
    # Run tests with coverage
    local test_output_file="/tmp/quality_test_output_$$.log"
    local coverage_file="/tmp/coverage_$$.out"
    
    if devbox run go test -v -cover -coverprofile="$coverage_file" ./pkg/analyzer/core > "$test_output_file" 2>&1; then
        log_success "Test suite completed successfully"
        
        # Extract coverage
        if [ -f "$coverage_file" ]; then
            local coverage=$(go tool cover -func="$coverage_file" | grep total | awk '{print $3}' | sed 's/%//')
            echo "$coverage" > "/tmp/coverage_percent_$$.txt"
            log_info "Current test coverage: ${coverage}%"
        fi
        
        # Parse test results
        local total_tests=$(grep -c "=== RUN" "$test_output_file" || echo "0")
        local passed_tests=$(grep -c "--- PASS:" "$test_output_file" || echo "0")
        local failed_tests=$(grep -c "--- FAIL:" "$test_output_file" || echo "0")
        
        echo "$total_tests:$passed_tests:$failed_tests" > "/tmp/test_stats_$$.txt"
        log_info "Test results: $passed_tests passed, $failed_tests failed out of $total_tests total"
        
        return 0
    else
        log_error "Test suite failed"
        cat "$test_output_file" | tail -20 >> "$LOG_FILE"
        return 1
    fi
}

run_performance_benchmarks() {
    log_info "Running performance benchmarks..."
    
    cd "$PROJECT_ROOT"
    
    local benchmark_output_file="/tmp/benchmark_output_$$.log"
    
    # Run benchmarks with timeout
    if timeout 300 devbox run go test -bench=BenchmarkAdvanced -benchmem -run=^$ ./pkg/analyzer/core > "$benchmark_output_file" 2>&1; then
        log_success "Performance benchmarks completed"
        
        # Extract performance metrics
        local files_per_sec=$(grep "files/sec" "$benchmark_output_file" | head -1 | grep -oE '[0-9]+\.[0-9]+' || echo "0")
        echo "$files_per_sec" > "/tmp/performance_$$.txt"
        log_info "Current performance: ${files_per_sec} files/sec"
        
        return 0
    else
        log_warning "Performance benchmarks failed or timed out"
        return 1
    fi
}

generate_quality_dashboard() {
    log_info "Generating quality dashboard report..."
    
    cd "$DASHBOARD_METRICS_DIR"
    
    local dashboard_output_file="quality_report_$(date +%Y%m%d_%H%M%S).log"
    local dashboard_json_file="quality_report_$(date +%Y%m%d_%H%M%S).json"
    
    # Run dashboard generator
    if timeout 120 go run "$PROJECT_ROOT/demos/dashboard/main.go" > "$dashboard_output_file" 2>&1; then
        log_success "Quality dashboard generated: $dashboard_output_file"
        
        # Extract key metrics
        local overall_score=$(grep "Overall Score:" "$dashboard_output_file" | grep -oE '[0-9]+\.[0-9]+' || echo "0")
        local critical_issues=$(grep -c "Critical:" "$dashboard_output_file" || echo "0")
        local status=$(grep "Status:" "$dashboard_output_file" | awk '{print $2}' || echo "unknown")
        
        echo "$overall_score:$critical_issues:$status" > "/tmp/dashboard_metrics_$$.txt"
        log_info "Dashboard metrics - Score: $overall_score, Critical Issues: $critical_issues, Status: $status"
        
        return 0
    else
        log_error "Quality dashboard generation failed"
        return 1
    fi
}

evaluate_quality_gates() {
    log_info "Evaluating quality gates..."
    
    local gates_passed=true
    local failed_gates=()
    local alerts=()
    
    # Load metrics
    local coverage=$(cat "/tmp/coverage_percent_$$.txt" 2>/dev/null || echo "0")
    local performance=$(cat "/tmp/performance_$$.txt" 2>/dev/null || echo "0")
    local dashboard_metrics=$(cat "/tmp/dashboard_metrics_$$.txt" 2>/dev/null || echo "0:0:unknown")
    
    IFS=':' read -r overall_score critical_issues status <<< "$dashboard_metrics"
    
    log_info "Quality Gate Evaluation:"
    log_info "  Coverage: ${coverage}% (threshold: ${COVERAGE_THRESHOLD}%)"
    log_info "  Performance: ${performance} files/sec (threshold: ${PERFORMANCE_THRESHOLD})"
    log_info "  Quality Score: ${overall_score} (threshold: ${QUALITY_SCORE_THRESHOLD})"
    log_info "  Critical Issues: ${critical_issues} (max allowed: ${MAX_CRITICAL_ISSUES})"
    
    # Coverage gate
    if (( $(echo "$coverage < $COVERAGE_THRESHOLD" | bc -l 2>/dev/null || echo "1") )); then
        gates_passed=false
        failed_gates+=("Coverage: ${coverage}% < ${COVERAGE_THRESHOLD}%")
        alerts+=("🔴 Coverage gate failed: ${coverage}% below threshold")
        log_error "Coverage gate failed: ${coverage}% < ${COVERAGE_THRESHOLD}%"
    else
        log_success "Coverage gate passed: ${coverage}% ≥ ${COVERAGE_THRESHOLD}%"
    fi
    
    # Performance gate
    if (( $(echo "$performance < $PERFORMANCE_THRESHOLD" | bc -l 2>/dev/null || echo "1") )); then
        gates_passed=false
        failed_gates+=("Performance: ${performance} files/sec < ${PERFORMANCE_THRESHOLD}")
        alerts+=("🟡 Performance gate failed: ${performance} files/sec below threshold")
        log_error "Performance gate failed: ${performance} files/sec < ${PERFORMANCE_THRESHOLD}"
    else
        log_success "Performance gate passed: ${performance} files/sec ≥ ${PERFORMANCE_THRESHOLD}"
    fi
    
    # Quality score gate
    if (( $(echo "$overall_score < $QUALITY_SCORE_THRESHOLD" | bc -l 2>/dev/null || echo "1") )); then
        gates_passed=false
        failed_gates+=("Quality Score: ${overall_score} < ${QUALITY_SCORE_THRESHOLD}")
        alerts+=("🔴 Quality score gate failed: ${overall_score} below threshold")
        log_error "Quality score gate failed: ${overall_score} < ${QUALITY_SCORE_THRESHOLD}"
    else
        log_success "Quality score gate passed: ${overall_score} ≥ ${QUALITY_SCORE_THRESHOLD}"
    fi
    
    # Critical issues gate
    if [ "$critical_issues" -gt "$MAX_CRITICAL_ISSUES" ]; then
        gates_passed=false
        failed_gates+=("Critical Issues: ${critical_issues} > ${MAX_CRITICAL_ISSUES}")
        alerts+=("🚨 Critical issues detected: ${critical_issues} issues found")
        log_error "Critical issues gate failed: ${critical_issues} > ${MAX_CRITICAL_ISSUES}"
    else
        log_success "Critical issues gate passed: ${critical_issues} ≤ ${MAX_CRITICAL_ISSUES}"
    fi
    
    # Save evaluation results
    echo "$gates_passed" > "/tmp/gates_passed_$$.txt"
    if [ ${#failed_gates[@]} -gt 0 ]; then
        printf '%s\n' "${failed_gates[@]}" > "/tmp/failed_gates_$$.txt"
    fi
    if [ ${#alerts[@]} -gt 0 ]; then
        printf '%s\n' "${alerts[@]}" > "/tmp/alerts_$$.txt"
    fi
    
    if [ "$gates_passed" = "true" ]; then
        log_success "All quality gates passed!"
        return 0
    else
        log_error "Quality gates failed: ${#failed_gates[@]} gates failed"
        return 1
    fi
}

send_notifications() {
    local gates_passed=$1
    
    if [ "$ENABLE_NOTIFICATIONS" != "true" ]; then
        log_info "Notifications disabled - skipping"
        return 0
    fi
    
    log_info "Sending quality monitoring notifications..."
    
    # Load metrics for notification
    local coverage=$(cat "/tmp/coverage_percent_$$.txt" 2>/dev/null || echo "0")
    local performance=$(cat "/tmp/performance_$$.txt" 2>/dev/null || echo "0")
    local dashboard_metrics=$(cat "/tmp/dashboard_metrics_$$.txt" 2>/dev/null || echo "0:0:unknown")
    IFS=':' read -r overall_score critical_issues status <<< "$dashboard_metrics"
    
    local notification_title
    local notification_color
    local notification_icon
    
    if [ "$gates_passed" = "true" ]; then
        notification_title="✅ Quality Gates Passed"
        notification_color="good"
        notification_icon=":white_check_mark:"
    else
        notification_title="❌ Quality Gates Failed"
        notification_color="danger"
        notification_icon=":x:"
    fi
    
    # Create notification message
    local message="$notification_icon *Quality Monitoring Report*\n\n"
    message+="*Project*: mobilecombackup\n"
    message+="*Timestamp*: $(date '+%Y-%m-%d %H:%M:%S')\n"
    message+="*Overall Status*: $status ($overall_score/100)\n\n"
    message+="*Quality Metrics:*\n"
    message+="• Coverage: ${coverage}% (threshold: ${COVERAGE_THRESHOLD}%)\n"
    message+="• Performance: ${performance} files/sec (threshold: ${PERFORMANCE_THRESHOLD})\n"
    message+="• Quality Score: ${overall_score} (threshold: ${QUALITY_SCORE_THRESHOLD})\n"
    message+="• Critical Issues: ${critical_issues}\n\n"
    
    if [ "$gates_passed" != "true" ] && [ -f "/tmp/failed_gates_$$.txt" ]; then
        message+="*Failed Gates:*\n"
        while IFS= read -r gate; do
            message+="• $gate\n"
        done < "/tmp/failed_gates_$$.txt"
    fi
    
    # Send Slack notification
    if [ -n "$SLACK_WEBHOOK_URL" ]; then
        local slack_payload="{
            \"text\": \"$notification_title\",
            \"attachments\": [{
                \"color\": \"$notification_color\",
                \"text\": \"$message\",
                \"footer\": \"FEAT-085 Quality Monitor\",
                \"ts\": $(date +%s)
            }]
        }"
        
        if curl -X POST -H 'Content-type: application/json' \
                --data "$slack_payload" \
                "$SLACK_WEBHOOK_URL" > /dev/null 2>&1; then
            log_success "Slack notification sent"
        else
            log_warning "Failed to send Slack notification"
        fi
    fi
    
    # Send email notification (if mail command is available)
    if [ -n "$EMAIL_RECIPIENTS" ] && command -v mail >/dev/null 2>&1; then
        local email_subject="Quality Monitor: $notification_title"
        local email_body=$(echo -e "$message" | sed 's/\*//g' | sed 's/:.*://g')
        
        if echo "$email_body" | mail -s "$email_subject" "$EMAIL_RECIPIENTS" > /dev/null 2>&1; then
            log_success "Email notification sent to $EMAIL_RECIPIENTS"
        else
            log_warning "Failed to send email notification"
        fi
    fi
}

generate_historical_report() {
    log_info "Generating historical quality report..."
    
    local report_file="$DASHBOARD_METRICS_DIR/historical_quality_report_$(date +%Y%m%d).md"
    
    cat > "$report_file" << EOF
# Quality Monitoring Historical Report

**Generated**: $(date '+%Y-%m-%d %H:%M:%S')
**Project**: mobilecombackup
**Monitoring System**: FEAT-085 Phase 5

## Current Quality Status

EOF
    
    # Add current metrics
    if [ -f "/tmp/dashboard_metrics_$$.txt" ]; then
        local dashboard_metrics=$(cat "/tmp/dashboard_metrics_$$.txt")
        IFS=':' read -r overall_score critical_issues status <<< "$dashboard_metrics"
        
        echo "- **Overall Score**: $overall_score/100" >> "$report_file"
        echo "- **Status**: $status" >> "$report_file"
        echo "- **Critical Issues**: $critical_issues" >> "$report_file"
    fi
    
    if [ -f "/tmp/coverage_percent_$$.txt" ]; then
        local coverage=$(cat "/tmp/coverage_percent_$$.txt")
        echo "- **Test Coverage**: ${coverage}%" >> "$report_file"
    fi
    
    if [ -f "/tmp/performance_$$.txt" ]; then
        local performance=$(cat "/tmp/performance_$$.txt")
        echo "- **Performance**: ${performance} files/sec" >> "$report_file"
    fi
    
    cat >> "$report_file" << EOF

## Quality Gates Status

EOF
    
    if [ -f "/tmp/gates_passed_$$.txt" ]; then
        local gates_passed=$(cat "/tmp/gates_passed_$$.txt")
        if [ "$gates_passed" = "true" ]; then
            echo "✅ All quality gates **PASSED**" >> "$report_file"
        else
            echo "❌ Quality gates **FAILED**" >> "$report_file"
            echo "" >> "$report_file"
            echo "### Failed Gates:" >> "$report_file"
            if [ -f "/tmp/failed_gates_$$.txt" ]; then
                while IFS= read -r gate; do
                    echo "- $gate" >> "$report_file"
                done < "/tmp/failed_gates_$$.txt"
            fi
        fi
    fi
    
    echo "" >> "$report_file"
    echo "---" >> "$report_file"
    echo "*Report generated by FEAT-085 Quality Monitoring System*" >> "$report_file"
    
    log_success "Historical report generated: $report_file"
}

cleanup_temp_files() {
    rm -f /tmp/coverage_percent_$$.txt
    rm -f /tmp/performance_$$.txt
    rm -f /tmp/dashboard_metrics_$$.txt
    rm -f /tmp/gates_passed_$$.txt
    rm -f /tmp/failed_gates_$$.txt
    rm -f /tmp/alerts_$$.txt
    rm -f /tmp/quality_test_output_$$.log
    rm -f /tmp/coverage_$$.out
    rm -f /tmp/benchmark_output_$$.log
    rm -f /tmp/test_stats_$$.txt
}

main() {
    local start_time=$(date +%s)
    
    echo -e "${PURPLE}🎯 FEAT-085 Quality Monitoring System${NC}"
    echo -e "${PURPLE}=====================================${NC}"
    log_info "Starting quality monitoring run..."
    
    # Trap to ensure cleanup
    trap cleanup_temp_files EXIT
    
    # Setup monitoring infrastructure
    if ! setup_monitoring; then
        log_error "Failed to set up monitoring infrastructure"
        exit 1
    fi
    
    # Run comprehensive tests
    if ! run_comprehensive_tests; then
        log_error "Test suite failed - aborting quality monitoring"
        exit 1
    fi
    
    # Run performance benchmarks
    run_performance_benchmarks  # Don't fail if benchmarks fail
    
    # Generate quality dashboard
    if ! generate_quality_dashboard; then
        log_warning "Quality dashboard generation failed - using test results only"
    fi
    
    # Evaluate quality gates
    local gates_passed=false
    if evaluate_quality_gates; then
        gates_passed=true
    fi
    
    # Send notifications
    send_notifications "$gates_passed"
    
    # Generate historical report
    generate_historical_report
    
    # Final summary
    local total_time=$(($(date +%s) - start_time))
    echo ""
    log_info "Quality monitoring completed in ${total_time}s"
    
    if [ "$gates_passed" = "true" ]; then
        echo -e "${GREEN}🎉 Quality monitoring passed - system is healthy!${NC}"
        exit 0
    else
        echo -e "${RED}⚠️  Quality monitoring detected issues - attention needed!${NC}"
        exit 1
    fi
}

# Handle command line arguments
case "${1:-}" in
    "setup")
        setup_monitoring
        ;;
    "test")
        run_comprehensive_tests
        ;;
    "benchmark")
        run_performance_benchmarks
        ;;
    "dashboard")
        generate_quality_dashboard
        ;;
    "gates")
        evaluate_quality_gates
        ;;
    "notify")
        send_notifications "${2:-false}"
        ;;
    "report")
        generate_historical_report
        ;;
    "help"|"--help"|"-h")
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  setup      - Set up monitoring infrastructure"
        echo "  test       - Run comprehensive test suite"
        echo "  benchmark  - Run performance benchmarks"
        echo "  dashboard  - Generate quality dashboard"
        echo "  gates      - Evaluate quality gates"
        echo "  notify     - Send notifications"
        echo "  report     - Generate historical report"
        echo "  help       - Show this help"
        echo ""
        echo "If no command is specified, runs full monitoring cycle."
        ;;
    *)
        main
        ;;
esac