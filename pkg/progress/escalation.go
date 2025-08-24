package progress

import (
	"fmt"
	"time"
)

// EscalationManager handles automatic escalation of blocked tasks and provides status reporting
type EscalationManager struct {
	tracker          *TaskTracker
	blockedThreshold time.Duration
	escalationLog    []EscalationEvent
	statusConfig     *StatusConfig
}

// EscalationEvent represents an escalation event
type EscalationEvent struct {
	TaskID     string        `json:"taskId"`
	EventType  string        `json:"eventType"` // "escalated", "resolved", "timeout"
	Timestamp  time.Time     `json:"timestamp"`
	Duration   time.Duration `json:"duration"` // How long task was blocked
	Reason     string        `json:"reason"`
	Resolution string        `json:"resolution,omitempty"`
	Severity   string        `json:"severity"` // "low", "medium", "high", "critical"
}

// StatusConfig configures status reporting behavior
type StatusConfig struct {
	ShowBlockedDetails  bool `json:"showBlockedDetails"`
	ShowVelocityMetrics bool `json:"showVelocityMetrics"`
	ShowCriticalPath    bool `json:"showCriticalPath"`
	ShowParallelTasks   bool `json:"showParallelTasks"`
	ShowTimeEstimates   bool `json:"showTimeEstimates"`
	CompactMode         bool `json:"compactMode"`
}

// GetDefaultStatusConfig returns default status reporting configuration
func GetDefaultStatusConfig() *StatusConfig {
	return &StatusConfig{
		ShowBlockedDetails:  true,
		ShowVelocityMetrics: true,
		ShowCriticalPath:    true,
		ShowParallelTasks:   true,
		ShowTimeEstimates:   true,
		CompactMode:         false,
	}
}

// NewEscalationManager creates a new escalation manager
func NewEscalationManager(tracker *TaskTracker) *EscalationManager {
	return &EscalationManager{
		tracker:          tracker,
		blockedThreshold: 15 * time.Minute, // Default 15 minute threshold
		escalationLog:    make([]EscalationEvent, 0),
		statusConfig:     GetDefaultStatusConfig(),
	}
}

// NewEscalationManagerWithConfig creates escalation manager with custom configuration
func NewEscalationManagerWithConfig(tracker *TaskTracker, threshold time.Duration, config *StatusConfig) *EscalationManager {
	return &EscalationManager{
		tracker:          tracker,
		blockedThreshold: threshold,
		escalationLog:    make([]EscalationEvent, 0),
		statusConfig:     config,
	}
}

// CheckForEscalations checks for tasks that need escalation and handles them
func (em *EscalationManager) CheckForEscalations() []EscalationEvent {
	blockedTasks := em.tracker.GetBlockedTasks(em.blockedThreshold)
	newEscalations := make([]EscalationEvent, 0, len(blockedTasks))

	for _, task := range blockedTasks {
		// Check if already escalated
		if em.isAlreadyEscalated(task.ID) {
			continue
		}

		escalation := EscalationEvent{
			TaskID:    task.ID,
			EventType: "escalated",
			Timestamp: time.Now(),
			Duration:  task.GetBlockedDuration(),
			Reason:    em.getBlockedReason(task),
			Severity:  em.calculateSeverity(task),
		}

		em.escalationLog = append(em.escalationLog, escalation)
		newEscalations = append(newEscalations, escalation)
	}

	return newEscalations
}

// isAlreadyEscalated checks if a task has already been escalated recently
func (em *EscalationManager) isAlreadyEscalated(taskID string) bool {
	recentThreshold := time.Now().Add(-30 * time.Minute) // Don't re-escalate within 30 minutes

	for _, event := range em.escalationLog {
		if event.TaskID == taskID && event.EventType == "escalated" && event.Timestamp.After(recentThreshold) {
			return true
		}
	}
	return false
}

// getBlockedReason extracts the blocked reason from a task
func (em *EscalationManager) getBlockedReason(task EnhancedTodo) string {
	if task.BlockedReason != nil {
		return *task.BlockedReason
	}
	return "Unknown blocking reason"
}

