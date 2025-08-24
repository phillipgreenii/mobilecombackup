package progress

import (
	"strings"
	"testing"
	"time"
)

func TestComplexityLevel_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		complexity ComplexityLevel
		expected   string
	}{
		{ComplexitySimple, "simple"},
		{ComplexityMedium, "medium"},
		{ComplexityComplex, "complex"},
	}

	for _, tt := range tests {
		t.Run(string(tt.complexity), func(t *testing.T) {
			if got := tt.complexity.String(); got != tt.expected {
				t.Errorf("ComplexityLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestComplexityLevel_GetEstimatedMinutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		complexity ComplexityLevel
		expected   int
	}{
		{ComplexitySimple, 15},
		{ComplexityMedium, 45},
		{ComplexityComplex, 120},
		{ComplexityLevel("unknown"), 30}, // Default case
	}

	for _, tt := range tests {
		t.Run(string(tt.complexity), func(t *testing.T) {
			if got := tt.complexity.GetEstimatedMinutes(); got != tt.expected {
				t.Errorf("ComplexityLevel.GetEstimatedMinutes() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPriorityLevel_GetWeight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		priority PriorityLevel
		expected int
	}{
		{PriorityLow, 1},
		{PriorityMedium, 2},
		{PriorityHigh, 3},
		{PriorityCritical, 4},
		{PriorityLevel("unknown"), 2}, // Default case
	}

	for _, tt := range tests {
		t.Run(string(tt.priority), func(t *testing.T) {
			if got := tt.priority.GetWeight(); got != tt.expected {
				t.Errorf("PriorityLevel.GetWeight() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTaskStatus_IsActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{StatusPending, false},
		{StatusInProgress, true},
		{StatusCompleted, false},
		{StatusBlocked, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsActive(); got != tt.expected {
				t.Errorf("TaskStatus.IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTaskStatus_IsComplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{StatusPending, false},
		{StatusInProgress, false},
		{StatusCompleted, true},
		{StatusBlocked, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsComplete(); got != tt.expected {
				t.Errorf("TaskStatus.IsComplete() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEnhancedTodo_GetEstimatedDuration(t *testing.T) {
	t.Parallel()

	// Test with explicit estimate
	explicitEstimate := 30 * time.Minute
	task1 := &EnhancedTodo{
		Estimate:   &explicitEstimate,
		Complexity: ComplexityMedium,
	}

	if got := task1.GetEstimatedDuration(); got != explicitEstimate {
		t.Errorf("GetEstimatedDuration() with explicit estimate = %v, want %v", got, explicitEstimate)
	}

	// Test with complexity-based estimate
	task2 := &EnhancedTodo{
		Complexity: ComplexityComplex,
	}

	expected := time.Duration(ComplexityComplex.GetEstimatedMinutes()) * time.Minute
	if got := task2.GetEstimatedDuration(); got != expected {
		t.Errorf("GetEstimatedDuration() with complexity = %v, want %v", got, expected)
	}
}

func TestEnhancedTodo_GetActualDuration(t *testing.T) {
	t.Parallel()

	now := time.Now()
	startTime := now.Add(-30 * time.Minute)
	completedTime := now.Add(-5 * time.Minute)

	// Test with completed task
	task1 := &EnhancedTodo{
		StartedAt:   &startTime,
		CompletedAt: &completedTime,
	}

	expected := completedTime.Sub(startTime)
	if got := task1.GetActualDuration(); got != expected {
		t.Errorf("GetActualDuration() for completed task = %v, want %v", got, expected)
	}

	// Test with in-progress task
	task2 := &EnhancedTodo{
		StartedAt: &startTime,
	}

	if got := task2.GetActualDuration(); got < 25*time.Minute || got > 35*time.Minute {
		t.Errorf("GetActualDuration() for in-progress task = %v, want ~30 minutes", got)
	}

	// Test with task not started
	task3 := &EnhancedTodo{}

	if got := task3.GetActualDuration(); got != 0 {
		t.Errorf("GetActualDuration() for non-started task = %v, want 0", got)
	}
}

func TestEnhancedTodo_IsBlocked(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{StatusPending, false},
		{StatusInProgress, false},
		{StatusCompleted, false},
		{StatusBlocked, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			task := &EnhancedTodo{Status: tt.status}
			if got := task.IsBlocked(); got != tt.expected {
				t.Errorf("IsBlocked() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEnhancedTodo_CanStart(t *testing.T) {
	t.Parallel()

	completedTasks := map[string]bool{
		"task1": true,
		"task2": true,
		"task3": false,
	}

	tests := []struct {
		name         string
		status       TaskStatus
		dependencies []string
		expected     bool
	}{
		{
			name:         "pending with no dependencies",
			status:       StatusPending,
			dependencies: []string{},
			expected:     true,
		},
		{
			name:         "pending with completed dependencies",
			status:       StatusPending,
			dependencies: []string{"task1", "task2"},
			expected:     true,
		},
		{
			name:         "pending with incomplete dependencies",
			status:       StatusPending,
			dependencies: []string{"task1", "task3"},
			expected:     false,
		},
		{
			name:         "completed task",
			status:       StatusCompleted,
			dependencies: []string{},
			expected:     false,
		},
		{
			name:         "in-progress task",
			status:       StatusInProgress,
			dependencies: []string{},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &EnhancedTodo{
				Status:       tt.status,
				Dependencies: tt.dependencies,
			}
			if got := task.CanStart(completedTasks); got != tt.expected {
				t.Errorf("CanStart() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProgressReport_String(t *testing.T) {
	t.Parallel()

	report := &ProgressReport{
		CompletedTasks:     3,
		TotalTasks:         10,
		Percentage:         30.0,
		TimeElapsed:        2 * time.Hour,
		EstimatedRemaining: 4 * time.Hour,
		Velocity:           1.5,
		BlockedTaskCount:   1,
	}

	result := report.String()

	// Check key components instead of exact string match due to duration formatting variations
	expectedComponents := []string{
		"Progress: 3/10 (30.0%)",
		"Elapsed:",
		"Remaining:",
		"Velocity: 1.5 tasks/h",
		"Blocked: 1",
	}

	for _, component := range expectedComponents {
		if !strings.Contains(result, component) {
			t.Errorf("ProgressReport.String() should contain %q, got %q", component, result)
		}
	}
}

func TestGetDefaultConfig(t *testing.T) {
	t.Parallel()

	config := GetDefaultConfig()

	// Test default values
	if config.DefaultComplexity != ComplexityMedium {
		t.Errorf("DefaultComplexity = %v, want %v", config.DefaultComplexity, ComplexityMedium)
	}

	if config.DefaultPriority != PriorityMedium {
		t.Errorf("DefaultPriority = %v, want %v", config.DefaultPriority, PriorityMedium)
	}

	// Test boolean flags
	if !config.ParseTasksSection {
		t.Error("ParseTasksSection should be true by default")
	}

	if !config.EstimateFromComplexity {
		t.Error("EstimateFromComplexity should be true by default")
	}

	// Test multipliers
	if len(config.ComplexityMultiplier) == 0 {
		t.Error("ComplexityMultiplier should not be empty")
	}

	if config.ComplexityMultiplier[ComplexitySimple] != 0.5 {
		t.Errorf("ComplexityMultiplier[Simple] = %v, want 0.5", config.ComplexityMultiplier[ComplexitySimple])
	}
}

func TestValidationResult_AddError(t *testing.T) {
	t.Parallel()

	result := &ValidationResult{IsValid: true}

	result.AddError("task1", "test error")

	if result.IsValid {
		t.Error("ValidationResult should be invalid after adding error")
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	if result.Errors[0].TaskID != "task1" {
		t.Errorf("Error TaskID = %v, want task1", result.Errors[0].TaskID)
	}

	if result.Errors[0].Message != "test error" {
		t.Errorf("Error Message = %v, want 'test error'", result.Errors[0].Message)
	}
}

func TestValidationResult_AddCircularError(t *testing.T) {
	t.Parallel()

	result := &ValidationResult{IsValid: true}
	chain := []string{"task1", "task2", "task3", "task1"}

	result.AddCircularError("task1", chain)

	if result.IsValid {
		t.Error("ValidationResult should be invalid after adding circular error")
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	err := result.Errors[0]
	if err.TaskID != "task1" {
		t.Errorf("Error TaskID = %v, want task1", err.TaskID)
	}

	if len(err.Circular) != 4 {
		t.Errorf("Circular chain length = %d, want 4", len(err.Circular))
	}
}

func TestDependencyError_Error(t *testing.T) {
	t.Parallel()

	// Test regular error
	err1 := &DependencyError{
		TaskID:  "task1",
		Message: "dependency not found",
	}

	expected1 := "task 'task1': dependency not found"
	if got := err1.Error(); got != expected1 {
		t.Errorf("DependencyError.Error() = %v, want %v", got, expected1)
	}

	// Test circular error
	err2 := &DependencyError{
		TaskID:   "task1",
		Message:  "circular dependency",
		Circular: []string{"task1", "task2", "task1"},
	}

	expected2 := "task 'task1': circular dependency detected: [task1 task2 task1]"
	if got := err2.Error(); got != expected2 {
		t.Errorf("DependencyError.Error() = %v, want %v", got, expected2)
	}
}
