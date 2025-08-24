package progress

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

// TaskGenerator generates tasks from issue specifications and documentation
type TaskGenerator struct {
	config *TaskGenerationConfig
}

// NewTaskGenerator creates a new task generator with default configuration
func NewTaskGenerator() *TaskGenerator {
	return &TaskGenerator{
		config: GetDefaultConfig(),
	}
}

// NewTaskGeneratorWithConfig creates a new task generator with custom configuration
func NewTaskGeneratorWithConfig(config *TaskGenerationConfig) *TaskGenerator {
	return &TaskGenerator{
		config: config,
	}
}

// ParsedIssue represents a parsed issue document
type ParsedIssue struct {
	Title              string
	Overview           string
	Tasks              []TaskDefinition
	AcceptanceCriteria []string
	Requirements       []string
	Dependencies       []string
	EstimatedEffort    string
	Priority           PriorityLevel
	Category           string
	Metadata           map[string]interface{}
}

// TaskDefinition represents a task definition found in an issue
type TaskDefinition struct {
	Content      string
	Completed    bool
	Complexity   ComplexityLevel
	Category     string
	Dependencies []string
	Estimate     *time.Duration
	Priority     PriorityLevel
	Tags         []string
}

// GenerateTasksFromIssue generates enhanced tasks from an issue specification
func (g *TaskGenerator) GenerateTasksFromIssue(issueContent string) ([]*EnhancedTodo, error) {
	return g.generateTasksFromParsedIssue(issueContent)
}

// generateTasksFromParsedIssue handles the main logic for task generation
func (g *TaskGenerator) generateTasksFromParsedIssue(issueContent string) ([]*EnhancedTodo, error) {
	parsed, err := g.parseIssueContent(issueContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse issue content: %w", err)
	}

	var tasks []*EnhancedTodo
	now := time.Now()

	// Generate tasks from parsed content
	if g.config.ParseTasksSection {
		for i, taskDef := range parsed.Tasks {
			task := &EnhancedTodo{
				ID:         fmt.Sprintf("task-%d", i+1),
				Content:    taskDef.Content,
				Status:     StatusPending,
				Complexity: taskDef.Complexity,
				Priority:   taskDef.Priority,
				Category:   taskDef.Category,
				Tags:       taskDef.Tags,
				CreatedAt:  now,
				ModifiedAt: now,
				Metadata:   make(map[string]interface{}),
			}

			// Set estimate
			if taskDef.Estimate != nil {
				task.Estimate = taskDef.Estimate
			} else if g.config.EstimateFromComplexity {
				estimate := g.calculateEstimate(task)
				task.Estimate = &estimate
			}

			// Set dependencies
			task.Dependencies = taskDef.Dependencies

			// Mark as completed if checked in original
			if taskDef.Completed {
				task.Status = StatusCompleted
				completedTime := now
				task.CompletedAt = &completedTime
			}

			// Add metadata
			task.Metadata["source"] = "issue-tasks"
			task.Metadata["issue-title"] = parsed.Title

			tasks = append(tasks, task)
		}
	}

	// Generate subtasks from acceptance criteria
	if g.config.ParseAcceptanceCriteria {
		for i, criteria := range parsed.AcceptanceCriteria {
			task := &EnhancedTodo{
				ID:         fmt.Sprintf("criteria-%d", i+1),
				Content:    fmt.Sprintf("Verify: %s", criteria),
				Status:     StatusPending,
				Complexity: ComplexitySimple, // Criteria are typically verification tasks
				Priority:   PriorityMedium,
				Category:   "verification",
				Tags:       []string{"acceptance-criteria", "verification"},
				CreatedAt:  now,
				ModifiedAt: now,
				Metadata:   make(map[string]interface{}),
			}

			// Criteria tasks depend on implementation tasks
			if len(tasks) > 0 {
				// Add dependency on the last implementation task
				for j := len(tasks) - 1; j >= 0; j-- {
					if tasks[j].Category != "verification" {
						task.Dependencies = append(task.Dependencies, tasks[j].ID)
						break
					}
				}
			}

			if g.config.EstimateFromComplexity {
				estimate := g.calculateEstimate(task)
				task.Estimate = &estimate
			}

			task.Metadata["source"] = "acceptance-criteria"
			task.Metadata["criteria-text"] = criteria

			tasks = append(tasks, task)
		}
	}

	// Apply ordering and grouping
	if g.config.OrderByDependency || g.config.OrderByPriority {
		tasks = g.orderTasks(tasks)
	}

	return tasks, nil
}

