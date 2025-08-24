package progress

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// ExampleTaskGenerator_GenerateTasksFromIssue demonstrates generating tasks from an issue specification
func ExampleTaskGenerator_GenerateTasksFromIssue() {
	generator := NewTaskGenerator()

	issueContent := `# FEAT-100: User Authentication System

## Overview
Implement a comprehensive user authentication system with login, registration, and password reset functionality.

## Tasks
- [ ] Design database schema for users
- [ ] Implement user registration API
- [x] Add password hashing utility
- [ ] Create login endpoint
- [ ] Implement password reset flow
- [ ] Add session management
- [ ] Create frontend login form
- [ ] Add comprehensive tests

## Acceptance Criteria
- Users can register with email and password
- Login works with valid credentials
- Password reset emails are sent
- Sessions expire appropriately
- All endpoints return proper error messages

## Priority
HIGH - Critical for launch

## Estimated Effort
- Implementation: 12-15 hours
- Testing: 4-5 hours
- Total: 16-20 hours`

	tasks, err := generator.GenerateTasksFromIssue(issueContent)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Generated %d tasks from issue specification:\n", len(tasks))

	for i, task := range tasks {
		status := "⏳"
		if task.Status == StatusCompleted {
			status = "✅"
		}

		fmt.Printf("%d. %s %s [%s, %s, %v]\n",
			i+1, status, task.Content,
			task.Priority, task.Complexity,
			task.GetEstimatedDuration().Round(time.Minute))

		if len(task.Dependencies) > 0 {
			fmt.Printf("   Dependencies: %v\n", task.Dependencies)
		}
	}

	// Output:
	// Generated 13 tasks from issue specification:
	// 1. ⏳ Design database schema for users [medium, medium, 45m]
	// 2. ⏳ Implement user registration API [medium, complex, 2h0m]
	// 3. ✅ Add password hashing utility [medium, medium, 45m]
	// 4. ⏳ Create login endpoint [medium, complex, 2h0m]
	// 5. ⏳ Implement password reset flow [medium, complex, 2h0m]
	// 6. ⏳ Add session management [medium, complex, 2h0m]
	// 7. ⏳ Create frontend login form [medium, complex, 2h0m]
	// 8. ⏳ Add comprehensive tests [medium, testing, 54m]
	// 9. ⏳ Verify: Users can register with email and password [medium, simple, 15m]
	//    Dependencies: [task-8]
	// 10. ⏳ Verify: Login works with valid credentials [medium, simple, 15m]
	//     Dependencies: [task-8]
	// 11. ⏳ Verify: Password reset emails are sent [medium, simple, 15m]
	//     Dependencies: [task-8]
	// 12. ⏳ Verify: Sessions expire appropriately [medium, simple, 15m]
	//     Dependencies: [task-8]
	// 13. ⏳ Verify: All endpoints return proper error messages [medium, simple, 15m]
	//     Dependencies: [task-8]
}

