// Package progress provides enhanced task tracking and progress management capabilities.
package progress

import (
	"fmt"
	"time"
)

// ComplexityLevel represents the complexity of a task
type ComplexityLevel string

const (
	ComplexitySimple  ComplexityLevel = "simple"
	ComplexityMedium  ComplexityLevel = "medium"
	ComplexityComplex ComplexityLevel = "complex"
)

// String returns the string representation of ComplexityLevel
func (c ComplexityLevel) String() string {
	return string(c)
}

// GetEstimatedMinutes returns estimated minutes for a complexity level
func (c ComplexityLevel) GetEstimatedMinutes() int {
	switch c {
	case ComplexitySimple:
		return 15 // 15 minutes
	case ComplexityMedium:
		return 45 // 45 minutes
	case ComplexityComplex:
		return 120 // 2 hours
	default:
		return 30 // Default to 30 minutes
	}
}

// PriorityLevel represents the priority of a task
type PriorityLevel string

const (
	PriorityLow      PriorityLevel = "low"
	PriorityMedium   PriorityLevel = "medium"
	PriorityHigh     PriorityLevel = "high"
	PriorityCritical PriorityLevel = "critical"
)

// String returns the string representation of PriorityLevel
func (p PriorityLevel) String() string {
	return string(p)
}

// GetWeight returns numerical weight for priority ordering
func (p PriorityLevel) GetWeight() int {
	switch p {
	case PriorityCritical:
		return 4
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 2
	}
}

// TaskStatus represents the current status of a task
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusCompleted  TaskStatus = "completed"
	StatusBlocked    TaskStatus = "blocked"
)

// String returns the string representation of TaskStatus
func (s TaskStatus) String() string {
	return string(s)
}

// IsActive returns true if the task is actively being worked on
func (s TaskStatus) IsActive() bool {
	return s == StatusInProgress
}

// IsComplete returns true if the task is finished
func (s TaskStatus) IsComplete() bool {
	return s == StatusCompleted
}

// EnhancedTodo represents a task with enhanced metadata and tracking
type EnhancedTodo struct {
	// Basic fields (compatible with existing TodoWrite)
	ID      string     `json:"id"`
	Content string     `json:"content"`
	Status  TaskStatus `json:"status"`

	// Enhanced fields
	Estimate     *time.Duration  `json:"estimate,omitempty"`     // Time estimate
	Complexity   ComplexityLevel `json:"complexity,omitempty"`   // Complexity level
	Dependencies []string        `json:"dependencies,omitempty"` // Task IDs this depends on
	Priority     PriorityLevel   `json:"priority,omitempty"`     // Priority level

	// Timing fields
	StartedAt   *time.Time `json:"startedAt,omitempty"`   // When task was started
	CompletedAt *time.Time `json:"completedAt,omitempty"` // When task was completed

	// Blocking fields
	BlockedReason *string `json:"blockedReason,omitempty"` // Reason for blocking
	RetryCount    int     `json:"retryCount,omitempty"`    // Number of retry attempts

	// Git integration
	LinkedCommits []string `json:"linkedCommits,omitempty"` // Associated commit hashes

	// Metadata
	Category   string                 `json:"category,omitempty"` // Task category
	Tags       []string               `json:"tags,omitempty"`     // Task tags
	Metadata   map[string]interface{} `json:"metadata,omitempty"` // Additional metadata
	CreatedAt  time.Time              `json:"createdAt"`          // Task creation time
	ModifiedAt time.Time              `json:"modifiedAt"`         // Last modification time
}

// GetEstimatedDuration returns the estimated duration for the task
func (t *EnhancedTodo) GetEstimatedDuration() time.Duration {
	if t.Estimate != nil {
		return *t.Estimate
	}

	// Use complexity-based estimate as fallback
	minutes := t.Complexity.GetEstimatedMinutes()
	return time.Duration(minutes) * time.Minute
}

// GetActualDuration returns the actual time spent on the task
func (t *EnhancedTodo) GetActualDuration() time.Duration {
	if t.StartedAt == nil {
		return 0
	}

	endTime := time.Now()
	if t.CompletedAt != nil {
		endTime = *t.CompletedAt
	}

	return endTime.Sub(*t.StartedAt)
}

// IsBlocked returns true if the task is currently blocked
func (t *EnhancedTodo) IsBlocked() bool {
	return t.Status == StatusBlocked
}

// GetBlockedDuration returns how long the task has been blocked
func (t *EnhancedTodo) GetBlockedDuration() time.Duration {
	if !t.IsBlocked() || t.StartedAt == nil {
		return 0
	}
	return time.Since(*t.StartedAt)
}

// CanStart returns true if the task can be started (dependencies satisfied)
func (t *EnhancedTodo) CanStart(completedTasks map[string]bool) bool {
	for _, depID := range t.Dependencies {
		if !completedTasks[depID] {
			return false
		}
	}
	return t.Status == StatusPending
}

