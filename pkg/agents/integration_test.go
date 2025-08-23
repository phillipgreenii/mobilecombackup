package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentProcessor_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	// Create a base template
	baseTemplateContent := `---
name: base-test-template
type: template
description: Base template for testing
model: sonnet
color: blue
tools:
  - Bash
  - Read
  - Write
---

# Base Test Template

This is a base template for testing inheritance.

## Core Behaviors

- Standard verification workflow
- Quality assurance requirements
- Tool usage preferences

## Shared Functionality

Base functionality that all agents inherit.`

	baseTemplatePath := filepath.Join(tempDir, "base-test-template.md")
	err := os.WriteFile(baseTemplatePath, []byte(baseTemplateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create base template: %v", err)
	}

	// Create a child agent that extends the template
	childAgentContent := `---
name: child-test-agent
description: Child agent for testing inheritance
extends: base-test-template
additional-tools:
  - Edit
  - MultiEdit
overrides:
  model: opus
  color: green
---

# Child Test Agent

This agent extends the base template.

## Specialized Behavior

- Additional functionality beyond the base template
- Specific tools and workflows

## Custom Implementation

Agent-specific implementation details.`

	childAgentPath := filepath.Join(tempDir, "child-test-agent.md")
	err = os.WriteFile(childAgentPath, []byte(childAgentContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create child agent: %v", err)
	}

	// Test the end-to-end workflow
	processor := NewAgentProcessor(tempDir)

	// Load template
	_, err = processor.LoadAgentFromFile(baseTemplatePath)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	// Load agent
	_, err = processor.LoadAgentFromFile(childAgentPath)
	if err != nil {
		t.Fatalf("Failed to load agent: %v", err)
	}

	// Validate all agents
	validationResult := processor.ValidateAgents()
	if !validationResult.IsValid {
		t.Fatalf("Validation failed: %v", validationResult.Errors)
	}

	// Resolve the child agent
	resolvedAgent, err := processor.ResolveAgent("child-test-agent")
	if err != nil {
		t.Fatalf("Failed to resolve agent: %v", err)
	}

	// Verify inheritance worked correctly
	if resolvedAgent.Metadata.Name != "child-test-agent" {
		t.Errorf("Expected name 'child-test-agent', got %s", resolvedAgent.Metadata.Name)
	}

	if resolvedAgent.Metadata.Model != "opus" {
		t.Errorf("Expected model 'opus', got %s", resolvedAgent.Metadata.Model)
	}

	if resolvedAgent.Metadata.Color != "green" {
		t.Errorf("Expected color 'green', got %s", resolvedAgent.Metadata.Color)
	}

	// Check that tools were merged
	expectedTools := map[string]bool{
		"Bash": true, "Read": true, "Write": true, "Edit": true, "MultiEdit": true,
	}

	if len(resolvedAgent.Metadata.Tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(resolvedAgent.Metadata.Tools))
	}

	for _, tool := range resolvedAgent.Metadata.Tools {
		if !expectedTools[tool] {
			t.Errorf("Unexpected tool: %s", tool)
		}
	}

	// Verify content is from child (child takes precedence)
	if !strings.Contains(resolvedAgent.Content, "Child Test Agent") {
		t.Error("Expected child content, but got different content")
	}

	// Verify inheritance fields are cleared
	if resolvedAgent.Metadata.Extends != "" {
		t.Error("Expected extends to be cleared after resolution")
	}

	if len(resolvedAgent.Metadata.AdditionalTools) != 0 {
		t.Error("Expected additional-tools to be cleared after resolution")
	}

	// Verify resolved flag is set
	if !resolvedAgent.Resolved {
		t.Error("Expected agent to be marked as resolved")
	}

	// Get processing stats
	stats := processor.GetProcessingStats()
	if stats.TemplatesLoaded != 1 {
		t.Errorf("Expected 1 template loaded, got %d", stats.TemplatesLoaded)
	}

	if stats.AgentsProcessed != 1 {
		t.Errorf("Expected 1 agent processed, got %d", stats.AgentsProcessed)
	}

	if stats.InheritanceChains != 1 {
		t.Errorf("Expected 1 inheritance chain, got %d", stats.InheritanceChains)
	}
}

func TestAgentProcessor_CircularDependencyDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	// Create agents with circular dependency
	agent1Content := `---
name: agent1
extends: agent2
---
Agent 1 content`

	agent2Content := `---
name: agent2  
extends: agent1
---
Agent 2 content`

	agent1Path := filepath.Join(tempDir, "agent1.md")
	agent2Path := filepath.Join(tempDir, "agent2.md")

	err := os.WriteFile(agent1Path, []byte(agent1Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create agent1: %v", err)
	}

	err = os.WriteFile(agent2Path, []byte(agent2Content), 0644)
	if err != nil {
		t.Fatalf("Failed to create agent2: %v", err)
	}

	processor := NewAgentProcessor(tempDir)

	// Load both agents (need to treat them as templates for this test)
	// First, load as regular agents
	_, err = processor.LoadAgentFromFile(agent1Path)
	if err != nil {
		t.Fatalf("Failed to load agent1: %v", err)
	}

	_, err = processor.LoadAgentFromFile(agent2Path)
	if err != nil {
		t.Fatalf("Failed to load agent2: %v", err)
	}

	// Manually add to templates registry for circular dependency test
	agent1 := processor.registry.agents["agent1"]
	agent2 := processor.registry.agents["agent2"]
	processor.registry.templates["agent1"] = agent1
	processor.registry.templates["agent2"] = agent2

	// Try to resolve - should detect circular dependency
	_, err = processor.ResolveAgent("agent1")
	if err == nil {
		t.Fatal("Expected circular dependency error, but got none")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("Expected circular dependency error, got: %v", err)
	}
}

