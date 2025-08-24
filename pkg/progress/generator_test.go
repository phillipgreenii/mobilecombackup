package progress

import (
	"strings"
	"testing"
	"time"
)

func TestNewTaskGenerator(t *testing.T) {
	t.Parallel()

	generator := NewTaskGenerator()

	if generator == nil {
		t.Fatal("NewTaskGenerator returned nil")
	}

	if generator.config == nil {
		t.Error("config should be initialized")
	}
}

func TestTaskGenerator_ParseIssueContent(t *testing.T) {
	t.Parallel()

	generator := NewTaskGenerator()

	issueContent := `# FEAT-123: Test Feature Implementation

## Overview
This is a test feature for parsing issue content and generating tasks automatically.

## Tasks
- [ ] Implement core functionality
- [x] Add unit tests
- [ ] Create documentation
- [ ] Review implementation

## Acceptance Criteria
- Feature works as expected
- All tests pass
- Documentation is complete

## Dependencies
- FEAT-100: Base framework
- FEAT-110: Authentication system

## Priority
HIGH - This is critical for the release

## Estimated Effort
- Implementation: 6-8 hours
- Testing: 2-3 hours
- Total: 8-11 hours`

	parsed, err := generator.parseIssueContent(issueContent)
	if err != nil {
		t.Fatalf("parseIssueContent failed: %v", err)
	}

	// Test title parsing
	if parsed.Title != "FEAT-123: Test Feature Implementation" {
		t.Errorf("Title = %v, want 'FEAT-123: Test Feature Implementation'", parsed.Title)
	}

	// Test category detection
	if parsed.Category != "feature" {
		t.Errorf("Category = %v, want 'feature'", parsed.Category)
	}

	// Test priority detection
	if parsed.Priority != PriorityHigh {
		t.Errorf("Priority = %v, want %v", parsed.Priority, PriorityHigh)
	}

	// Test overview parsing
	if !strings.Contains(parsed.Overview, "test feature") {
		t.Error("Overview should contain parsed text")
	}

	// Test tasks parsing
	if len(parsed.Tasks) != 4 {
		t.Errorf("Tasks count = %d, want 4", len(parsed.Tasks))
	}

	// Check task completion status
	if parsed.Tasks[1].Content != "Add unit tests" {
		t.Errorf("Task content = %v, want 'Add unit tests'", parsed.Tasks[1].Content)
	}

	if !parsed.Tasks[1].Completed {
		t.Error("Second task should be marked as completed")
	}

	// Test acceptance criteria parsing
	if len(parsed.AcceptanceCriteria) < 2 {
		t.Errorf("AcceptanceCriteria count = %d, want at least 2", len(parsed.AcceptanceCriteria))
	}

	// Test dependencies parsing
	if len(parsed.Dependencies) != 2 {
		t.Errorf("Dependencies count = %d, want 2", len(parsed.Dependencies))
	}

	expectedDeps := []string{"FEAT-100", "FEAT-110"}
	for _, dep := range expectedDeps {
		found := false
		for _, parsedDep := range parsed.Dependencies {
			if strings.Contains(parsedDep, dep) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Dependency %s not found in parsed dependencies", dep)
		}
	}
}

func TestTaskGenerator_GenerateTasksFromIssue(t *testing.T) {
	t.Parallel()

	generator := NewTaskGenerator()

	issueContent := `# FEAT-456: Database Migration Tool

## Tasks
- [ ] Design migration schema
- [ ] Implement migration runner
- [x] Add rollback functionality
- [ ] Create CLI interface
- [ ] Add comprehensive tests

## Acceptance Criteria  
- Migrations run successfully
- Rollbacks work correctly
- CLI is user-friendly
- Full test coverage achieved`

	tasks, err := generator.GenerateTasksFromIssue(issueContent)
	if err != nil {
		t.Fatalf("GenerateTasksFromIssue failed: %v", err)
	}

	// Should generate tasks from both tasks section and acceptance criteria
	expectedTaskCount := 5 + 4 // 5 tasks + 4 acceptance criteria
	if len(tasks) != expectedTaskCount {
		t.Errorf("Generated tasks count = %d, want %d", len(tasks), expectedTaskCount)
	}

	// Check first task
	firstTask := tasks[0]
	if firstTask.ID != "task-1" {
		t.Errorf("First task ID = %v, want 'task-1'", firstTask.ID)
	}

	if firstTask.Content != "Design migration schema" {
		t.Errorf("First task content = %v, want 'Design migration schema'", firstTask.Content)
	}

	if firstTask.Status != StatusPending {
		t.Errorf("First task status = %v, want %v", firstTask.Status, StatusPending)
	}

	// Check completed task
	var completedTask *EnhancedTodo
	for _, task := range tasks {
		if strings.Contains(task.Content, "rollback functionality") {
			completedTask = task
			break
		}
	}

	if completedTask == nil {
		t.Error("Should find completed rollback task")
	} else if completedTask.Status != StatusCompleted {
		t.Errorf("Rollback task status = %v, want %v", completedTask.Status, StatusCompleted)
	}

	// Check acceptance criteria tasks
	criteriaCount := 0
	for _, task := range tasks {
		if task.Category == "verification" {
			criteriaCount++
		}
	}

	if criteriaCount != 4 {
		t.Errorf("Criteria tasks count = %d, want 4", criteriaCount)
	}

	// Check that criteria tasks have dependencies on implementation tasks
	var criteriaTask *EnhancedTodo
	for _, task := range tasks {
		if strings.Contains(task.Content, "Verify: Migrations run successfully") {
			criteriaTask = task
			break
		}
	}

	if criteriaTask != nil && len(criteriaTask.Dependencies) == 0 {
		t.Error("Criteria task should have dependencies")
	}
}