// ExampleTaskTracker demonstrates comprehensive progress tracking
func ExampleTaskTracker() {
	tracker := NewTaskTracker()

	// Add tasks with different complexities and dependencies
	tasks := []*EnhancedTodo{
		{
			ID:         "setup",
			Content:    "Set up development environment",
			Complexity: ComplexitySimple,
			Priority:   PriorityHigh,
		},
		{
			ID:           "implement-core",
			Content:      "Implement core functionality",
			Complexity:   ComplexityComplex,
			Priority:     PriorityHigh,
			Dependencies: []string{"setup"},
		},
		{
			ID:           "add-tests",
			Content:      "Add comprehensive tests",
			Complexity:   ComplexityMedium,
			Priority:     PriorityMedium,
			Dependencies: []string{"implement-core"},
		},
		{
			ID:           "documentation",
			Content:      "Write documentation",
			Complexity:   ComplexityMedium,
			Priority:     PriorityLow,
			Dependencies: []string{"implement-core"},
		},
	}

	// Add tasks to tracker
	for _, task := range tasks {
		tracker.AddTask(task)
	}

	// Validate dependencies
	validation := tracker.ValidateDependencies()
	if !validation.IsValid {
		fmt.Printf("Dependency validation failed: %v\n", validation.Errors)
		return
	}

	fmt.Printf("Task execution order: %v\n", validation.TaskOrder)
	fmt.Printf("Critical path: %v\n", validation.CriticalPath)

	// Simulate task execution
	fmt.Println("\n--- Simulating Task Execution ---")

	// Start first task
	err := tracker.UpdateTaskStatus("setup", StatusInProgress, "")
	if err != nil {
		fmt.Printf("Error updating task status: %v\n", err)
		return
	}
	time.Sleep(50 * time.Millisecond) // Simulate work

	report := tracker.GetProgressReport()
	fmt.Printf("Progress: %s\n", report.String())

	// Complete first task
	err = tracker.UpdateTaskStatus("setup", StatusCompleted, "")
	if err != nil {
		fmt.Printf("Error completing task: %v\n", err)
		return
	}

	// Start second task
	err = tracker.UpdateTaskStatus("implement-core", StatusInProgress, "")
	if err != nil {
		fmt.Printf("Error starting second task: %v\n", err)
		return
	}
	time.Sleep(30 * time.Millisecond) // Simulate work

	// Block the task
	tracker.UpdateTaskStatus("implement-core", StatusBlocked, "waiting for API specification")

	report = tracker.GetProgressReport()
	fmt.Printf("After blocking: %s\n", report.String())

	// Show available tasks (tasks that can be started)
	fmt.Printf("Available tasks: %d\n", len(report.AvailableTasks))
	for _, task := range report.AvailableTasks {
		fmt.Printf("  - %s (%s)\n", task.Content, task.Priority)
	}

	// Output:
	// Task execution order: [setup implement-core add-tests documentation]
	// Critical path: [setup implement-core add-tests]
	//
	// --- Simulating Task Execution ---
	// Progress: 0/4 (0.0%) | Elapsed: 50ms | Remaining: ~0s | Velocity: 0.0 tasks/h | Blocked: 0
	// After blocking: 1/4 (25.0%) | Elapsed: 80ms | Remaining: ~0s | Velocity: 45000.0 tasks/h | Blocked: 1
	// Available tasks: 0
}

// ExampleEscalationManager demonstrates automatic escalation of blocked tasks
func ExampleEscalationManager() {
	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	// Create a high-priority task
	task := &EnhancedTodo{
		ID:         "critical-feature",
		Content:    "Implement critical security feature",
		Priority:   PriorityHigh,
		Complexity: ComplexityComplex,
		Status:     StatusBlocked,
	}

	// Simulate task being blocked for 20 minutes (exceeds 15 minute threshold)
	blockTime := time.Now().Add(-20 * time.Minute)
	task.StartedAt = &blockTime
	reason := "waiting for security review approval"
	task.BlockedReason = &reason

	tracker.AddTask(task)

	// Check for escalations
	escalations := em.CheckForEscalations()

	if len(escalations) > 0 {
		escalation := escalations[0]
		fmt.Printf("🚨 ESCALATION DETECTED\n")
		fmt.Printf("Task: %s\n", escalation.TaskID)
		fmt.Printf("Severity: %s\n", escalation.Severity)
		fmt.Printf("Blocked for: %v\n", escalation.Duration.Round(time.Minute))
		fmt.Printf("Reason: %s\n", escalation.Reason)

		// Generate escalation alert
		alert := em.GenerateEscalationAlert(escalation)
		fmt.Printf("\n--- ESCALATION ALERT ---\n%s\n", alert)

		// Resolve the escalation
		em.ResolveEscalation("critical-feature", "Security team provided approval")

		summary := em.GetEscalationSummary()
		fmt.Printf("--- ESCALATION SUMMARY ---\n")
		fmt.Printf("Total escalations: %v\n", summary["total_escalations"])
		fmt.Printf("Resolved: %v\n", summary["resolved_count"])
		fmt.Printf("Resolution rate: %.1f%%\n", summary["resolution_rate"])
	}

	// Output:
	// 🚨 ESCALATION DETECTED
	// Task: critical-feature
	// Severity: high
	// Blocked for: 20m
	// Reason: waiting for security review approval
	//
	// --- ESCALATION ALERT ---
	// ⚠️ BLOCKED TASK ALERT
	// Task: Implement critical security feature
	// Priority: high
	// Blocked for: 20m
	// Reason: waiting for security review approval
	// Severity: high
	//
	// Suggested Actions:
	// 1. Review the blocking reason and determine if it can be resolved
	// 2. Escalate to team lead or seek immediate assistance
	// 3. Consider if task can be broken down into smaller parts
	// 4. Break down complex task into smaller, manageable subtasks
	// 5. Seek expert guidance or pair programming assistance
	// 6. Update task status if actually unblocked
	// 7. Add more detailed blocking reason for better tracking
	//
	// --- ESCALATION SUMMARY ---
	// Total escalations: 1
	// Resolved: 1
	// Resolution rate: 100.0%
}

