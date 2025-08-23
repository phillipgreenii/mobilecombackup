package agents

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Constants
const yamlDelimiter = "---"

// AgentProcessor handles loading, parsing, and processing agent templates
type AgentProcessor struct {
	registry *TemplateRegistry
	stats    ProcessingStats
}

// NewAgentProcessor creates a new agent processor
func NewAgentProcessor(basePath string) *AgentProcessor {
	return &AgentProcessor{
		registry: NewTemplateRegistry(basePath),
		stats:    ProcessingStats{},
	}
}

// LoadAgentFromFile loads an agent definition from a file
func (p *AgentProcessor) LoadAgentFromFile(filePath string) (*AgentDefinition, error) {
	file, err := os.Open(filePath) // #nosec G304 - File path is from controlled source
	if err != nil {
		return nil, fmt.Errorf("failed to open agent file %s: %w", filePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file %s: %v\n", filePath, err)
		}
	}()

	content, metadata, err := p.parseAgentFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse agent file %s: %w", filePath, err)
	}

	agent := &AgentDefinition{
		Metadata: metadata,
		Content:  content,
		FilePath: filePath,
		Resolved: false,
	}

	// Store in appropriate registry
	if agent.IsTemplate() {
		p.registry.templates[agent.Metadata.Name] = agent
		p.stats.TemplatesLoaded++
	} else {
		p.registry.agents[agent.Metadata.Name] = agent
	}

	return agent, nil
}

// LoadAgentsFromDirectory loads all agent definitions from a directory
func (p *AgentProcessor) LoadAgentsFromDirectory(dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			// Check if file starts with YAML frontmatter before trying to load
			if p.isAgentFile(path) {
				_, err := p.LoadAgentFromFile(path)
				if err != nil {
					return fmt.Errorf("failed to load agent from %s: %w", path, err)
				}
			}
		}

		return nil
	})
}

// isAgentFile checks if a markdown file has YAML frontmatter
func (p *AgentProcessor) isAgentFile(filePath string) bool {
	file, err := os.Open(filePath) // #nosec G304 - File path is from controlled source
	if err != nil {
		return false
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file %s: %v\n", filePath, err)
		}
	}()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		firstLine := strings.TrimSpace(scanner.Text())
		return firstLine == yamlDelimiter
	}

	return false
}

// parseAgentFile parses an agent file and extracts YAML frontmatter and content
func (p *AgentProcessor) parseAgentFile(file *os.File) (content string, metadata AgentMetadata, err error) {
	scanner := bufio.NewScanner(file)

	// First line should be yamlDelimiter
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != yamlDelimiter {
		return "", metadata, fmt.Errorf("agent file must start with YAML frontmatter delimiter '%s'", yamlDelimiter)
	}

	// Read YAML frontmatter
	var yamlLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == yamlDelimiter {
			break
		}
		yamlLines = append(yamlLines, line)
	}

	// Parse YAML
	yamlContent := strings.Join(yamlLines, "\n")
	if err := yaml.Unmarshal([]byte(yamlContent), &metadata); err != nil {
		return "", metadata, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// Read markdown content
	var contentLines []string
	for scanner.Scan() {
		contentLines = append(contentLines, scanner.Text())
	}

	content = strings.Join(contentLines, "\n")
	return content, metadata, scanner.Err()
}

// ResolveAgent resolves template inheritance for an agent
func (p *AgentProcessor) ResolveAgent(agentName string) (*AgentDefinition, error) {
	start := time.Now()
	defer func() {
		p.stats.ProcessingTime += time.Since(start)
	}()

	agent, exists := p.registry.agents[agentName]
	if !exists {
		return nil, fmt.Errorf("agent '%s' not found", agentName)
	}

	if agent.Resolved {
		return agent, nil
	}

	resolvedAgent, err := p.resolveAgentRecursive(agent, make(map[string]bool))
	if err != nil {
		return nil, err
	}

	resolvedAgent.Resolved = true
	p.registry.agents[agentName] = resolvedAgent
	p.stats.AgentsProcessed++

	return resolvedAgent, nil
}

// resolveAgentRecursive recursively resolves template inheritance
func (p *AgentProcessor) resolveAgentRecursive(agent *AgentDefinition, visited map[string]bool) (*AgentDefinition, error) {
	// Check for circular dependencies
	if visited[agent.Metadata.Name] {
		return nil, fmt.Errorf("circular dependency detected in agent inheritance chain for '%s'", agent.Metadata.Name)
	}

	// If no parent, return as-is
	if !agent.HasParent() {
		return agent, nil
	}

	visited[agent.Metadata.Name] = true
	p.stats.InheritanceChains++

	// Get parent template
	parentTemplate, exists := p.registry.templates[agent.GetParentName()]
	if !exists {
		return nil, fmt.Errorf("template '%s' not found (required by agent '%s')",
			agent.GetParentName(), agent.Metadata.Name)
	}

	// Resolve parent if it has a parent (templates can't extend templates per validation,
	// but we'll handle it for robustness)
	resolvedParent := parentTemplate
	if parentTemplate.HasParent() {
		var err error
		resolvedParent, err = p.resolveAgentRecursive(parentTemplate, visited)
		if err != nil {
			return nil, err
		}
	}

	// Merge parent and child
	mergedAgent := p.mergeAgentDefinitions(resolvedParent, agent)

	delete(visited, agent.Metadata.Name)
	return mergedAgent, nil
}

