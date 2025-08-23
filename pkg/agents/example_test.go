package agents

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSpecImplementationEngineerInheritance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test the refactored spec-implementation-engineer against base template
	templatesDir := filepath.Join("..", "..", ".claude", "agents", "templates")
	agentsDir := filepath.Join("..", "..", ".claude", "agents")

	processor := NewAgentProcessor(templatesDir)

	// Load templates
	err := processor.LoadAgentsFromDirectory(templatesDir)
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	// Load the refactored agent
	refactoredAgentPath := filepath.Join(agentsDir, "spec-implementation-engineer-v2.md")
	_, err = processor.LoadAgentFromFile(refactoredAgentPath)
	if err != nil {
		t.Fatalf("Failed to load refactored agent: %v", err)
	}

	// Validate
	validationResult := processor.ValidateAgents()
	if !validationResult.IsValid {
		t.Fatalf("Validation failed: %v", validationResult.Errors)
	}

	// Resolve the agent
	resolvedAgent, err := processor.ResolveAgent("spec-implementation-engineer")
	if err != nil {
		t.Fatalf("Failed to resolve agent: %v", err)
	}

	// Verify inheritance worked
	if resolvedAgent.Metadata.Model != "sonnet" {
		t.Errorf("Expected model 'sonnet' from base template, got %s", resolvedAgent.Metadata.Model)
	}

	if resolvedAgent.Metadata.Color != "green" {
		t.Errorf("Expected color 'green' from base template, got %s", resolvedAgent.Metadata.Color)
	}

	// Check that tools include both base tools and additional tools
	toolsMap := make(map[string]bool)
	for _, tool := range resolvedAgent.Metadata.Tools {
		toolsMap[tool] = true
	}

	// Should have base tools
	expectedBaseTools := []string{"Bash", "Read", "Write", "Edit", "MultiEdit"}
	for _, tool := range expectedBaseTools {
		if !toolsMap[tool] {
			t.Errorf("Expected base tool '%s' not found", tool)
		}
	}

	// Should have additional tools
	expectedAdditionalTools := []string{"mcp__serena__check_onboarding_performed", "mcp__serena__onboarding"}
	for _, tool := range expectedAdditionalTools {
		if !toolsMap[tool] {
			t.Errorf("Expected additional tool '%s' not found", tool)
		}
	}

	// Should have Serena MCP tools from base template
	expectedSerenaTools := []string{
		"mcp__serena__get_symbols_overview",
		"mcp__serena__find_symbol",
		"mcp__serena__replace_symbol_body",
	}
	for _, tool := range expectedSerenaTools {
		if !toolsMap[tool] {
			t.Errorf("Expected Serena MCP tool '%s' not found", tool)
		}
	}

	// Verify inheritance fields are cleared
	if resolvedAgent.Metadata.Extends != "" {
		t.Error("Expected extends to be cleared after resolution")
	}

	if len(resolvedAgent.Metadata.AdditionalTools) != 0 {
		t.Error("Expected additional-tools to be cleared after resolution")
	}

	// Content should be from the child agent
	if !strings.Contains(resolvedAgent.Content, "Specification Implementation Engineer") {
		t.Error("Expected child agent content")
	}

	t.Logf("Resolved agent has %d tools", len(resolvedAgent.Metadata.Tools))
	t.Logf("Processing successful: %s", processor.GetProcessingStats().String())
}

func TestProductDocSyncInheritanceExample(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a demo refactor of product-doc-sync using base-documentation-agent
	templatesDir := filepath.Join("..", "..", ".claude", "agents", "templates")

	processor := NewAgentProcessor(templatesDir)

	// Load templates
	err := processor.LoadAgentsFromDirectory(templatesDir)
	if err != nil {
		t.Fatalf("Failed to load templates: %v", err)
	}

	// Create a demo agent definition using inheritance
	demoAgent := &AgentDefinition{
		Metadata: AgentMetadata{
			Name:            "product-doc-sync-demo",
			Description:     "Demo of product-doc-sync using inheritance",
			Extends:         "base-documentation-agent",
			AdditionalTools: []string{"Task"}, // Add Task tool for launching other agents
		},
		Content: `# Product Documentation Sync Agent

*This agent extends the base-documentation-agent template.*

## Specialized Behavior

Ensures that product documentation remains synchronized with code changes.

### Unique Responsibilities
- Detect code changes that affect documentation
- Update specifications and README files
- Sync agent definitions and examples
- Coordinate with implementation agents`,
		FilePath: "demo-agent.md",
	}

	// Add to registry
	processor.registry.agents[demoAgent.Metadata.Name] = demoAgent

	// Resolve
	resolvedAgent, err := processor.ResolveAgent("product-doc-sync-demo")
	if err != nil {
		t.Fatalf("Failed to resolve demo agent: %v", err)
	}

	// Should inherit model and color from base template
	if resolvedAgent.Metadata.Model != "sonnet" {
		t.Errorf("Expected model 'sonnet' from base template, got %s", resolvedAgent.Metadata.Model)
	}

	if resolvedAgent.Metadata.Color != "cyan" {
		t.Errorf("Expected color 'cyan' from base template, got %s", resolvedAgent.Metadata.Color)
	}

	// Should have documentation tools plus additional Task tool
	toolsMap := make(map[string]bool)
	for _, tool := range resolvedAgent.Metadata.Tools {
		toolsMap[tool] = true
	}

	// Should have base documentation tools
	expectedTools := []string{"Read", "Write", "Edit", "Task"}
	for _, tool := range expectedTools {
		if !toolsMap[tool] {
			t.Errorf("Expected tool '%s' not found", tool)
		}
	}

	t.Logf("Demo agent resolved successfully with %d tools", len(resolvedAgent.Metadata.Tools))
}
