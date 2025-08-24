package progress

import (
	"strings"
	"testing"
	"time"
)

func TestNewEscalationManager(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	if em == nil {
		t.Fatal("NewEscalationManager returned nil")
	}

	if em.tracker != tracker {
		t.Error("tracker not set correctly")
	}

	if em.blockedThreshold != 15*time.Minute {
		t.Errorf("blockedThreshold = %v, want 15m", em.blockedThreshold)
	}

	if em.statusConfig == nil {
		t.Error("statusConfig should be initialized")
	}
}

func TestEscalationManager_CheckForEscalations(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	// Add a blocked task
	task := &EnhancedTodo{
		ID:       "blocked-task",
		Content:  "Blocked task",
		Status:   StatusBlocked,
		Priority: PriorityHigh,
	}

	// Set start time to 20 minutes ago (exceeds 15 minute threshold)
	blockTime := time.Now().Add(-20 * time.Minute)
	task.StartedAt = &blockTime
	reason := "test blocking reason"
	task.BlockedReason = &reason

	err := tracker.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Check for escalations
	escalations := em.CheckForEscalations()

	if len(escalations) != 1 {
		t.Errorf("Expected 1 escalation, got %d", len(escalations))
	}

	if len(escalations) > 0 {
		escalation := escalations[0]
		if escalation.TaskID != "blocked-task" {
			t.Errorf("Escalation TaskID = %v, want 'blocked-task'", escalation.TaskID)
		}

		if escalation.EventType != "escalated" {
			t.Errorf("Escalation EventType = %v, want 'escalated'", escalation.EventType)
		}

		if escalation.Severity == "" {
			t.Error("Escalation should have severity set")
		}

		// Should be high severity due to PriorityHigh and >15min duration
		if escalation.Severity != "high" && escalation.Severity != "critical" {
			t.Errorf("Expected high/critical severity, got %v", escalation.Severity)
		}
	}

	// Check that second call doesn't re-escalate the same task
	escalations2 := em.CheckForEscalations()
	if len(escalations2) != 0 {
		t.Errorf("Expected no new escalations, got %d", len(escalations2))
	}
}

func TestEscalationManager_CalculateSeverity(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	tests := []struct {
		priority      PriorityLevel
		blockedTime   time.Duration
		expectedLevel string
	}{
		{PriorityCritical, 5 * time.Minute, "high"},
		{PriorityCritical, 15 * time.Minute, "critical"},
		{PriorityHigh, 10 * time.Minute, "medium"},
		{PriorityHigh, 20 * time.Minute, "high"},
		{PriorityHigh, 45 * time.Minute, "critical"},
		{PriorityMedium, 20 * time.Minute, "low"},
		{PriorityMedium, 45 * time.Minute, "medium"},
		{PriorityMedium, 90 * time.Minute, "high"},
		{PriorityLow, 90 * time.Minute, "low"},
		{PriorityLow, 150 * time.Minute, "medium"},
	}

	for _, tt := range tests {
		t.Run(string(tt.priority)+"_"+tt.blockedTime.String(), func(t *testing.T) {
			task := EnhancedTodo{
				Priority: tt.priority,
				Status:   StatusBlocked,
			}

			// Set blocked time
			startTime := time.Now().Add(-tt.blockedTime)
			task.StartedAt = &startTime

			severity := em.calculateSeverity(task)
			if severity != tt.expectedLevel {
				t.Errorf("calculateSeverity() = %v, want %v", severity, tt.expectedLevel)
			}
		})
	}
}

func TestEscalationManager_GenerateEscalationAlert(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	// Add a blocked task
	task := &EnhancedTodo{
		ID:       "test-task",
		Content:  "Test blocked task",
		Status:   StatusBlocked,
		Priority: PriorityHigh,
	}

	blockTime := time.Now().Add(-30 * time.Minute)
	task.StartedAt = &blockTime
	reason := "waiting for external service"
	task.BlockedReason = &reason

	err := tracker.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	escalation := EscalationEvent{
		TaskID:    "test-task",
		EventType: "escalated",
		Duration:  30 * time.Minute,
		Reason:    reason,
		Severity:  "high",
	}

	alert := em.GenerateEscalationAlert(escalation)

	// Check that alert contains expected information
	expectedContent := []string{
		"BLOCKED TASK ALERT",
		"Test blocked task",
		"high",
		"30m",
		"waiting for external service",
		"Suggested Actions",
	}

	for _, content := range expectedContent {
		if !strings.Contains(alert, content) {
			t.Errorf("Alert should contain '%s', got:\n%s", content, alert)
		}
	}
}