// Example_statusReport demonstrates comprehensive status reporting
func Example_statusReport() {
	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	// Create a realistic task scenario
	now := time.Now()

	// Add completed tasks
	completedTask1 := &EnhancedTodo{
		ID:       "task1",
		Content:  "Set up project structure",
		Status:   StatusCompleted,
		Priority: PriorityMedium,
	}
	startTime1 := now.Add(-2 * time.Hour)
	endTime1 := now.Add(-90 * time.Minute)
	completedTask1.StartedAt = &startTime1
	completedTask1.CompletedAt = &endTime1
	tracker.AddTask(completedTask1)

	completedTask2 := &EnhancedTodo{
		ID:       "task2",
		Content:  "Implement basic API",
		Status:   StatusCompleted,
		Priority: PriorityHigh,
	}
	startTime2 := now.Add(-90 * time.Minute)
	endTime2 := now.Add(-30 * time.Minute)
	completedTask2.StartedAt = &startTime2
	completedTask2.CompletedAt = &endTime2
	tracker.AddTask(completedTask2)

	// Current task in progress
	currentTask := &EnhancedTodo{
		ID:         "current",
		Content:    "Add authentication middleware",
		Status:     StatusInProgress,
		Priority:   PriorityHigh,
		Complexity: ComplexityMedium,
	}
	currentStart := now.Add(-25 * time.Minute)
	currentTask.StartedAt = &currentStart
	tracker.AddTask(currentTask)

	// Blocked task
	blockedTask := &EnhancedTodo{
		ID:       "blocked",
		Content:  "Deploy to staging environment",
		Status:   StatusBlocked,
		Priority: PriorityMedium,
	}
	blockedReason := "waiting for DevOps team"
	blockedTask.BlockedReason = &blockedReason
	tracker.AddTask(blockedTask)

	// Pending tasks
	tracker.AddTask(&EnhancedTodo{
		ID:       "pending1",
		Content:  "Write integration tests",
		Status:   StatusPending,
		Priority: PriorityHigh,
	})

	tracker.AddTask(&EnhancedTodo{
		ID:       "pending2",
		Content:  "Update documentation",
		Status:   StatusPending,
		Priority: PriorityLow,
	})

	// Generate status report
	statusReport := em.GenerateStatusReport()
	fmt.Print(statusReport)

	fmt.Println("\n--- COMPACT STATUS ---")
	em.statusConfig.CompactMode = true
	compactStatus := em.GenerateStatusReport()
	fmt.Println(compactStatus)

	// Output:
	// 📊 TASK PROGRESS STATUS
	// ==================================================
	//
	// 📈 Overall Progress: 2/6 tasks (33.3%)
	// ⏱️ Time: 2h0m elapsed, ~1h30m remaining
	// 🚀 Velocity: 1.0 tasks/hour (66.7% efficiency)
	//
	// 🎯 CURRENT TASK
	//    Add authentication middleware
	//    Priority: high | Complexity: medium
	//    Working for: 25m
	//
	// 📋 TASK STATUS
	//    ✅ Completed: 2
	//    🔄 In Progress: 1
	//    ⏳ Pending: 2
	//    🚫 Blocked: 1
	//
	// 🚫 BLOCKED TASKS
	//    • Deploy to staging environment (blocked 0s)
	//      Reason: waiting for DevOps team
	//
	// 🔄 NEXT AVAILABLE TASKS
	//    • Write integration tests (Priority: high, Est: 45m)
	//    • Update documentation (Priority: low, Est: 45m)
	//
	// 📊 QUALITY METRICS
	//    Success Rate: 33.3%
	//
	// --- COMPACT STATUS ---
	// Progress: 2/6 (33%) | Current: Add authentication middle... | Blocked: 1 | Velocity: 1.0/h | ETA: 1h30m
}