// calculateSeverity determines escalation severity based on task properties
func (em *EscalationManager) calculateSeverity(task EnhancedTodo) string {
	duration := task.GetBlockedDuration()

	// Base severity on priority and duration
	switch task.Priority {
	case PriorityCritical:
		if duration > 10*time.Minute {
			return "critical"
		}
		return "high"
	case PriorityHigh:
		if duration > 30*time.Minute {
			return "critical"
		} else if duration > 15*time.Minute {
			return "high"
		}
		return string(ComplexityMedium)
	case PriorityMedium:
		if duration > 1*time.Hour {
			return "high"
		} else if duration > 30*time.Minute {
			return string(ComplexityMedium)
		}
		return "low"
	default: // Low priority
		if duration > 2*time.Hour {
			return string(ComplexityMedium)
		}
		return "low"
	}
}

// GenerateEscalationAlert generates a user-friendly escalation alert
func (em *EscalationManager) GenerateEscalationAlert(escalation EscalationEvent) string {
	task, exists := em.tracker.GetTask(escalation.TaskID)
	if !exists {
		return fmt.Sprintf("⚠️ Task %s has been blocked for %v", escalation.TaskID, escalation.Duration.Round(time.Minute))
	}

	var severityIcon string
	switch escalation.Severity {
	case "critical":
		severityIcon = "🚨"
	case "high":
		severityIcon = "⚠️"
	case "medium":
		severityIcon = "⏰"
	default:
		severityIcon = "ℹ️"
	}

	alert := fmt.Sprintf("%s BLOCKED TASK ALERT\n", severityIcon)
	alert += fmt.Sprintf("Task: %s\n", task.Content)
	alert += fmt.Sprintf("Priority: %s\n", task.Priority)
	alert += fmt.Sprintf("Blocked for: %v\n", escalation.Duration.Round(time.Minute))
	alert += fmt.Sprintf("Reason: %s\n", escalation.Reason)
	alert += fmt.Sprintf("Severity: %s\n", escalation.Severity)

	// Add suggested actions based on severity and context
	alert += "\nSuggested Actions:\n"
	suggestions := em.generateResolutionSuggestions(task, escalation)
	for i, suggestion := range suggestions {
		alert += fmt.Sprintf("%d. %s\n", i+1, suggestion)
	}

	return alert
}

// generateResolutionSuggestions provides context-specific resolution suggestions
func (em *EscalationManager) generateResolutionSuggestions(task *EnhancedTodo, escalation EscalationEvent) []string {
	var suggestions []string

	// Generic suggestions
	suggestions = append(suggestions, "Review the blocking reason and determine if it can be resolved")

	if len(task.Dependencies) > 0 {
		suggestions = append(suggestions, "Check if dependency tasks are actually completed")
		suggestions = append(suggestions, "Consider if dependencies can be partially satisfied")
	}

	// Priority-based suggestions
	switch task.Priority {
	case PriorityCritical, PriorityHigh:
		suggestions = append(suggestions, "Escalate to team lead or seek immediate assistance")
		suggestions = append(suggestions, "Consider if task can be broken down into smaller parts")
	case PriorityMedium:
		suggestions = append(suggestions, "Reassess priority and consider deferring if appropriate")
		suggestions = append(suggestions, "Look for alternative approaches or workarounds")
	case PriorityLow:
		suggestions = append(suggestions, "Consider skipping this task for now and returning later")
	}

	// Complexity-based suggestions
	switch task.Complexity {
	case ComplexityComplex:
		suggestions = append(suggestions, "Break down complex task into smaller, manageable subtasks")
		suggestions = append(suggestions, "Seek expert guidance or pair programming assistance")
	case ComplexitySimple:
		suggestions = append(suggestions, "Simple task blocked - check for environment issues")
	}

	// Duration-based suggestions
	if escalation.Duration > 1*time.Hour {
		suggestions = append(suggestions, "Task blocked for over 1 hour - consider context switching")
		suggestions = append(suggestions, "Document current progress and move to different task")
	}

	// Common resolution suggestions
	suggestions = append(suggestions, "Update task status if actually unblocked")
	suggestions = append(suggestions, "Add more detailed blocking reason for better tracking")

	return suggestions
}