func TestAgentProcessor_LoadAgentsFromDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	// Create multiple agent files
	files := map[string]string{
		"template1.md": `---
name: template1
type: template
---
Template 1`,
		"template2.md": `---
name: template2  
type: template
---
Template 2`,
		"agent1.md": `---
name: agent1
extends: template1
---
Agent 1`,
		"agent2.md": `---
name: agent2
extends: template2
---
Agent 2`,
		"README.md": "This is not an agent file", // Should be ignored
	}

	// Create subdirectory with more agents
	subDir := filepath.Join(tempDir, "subdirectory")
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	files["subdirectory/agent3.md"] = `---
name: agent3
---
Agent 3`

	// Write all files
	for filename, content := range files {
		fullPath := filepath.Join(tempDir, filename)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
	}

	// Load all agents from directory
	processor := NewAgentProcessor(tempDir)
	err = processor.LoadAgentsFromDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to load agents from directory: %v", err)
	}

	// Verify all agents were loaded
	expectedTemplates := []string{"template1", "template2"}
	expectedAgents := []string{"agent1", "agent2", "agent3"}

	templates := processor.registry.ListTemplates()
	if len(templates) != len(expectedTemplates) {
		t.Errorf("Expected %d templates, got %d", len(expectedTemplates), len(templates))
	}

	agents := processor.registry.ListAgents()
	if len(agents) != len(expectedAgents) {
		t.Errorf("Expected %d agents, got %d", len(expectedAgents), len(agents))
	}

	// Test resolving all agents
	err = processor.ResolveAllAgents()
	if err != nil {
		t.Fatalf("Failed to resolve all agents: %v", err)
	}

	// Verify stats
	stats := processor.GetProcessingStats()
	if stats.TemplatesLoaded != 2 {
		t.Errorf("Expected 2 templates loaded, got %d", stats.TemplatesLoaded)
	}

	// All 3 agents should be processed (agent3 doesn't need inheritance but still gets processed)
	if stats.AgentsProcessed != 3 {
		t.Errorf("Expected 3 agents processed, got %d", stats.AgentsProcessed)
	}
}

func TestAgentGenerator_GenerateAgentFromTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	// Create a base template
	templateContent := `---
name: base-template
type: template
model: sonnet
color: blue
tools:
  - Bash
  - Read
---

# Base Template

Template content with standard behaviors.`

	templatePath := filepath.Join(tempDir, "base-template.md")
	err := os.WriteFile(templatePath, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	// Set up processor and generator
	processor := NewAgentProcessor(tempDir)
	_, err = processor.LoadAgentFromFile(templatePath)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	generator := NewAgentGenerator(processor)

	// Generate a new agent from the template
	outputPath := filepath.Join(tempDir, "generated-agent.md")
	overrides := GenerationOverrides{
		Model:           "opus",
		AdditionalTools: []string{"Write", "Edit"},
	}

	err = generator.GenerateAgentFromTemplate(
		"base-template",
		"generated-agent",
		"A generated agent for testing",
		outputPath,
		overrides,
	)
	if err != nil {
		t.Fatalf("Failed to generate agent: %v", err)
	}

	// Verify the generated file exists and has correct content
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Generated agent file does not exist")
	}

	// Load and verify the generated agent
	generatedAgent, err := processor.LoadAgentFromFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to load generated agent: %v", err)
	}

	if generatedAgent.Metadata.Name != "generated-agent" {
		t.Errorf("Expected name 'generated-agent', got %s", generatedAgent.Metadata.Name)
	}

	if generatedAgent.Metadata.Extends != "base-template" {
		t.Errorf("Expected extends 'base-template', got %s", generatedAgent.Metadata.Extends)
	}

	if generatedAgent.Metadata.Overrides.Model != "opus" {
		t.Errorf("Expected override model 'opus', got %s", generatedAgent.Metadata.Overrides.Model)
	}

	if len(generatedAgent.Metadata.AdditionalTools) != 2 {
		t.Errorf("Expected 2 additional tools, got %d", len(generatedAgent.Metadata.AdditionalTools))
	}

	// Verify generated content structure
	if !strings.Contains(generatedAgent.Content, "generated-agent") {
		t.Error("Generated content should contain agent name")
	}

	if !strings.Contains(generatedAgent.Content, "base-template") {
		t.Error("Generated content should reference parent template")
	}
}

func TestAgentTemplateSystemWithRealTemplates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with our actual template directory structure
	templatesDir := filepath.Join("..", "..", ".claude", "agents", "templates")

	// Check if templates directory exists (test should work from pkg/agents)
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		t.Skip("Templates directory not found, skipping real template test")
	}

	processor := NewAgentProcessor(templatesDir)
	err := processor.LoadAgentsFromDirectory(templatesDir)
	if err != nil {
		t.Fatalf("Failed to load real templates: %v", err)
	}

	// Validate all templates
	validationResult := processor.ValidateAgents()
	if !validationResult.IsValid {
		t.Errorf("Real template validation failed: %v", validationResult.Errors)
	}

	// Check that we loaded the expected base templates
	expectedTemplates := []string{
		"base-implementation-agent",
		"base-review-agent",
		"base-documentation-agent",
		"base-orchestration-agent",
	}

	loadedTemplates := processor.registry.ListTemplates()
	for _, expected := range expectedTemplates {
		found := false
		for _, loaded := range loadedTemplates {
			if loaded == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected template '%s' not found in loaded templates", expected)
		}
	}

	// Test template generator with real templates
	generator := NewAgentGenerator(processor)
	templates := generator.ListAvailableTemplates()

	if len(templates) == 0 {
		t.Error("No templates available for generation")
	}

	for name, description := range templates {
		if description == "" {
			t.Errorf("Template '%s' has empty description", name)
		}
	}
}
