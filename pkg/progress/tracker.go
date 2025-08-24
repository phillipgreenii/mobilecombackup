package progress

import (
	"fmt"
	"sort"
	"time"
)

// TaskTracker manages enhanced task tracking with progress reporting and dependency management
type TaskTracker struct {
	tasks     map[string]*EnhancedTodo
	startTime time.Time
	config    *TaskGenerationConfig
	history   []TaskEvent // Historical events for analytics
}

// TaskEvent represents a historical event in task execution
type TaskEvent struct {
	TaskID    string    `json:"taskId"`
	Event     string    `json:"event"` // "started", "completed", "blocked", "resumed"
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"`
}

// NewTaskTracker creates a new task tracker
func NewTaskTracker() *TaskTracker {
	return &TaskTracker{
		tasks:     make(map[string]*EnhancedTodo),
		startTime: time.Now(),
		config:    GetDefaultConfig(),
		history:   make([]TaskEvent, 0),
	}
}

// NewTaskTrackerWithConfig creates a new task tracker with custom configuration
func NewTaskTrackerWithConfig(config *TaskGenerationConfig) *TaskTracker {
	return &TaskTracker{
		tasks:     make(map[string]*EnhancedTodo),
		startTime: time.Now(),
		config:    config,
		history:   make([]TaskEvent, 0),
	}
}

// AddTask adds a task to the tracker
func (t *TaskTracker) AddTask(task *EnhancedTodo) error {
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	task.ModifiedAt = time.Now()

	// Apply default values if not set
	if task.Priority == "" {
		task.Priority = t.config.DefaultPriority
	}
	if task.Complexity == "" {
		task.Complexity = t.config.DefaultComplexity
	}

	// Set estimate based on complexity if not provided
	if task.Estimate == nil && t.config.EstimateFromComplexity {
		estimate := t.calculateEstimate(task)
		task.Estimate = &estimate
	}

	t.tasks[task.ID] = task
	return nil
}

// calculateEstimate calculates time estimate based on complexity and category
func (t *TaskTracker) calculateEstimate(task *EnhancedTodo) time.Duration {
	baseDuration := task.Complexity.GetEstimatedMinutes()

	// Apply complexity multiplier
	if multiplier, ok := t.config.ComplexityMultiplier[task.Complexity]; ok {
		baseDuration = int(float64(baseDuration) * multiplier)
	}

	// Apply category multiplier
	if task.Category != "" {
		if multiplier, ok := t.config.CategoryMultiplier[task.Category]; ok {
			baseDuration = int(float64(baseDuration) * multiplier)
		}
	}

	return time.Duration(baseDuration) * time.Minute
}

// GetTask retrieves a task by ID
func (t *TaskTracker) GetTask(taskID string) (*EnhancedTodo, bool) {
	task, exists := t.tasks[taskID]
	return task, exists
}

// UpdateTaskStatus updates a task's status and handles timing
func (t *TaskTracker) UpdateTaskStatus(taskID string, status TaskStatus, reason string) error {
	task, exists := t.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	now := time.Now()
	oldStatus := task.Status

	// Handle status transitions
	switch status {
	case StatusInProgress:
		if task.StartedAt == nil {
			task.StartedAt = &now
		}
		if oldStatus == StatusBlocked {
			// Resuming from blocked state
			t.addHistoryEvent(taskID, "resumed", fmt.Sprintf("resumed after %v", task.GetBlockedDuration()))
		} else {
			t.addHistoryEvent(taskID, "started", "")
		}

	case StatusCompleted:
		if task.CompletedAt == nil {
			task.CompletedAt = &now
		}
		t.addHistoryEvent(taskID, "completed", fmt.Sprintf("duration: %v", task.GetActualDuration()))

	case StatusBlocked:
		task.BlockedReason = &reason
		t.addHistoryEvent(taskID, "blocked", reason)

	case StatusPending:
		// Reset timing if returning to pending
		task.StartedAt = nil
		task.CompletedAt = nil
		task.BlockedReason = nil
	}

	task.Status = status
	task.ModifiedAt = now

	return nil
}

