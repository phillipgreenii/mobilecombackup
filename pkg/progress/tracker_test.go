package progress

import (
	"testing"
	"time"
)

func TestNewTaskTracker(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()

	if tracker == nil {
		t.Fatal("NewTaskTracker returned nil")
	}

	if tracker.tasks == nil {
		t.Error("tasks map should be initialized")
	}

	if tracker.config == nil {
		t.Error("config should be initialized")
	}

	if tracker.startTime.IsZero() {
		t.Error("startTime should be set")
	}
}

func TestTaskTracker_AddTask(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()

	// Test adding valid task
	task1 := &EnhancedTodo{
		ID:      "task1",
		Content: "Test task",
	}

	err := tracker.AddTask(task1)
	if err != nil {
		t.Errorf("AddTask failed: %v", err)
	}

	// Verify task was added
	retrieved, exists := tracker.GetTask("task1")
	if !exists {
		t.Error("Task was not stored")
	}

	if retrieved.ID != "task1" {
		t.Errorf("Retrieved task ID = %v, want task1", retrieved.ID)
	}

	// Verify defaults were applied
	if retrieved.Priority != PriorityMedium {
		t.Errorf("Default priority not applied, got %v", retrieved.Priority)
	}

	if retrieved.Complexity != ComplexityMedium {
		t.Errorf("Default complexity not applied, got %v", retrieved.Complexity)
	}

	if retrieved.Estimate == nil {
		t.Error("Estimate should be calculated from complexity")
	}

	// Test adding task with empty ID
	task2 := &EnhancedTodo{
		Content: "Invalid task",
	}

	err = tracker.AddTask(task2)
	if err == nil {
		t.Error("Expected error for empty task ID")
	}
}

func TestTaskTracker_UpdateTaskStatus(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()

	// Add a test task
	task := &EnhancedTodo{
		ID:      "task1",
		Content: "Test task",
	}
	tracker.AddTask(task)

	// Test starting task
	err := tracker.UpdateTaskStatus("task1", StatusInProgress, "")
	if err != nil {
		t.Errorf("UpdateTaskStatus failed: %v", err)
	}

	updated, _ := tracker.GetTask("task1")
	if updated.Status != StatusInProgress {
		t.Errorf("Status not updated, got %v", updated.Status)
	}

	if updated.StartedAt == nil {
		t.Error("StartedAt should be set when task starts")
	}

	// Test completing task
	time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps

	err = tracker.UpdateTaskStatus("task1", StatusCompleted, "")
	if err != nil {
		t.Errorf("UpdateTaskStatus failed: %v", err)
	}

	completed, _ := tracker.GetTask("task1")
	if completed.Status != StatusCompleted {
		t.Errorf("Status not updated to completed, got %v", completed.Status)
	}

	if completed.CompletedAt == nil {
		t.Error("CompletedAt should be set when task completes")
	}

	// Test blocking task
	tracker.AddTask(&EnhancedTodo{ID: "task2", Content: "Test task 2"})
	tracker.UpdateTaskStatus("task2", StatusInProgress, "")

	err = tracker.UpdateTaskStatus("task2", StatusBlocked, "test blocker")
	if err != nil {
		t.Errorf("UpdateTaskStatus failed: %v", err)
	}

	blocked, _ := tracker.GetTask("task2")
	if blocked.Status != StatusBlocked {
		t.Errorf("Status not updated to blocked, got %v", blocked.Status)
	}

	if blocked.BlockedReason == nil || *blocked.BlockedReason != "test blocker" {
		t.Errorf("BlockedReason not set correctly, got %v", blocked.BlockedReason)
	}

	// Test updating non-existent task
	err = tracker.UpdateTaskStatus("nonexistent", StatusCompleted, "")
	if err == nil {
		t.Error("Expected error for non-existent task")
	}
}

func TestTaskTracker_ValidateDependencies(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()

	// Add tasks with valid dependencies
	tracker.AddTask(&EnhancedTodo{ID: "task1", Content: "Task 1"})
	tracker.AddTask(&EnhancedTodo{ID: "task2", Content: "Task 2", Dependencies: []string{"task1"}})
	tracker.AddTask(&EnhancedTodo{ID: "task3", Content: "Task 3", Dependencies: []string{"task2"}})

	result := tracker.ValidateDependencies()
	if !result.IsValid {
		t.Errorf("Validation failed for valid dependencies: %v", result.Errors)
	}

	if len(result.TaskOrder) != 3 {
		t.Errorf("TaskOrder length = %d, want 3", len(result.TaskOrder))
	}

	// Verify topological order
	order := result.TaskOrder
	task1Index := indexOf(order, "task1")
	task2Index := indexOf(order, "task2")
	task3Index := indexOf(order, "task3")

	if task1Index == -1 || task2Index == -1 || task3Index == -1 {
		t.Error("All tasks should be in the order")
	}

	if task1Index > task2Index || task2Index > task3Index {
		t.Errorf("Tasks not in correct dependency order: %v", order)
	}
}