func TestTaskGenerator_AnalyzeTaskComplexity(t *testing.T) {
	t.Parallel()

	generator := NewTaskGenerator()

	tests := []struct {
		content  string
		expected ComplexityLevel
	}{
		{"Implement comprehensive authentication system", ComplexityComplex},
		{"Design full database architecture", ComplexityComplex},
		{"Add simple logging", ComplexitySimple},
		{"Fix minor bug in validation", ComplexitySimple},
		{"Update documentation", ComplexitySimple},
		{"Create user interface", ComplexityMedium},
		{"Refactor code structure", ComplexityMedium},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			result := generator.analyzeTaskComplexity(tt.content)
			if result != tt.expected {
				t.Errorf("analyzeTaskComplexity(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestTaskGenerator_AnalyzeTaskCategory(t *testing.T) {
	t.Parallel()

	generator := NewTaskGenerator()

	tests := []struct {
		content  string
		expected string
	}{
		{"Implement user authentication", "implementation"},
		{"Test login functionality", "testing"},
		{"Document API endpoints", "documentation"},
		{"Plan system architecture", "planning"},
		{"Review code quality", "review"},
		{"Deploy to production", "deployment"},
		{"Refactor legacy code", "maintenance"},
		{"Random task content", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			result := generator.analyzeTaskCategory(tt.content)
			if result != tt.expected {
				t.Errorf("analyzeTaskCategory(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestTaskGenerator_AnalyzeTaskTags(t *testing.T) {
	t.Parallel()

	generator := NewTaskGenerator()

	tests := []struct {
		content     string
		expectedTag string
	}{
		{"Create API endpoint", "backend"},
		{"Update UI components", "frontend"},
		{"Add test coverage", "testing"},
		{"Write documentation", "docs"},
		{"Implement security features", "security"},
		{"Optimize performance", "performance"},
		{"Integration with external service", "integration"},
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			result := generator.analyzeTaskTags(tt.content)
			found := false
			for _, tag := range result {
				if tag == tt.expectedTag {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("analyzeTaskTags(%q) = %v, expected to contain %v", tt.content, result, tt.expectedTag)
			}
		})
	}
}

func TestTaskGenerator_EstimateIssueEffort(t *testing.T) {
	t.Parallel()

	generator := NewTaskGenerator()

	issueContent := `# FEAT-789: Simple Task List

## Tasks
- [ ] Add new field (simple task)
- [ ] Update validation logic (medium complexity)
- [ ] Implement complex algorithm (high complexity)`

	totalEffort, err := generator.EstimateIssueEffort(issueContent)
	if err != nil {
		t.Fatalf("EstimateIssueEffort failed: %v", err)
	}

	// Should be sum of estimates for all generated tasks
	// Simple (15min) + Medium (45min) + Complex (120min) = 180min = 3 hours
	// Plus any acceptance criteria tasks
	if totalEffort < 2*time.Hour || totalEffort > 5*time.Hour {
		t.Errorf("Total effort = %v, expected between 2-5 hours", totalEffort)
	}
}

func TestTaskGenerator_ProcessSection(t *testing.T) {
	t.Parallel()

	generator := NewTaskGenerator()
	issue := &ParsedIssue{
		Metadata: make(map[string]interface{}),
	}

	// Test overview processing
	generator.processSection(issue, "overview", "This is the overview content")
	if issue.Overview != "This is the overview content" {
		t.Errorf("Overview = %v, want 'This is the overview content'", issue.Overview)
	}

	// Test priority processing
	generator.processSection(issue, "priority", "CRITICAL - This is very important")
	if issue.Priority != PriorityCritical {
		t.Errorf("Priority = %v, want %v", issue.Priority, PriorityCritical)
	}

	// Test custom section in metadata
	generator.processSection(issue, "custom-section", "Custom content here")
	if customValue, exists := issue.Metadata["custom-section"]; !exists {
		t.Error("Custom section should be stored in metadata")
	} else if customValue != "Custom content here" {
		t.Errorf("Custom section content = %v, want 'Custom content here'", customValue)
	}
}

func TestTaskGenerator_ParseTasks(t *testing.T) {
	t.Parallel()

	generator := NewTaskGenerator()

	content := `- [ ] Implement feature A
- [x] Add tests for B  
- [ ] Document C functionality
- [x] Review D implementation`

	tasks := generator.parseTasks(content)

	if len(tasks) != 4 {
		t.Errorf("Parsed tasks count = %d, want 4", len(tasks))
	}

	// Check first task
	if tasks[0].Content != "Implement feature A" {
		t.Errorf("First task content = %v, want 'Implement feature A'", tasks[0].Content)
	}

	if tasks[0].Completed {
		t.Error("First task should not be completed")
	}

	// Check completed task
	if !tasks[1].Completed {
		t.Error("Second task should be completed")
	}

	if tasks[1].Content != "Add tests for B" {
		t.Errorf("Second task content = %v, want 'Add tests for B'", tasks[1].Content)
	}

	// Test complexity analysis
	implementTask := tasks[0]
	if implementTask.Complexity == ComplexitySimple {
		t.Error("Implement task should not be simple complexity")
	}

	// Test category analysis
	testTask := tasks[1]
	if testTask.Category != "testing" {
		t.Errorf("Test task category = %v, want 'testing'", testTask.Category)
	}

	docTask := tasks[2]
	if docTask.Category != "documentation" {
		t.Errorf("Doc task category = %v, want 'documentation'", docTask.Category)
	}
}

func TestTaskGenerator_ParseAcceptanceCriteria(t *testing.T) {
	t.Parallel()

	generator := NewTaskGenerator()

	content := `- Feature works correctly
- All edge cases handled  
- Performance is acceptable
Error handling is robust. System is maintainable.`

	criteria := generator.parseAcceptanceCriteria(content)

	// Should parse both list items and sentences
	if len(criteria) < 3 {
		t.Errorf("Parsed criteria count = %d, want at least 3", len(criteria))
	}

	expectedCriteria := []string{
		"Feature works correctly",
		"All edge cases handled",
		"Performance is acceptable",
	}

	for _, expected := range expectedCriteria {
		found := false
		for _, actual := range criteria {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected criteria '%s' not found in %v", expected, criteria)
		}
	}
}

func TestTaskGenerator_OrderTasks(t *testing.T) {
	t.Parallel()

	config := &TaskGenerationConfig{
		OrderByPriority:   true,
		OrderByDependency: true,
	}
	generator := NewTaskGeneratorWithConfig(config)

	tasks := []*EnhancedTodo{
		{ID: "low1", Priority: PriorityLow, Dependencies: []string{"high1"}},
		{ID: "high1", Priority: PriorityHigh, Dependencies: []string{}},
		{ID: "medium1", Priority: PriorityMedium, Dependencies: []string{}},
	}

	ordered := generator.orderTasks(tasks)

	// Should order by dependencies first (tasks with no deps), then by priority
	if ordered[0].ID != "high1" && ordered[0].ID != "medium1" {
		t.Errorf("First task should be high1 or medium1, got %v", ordered[0].ID)
	}

	// Task with dependencies should come after its dependencies
	low1Index := -1
	high1Index := -1
	for i, task := range ordered {
		if task.ID == "low1" {
			low1Index = i
		}
		if task.ID == "high1" {
			high1Index = i
		}
	}

	if low1Index != -1 && high1Index != -1 && low1Index < high1Index {
		t.Error("Task with dependency should come after its dependency")
	}
}

func TestTaskGenerator_ConfigurableGeneration(t *testing.T) {
	t.Parallel()

	// Test with tasks section disabled
	config := &TaskGenerationConfig{
		ParseTasksSection:       false,
		ParseAcceptanceCriteria: true,
		EstimateFromComplexity:  true,
		DefaultComplexity:       ComplexitySimple,
		DefaultPriority:         PriorityLow,
	}

	generator := NewTaskGeneratorWithConfig(config)

	issueContent := `# Test Issue

## Tasks
- [ ] Should be ignored
- [ ] Also ignored

## Acceptance Criteria
- Should be parsed
- Also parsed`

	tasks, err := generator.GenerateTasksFromIssue(issueContent)
	if err != nil {
		t.Fatalf("GenerateTasksFromIssue failed: %v", err)
	}

	// Should only generate tasks from acceptance criteria
	if len(tasks) != 2 {
		t.Errorf("Generated tasks count = %d, want 2", len(tasks))
	}

	for _, task := range tasks {
		if task.Category != "verification" {
			t.Errorf("Task category = %v, want 'verification'", task.Category)
		}

		if task.Priority != PriorityLow {
			t.Errorf("Task priority = %v, want %v", task.Priority, PriorityLow)
		}
	}
}
