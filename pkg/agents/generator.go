package agents

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// AgentGenerator provides functionality to generate new agents from templates
type AgentGenerator struct {
	processor *AgentProcessor
}

// NewAgentGenerator creates a new agent generator
func NewAgentGenerator(processor *AgentProcessor) *AgentGenerator {
	return &AgentGenerator{
		processor: processor,
	}
}

// GenerateAgentFromTemplate creates a new agent file based on a template
func (g *AgentGenerator) GenerateAgentFromTemplate(
	templateName, agentName, description, outputPath string, overrides GenerationOverrides,
) error {
	// Get the template
	template, exists := g.processor.registry.templates[templateName]
	if !exists {
		return fmt.Errorf("template '%s' not found", templateName)
	}

	// Create the new agent definition
	newAgent := &AgentDefinition{
		Metadata: AgentMetadata{
			Name:        agentName,
			Description: description,
			Extends:     templateName,
		},
	}

	// Apply overrides
	if overrides.Model != "" {
		newAgent.Metadata.Overrides.Model = overrides.Model
	}
	if overrides.Color != "" {
		newAgent.Metadata.Overrides.Color = overrides.Color
	}
	if len(overrides.AdditionalTools) > 0 {
		newAgent.Metadata.AdditionalTools = overrides.AdditionalTools
	}
	if len(overrides.ToolsOverride) > 0 {
		newAgent.Metadata.Overrides.Tools = overrides.ToolsOverride
	}

	// Generate content
	content := g.generateAgentContent(template, agentName, description, overrides)
	newAgent.Content = content

	// Write to file
	return g.writeAgentToFile(newAgent, outputPath)
}

// GenerationOverrides allows customizing the generated agent
type GenerationOverrides struct {
	Model           string   `yaml:"model,omitempty"`
	Color           string   `yaml:"color,omitempty"`
	AdditionalTools []string `yaml:"additional-tools,omitempty"`
	ToolsOverride   []string `yaml:"tools-override,omitempty"`
	CustomContent   string   `yaml:"custom-content,omitempty"`
}

// generateAgentContent creates the markdown content for a new agent
func (g *AgentGenerator) generateAgentContent(
	template *AgentDefinition, agentName, description string, overrides GenerationOverrides,
) string {
	var content strings.Builder

	if overrides.CustomContent != "" {
		return overrides.CustomContent
	}

	// Generate default content based on template
	content.WriteString(fmt.Sprintf("# %s\n\n", agentName))
	content.WriteString(fmt.Sprintf("%s\n\n", description))

	content.WriteString("## Specialized Behavior\n\n")
	content.WriteString("*This agent extends the ")
	content.WriteString(fmt.Sprintf("[%s](templates/%s.md) template", template.Metadata.Name, template.Metadata.Name))
	content.WriteString(" and inherits all its core behaviors.*\n\n")

	content.WriteString("### Unique Responsibilities\n\n")
	content.WriteString("- [Add specific responsibilities for this agent]\n")
	content.WriteString("- [Customize based on domain/requirements]\n")
	content.WriteString("- [Include any special workflows or processes]\n\n")

	content.WriteString("### Domain Expertise\n\n")
	content.WriteString("- [List domain-specific knowledge areas]\n")
	content.WriteString("- [Include relevant technologies/frameworks]\n")
	content.WriteString("- [Note any special considerations]\n\n")

	content.WriteString("### Implementation Notes\n\n")
	content.WriteString("- [Any agent-specific implementation guidelines]\n")
	content.WriteString("- [Special tools or processes to use]\n")
	content.WriteString("- [Integration points with other systems]\n\n")

	content.WriteString("## Examples\n\n")
	content.WriteString("*Add specific examples of when and how to use this agent.*\n\n")

	return content.String()
}

// writeAgentToFile writes an agent definition to a markdown file
func (g *AgentGenerator) writeAgentToFile(agent *AgentDefinition, outputPath string) error {
	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create file
	file, err := os.Create(outputPath) // #nosec G304 - Output path is controlled
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file %s: %v\n", outputPath, err)
		}
	}()

	writer := bufio.NewWriter(file)
	defer func() {
		if err := writer.Flush(); err != nil {
			fmt.Printf("Warning: failed to flush writer: %v\n", err)
		}
	}()

	// Write YAML frontmatter
	if _, err := writer.WriteString("---\n"); err != nil {
		return fmt.Errorf("failed to write frontmatter start: %w", err)
	}

	yamlData, err := yaml.Marshal(agent.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if _, err := writer.Write(yamlData); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	if _, err := writer.WriteString("---\n\n"); err != nil {
		return fmt.Errorf("failed to write frontmatter end: %w", err)
	}

	// Write content
	if _, err := writer.WriteString(agent.Content); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	return nil
}

// ValidateTemplate validates that a template can be used for generation
func (g *AgentGenerator) ValidateTemplate(templateName string) error {
	template, exists := g.processor.registry.templates[templateName]
	if !exists {
		return fmt.Errorf("template '%s' not found", templateName)
	}

	// Basic validation
	result := template.Validate()
	if !result.IsValid {
		return fmt.Errorf("template '%s' is invalid: %v", templateName, result.Errors)
	}

	return nil
}

// ListAvailableTemplates returns a list of available templates with descriptions
func (g *AgentGenerator) ListAvailableTemplates() map[string]string {
	templates := make(map[string]string)
	for name, template := range g.processor.registry.templates {
		templates[name] = template.Metadata.Description
	}
	return templates
}

// GenerateAgentInteractive provides an interactive way to generate agents
func (g *AgentGenerator) GenerateAgentInteractive() error {
	scanner := bufio.NewScanner(os.Stdin)

	// List available templates
	fmt.Println("Available templates:")
	templates := g.ListAvailableTemplates()
	for name, desc := range templates {
		fmt.Printf("  %s: %s\n", name, desc)
	}
	fmt.Println()

	// Get template selection
	fmt.Print("Select template: ")
	scanner.Scan()
	templateName := strings.TrimSpace(scanner.Text())

	if err := g.ValidateTemplate(templateName); err != nil {
		return fmt.Errorf("invalid template selection: %w", err)
	}

	// Get agent details
	fmt.Print("Agent name: ")
	scanner.Scan()
	agentName := strings.TrimSpace(scanner.Text())

	fmt.Print("Agent description: ")
	scanner.Scan()
	description := strings.TrimSpace(scanner.Text())

	fmt.Print("Output file path: ")
	scanner.Scan()
	outputPath := strings.TrimSpace(scanner.Text())

	// Get optional overrides
	var overrides GenerationOverrides

	fmt.Print("Model override (optional, press enter to skip): ")
	scanner.Scan()
	if model := strings.TrimSpace(scanner.Text()); model != "" {
		overrides.Model = model
	}

	fmt.Print("Color override (optional, press enter to skip): ")
	scanner.Scan()
	if color := strings.TrimSpace(scanner.Text()); color != "" {
		overrides.Color = color
	}

	// Generate the agent
	return g.GenerateAgentFromTemplate(templateName, agentName, description, outputPath, overrides)
}