func TestEscalationManager_GenerateResolutionSuggestions(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	// Test with high priority task with dependencies
	task := &EnhancedTodo{
		ID:           "task-with-deps",
		Content:      "Task with dependencies",
		Priority:     PriorityHigh,
		Complexity:   ComplexityComplex,
		Dependencies: []string{"dep1", "dep2"},
	}

	escalation := EscalationEvent{
		Duration: 2 * time.Hour,
		Severity: "high",
	}

	suggestions := em.generateResolutionSuggestions(task, escalation)

	// Check for expected suggestion types
	suggestionText := strings.Join(suggestions, " ")

	expectedSuggestions := []string{
		"dependencies", // Should mention dependencies
		"escalate",     // High priority should suggest escalation
		"complex",      // Complex task should suggest breaking down
		"1 hour",       // Long duration should suggest context switching
	}

	for _, expected := range expectedSuggestions {
		if !strings.Contains(strings.ToLower(suggestionText), expected) {
			t.Errorf("Suggestions should contain reference to '%s', got: %v", expected, suggestions)
		}
	}

	if len(suggestions) < 3 {
		t.Errorf("Expected at least 3 suggestions, got %d", len(suggestions))
	}
}

func TestEscalationManager_GenerateStatusReport(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	// Add various tasks to create interesting status
	now := time.Now()

	// Completed task
	completedTask := &EnhancedTodo{
		ID:      "completed",
		Content: "Completed task",
		Status:  StatusCompleted,
	}
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(-30 * time.Minute)
	completedTask.StartedAt = &startTime
	completedTask.CompletedAt = &endTime
	err := tracker.AddTask(completedTask)
	if err != nil {
		t.Fatalf("Failed to add completed task: %v", err)
	}

	// In-progress task
	inProgressTask := &EnhancedTodo{
		ID:         "in-progress",
		Content:    "Current working task",
		Status:     StatusInProgress,
		Priority:   PriorityHigh,
		Complexity: ComplexityMedium,
	}
	workStartTime := now.Add(-20 * time.Minute)
	inProgressTask.StartedAt = &workStartTime
	tracker.AddTask(inProgressTask)

	// Blocked task
	blockedTask := &EnhancedTodo{
		ID:      "blocked",
		Content: "Blocked task needing attention",
		Status:  StatusBlocked,
	}
	blockedReason := "waiting for review"
	blockedTask.BlockedReason = &blockedReason
	tracker.AddTask(blockedTask)

	// Pending task
	tracker.AddTask(&EnhancedTodo{
		ID:       "pending",
		Content:  "Next task to do",
		Status:   StatusPending,
		Priority: PriorityMedium,
	})

	// Generate status report
	report := em.GenerateStatusReport()

	// Check that report contains expected sections
	expectedSections := []string{
		"TASK PROGRESS STATUS",
		"Overall Progress:",
		"CURRENT TASK",
		"Current working task", // Current task content
		"TASK STATUS",
		"Completed: 1",
		"In Progress: 1",
		"Pending: 1",
		"Blocked: 1",
		"BLOCKED TASKS",
		"Blocked task needing attention",
		"waiting for review",
		"NEXT AVAILABLE TASKS",
	}

	for _, section := range expectedSections {
		if !strings.Contains(report, section) {
			t.Errorf("Status report should contain '%s', got:\n%s", section, report)
		}
	}

	// Test compact mode
	em.statusConfig.CompactMode = true
	compactReport := em.GenerateStatusReport()

	if strings.Contains(compactReport, "\n") {
		t.Error("Compact report should be single line")
	}

	if !strings.Contains(compactReport, "Progress:") {
		t.Error("Compact report should contain progress information")
	}
}

func TestEscalationManager_ResolveEscalation(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	// First, escalate a task
	task := &EnhancedTodo{
		ID:     "test-task",
		Status: StatusBlocked,
	}
	blockTime := time.Now().Add(-30 * time.Minute)
	task.StartedAt = &blockTime
	err := tracker.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	em.CheckForEscalations() // This should create an escalation

	// Now resolve it
	em.ResolveEscalation("test-task", "Issue was resolved by restarting service")

	// Check escalation history
	history := em.GetEscalationHistory()

	resolvedFound := false
	for _, event := range history {
		if event.TaskID == "test-task" && event.EventType == "resolved" {
			resolvedFound = true
			if event.Resolution != "Issue was resolved by restarting service" {
				t.Errorf("Resolution = %v, want 'Issue was resolved by restarting service'", event.Resolution)
			}
		}
	}

	if !resolvedFound {
		t.Error("Resolved escalation event not found in history")
	}
}