// addHistoryEvent adds an event to the task history
func (t *TaskTracker) addHistoryEvent(taskID, event, details string) {
	t.history = append(t.history, TaskEvent{
		TaskID:    taskID,
		Event:     event,
		Timestamp: time.Now(),
		Details:   details,
	})
}

// ValidateDependencies validates all task dependencies and detects circular dependencies
func (t *TaskTracker) ValidateDependencies() *ValidationResult {
	result := &ValidationResult{IsValid: true}

	// Check for missing dependencies
	for taskID, task := range t.tasks {
		for _, depID := range task.Dependencies {
			if _, exists := t.tasks[depID]; !exists {
				result.AddError(taskID, fmt.Sprintf("dependency not found: %s", depID))
			}
		}
	}

	// Check for circular dependencies using DFS
	visited := make(map[string]int) // 0: unvisited, 1: visiting, 2: visited
	var stack []string

	var dfs func(string) bool
	dfs = func(taskID string) bool {
		if visited[taskID] == 1 {
			// Found circular dependency
			circularChain := make([]string, 0)
			startIndex := -1
			for i, id := range stack {
				if id == taskID {
					startIndex = i
					break
				}
			}
			if startIndex >= 0 {
				circularChain = append(circularChain, stack[startIndex:]...)
				circularChain = append(circularChain, taskID) // Complete the circle
			}
			result.AddCircularError(taskID, circularChain)
			return false
		}

		if visited[taskID] == 2 {
			return true // Already fully processed
		}

		visited[taskID] = 1 // Mark as visiting
		stack = append(stack, taskID)

		task, exists := t.tasks[taskID]
		if !exists {
			return true // Skip non-existent tasks
		}
		for _, depID := range task.Dependencies {
			if !dfs(depID) {
				return false
			}
		}

		visited[taskID] = 2          // Mark as fully visited
		stack = stack[:len(stack)-1] // Remove from stack
		return true
	}

	// Run DFS on all unvisited nodes
	for taskID := range t.tasks {
		if visited[taskID] == 0 {
			dfs(taskID)
		}
	}

	// Generate execution order using topological sort if valid
	if result.IsValid {
		result.TaskOrder = t.topologicalSort()
		result.CriticalPath = t.calculateCriticalPath()
	}

	return result
}

// topologicalSort returns tasks in dependency-safe execution order
func (t *TaskTracker) topologicalSort() []string {
	inDegree := make(map[string]int)

	// Initialize in-degree count
	for taskID := range t.tasks {
		inDegree[taskID] = 0
	}

	// Calculate in-degrees
	for _, task := range t.tasks {
		for _, depID := range task.Dependencies {
			inDegree[depID]++
		}
	}

	// Find tasks with no dependencies
	var queue []string
	for taskID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, taskID)
		}
	}

	// Sort initial queue by priority
	sort.Slice(queue, func(i, j int) bool {
		taskA := t.tasks[queue[i]]
		taskB := t.tasks[queue[j]]
		return taskA.Priority.GetWeight() > taskB.Priority.GetWeight()
	})

	var result []string
	for len(queue) > 0 {
		// Take task with highest priority
		taskID := queue[0]
		queue = queue[1:]
		result = append(result, taskID)

		// Remove this task from dependency graph
		task := t.tasks[taskID]
		for _, depID := range task.Dependencies {
			inDegree[depID]--
			if inDegree[depID] == 0 {
				// Insert in priority order
				inserted := false
				for i, queuedID := range queue {
					if t.tasks[depID].Priority.GetWeight() > t.tasks[queuedID].Priority.GetWeight() {
						queue = append(queue[:i], append([]string{depID}, queue[i:]...)...)
						inserted = true
						break
					}
				}
				if !inserted {
					queue = append(queue, depID)
				}
			}
		}
	}

	return result
}