// ResolveEscalation marks an escalation as resolved
func (em *EscalationManager) ResolveEscalation(taskID, resolution string) {
	resolveEvent := EscalationEvent{
		TaskID:     taskID,
		EventType:  "resolved",
		Timestamp:  time.Now(),
		Resolution: resolution,
	}

	em.escalationLog = append(em.escalationLog, resolveEvent)
}

// GetEscalationHistory returns the escalation history for analysis
func (em *EscalationManager) GetEscalationHistory() []EscalationEvent {
	// Return copy to prevent modification
	history := make([]EscalationEvent, len(em.escalationLog))
	copy(history, em.escalationLog)
	return history
}

// GenerateStatusReport generates a comprehensive status report
func (em *EscalationManager) GenerateStatusReport() string {
	report := em.tracker.GetProgressReport()
	var output string

	if em.statusConfig.CompactMode {
		return em.generateCompactStatus(report)
	}

	// Header
	output += "📊 TASK PROGRESS STATUS\n"
	output += "=" + repeatString("=", 50) + "\n\n"

	// Overall progress
	output += fmt.Sprintf("📈 Overall Progress: %d/%d tasks (%.1f%%)\n",
		report.CompletedTasks, report.TotalTasks, report.Percentage)

	if em.statusConfig.ShowTimeEstimates {
		output += fmt.Sprintf("⏱️ Time: %v elapsed, ~%v remaining\n",
			report.TimeElapsed.Round(time.Minute),
			report.EstimatedRemaining.Round(time.Minute))
	}

	if em.statusConfig.ShowVelocityMetrics {
		output += fmt.Sprintf("🚀 Velocity: %.1f tasks/hour (%.1f%% efficiency)\n",
			report.Velocity, report.EfficiencyRatio*100)
	}

	output += "\n"

	// Current task
	if report.CurrentTask != nil {
		output += "🎯 CURRENT TASK\n"
		output += fmt.Sprintf("   %s\n", report.CurrentTask.Content)
		output += fmt.Sprintf("   Priority: %s | Complexity: %s\n",
			report.CurrentTask.Priority, report.CurrentTask.Complexity)
		if report.CurrentTask.StartedAt != nil {
			elapsed := time.Since(*report.CurrentTask.StartedAt)
			output += fmt.Sprintf("   Working for: %v\n", elapsed.Round(time.Minute))
		}
		output += "\n"
	}

	// Task status breakdown
	output += "📋 TASK STATUS\n"
	output += fmt.Sprintf("   ✅ Completed: %d\n", report.CompletedTasks)
	output += fmt.Sprintf("   🔄 In Progress: %d\n", report.InProgressTasks)
	output += fmt.Sprintf("   ⏳ Pending: %d\n", report.PendingTasks)
	output += fmt.Sprintf("   🚫 Blocked: %d\n", report.BlockedTaskCount)
	output += "\n"

	// Blocked tasks details
	if em.statusConfig.ShowBlockedDetails && len(report.BlockedTasks) > 0 {
		output += "🚫 BLOCKED TASKS\n"
		for i, task := range report.BlockedTasks {
			if i >= 3 { // Limit to 3 blocked tasks in status
				output += fmt.Sprintf("   ... and %d more\n", len(report.BlockedTasks)-3)
				break
			}

			duration := task.GetBlockedDuration()
			reason := "Unknown reason"
			if task.BlockedReason != nil {
				reason = *task.BlockedReason
			}

			output += fmt.Sprintf("   • %s (blocked %v)\n",
				truncateString(task.Content, 40), duration.Round(time.Minute))
			output += fmt.Sprintf("     Reason: %s\n", reason)
		}
		output += "\n"
	}

	// Available tasks
	if len(report.AvailableTasks) > 0 {
		output += "🔄 NEXT AVAILABLE TASKS\n"
		count := len(report.AvailableTasks)
		if count > 3 {
			count = 3 // Show top 3
		}

		for i := 0; i < count; i++ {
			task := report.AvailableTasks[i]
			output += fmt.Sprintf("   • %s (Priority: %s, Est: %v)\n",
				truncateString(task.Content, 40),
				task.Priority,
				task.GetEstimatedDuration().Round(time.Minute))
		}
		if len(report.AvailableTasks) > 3 {
			output += fmt.Sprintf("   ... and %d more available\n", len(report.AvailableTasks)-3)
		}
		output += "\n"
	}

	// Critical path
	if em.statusConfig.ShowCriticalPath && len(report.CriticalPath) > 0 {
		output += "🎯 CRITICAL PATH\n"
		output += "   "
		for i, taskID := range report.CriticalPath {
			if i > 0 {
				output += " → "
			}
			if task, exists := em.tracker.GetTask(taskID); exists {
				output += truncateString(task.Content, 15)
			} else {
				output += taskID
			}
		}
		output += "\n\n"
	}

	// Parallel opportunities
	if em.statusConfig.ShowParallelTasks && len(report.ParallelizableSets) > 0 {
		output += "⚡ PARALLELIZATION OPPORTUNITIES\n"
		for i, set := range report.ParallelizableSets {
			if i >= 2 { // Limit to 2 sets
				break
			}
			output += fmt.Sprintf("   Set %d: ", i+1)
			for j, taskID := range set {
				if j > 0 {
					output += " + "
				}
				if task, exists := em.tracker.GetTask(taskID); exists {
					output += truncateString(task.Content, 12)
				} else {
					output += taskID
				}
			}
			output += "\n"
		}
		output += "\n"
	}

	// Quality metrics
	if report.SuccessRate > 0 || report.RetryRate > 0 {
		output += "📊 QUALITY METRICS\n"
		output += fmt.Sprintf("   Success Rate: %.1f%%\n", report.SuccessRate)
		if report.RetryRate > 0 {
			output += fmt.Sprintf("   Retry Rate: %.1f%% (avg %.1f retries/task)\n",
				report.RetryRate, report.AverageRetries)
		}
		output += "\n"
	}

	// Recent escalations
	recentEscalations := em.getRecentEscalations(1 * time.Hour)
	if len(recentEscalations) > 0 {
		output += "⚠️ RECENT ESCALATIONS\n"
		for _, escalation := range recentEscalations {
			output += fmt.Sprintf("   • %s (%s) - %s\n",
				escalation.TaskID, escalation.Severity, escalation.Reason)
		}
		output += "\n"
	}

	return output
}