func TestTaskTracker_ValidateDependencies_CircularDependency(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()

	// Add tasks with circular dependencies
	tracker.AddTask(&EnhancedTodo{ID: "task1", Content: "Task 1", Dependencies: []string{"task2"}})
	tracker.AddTask(&EnhancedTodo{ID: "task2", Content: "Task 2", Dependencies: []string{"task1"}})

	result := tracker.ValidateDependencies()
	if result.IsValid {
		t.Error("Validation should fail for circular dependencies")
	}

	if len(result.Errors) == 0 {
		t.Error("Should have validation errors")
	}

	// Check for circular dependency error
	foundCircular := false
	for _, err := range result.Errors {
		if len(err.Circular) > 0 {
			foundCircular = true
			break
		}
	}

	if !foundCircular {
		t.Error("Should detect circular dependency")
	}
}

func TestTaskTracker_ValidateDependencies_MissingDependency(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()

	// Add task with missing dependency
	tracker.AddTask(&EnhancedTodo{ID: "task1", Content: "Task 1", Dependencies: []string{"nonexistent"}})

	result := tracker.ValidateDependencies()
	if result.IsValid {
		t.Error("Validation should fail for missing dependencies")
	}

	if len(result.Errors) == 0 {
		t.Error("Should have validation errors")
	}
}

func TestTaskTracker_GetProgressReport(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()

	// Add various tasks
	now := time.Now()
	startTime := now.Add(-1 * time.Hour)

	// Completed task
	completedTask := &EnhancedTodo{
		ID:       "completed",
		Content:  "Completed task",
		Status:   StatusCompleted,
		Estimate: durationPtr(30 * time.Minute),
	}
	completedTask.StartedAt = &startTime
	completedTask.CompletedAt = &now
	tracker.AddTask(completedTask)

	// In-progress task
	inProgressTask := &EnhancedTodo{
		ID:       "inprogress",
		Content:  "In progress task",
		Status:   StatusInProgress,
		Priority: PriorityHigh,
		Estimate: durationPtr(45 * time.Minute),
	}
	inProgressTask.StartedAt = &startTime
	tracker.AddTask(inProgressTask)

	// Blocked task
	blockedTask := &EnhancedTodo{
		ID:      "blocked",
		Content: "Blocked task",
		Status:  StatusBlocked,
	}
	reason := "waiting for dependency"
	blockedTask.BlockedReason = &reason
	tracker.AddTask(blockedTask)

	// Pending task
	tracker.AddTask(&EnhancedTodo{
		ID:      "pending",
		Content: "Pending task",
		Status:  StatusPending,
	})

	report := tracker.GetProgressReport()

	// Test basic counts
	if report.CompletedTasks != 1 {
		t.Errorf("CompletedTasks = %d, want 1", report.CompletedTasks)
	}

	if report.TotalTasks != 4 {
		t.Errorf("TotalTasks = %d, want 4", report.TotalTasks)
	}

	if report.InProgressTasks != 1 {
		t.Errorf("InProgressTasks = %d, want 1", report.InProgressTasks)
	}

	if report.BlockedTaskCount != 1 {
		t.Errorf("BlockedTaskCount = %d, want 1", report.BlockedTaskCount)
	}

	if report.PendingTasks != 1 {
		t.Errorf("PendingTasks = %d, want 1", report.PendingTasks)
	}

	// Test percentage calculation
	expectedPercentage := 25.0 // 1 completed out of 4 total
	if report.Percentage != expectedPercentage {
		t.Errorf("Percentage = %v, want %v", report.Percentage, expectedPercentage)
	}

	// Test current task (should be highest priority in-progress task)
	if report.CurrentTask == nil {
		t.Error("CurrentTask should not be nil")
	} else if report.CurrentTask.ID != "inprogress" {
		t.Errorf("CurrentTask ID = %v, want inprogress", report.CurrentTask.ID)
	}

	// Test blocked tasks
	if len(report.BlockedTasks) != 1 {
		t.Errorf("BlockedTasks length = %d, want 1", len(report.BlockedTasks))
	}

	// Test velocity calculation
	if report.Velocity <= 0 {
		t.Errorf("Velocity should be positive, got %v", report.Velocity)
	}
}