// calculateCriticalPath calculates the critical path through the task dependency graph
func (t *TaskTracker) calculateCriticalPath() []string {
	// Build adjacency list for forward traversal
	dependents := make(map[string][]string)
	for taskID, task := range t.tasks {
		for _, depID := range task.Dependencies {
			dependents[depID] = append(dependents[depID], taskID)
		}
	}

	// Calculate earliest start times using dynamic programming
	earliestStart := make(map[string]time.Duration)
	var calculateEarliest func(string) time.Duration
	calculateEarliest = func(taskID string) time.Duration {
		if start, exists := earliestStart[taskID]; exists {
			return start
		}

		task := t.tasks[taskID]
		maxDepTime := time.Duration(0)

		// Find maximum dependency completion time
		for _, depID := range task.Dependencies {
			depTask := t.tasks[depID]
			depStart := calculateEarliest(depID)
			depEnd := depStart + depTask.GetEstimatedDuration()
			if depEnd > maxDepTime {
				maxDepTime = depEnd
			}
		}

		earliestStart[taskID] = maxDepTime
		return maxDepTime
	}

	// Calculate earliest start times for all tasks
	for taskID := range t.tasks {
		calculateEarliest(taskID)
	}

	// Find critical path by following longest path
	var criticalPath []string
	endTasks := make([]string, 0)

	// Find tasks with no dependents (end tasks)
	for taskID := range t.tasks {
		if len(dependents[taskID]) == 0 {
			endTasks = append(endTasks, taskID)
		}
	}

	// Find end task with latest completion time
	latestEnd := time.Duration(0)
	var criticalEndTask string
	for _, taskID := range endTasks {
		task := t.tasks[taskID]
		completion := earliestStart[taskID] + task.GetEstimatedDuration()
		if completion > latestEnd {
			latestEnd = completion
			criticalEndTask = taskID
		}
	}

	// Backtrack to find critical path
	if criticalEndTask != "" {
		visited := make(map[string]bool)
		var buildPath func(string)
		buildPath = func(taskID string) {
			if visited[taskID] {
				return
			}
			visited[taskID] = true
			criticalPath = append([]string{taskID}, criticalPath...)

			task := t.tasks[taskID]
			criticalTime := earliestStart[taskID]

			// Find critical dependency (one that determines start time)
			for _, depID := range task.Dependencies {
				depTask := t.tasks[depID]
				depCompletion := earliestStart[depID] + depTask.GetEstimatedDuration()
				if depCompletion == criticalTime {
					buildPath(depID)
					break
				}
			}
		}

		buildPath(criticalEndTask)
	}

	return criticalPath
}

// GetProgressReport generates a comprehensive progress report
func (t *TaskTracker) GetProgressReport() *ProgressReport {
	now := time.Now()
	report := &ProgressReport{
		StartTime:   t.startTime,
		CurrentTime: now,
		TimeElapsed: now.Sub(t.startTime),
	}

	// Count task statuses
	var completedCount, pendingCount, inProgressCount, blockedCount int
	var totalEstimated, totalActual time.Duration
	var retrySum int

	completedTasks := make(map[string]bool)
	var currentTask *EnhancedTodo
	var blockedTasks, availableTasks []EnhancedTodo

	for _, task := range t.tasks {
		report.TotalTasks++

		switch task.Status {
		case StatusCompleted:
			completedCount++
			completedTasks[task.ID] = true
			totalActual += task.GetActualDuration()
		case StatusPending:
			pendingCount++
			if task.CanStart(completedTasks) {
				availableTasks = append(availableTasks, *task)
			}
		case StatusInProgress:
			inProgressCount++
			if currentTask == nil || task.Priority.GetWeight() > currentTask.Priority.GetWeight() {
				currentTask = task
			}
		case StatusBlocked:
			blockedCount++
			blockedTasks = append(blockedTasks, *task)
		}

		totalEstimated += task.GetEstimatedDuration()
		retrySum += task.RetryCount
	}

	// Basic metrics
	report.CompletedTasks = completedCount
	report.PendingTasks = pendingCount
	report.InProgressTasks = inProgressCount
	report.BlockedTaskCount = blockedCount

	if report.TotalTasks > 0 {
		report.Percentage = float64(completedCount) / float64(report.TotalTasks) * 100
	}

	// Time estimates
	if completedCount > 0 {
		report.AverageTaskTime = totalActual / time.Duration(completedCount)

		// Estimate remaining time based on average actual time
		remainingTasks := report.TotalTasks - completedCount
		report.EstimatedRemaining = report.AverageTaskTime * time.Duration(remainingTasks)
	} else if totalEstimated > 0 {
		// Use original estimates if no completed tasks yet
		remainingEstimate := totalEstimated * time.Duration(report.TotalTasks-completedCount) / time.Duration(report.TotalTasks)
		report.EstimatedRemaining = remainingEstimate
	}

	report.EstimatedTotal = report.TimeElapsed + report.EstimatedRemaining

	// Velocity calculations
	if report.TimeElapsed > 0 {
		hoursElapsed := report.TimeElapsed.Hours()
		report.Velocity = float64(completedCount) / hoursElapsed
		report.CompletionRate = report.Velocity
	}

	// Efficiency ratio
	if totalEstimated > 0 && totalActual > 0 {
		report.EfficiencyRatio = totalEstimated.Seconds() / totalActual.Seconds()
	}

	// Quality metrics
	if report.TotalTasks > 0 {
		successfulTasks := completedCount
		if blockedCount == 0 && inProgressCount == 0 {
			// All done - count non-retried tasks as successful
			successfulTasks = 0
			for _, task := range t.tasks {
				if task.RetryCount == 0 {
					successfulTasks++
				}
			}
		}
		report.SuccessRate = float64(successfulTasks) / float64(report.TotalTasks) * 100

		tasksWithRetries := 0
		for _, task := range t.tasks {
			if task.RetryCount > 0 {
				tasksWithRetries++
			}
		}
		report.RetryRate = float64(tasksWithRetries) / float64(report.TotalTasks) * 100
		report.AverageRetries = float64(retrySum) / float64(report.TotalTasks)
	}

	// Set task arrays
	report.CurrentTask = currentTask
	report.BlockedTasks = blockedTasks
	report.AvailableTasks = availableTasks

	// Add dependency information
	validation := t.ValidateDependencies()
	if validation.IsValid {
		report.CriticalPath = validation.CriticalPath
		report.ParallelizableSets = t.findParallelizableTasks(validation.TaskOrder)
	}

	return report
}