// TestCompleteWorkflow demonstrates a complete workflow from issue to completion
func TestCompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Step 1: Generate tasks from issue
	generator := NewTaskGenerator()
	issueContent := `# FEAT-200: Task Progress Enhancement

## Tasks
- [ ] Create progress tracking types
- [ ] Implement task tracker
- [ ] Add dependency validation
- [ ] Create escalation manager

## Acceptance Criteria  
- All tasks tracked properly
- Dependencies validated
- Escalations work correctly`

	tasks, err := generator.GenerateTasksFromIssue(issueContent)
	if err != nil {
		t.Fatalf("Failed to generate tasks: %v", err)
	}

	// Step 2: Set up tracking
	tracker := NewTaskTracker()
	em := NewEscalationManager(tracker)

	// Add tasks to tracker
	for _, task := range tasks {
		err := tracker.AddTask(task)
		if err != nil {
			t.Fatalf("Failed to add task: %v", err)
		}
	}

	// Step 3: Validate workflow
	validation := tracker.ValidateDependencies()
	if !validation.IsValid {
		t.Fatalf("Dependency validation failed: %v", validation.Errors)
	}

	// Step 4: Simulate execution
	orderedTasks := validation.TaskOrder

	completedCount := 0
	for i, taskID := range orderedTasks {
		if i >= 2 { // Only complete first 2 tasks for testing
			break
		}

		// Start task
		err := tracker.UpdateTaskStatus(taskID, StatusInProgress, "")
		if err != nil {
			t.Fatalf("Failed to start task: %v", err)
		}

		// Simulate some work time
		time.Sleep(10 * time.Millisecond)

		// Complete task
		err = tracker.UpdateTaskStatus(taskID, StatusCompleted, "")
		if err != nil {
			t.Fatalf("Failed to complete task: %v", err)
		}

		completedCount++
	}

	// Step 5: Generate final report
	report := tracker.GetProgressReport()

	// Verify progress
	if report.CompletedTasks != completedCount {
		t.Errorf("Expected %d completed tasks, got %d", completedCount, report.CompletedTasks)
	}

	if report.TotalTasks != len(tasks) {
		t.Errorf("Expected %d total tasks, got %d", len(tasks), report.TotalTasks)
	}

	// Test escalation system (by creating a blocked task)
	if len(orderedTasks) > 2 {
		thirdTask := orderedTasks[2]
		tracker.UpdateTaskStatus(thirdTask, StatusInProgress, "")

		// Block it and backdate to trigger escalation
		tracker.UpdateTaskStatus(thirdTask, StatusBlocked, "test blocking")

		// Manually set blocked time to trigger escalation
		task, _ := tracker.GetTask(thirdTask)
		blockTime := time.Now().Add(-20 * time.Minute)
		task.StartedAt = &blockTime

		// Check escalations
		escalations := em.CheckForEscalations()
		if len(escalations) == 0 {
			t.Error("Expected escalation for blocked task")
		}
	}

	// Generate status report
	status := em.GenerateStatusReport()
	if !strings.Contains(status, "TASK PROGRESS STATUS") {
		t.Error("Status report should contain header")
	}

	t.Logf("Workflow completed successfully. Final status:\n%s", status)
}