// mergeAgentDefinitions merges a child agent with its parent template
func (p *AgentProcessor) mergeAgentDefinitions(parent, child *AgentDefinition) *AgentDefinition {
	merged := &AgentDefinition{
		FilePath: child.FilePath,
		Resolved: false,
	}

	// Start with parent metadata
	merged.Metadata = parent.Metadata

	// Override with child metadata
	if child.Metadata.Name != "" {
		merged.Metadata.Name = child.Metadata.Name
	}
	if child.Metadata.Description != "" {
		merged.Metadata.Description = child.Metadata.Description
	}

	// Merge tools (union of parent tools and child additional tools)
	toolsSet := make(map[string]bool)

	// Add parent tools
	for _, tool := range parent.Metadata.Tools {
		toolsSet[tool] = true
	}

	// Add child tools and additional tools
	for _, tool := range child.Metadata.Tools {
		toolsSet[tool] = true
	}
	for _, tool := range child.Metadata.AdditionalTools {
		toolsSet[tool] = true
	}

	// Convert back to slice
	merged.Metadata.Tools = make([]string, 0, len(toolsSet))
	for tool := range toolsSet {
		merged.Metadata.Tools = append(merged.Metadata.Tools, tool)
	}

	// Apply overrides
	if child.Metadata.Overrides.Model != "" {
		merged.Metadata.Model = child.Metadata.Overrides.Model
	} else if child.Metadata.Model != "" {
		merged.Metadata.Model = child.Metadata.Model
	}

	if child.Metadata.Overrides.Color != "" {
		merged.Metadata.Color = child.Metadata.Overrides.Color
	} else if child.Metadata.Color != "" {
		merged.Metadata.Color = child.Metadata.Color
	}

	// Handle tools override (replaces all tools if specified)
	if len(child.Metadata.Overrides.Tools) > 0 {
		merged.Metadata.Tools = make([]string, len(child.Metadata.Overrides.Tools))
		copy(merged.Metadata.Tools, child.Metadata.Overrides.Tools)
	}

	// Merge content - child content takes precedence, but we can append if needed
	if child.Content != "" {
		merged.Content = child.Content
	} else {
		merged.Content = parent.Content
	}

	// Child type always wins (agents extending templates become agents)
	if child.Metadata.Type != "" {
		merged.Metadata.Type = child.Metadata.Type
	}

	// Clear inheritance-related fields in merged result
	merged.Metadata.Extends = ""
	merged.Metadata.AdditionalTools = nil
	merged.Metadata.Overrides = AgentOverrides{}

	return merged
}

// ValidateAgents validates all loaded agents and templates
func (p *AgentProcessor) ValidateAgents() *ValidationResult {
	overallResult := &ValidationResult{IsValid: true}

	// Validate templates first
	for name, template := range p.registry.templates {
		result := template.Validate()
		if !result.IsValid {
			overallResult.IsValid = false
			for _, err := range result.Errors {
				err.AgentName = name
				overallResult.Errors = append(overallResult.Errors, err)
			}
		}
		overallResult.Warnings = append(overallResult.Warnings, result.Warnings...)
	}

	// Validate agents
	for name, agent := range p.registry.agents {
		result := agent.Validate()
		if !result.IsValid {
			overallResult.IsValid = false
			for _, err := range result.Errors {
				err.AgentName = name
				overallResult.Errors = append(overallResult.Errors, err)
			}
		}
		overallResult.Warnings = append(overallResult.Warnings, result.Warnings...)

		// Additional validation for inheritance
		if agent.HasParent() {
			if _, exists := p.registry.templates[agent.GetParentName()]; !exists {
				overallResult.IsValid = false
				overallResult.Errors = append(overallResult.Errors, ValidationError{
					AgentName: name,
					Field:     "extends",
					Message:   fmt.Sprintf("template '%s' not found", agent.GetParentName()),
				})
			}
		}
	}

	p.stats.ValidationErrors = len(overallResult.Errors)
	return overallResult
}

// GetProcessingStats returns processing statistics
func (p *AgentProcessor) GetProcessingStats() ProcessingStats {
	return p.stats
}

// GetRegistry returns the template registry (read-only access)
func (p *AgentProcessor) GetRegistry() *TemplateRegistry {
	return p.registry
}

// ResolveAllAgents resolves template inheritance for all loaded agents
func (p *AgentProcessor) ResolveAllAgents() error {
	for name := range p.registry.agents {
		_, err := p.ResolveAgent(name)
		if err != nil {
			return fmt.Errorf("failed to resolve agent '%s': %w", name, err)
		}
	}
	return nil
}