// findParallelizableTasks identifies sets of tasks that can be executed in parallel
func (t *TaskTracker) findParallelizableTasks(orderedTasks []string) [][]string {
	var parallelSets [][]string

	// Group tasks by their dependency level
	levels := make(map[int][]string)
	taskLevel := make(map[string]int)

	// Calculate dependency levels
	for _, taskID := range orderedTasks {
		maxLevel := 0
		task := t.tasks[taskID]

		for _, depID := range task.Dependencies {
			if level, exists := taskLevel[depID]; exists && level >= maxLevel {
				maxLevel = level + 1
			}
		}

		taskLevel[taskID] = maxLevel
		levels[maxLevel] = append(levels[maxLevel], taskID)
	}

	// Convert levels to parallel sets (tasks in same level can run in parallel)
	for level := 0; level < len(levels); level++ {
		if tasks, exists := levels[level]; exists && len(tasks) > 1 {
			parallelSets = append(parallelSets, tasks)
		}
	}

	return parallelSets
}

// GetBlockedTasks returns tasks that have been blocked for too long
func (t *TaskTracker) GetBlockedTasks(threshold time.Duration) []EnhancedTodo {
	var longBlocked []EnhancedTodo

	for _, task := range t.tasks {
		if task.IsBlocked() && task.GetBlockedDuration() > threshold {
			longBlocked = append(longBlocked, *task)
		}
	}

	// Sort by blocked duration (longest first)
	sort.Slice(longBlocked, func(i, j int) bool {
		return longBlocked[i].GetBlockedDuration() > longBlocked[j].GetBlockedDuration()
	})

	return longBlocked
}

// GetHistory returns task execution history
func (t *TaskTracker) GetHistory() []TaskEvent {
	// Return copy to prevent modification
	historyCopy := make([]TaskEvent, len(t.history))
	copy(historyCopy, t.history)
	return historyCopy
}

// GetTasks returns all tasks (copy to prevent modification)
func (t *TaskTracker) GetTasks() map[string]EnhancedTodo {
	tasks := make(map[string]EnhancedTodo)
	for id, task := range t.tasks {
		tasks[id] = *task
	}
	return tasks
}

// Reset resets the tracker state
func (t *TaskTracker) Reset() {
	t.tasks = make(map[string]*EnhancedTodo)
	t.startTime = time.Now()
	t.history = make([]TaskEvent, 0)
}
