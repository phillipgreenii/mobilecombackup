package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAgentDefinition_IsTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		agentType string
		expected  bool
	}{
		{"template type", "template", true},
		{"Template type", "Template", true},
		{"TEMPLATE type", "TEMPLATE", true},
		{"agent type", "agent", false},
		{"empty type", "", false},
		{"other type", "other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &AgentDefinition{
				Metadata: AgentMetadata{Type: tt.agentType},
			}
			if got := agent.IsTemplate(); got != tt.expected {
				t.Errorf("IsTemplate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgentDefinition_HasParent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		extends  string
		expected bool
	}{
		{"has parent", "base-template", true},
		{"no parent", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &AgentDefinition{
				Metadata: AgentMetadata{Extends: tt.extends},
			}
			if got := agent.HasParent(); got != tt.expected {
				t.Errorf("HasParent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAgentDefinition_GetAllTools(t *testing.T) {
	t.Parallel()

	agent := &AgentDefinition{
		Metadata: AgentMetadata{
			Tools:           []string{"Bash", "Read"},
			AdditionalTools: []string{"Write", "Edit"},
		},
	}

	tools := agent.GetAllTools()
	expected := []string{"Bash", "Read", "Write", "Edit"}

	if len(tools) != len(expected) {
		t.Errorf("GetAllTools() returned %d tools, want %d", len(tools), len(expected))
	}

	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool] = true
	}

	for _, expectedTool := range expected {
		if !toolMap[expectedTool] {
			t.Errorf("GetAllTools() missing expected tool: %s", expectedTool)
		}
	}
}

func TestAgentDefinition_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		agent     *AgentDefinition
		wantValid bool
		wantError string
	}{
		{
			name: "valid agent",
			agent: &AgentDefinition{
				Metadata: AgentMetadata{
					Name:  "test-agent",
					Model: "sonnet",
				},
				Content: "Test content",
			},
			wantValid: true,
		},
		{
			name: "missing name",
			agent: &AgentDefinition{
				Content: "Test content",
			},
			wantValid: false,
			wantError: "name is required",
		},
		{
			name: "agent without content",
			agent: &AgentDefinition{
				Metadata: AgentMetadata{
					Name: "test-agent",
				},
			},
			wantValid: false,
			wantError: "agent content cannot be empty",
		},
		{
			name: "template extending template",
			agent: &AgentDefinition{
				Metadata: AgentMetadata{
					Name:    "test-template",
					Type:    "template",
					Extends: "base-template",
				},
				Content: "Template content",
			},
			wantValid: false,
			wantError: "templates cannot extend other templates",
		},
		{
			name: "circular reference",
			agent: &AgentDefinition{
				Metadata: AgentMetadata{
					Name:    "self-referencing",
					Extends: "self-referencing",
				},
				Content: "Content",
			},
			wantValid: false,
			wantError: "agent cannot extend itself",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.agent.Validate()

			if result.IsValid != tt.wantValid {
				t.Errorf("Validate().IsValid = %v, want %v", result.IsValid, tt.wantValid)
			}

			if !tt.wantValid && tt.wantError != "" {
				found := false
				for _, err := range result.Errors {
					if strings.Contains(err.Message, tt.wantError) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Validate() expected error containing '%s', got errors: %v",
						tt.wantError, result.Errors)
				}
			}
		})
	}
}

func TestNewAgentProcessor(t *testing.T) {
	t.Parallel()

	processor := NewAgentProcessor("/test/path")

	if processor == nil {
		t.Fatal("NewAgentProcessor returned nil")
	}

	if processor.registry == nil {
		t.Error("processor.registry is nil")
	}

	if processor.registry.basePath != "/test/path" {
		t.Errorf("processor.registry.basePath = %s, want /test/path", processor.registry.basePath)
	}
}

func TestAgentProcessor_parseAgentFile(t *testing.T) {
	t.Parallel()

	// Create a temporary file with agent content
	tempDir := t.TempDir()
	agentFile := filepath.Join(tempDir, "test-agent.md")

	agentContent := `---
name: test-agent
description: Test agent for testing
model: sonnet
color: blue
tools:
  - Bash
  - Read
  - Write
type: agent
extends: base-template
additional-tools:
  - Edit
overrides:
  model: opus
---

# Test Agent

This is the content of the test agent.

## Responsibilities
- Test functionality
- Validate parsing`

	err := os.WriteFile(agentFile, []byte(agentContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	processor := NewAgentProcessor(tempDir)
	file, err := os.Open(agentFile)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("Failed to close file: %v", err)
		}
	}()

	content, metadata, err := processor.parseAgentFile(file)
	if err != nil {
		t.Fatalf("parseAgentFile failed: %v", err)
	}

	// Validate metadata
	if metadata.Name != "test-agent" {
		t.Errorf("metadata.Name = %s, want test-agent", metadata.Name)
	}

	if metadata.Description != "Test agent for testing" {
		t.Errorf("metadata.Description = %s, want 'Test agent for testing'", metadata.Description)
	}

	if metadata.Model != "sonnet" {
		t.Errorf("metadata.Model = %s, want sonnet", metadata.Model)
	}

	if len(metadata.Tools) != 3 {
		t.Errorf("len(metadata.Tools) = %d, want 3", len(metadata.Tools))
	}

	if metadata.Extends != "base-template" {
		t.Errorf("metadata.Extends = %s, want base-template", metadata.Extends)
	}

	if len(metadata.AdditionalTools) != 1 || metadata.AdditionalTools[0] != "Edit" {
		t.Errorf("metadata.AdditionalTools = %v, want [Edit]", metadata.AdditionalTools)
	}

	if metadata.Overrides.Model != "opus" {
		t.Errorf("metadata.Overrides.Model = %s, want opus", metadata.Overrides.Model)
	}

	// Validate content
	if !strings.Contains(content, "# Test Agent") {
		t.Error("Content missing expected header")
	}

	if !strings.Contains(content, "Test functionality") {
		t.Error("Content missing expected text")
	}
}