// parseIssueContent parses issue markdown content into structured data
func (g *TaskGenerator) parseIssueContent(content string) (*ParsedIssue, error) {
	issue := &ParsedIssue{
		Metadata: make(map[string]interface{}),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))

	var currentSection string
	var sectionContent strings.Builder

	// Regex patterns for parsing
	titlePattern := regexp.MustCompile(`^# (.+)$`)
	sectionPattern := regexp.MustCompile(`^## (.+)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Extract title
		if matches := titlePattern.FindStringSubmatch(line); matches != nil {
			issue.Title = strings.TrimSpace(matches[1])

			// Extract issue type and number from title
			switch {
			case strings.Contains(issue.Title, "FEAT-"):
				issue.Category = "feature"
				issue.Priority = PriorityMedium
			case strings.Contains(issue.Title, "BUG-"):
				issue.Category = "bugfix"
				issue.Priority = PriorityHigh
			case strings.Contains(issue.Title, "DOC-"):
				issue.Category = "documentation"
				issue.Priority = PriorityLow
			}
			continue
		}

		// Extract sections
		if matches := sectionPattern.FindStringSubmatch(line); matches != nil {
			// Process previous section
			if currentSection != "" {
				g.processSection(issue, currentSection, sectionContent.String())
			}

			currentSection = strings.ToLower(strings.TrimSpace(matches[1]))
			sectionContent.Reset()
			continue
		}

		// Add line to current section
		if currentSection != "" {
			if sectionContent.Len() > 0 {
				sectionContent.WriteString("\n")
			}
			sectionContent.WriteString(line)
		}
	}

	// Process final section
	if currentSection != "" {
		g.processSection(issue, currentSection, sectionContent.String())
	}

	return issue, scanner.Err()
}

// processSection processes a specific section of the issue
func (g *TaskGenerator) processSection(issue *ParsedIssue, sectionName, content string) {
	lines := strings.Split(content, "\n")

	switch sectionName {
	case "overview":
		issue.Overview = strings.TrimSpace(content)

	case "tasks":
		issue.Tasks = g.parseTasks(content)

	case "acceptance criteria":
		issue.AcceptanceCriteria = g.parseAcceptanceCriteria(content)

	case "requirements", "functional requirements":
		issue.Requirements = g.parseRequirements(content)

	case "dependencies":
		issue.Dependencies = g.parseDependencies(content)

	case "estimated effort", "effort":
		issue.EstimatedEffort = strings.TrimSpace(content)

		// Extract priority from effort section
		if strings.Contains(strings.ToLower(content), "high") || strings.Contains(strings.ToLower(content), "critical") {
			issue.Priority = PriorityHigh
		} else if strings.Contains(strings.ToLower(content), "low") {
			issue.Priority = PriorityLow
		}

	case "priority":
		priorityText := strings.ToLower(strings.TrimSpace(content))
		switch {
		case strings.Contains(priorityText, "critical"):
			issue.Priority = PriorityCritical
		case strings.Contains(priorityText, "high"):
			issue.Priority = PriorityHigh
		case strings.Contains(priorityText, "low"):
			issue.Priority = PriorityLow
		default:
			issue.Priority = PriorityMedium
		}

	default:
		// Store other sections in metadata
		if len(lines) > 0 && strings.TrimSpace(content) != "" {
			issue.Metadata[sectionName] = strings.TrimSpace(content)
		}
	}
}

// parseTasks parses task list from markdown content
func (g *TaskGenerator) parseTasks(content string) []TaskDefinition {
	var tasks []TaskDefinition
	lines := strings.Split(content, "\n")

	taskPattern := regexp.MustCompile(`^- \[([ x])\] (.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := taskPattern.FindStringSubmatch(line); matches != nil {
			task := TaskDefinition{
				Content:   strings.TrimSpace(matches[2]),
				Completed: matches[1] == "x",
				Priority:  g.config.DefaultPriority,
			}

			// Analyze task content for complexity hints
			task.Complexity = g.analyzeTaskComplexity(task.Content)
			task.Category = g.analyzeTaskCategory(task.Content)
			task.Tags = g.analyzeTaskTags(task.Content)

			tasks = append(tasks, task)
		}
	}

	return tasks
}

// analyzeTaskComplexity determines complexity from task content
func (g *TaskGenerator) analyzeTaskComplexity(content string) ComplexityLevel {
	contentLower := strings.ToLower(content)

	// Complex indicators
	complexKeywords := []string{
		"implement", "create", "build", "design", "architect",
		"integration", "system", "framework", "algorithm",
		"comprehensive", "end-to-end", "full", "complete",
	}

	// Simple indicators
	simpleKeywords := []string{
		"add", "update", "fix", "modify", "change", "adjust",
		"document", "write", "format", "clean", "organize",
		"test", "verify", "check", "validate", "review",
	}

	complexCount := 0
	simpleCount := 0

	for _, keyword := range complexKeywords {
		if strings.Contains(contentLower, keyword) {
			complexCount++
		}
	}

	for _, keyword := range simpleKeywords {
		if strings.Contains(contentLower, keyword) {
			simpleCount++
		}
	}

	// Determine complexity based on keyword analysis
	if complexCount > simpleCount && complexCount > 0 {
		return ComplexityComplex
	} else if simpleCount > 0 {
		return ComplexitySimple
	}

	return ComplexityMedium // Default
}