func TestEscalationManager_GetEscalationSummary(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	// Create some escalation events
	em.escalationLog = []EscalationEvent{
		{TaskID: "task1", EventType: "escalated", Severity: "high"},
		{TaskID: "task2", EventType: "escalated", Severity: "medium"},
		{TaskID: "task3", EventType: "escalated", Severity: "high"},
		{TaskID: "task1", EventType: "resolved"},
		{TaskID: "task2", EventType: "resolved"},
	}

	summary := em.GetEscalationSummary()

	// Check summary contents
	if totalEscalations, ok := summary["total_escalations"].(int); !ok || totalEscalations != 3 {
		t.Errorf("total_escalations = %v, want 3", summary["total_escalations"])
	}

	if resolvedCount, ok := summary["resolved_count"].(int); !ok || resolvedCount != 2 {
		t.Errorf("resolved_count = %v, want 2", summary["resolved_count"])
	}

	if resolutionRate, ok := summary["resolution_rate"].(float64); !ok || resolutionRate < 66 || resolutionRate > 67 {
		t.Errorf("resolution_rate = %v, want ~66.67", summary["resolution_rate"])
	}

	severityBreakdown, ok := summary["severity_breakdown"].(map[string]int)
	if !ok {
		t.Error("severity_breakdown should be map[string]int")
	} else {
		if severityBreakdown["high"] != 2 {
			t.Errorf("high severity count = %d, want 2", severityBreakdown["high"])
		}
		if severityBreakdown["medium"] != 1 {
			t.Errorf("medium severity count = %d, want 1", severityBreakdown["medium"])
		}
	}
}

func TestGetDefaultStatusConfig(t *testing.T) {
	t.Parallel()

	config := GetDefaultStatusConfig()

	if config == nil {
		t.Fatal("GetDefaultStatusConfig returned nil")
	}

	// Check default values
	if !config.ShowBlockedDetails {
		t.Error("ShowBlockedDetails should be true by default")
	}

	if !config.ShowVelocityMetrics {
		t.Error("ShowVelocityMetrics should be true by default")
	}

	if config.CompactMode {
		t.Error("CompactMode should be false by default")
	}
}

func TestUtilityFunctions(t *testing.T) {
	t.Parallel()

	// Test repeatString
	result := repeatString("=", 5)
	expected := "====="
	if result != expected {
		t.Errorf("repeatString('=', 5) = %v, want %v", result, expected)
	}

	// Test truncateString
	longString := "This is a very long string that should be truncated"
	truncated := truncateString(longString, 20)
	if len(truncated) > 20 {
		t.Errorf("truncateString should limit length to 20, got %d", len(truncated))
	}

	if !strings.HasSuffix(truncated, "...") {
		t.Error("truncateString should add ... suffix")
	}

	// Test with string shorter than limit
	shortString := "Short"
	notTruncated := truncateString(shortString, 20)
	if notTruncated != shortString {
		t.Errorf("truncateString should not modify short strings, got %v", notTruncated)
	}
}

func TestEscalationManager_GetRecentEscalations(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	now := time.Now()

	// Add escalations at different times
	em.escalationLog = []EscalationEvent{
		{TaskID: "old", EventType: "escalated", Timestamp: now.Add(-2 * time.Hour)},
		{TaskID: "recent1", EventType: "escalated", Timestamp: now.Add(-30 * time.Minute)},
		{TaskID: "recent2", EventType: "escalated", Timestamp: now.Add(-10 * time.Minute)},
		{TaskID: "resolved", EventType: "resolved", Timestamp: now.Add(-5 * time.Minute)},
	}

	// Get escalations from last hour
	recent := em.getRecentEscalations(1 * time.Hour)

	// Should only include recent1 and recent2 (not old or resolved)
	if len(recent) != 2 {
		t.Errorf("Expected 2 recent escalations, got %d", len(recent))
	}

	for _, escalation := range recent {
		if escalation.TaskID == "old" {
			t.Error("Should not include old escalation")
		}
		if escalation.EventType != "escalated" {
			t.Error("Should only include escalated events")
		}
	}
}