// ProgressReport provides comprehensive progress information
type ProgressReport struct {
	// Basic progress
	CompletedTasks int     `json:"completedTasks"`
	TotalTasks     int     `json:"totalTasks"`
	Percentage     float64 `json:"percentage"`

	// Timing
	StartTime          time.Time     `json:"startTime"`
	CurrentTime        time.Time     `json:"currentTime"`
	TimeElapsed        time.Duration `json:"timeElapsed"`
	EstimatedRemaining time.Duration `json:"estimatedRemaining"`
	EstimatedTotal     time.Duration `json:"estimatedTotal"`

	// Performance metrics
	Velocity        float64       `json:"velocity"`        // Tasks per hour
	AverageTaskTime time.Duration `json:"averageTaskTime"` // Average time per task
	EfficiencyRatio float64       `json:"efficiencyRatio"` // Actual vs estimated time
	CompletionRate  float64       `json:"completionRate"`  // Tasks completed per hour

	// Current state
	CurrentTask        *EnhancedTodo  `json:"currentTask,omitempty"`
	BlockedTasks       []EnhancedTodo `json:"blockedTasks,omitempty"`
	AvailableTasks     []EnhancedTodo `json:"availableTasks,omitempty"`
	CriticalPath       []string       `json:"criticalPath,omitempty"`       // Task IDs on critical path
	ParallelizableSets [][]string     `json:"parallelizableSets,omitempty"` // Sets of tasks that can run in parallel

	// Status counts
	PendingTasks     int `json:"pendingTasks"`
	InProgressTasks  int `json:"inProgressTasks"`
	BlockedTaskCount int `json:"blockedTaskCount"`

	// Quality metrics
	SuccessRate    float64 `json:"successRate"`    // Percentage of tasks completed without blocking
	RetryRate      float64 `json:"retryRate"`      // Percentage of tasks that required retries
	AverageRetries float64 `json:"averageRetries"` // Average retries per task
}

// String returns a formatted string representation of the progress report
func (p *ProgressReport) String() string {
	return fmt.Sprintf(
		"Progress: %d/%d (%.1f%%) | Elapsed: %v | Remaining: ~%v | Velocity: %.1f tasks/h | Blocked: %d",
		p.CompletedTasks, p.TotalTasks, p.Percentage,
		p.TimeElapsed.Round(time.Minute),
		p.EstimatedRemaining.Round(time.Minute),
		p.Velocity,
		p.BlockedTaskCount,
	)
}

// TaskGenerationConfig controls how tasks are generated from issues
type TaskGenerationConfig struct {
	// Parsing options
	ParseTasksSection       bool `json:"parseTasksSection"`       // Parse tasks from "Tasks" section
	ParseAcceptanceCriteria bool `json:"parseAcceptanceCriteria"` // Parse acceptance criteria as subtasks
	EstimateFromComplexity  bool `json:"estimateFromComplexity"`  // Use complexity for time estimates

	// Task ordering
	OrderByPriority   bool `json:"orderByPriority"`   // Order tasks by priority
	OrderByDependency bool `json:"orderByDependency"` // Order tasks by dependencies
	GroupByCategory   bool `json:"groupByCategory"`   // Group similar tasks together

	// Default values
	DefaultComplexity ComplexityLevel `json:"defaultComplexity"` // Default complexity for tasks
	DefaultPriority   PriorityLevel   `json:"defaultPriority"`   // Default priority for tasks

	// Estimation parameters
	ComplexityMultiplier map[ComplexityLevel]float64 `json:"complexityMultiplier"` // Multipliers for complexity
	CategoryMultiplier   map[string]float64          `json:"categoryMultiplier"`   // Multipliers for categories
}

// GetDefaultConfig returns a default task generation configuration
func GetDefaultConfig() *TaskGenerationConfig {
	return &TaskGenerationConfig{
		ParseTasksSection:       true,
		ParseAcceptanceCriteria: true,
		EstimateFromComplexity:  true,
		OrderByPriority:         true,
		OrderByDependency:       true,
		GroupByCategory:         false,
		DefaultComplexity:       ComplexityMedium,
		DefaultPriority:         PriorityMedium,
		ComplexityMultiplier: map[ComplexityLevel]float64{
			ComplexitySimple:  0.5,
			ComplexityMedium:  1.0,
			ComplexityComplex: 2.5,
		},
		CategoryMultiplier: map[string]float64{
			"implementation": 1.5,
			"testing":        1.2,
			"documentation":  0.8,
			"planning":       0.6,
			"review":         0.7,
		},
	}
}

// DependencyError represents an error in task dependencies
type DependencyError struct {
	TaskID   string
	Message  string
	Circular []string // Circular dependency chain if applicable
}

// Error returns the error message
func (e *DependencyError) Error() string {
	if len(e.Circular) > 0 {
		return fmt.Sprintf("task '%s': circular dependency detected: %v", e.TaskID, e.Circular)
	}
	return fmt.Sprintf("task '%s': %s", e.TaskID, e.Message)
}

// ValidationResult contains results from task validation
type ValidationResult struct {
	IsValid      bool              `json:"isValid"`
	Errors       []DependencyError `json:"errors,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
	TaskOrder    []string          `json:"taskOrder,omitempty"`    // Valid execution order
	CriticalPath []string          `json:"criticalPath,omitempty"` // Critical path through tasks
}

// AddError adds a validation error
func (v *ValidationResult) AddError(taskID, message string) {
	v.IsValid = false
	v.Errors = append(v.Errors, DependencyError{
		TaskID:  taskID,
		Message: message,
	})
}

// AddCircularError adds a circular dependency error
func (v *ValidationResult) AddCircularError(taskID string, chain []string) {
	v.IsValid = false
	v.Errors = append(v.Errors, DependencyError{
		TaskID:   taskID,
		Message:  "circular dependency detected",
		Circular: chain,
	})
}

// AddWarning adds a validation warning
func (v *ValidationResult) AddWarning(message string) {
	v.Warnings = append(v.Warnings, message)
}