// analyzeTaskCategory determines category from task content
func (g *TaskGenerator) analyzeTaskCategory(content string) string {
	contentLower := strings.ToLower(content)

	categoryKeywords := map[string][]string{
		"implementation": {"implement", "create", "build", "develop", "code", "program"},
		"testing":        {"test", "verify", "validate", "check", "ensure", "confirm"},
		"documentation":  {"document", "write", "update", "create", "readme", "docs", "spec"},
		"planning":       {"plan", "design", "analyze", "research", "investigate", "explore"},
		"review":         {"review", "evaluate", "assess", "examine", "audit", "inspect"},
		"deployment":     {"deploy", "release", "publish", "install", "configure", "setup"},
		"maintenance":    {"refactor", "optimize", "clean", "organize", "restructure", "improve"},
	}

	for category, keywords := range categoryKeywords {
		for _, keyword := range keywords {
			if strings.Contains(contentLower, keyword) {
				return category
			}
		}
	}

	return "general"
}

// analyzeTaskTags extracts tags from task content
func (g *TaskGenerator) analyzeTaskTags(content string) []string {
	var tags []string
	contentLower := strings.ToLower(content)

	tagPatterns := map[string][]string{
		"backend":     {"api", "server", "database", "service", "backend"},
		"frontend":    {"ui", "frontend", "interface", "client", "web"},
		"testing":     {"test", "spec", "verify", "validate"},
		"docs":        {"documentation", "readme", "docs", "guide"},
		"security":    {"security", "auth", "permission", "secure"},
		"performance": {"performance", "optimize", "speed", "efficient"},
		"integration": {"integration", "connect", "interface", "link"},
	}

	for tag, patterns := range tagPatterns {
		for _, pattern := range patterns {
			if strings.Contains(contentLower, pattern) {
				tags = append(tags, tag)
				break
			}
		}
	}

	return tags
}

// parseAcceptanceCriteria parses acceptance criteria from content
func (g *TaskGenerator) parseAcceptanceCriteria(content string) []string {
	var criteria []string
	lines := strings.Split(content, "\n")

	listPattern := regexp.MustCompile(`^- (.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := listPattern.FindStringSubmatch(line); matches != nil {
			criteria = append(criteria, strings.TrimSpace(matches[1]))
		} else if line != "" && !strings.HasPrefix(line, "#") {
			// Handle paragraph-style criteria
			sentences := strings.Split(line, ".")
			for _, sentence := range sentences {
				sentence = strings.TrimSpace(sentence)
				if len(sentence) > 10 { // Ignore very short fragments
					criteria = append(criteria, sentence)
				}
			}
		}
	}

	return criteria
}

// parseRequirements parses requirements from content
func (g *TaskGenerator) parseRequirements(content string) []string {
	return g.parseAcceptanceCriteria(content) // Same parsing logic
}

// parseDependencies parses dependencies from content
func (g *TaskGenerator) parseDependencies(content string) []string {
	var dependencies []string
	lines := strings.Split(content, "\n")

	// Look for FEAT-XXX, BUG-XXX, etc. patterns
	depPattern := regexp.MustCompile(`([A-Z]+-\d+)`)

	for _, line := range lines {
		matches := depPattern.FindAllString(line, -1)
		dependencies = append(dependencies, matches...)
	}

	return dependencies
}

// calculateEstimate calculates time estimate for a task
func (g *TaskGenerator) calculateEstimate(task *EnhancedTodo) time.Duration {
	baseDuration := task.Complexity.GetEstimatedMinutes()

	// Apply complexity multiplier
	if multiplier, ok := g.config.ComplexityMultiplier[task.Complexity]; ok {
		baseDuration = int(float64(baseDuration) * multiplier)
	}

	// Apply category multiplier
	if task.Category != "" {
		if multiplier, ok := g.config.CategoryMultiplier[task.Category]; ok {
			baseDuration = int(float64(baseDuration) * multiplier)
		}
	}

	return time.Duration(baseDuration) * time.Minute
}

// orderTasks orders tasks based on configuration
func (g *TaskGenerator) orderTasks(tasks []*EnhancedTodo) []*EnhancedTodo {
	// Create a copy to avoid modifying original
	ordered := make([]*EnhancedTodo, len(tasks))
	copy(ordered, tasks)

	if g.config.OrderByDependency {
		// Simple dependency ordering (tasks with no deps first)
		sort.SliceStable(ordered, func(i, j int) bool {
			return len(ordered[i].Dependencies) < len(ordered[j].Dependencies)
		})
	}

	if g.config.OrderByPriority {
		// Secondary sort by priority (stable sort preserves dependency order)
		sort.SliceStable(ordered, func(i, j int) bool {
			return ordered[i].Priority.GetWeight() > ordered[j].Priority.GetWeight()
		})
	}

	return ordered
}

// GenerateTasksFromReader generates tasks from an io.Reader containing issue content
func (g *TaskGenerator) GenerateTasksFromReader(reader io.Reader) ([]*EnhancedTodo, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	return g.GenerateTasksFromIssue(string(content))
}

// EstimateIssueEffort estimates total effort for an issue
func (g *TaskGenerator) EstimateIssueEffort(issueContent string) (time.Duration, error) {
	tasks, err := g.GenerateTasksFromIssue(issueContent)
	if err != nil {
		return 0, err
	}

	var total time.Duration
	for _, task := range tasks {
		total += task.GetEstimatedDuration()
	}

	return total, nil
}