func TestTaskTracker_GetBlockedTasks(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()

	now := time.Now()
	longBlockedTime := now.Add(-30 * time.Minute)
	shortBlockedTime := now.Add(-5 * time.Minute)

	// Add long-blocked task
	longBlockedTask := &EnhancedTodo{
		ID:      "long-blocked",
		Content: "Long blocked task",
		Status:  StatusBlocked,
	}
	longBlockedTask.StartedAt = &longBlockedTime
	tracker.AddTask(longBlockedTask)

	// Add short-blocked task
	shortBlockedTask := &EnhancedTodo{
		ID:      "short-blocked",
		Content: "Short blocked task",
		Status:  StatusBlocked,
	}
	shortBlockedTask.StartedAt = &shortBlockedTime
	tracker.AddTask(shortBlockedTask)

	// Add non-blocked task
	tracker.AddTask(&EnhancedTodo{
		ID:      "normal",
		Content: "Normal task",
		Status:  StatusPending,
	})

	// Get tasks blocked longer than 10 minutes
	threshold := 10 * time.Minute
	blockedTasks := tracker.GetBlockedTasks(threshold)

	// Should only return the long-blocked task
	if len(blockedTasks) != 1 {
		t.Errorf("Expected 1 long-blocked task, got %d", len(blockedTasks))
	}

	if len(blockedTasks) > 0 && blockedTasks[0].ID != "long-blocked" {
		t.Errorf("Expected long-blocked task, got %v", blockedTasks[0].ID)
	}
}

func TestTaskTracker_TopologicalSort(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()

	// Create a more complex dependency graph
	tracker.AddTask(&EnhancedTodo{ID: "a", Content: "Task A", Priority: PriorityLow})
	tracker.AddTask(&EnhancedTodo{ID: "b", Content: "Task B", Dependencies: []string{"a"}, Priority: PriorityHigh})
	tracker.AddTask(&EnhancedTodo{ID: "c", Content: "Task C", Dependencies: []string{"a"}, Priority: PriorityMedium})
	tracker.AddTask(&EnhancedTodo{ID: "d", Content: "Task D", Dependencies: []string{"b", "c"}})

	order := tracker.topologicalSort()

	// Verify all tasks are included
	if len(order) != 4 {
		t.Errorf("Order length = %d, want 4", len(order))
	}

	// Verify dependency constraints
	aIndex := indexOf(order, "a")
	bIndex := indexOf(order, "b")
	cIndex := indexOf(order, "c")
	dIndex := indexOf(order, "d")

	if aIndex == -1 || bIndex == -1 || cIndex == -1 || dIndex == -1 {
		t.Error("All tasks should be in the order")
	}

	// A should come before B and C
	if aIndex > bIndex || aIndex > cIndex {
		t.Error("Task A should come before B and C")
	}

	// B and C should come before D
	if bIndex > dIndex || cIndex > dIndex {
		t.Error("Tasks B and C should come before D")
	}
}

func TestTaskTracker_CalculateCriticalPath(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()

	// Create tasks with different durations
	tracker.AddTask(&EnhancedTodo{
		ID:       "a",
		Content:  "Task A",
		Estimate: durationPtr(1 * time.Hour),
	})
	tracker.AddTask(&EnhancedTodo{
		ID:           "b",
		Content:      "Task B",
		Dependencies: []string{"a"},
		Estimate:     durationPtr(2 * time.Hour),
	})
	tracker.AddTask(&EnhancedTodo{
		ID:           "c",
		Content:      "Task C",
		Dependencies: []string{"a"},
		Estimate:     durationPtr(30 * time.Minute),
	})
	tracker.AddTask(&EnhancedTodo{
		ID:           "d",
		Content:      "Task D",
		Dependencies: []string{"b"},
		Estimate:     durationPtr(1 * time.Hour),
	})

	criticalPath := tracker.calculateCriticalPath()

	// Critical path should be A -> B -> D (longest path)
	expected := []string{"a", "b", "d"}
	if !equalSlices(criticalPath, expected) {
		t.Errorf("Critical path = %v, want %v", criticalPath, expected)
	}
}

func TestTaskTracker_FindParallelizableTasks(t *testing.T) {
	t.Parallel()

	tracker := NewTaskTracker()

	// Create tasks that can be parallelized
	tracker.AddTask(&EnhancedTodo{ID: "a", Content: "Task A"})
	tracker.AddTask(&EnhancedTodo{ID: "b", Content: "Task B"}) // Parallel with A
	tracker.AddTask(&EnhancedTodo{ID: "c", Content: "Task C", Dependencies: []string{"a"}})
	tracker.AddTask(&EnhancedTodo{ID: "d", Content: "Task D", Dependencies: []string{"b"}}) // Parallel with C

	order := []string{"a", "b", "c", "d"} // Assume this is a valid topological order
	parallelSets := tracker.findParallelizableTasks(order)

	// Should find that A and B can run in parallel, and C and D can run in parallel
	if len(parallelSets) < 1 {
		t.Error("Should find at least one set of parallelizable tasks")
	}

	// Find the parallel set containing A and B
	foundAB := false
	for _, set := range parallelSets {
		if contains(set, "a") && contains(set, "b") {
			foundAB = true
			break
		}
	}

	if !foundAB {
		t.Error("Should find A and B as parallelizable")
	}
}

// Helper functions

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(slice []string, item string) bool {
	return indexOf(slice, item) != -1
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}