func TestAgentProcessor_LoadAgentFromFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()

	// Create test template
	templateContent := `---
name: base-test-template
type: template
model: sonnet
tools:
  - Bash
  - Read
---

# Base Template

Base template content.`

	templateFile := filepath.Join(tempDir, "base-test-template.md")
	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	// Create test agent
	agentContent := `---
name: test-agent
extends: base-test-template
additional-tools:
  - Write
---

# Test Agent

Agent content.`

	agentFile := filepath.Join(tempDir, "test-agent.md")
	err = os.WriteFile(agentFile, []byte(agentContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create agent file: %v", err)
	}

	processor := NewAgentProcessor(tempDir)

	// Load template first
	template, err := processor.LoadAgentFromFile(templateFile)
	if err != nil {
		t.Fatalf("Failed to load template: %v", err)
	}

	if !template.IsTemplate() {
		t.Error("Loaded template should be identified as template")
	}

	// Load agent
	agent, err := processor.LoadAgentFromFile(agentFile)
	if err != nil {
		t.Fatalf("Failed to load agent: %v", err)
	}

	if agent.IsTemplate() {
		t.Error("Loaded agent should not be identified as template")
	}

	if !agent.HasParent() {
		t.Error("Agent should have parent")
	}

	if agent.GetParentName() != "base-test-template" {
		t.Errorf("Agent parent = %s, want base-test-template", agent.GetParentName())
	}
}

func TestAgentProcessor_mergeAgentDefinitions(t *testing.T) {
	t.Parallel()

	parent := &AgentDefinition{
		Metadata: AgentMetadata{
			Name:  "base-template",
			Model: "sonnet",
			Color: "blue",
			Tools: []string{"Bash", "Read"},
		},
		Content: "Base content",
	}

	child := &AgentDefinition{
		Metadata: AgentMetadata{
			Name:            "child-agent",
			Description:     "Child agent",
			AdditionalTools: []string{"Write", "Edit"},
			Overrides: AgentOverrides{
				Model: "opus",
			},
		},
		Content: "Child content",
	}

	processor := NewAgentProcessor("/test")
	merged := processor.mergeAgentDefinitions(parent, child)

	// Check name override
	if merged.Metadata.Name != "child-agent" {
		t.Errorf("merged.Name = %s, want child-agent", merged.Metadata.Name)
	}

	// Check model override
	if merged.Metadata.Model != "opus" {
		t.Errorf("merged.Model = %s, want opus", merged.Metadata.Model)
	}

	// Check color inheritance
	if merged.Metadata.Color != "blue" {
		t.Errorf("merged.Color = %s, want blue", merged.Metadata.Color)
	}

	// Check tools merging
	expectedTools := map[string]bool{
		"Bash": true, "Read": true, "Write": true, "Edit": true,
	}

	if len(merged.Metadata.Tools) != len(expectedTools) {
		t.Errorf("len(merged.Tools) = %d, want %d", len(merged.Metadata.Tools), len(expectedTools))
	}

	for _, tool := range merged.Metadata.Tools {
		if !expectedTools[tool] {
			t.Errorf("Unexpected tool in merged result: %s", tool)
		}
	}

	// Check content override
	if merged.Content != "Child content" {
		t.Errorf("merged.Content = %s, want 'Child content'", merged.Content)
	}

	// Check inheritance fields are cleared
	if merged.Metadata.Extends != "" {
		t.Errorf("merged.Extends should be empty, got %s", merged.Metadata.Extends)
	}

	if len(merged.Metadata.AdditionalTools) != 0 {
		t.Errorf("merged.AdditionalTools should be empty, got %v", merged.Metadata.AdditionalTools)
	}
}

func TestProcessingStats_String(t *testing.T) {
	t.Parallel()

	stats := ProcessingStats{
		TemplatesLoaded:   3,
		AgentsProcessed:   5,
		InheritanceChains: 2,
		ProcessingTime:    100 * time.Millisecond,
		ValidationErrors:  1,
	}

	result := stats.String()
	expected := "Templates: 3, Agents: 5, Chains: 2, Time: 100ms, Errors: 1"

	if result != expected {
		t.Errorf("ProcessingStats.String() = %s, want %s", result, expected)
	}
}