// generateCompactStatus generates a compact one-line status
func (em *EscalationManager) generateCompactStatus(report *ProgressReport) string {
	status := fmt.Sprintf("Progress: %d/%d (%.0f%%)",
		report.CompletedTasks, report.TotalTasks, report.Percentage)

	if report.CurrentTask != nil {
		status += fmt.Sprintf(" | Current: %s", truncateString(report.CurrentTask.Content, 25))
	}

	if report.BlockedTaskCount > 0 {
		status += fmt.Sprintf(" | Blocked: %d", report.BlockedTaskCount)
	}

	if report.Velocity > 0 {
		status += fmt.Sprintf(" | Velocity: %.1f/h", report.Velocity)
	}

	status += fmt.Sprintf(" | ETA: %v", report.EstimatedRemaining.Round(time.Minute))

	return status
}

// getRecentEscalations returns escalations within the specified time window
func (em *EscalationManager) getRecentEscalations(window time.Duration) []EscalationEvent {
	var recent []EscalationEvent
	cutoff := time.Now().Add(-window)

	for _, escalation := range em.escalationLog {
		if escalation.Timestamp.After(cutoff) && escalation.EventType == "escalated" {
			recent = append(recent, escalation)
		}
	}

	return recent
}

// Utility functions

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// GetEscalationSummary returns a summary of escalation metrics
func (em *EscalationManager) GetEscalationSummary() map[string]interface{} {
	totalEscalations := 0
	severityCounts := make(map[string]int)
	resolvedCount := 0

	for _, event := range em.escalationLog {
		switch event.EventType {
		case "escalated":
			totalEscalations++
			severityCounts[event.Severity]++
		case "resolved":
			resolvedCount++
		}
	}

	return map[string]interface{}{
		"total_escalations":  totalEscalations,
		"resolved_count":     resolvedCount,
		"severity_breakdown": severityCounts,
		"resolution_rate":    float64(resolvedCount) / float64(totalEscalations) * 100,
	}
}
