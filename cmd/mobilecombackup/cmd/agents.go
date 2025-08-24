// Package cmd contains the command-line interface for mobilecombackup
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/phillipgreenii/mobilecombackup/pkg/agents"
	"github.com/spf13/cobra"
)

var (
	// Global flags for agents command
	agentsDir        string
	templatesDir     string
	agentsOutputJSON bool

	// Generation flags
	agentName        string
	agentDescription string
	templateName     string
	outputPath       string
	overrideModel    string
	overrideColor    string
	additionalTools  []string
	toolsOverride    []string
	customContent    string
	interactive      bool
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage agent templates and generate new agents",
	Long: `Manage agent templates and generate new agents from base templates.
	
This command provides functionality to:
- Validate agent template definitions
- List available templates with descriptions
- Generate new agents from templates with customization
- Show template hierarchy and inheritance relationships
- Validate existing agent definitions

Agent templates use YAML frontmatter to define metadata and inheritance,
allowing for consistent agent creation with reduced duplication.`,
	Example: `  # List all available templates
  mobilecombackup agents list-templates
  
  # Validate all templates in directory
  mobilecombackup agents validate --templates-dir .claude/agents/templates
  
  # Generate new agent from template
  mobilecombackup agents generate --template base-implementation-agent \
    --name my-new-agent --description "Custom agent for my tasks" \
    --output-path agents/my-new-agent.md
  
  # Interactive agent generation
  mobilecombackup agents generate --interactive`,
}

var listTemplatesCmd = &cobra.Command{
	Use:   "list-templates",
	Short: "List available agent templates",
	Long: `List all available agent templates with their descriptions.
	
Shows template names, descriptions, and basic metadata to help you
choose the appropriate base template for generating new agents.`,
	RunE: runListTemplates,
}

var validateTemplatesCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate agent templates and definitions",
	Long: `Validate agent template definitions and check for common issues.
	
Performs comprehensive validation including:
- YAML frontmatter syntax validation
- Required field checking (name, description, etc.)
- Template inheritance validation
- Circular dependency detection
- Tool list validation
- Model and color validation

Returns detailed error reports for any validation failures.`,
	RunE: runValidateTemplates,
}

var generateAgentCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new agent from a template",
	Long: `Generate a new agent definition file from an existing template.
	
Creates a new agent file with YAML frontmatter that extends the specified
template. The generated agent inherits all base behaviors and tools from
the template while allowing customization through overrides.

You can specify additional tools, override model/color settings, and 
provide custom content for the generated agent.`,
	Example: `  # Generate with basic settings
  mobilecombackup agents generate --template base-implementation-agent \
    --name database-agent --description "Database management agent" \
    --output-path agents/database-agent.md
  
  # Generate with overrides and additional tools
  mobilecombackup agents generate --template base-review-agent \
    --name security-reviewer --description "Security code reviewer" \
    --override-model opus --override-color red \
    --additional-tools SecurityScanner,VulnDB \
    --output-path agents/security-reviewer.md
  
  # Interactive generation
  mobilecombackup agents generate --interactive`,
	RunE: runGenerateAgent,
}

var showHierarchyCmd = &cobra.Command{
	Use:   "show-hierarchy",
	Short: "Show template hierarchy and inheritance relationships",
	Long: `Display the template hierarchy showing inheritance relationships.
	
Shows which templates exist, which agents extend which templates,
and provides an overview of the inheritance structure to help
understand the agent ecosystem.`,
	RunE: runShowHierarchy,
}

func init() {
	// Add subcommands to agents command
	agentsCmd.AddCommand(listTemplatesCmd)
	agentsCmd.AddCommand(validateTemplatesCmd)
	agentsCmd.AddCommand(generateAgentCmd)
	agentsCmd.AddCommand(showHierarchyCmd)

	// Global flags for agents command
	agentsCmd.PersistentFlags().StringVar(&templatesDir, "templates-dir", ".claude/agents/templates", "Directory containing agent templates")
	agentsCmd.PersistentFlags().StringVar(&agentsDir, "agents-dir", ".claude/agents", "Directory containing agent definitions")
	agentsCmd.PersistentFlags().BoolVar(&agentsOutputJSON, "json", false, "Output results as JSON")

	// Flags for validate command
	validateTemplatesCmd.Flags().StringVar(&templatesDir, "templates-dir", ".claude/agents/templates", "Directory containing templates to validate")
	validateTemplatesCmd.Flags().StringVar(&agentsDir, "agents-dir", ".claude/agents", "Directory containing agents to validate")

	// Flags for generate command
	generateAgentCmd.Flags().StringVar(&templateName, "template", "", "Template name to use for generation (required unless --interactive)")
	generateAgentCmd.Flags().StringVar(&agentName, "name", "", "Name for the generated agent (required unless --interactive)")
	generateAgentCmd.Flags().StringVar(&agentDescription, "description", "", "Description for the generated agent (required unless --interactive)")
	generateAgentCmd.Flags().StringVar(&outputPath, "output-path", "", "Output file path for generated agent (required unless --interactive)")
	generateAgentCmd.Flags().StringVar(&overrideModel, "override-model", "", "Override the model setting")
	generateAgentCmd.Flags().StringVar(&overrideColor, "override-color", "", "Override the color setting")
	generateAgentCmd.Flags().StringSliceVar(&additionalTools, "additional-tools", []string{}, "Additional tools to add to the agent")
	generateAgentCmd.Flags().StringSliceVar(&toolsOverride, "tools-override", []string{}, "Override all tools (replaces template tools)")
	generateAgentCmd.Flags().StringVar(&customContent, "custom-content", "", "Custom markdown content for the agent")
	generateAgentCmd.Flags().BoolVar(&interactive, "interactive", false, "Use interactive mode for agent generation")

	// Register the agents command with root command
	rootCmd.AddCommand(agentsCmd)
}

func runListTemplates(cmd *cobra.Command, args []string) error {
	processor := agents.NewAgentProcessor(templatesDir)

	// Load templates from directory
	if err := processor.LoadAgentsFromDirectory(templatesDir); err != nil {
		return fmt.Errorf("failed to load templates from %s: %w", templatesDir, err)
	}

	generator := agents.NewAgentGenerator(processor)
	templates := generator.ListAvailableTemplates()

	if len(templates) == 0 {
		fmt.Printf("No templates found in %s\n", templatesDir)
		return nil
	}

	if agentsOutputJSON {
		return outputTemplatesJSON(templates)
	}

	return outputTemplatesText(templates)
}

func runValidateTemplates(cmd *cobra.Command, args []string) error {
	processor := agents.NewAgentProcessor(templatesDir)

	// Load templates
	if err := processor.LoadAgentsFromDirectory(templatesDir); err != nil {
		return fmt.Errorf("failed to load templates from %s: %w", templatesDir, err)
	}

	// Load agents if agents directory exists
	if _, err := os.Stat(agentsDir); err == nil {
		if err := processor.LoadAgentsFromDirectory(agentsDir); err != nil {
			return fmt.Errorf("failed to load agents from %s: %w", agentsDir, err)
		}
	}

	// Validate all agents and templates
	result := processor.ValidateAgents()

	if agentsOutputJSON {
		return outputValidationJSON(result, processor.GetProcessingStats())
	}

	return outputValidationText(result, processor.GetProcessingStats())
}

func runGenerateAgent(cmd *cobra.Command, args []string) error {
	processor := agents.NewAgentProcessor(templatesDir)

	// Load templates
	if err := processor.LoadAgentsFromDirectory(templatesDir); err != nil {
		return fmt.Errorf("failed to load templates from %s: %w", templatesDir, err)
	}

	generator := agents.NewAgentGenerator(processor)

	// Handle interactive mode
	if interactive {
		return generator.GenerateAgentInteractive()
	}

	// Validate required flags for non-interactive mode
	if templateName == "" {
		return fmt.Errorf("--template is required when not using --interactive mode")
	}
	if agentName == "" {
		return fmt.Errorf("--name is required when not using --interactive mode")
	}
	if agentDescription == "" {
		return fmt.Errorf("--description is required when not using --interactive mode")
	}
	if outputPath == "" {
		return fmt.Errorf("--output-path is required when not using --interactive mode")
	}

	// Create generation overrides
	overrides := agents.GenerationOverrides{
		Model:           overrideModel,
		Color:           overrideColor,
		AdditionalTools: additionalTools,
		ToolsOverride:   toolsOverride,
		CustomContent:   customContent,
	}

	// Generate the agent
	if err := generator.GenerateAgentFromTemplate(templateName, agentName, agentDescription, outputPath, overrides); err != nil {
		return fmt.Errorf("failed to generate agent: %w", err)
	}

	fmt.Printf("Successfully generated agent '%s' from template '%s'\n", agentName, templateName)
	fmt.Printf("Output file: %s\n", outputPath)

	return nil
}

func runShowHierarchy(cmd *cobra.Command, args []string) error {
	processor := agents.NewAgentProcessor(templatesDir)

	// Load templates
	if err := processor.LoadAgentsFromDirectory(templatesDir); err != nil {
		return fmt.Errorf("failed to load templates from %s: %w", templatesDir, err)
	}

	// Load agents if directory exists
	if _, err := os.Stat(agentsDir); err == nil {
		if err := processor.LoadAgentsFromDirectory(agentsDir); err != nil {
			return fmt.Errorf("failed to load agents from %s: %w", agentsDir, err)
		}
	}

	registry := processor.GetRegistry()
	templates := registry.ListTemplates()
	agentsList := registry.ListAgents()

	if agentsOutputJSON {
		return outputHierarchyJSON(templates, agentsList, registry)
	}

	return outputHierarchyText(templates, agentsList, registry)
}

// Output helper functions

func outputTemplatesJSON(templates map[string]string) error {
	type templateInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	var templateList []templateInfo
	for name, desc := range templates {
		templateList = append(templateList, templateInfo{
			Name:        name,
			Description: desc,
		})
	}

	// Sort by name for consistent output
	sort.Slice(templateList, func(i, j int) bool {
		return templateList[i].Name < templateList[j].Name
	})

	output := map[string]interface{}{
		"templates": templateList,
		"count":     len(templateList),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputTemplatesText(templates map[string]string) error {
	fmt.Printf("Available Templates (%d):\n\n", len(templates))

	// Sort template names for consistent output
	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		desc := templates[name]
		fmt.Printf("  %s\n", name)
		if desc != "" {
			fmt.Printf("    %s\n", desc)
		}
		fmt.Println()
	}
	return nil
}

func outputValidationJSON(result *agents.ValidationResult, stats agents.ProcessingStats) error {
	output := map[string]interface{}{
		"valid":    result.IsValid,
		"errors":   result.Errors,
		"warnings": result.Warnings,
		"stats":    stats,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputValidationText(result *agents.ValidationResult, stats agents.ProcessingStats) error {
	fmt.Printf("Validation Results:\n\n")

	if result.IsValid {
		fmt.Printf("✅ All templates and agents are valid\n\n")
	} else {
		fmt.Printf("❌ Validation failed\n\n")
	}

	if len(result.Errors) > 0 {
		fmt.Printf("Errors (%d):\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  • %s\n", err)
		}
		fmt.Println()
	}

	if len(result.Warnings) > 0 {
		fmt.Printf("Warnings (%d):\n", len(result.Warnings))
		for _, warning := range result.Warnings {
			fmt.Printf("  • %s\n", warning)
		}
		fmt.Println()
	}

	fmt.Printf("Processing Statistics:\n")
	fmt.Printf("  Templates loaded: %d\n", stats.TemplatesLoaded)
	fmt.Printf("  Agents processed: %d\n", stats.AgentsProcessed)
	fmt.Printf("  Inheritance chains: %d\n", stats.InheritanceChains)
	fmt.Printf("  Processing time: %v\n", stats.ProcessingTime)

	if !result.IsValid {
		os.Exit(1)
	}

	return nil
}

func outputHierarchyJSON(templates []string, agentsList []string, registry *agents.TemplateRegistry) error {
	hierarchyInfo := make(map[string]interface{})

	// Get template details
	templateDetails := make(map[string]interface{})
	for _, templateName := range templates {
		if template, exists := registry.GetTemplate(templateName); exists {
			templateDetails[templateName] = map[string]interface{}{
				"description": template.Metadata.Description,
				"model":       template.Metadata.Model,
				"color":       template.Metadata.Color,
				"tools":       template.Metadata.Tools,
			}
		}
	}

	// Get agent details and inheritance
	agentDetails := make(map[string]interface{})
	for _, agentName := range agentsList {
		if agent, exists := registry.GetAgent(agentName); exists {
			agentDetails[agentName] = map[string]interface{}{
				"description": agent.Metadata.Description,
				"extends":     agent.Metadata.Extends,
				"model":       agent.Metadata.Model,
				"color":       agent.Metadata.Color,
			}
		}
	}

	hierarchyInfo["templates"] = templateDetails
	hierarchyInfo["agents"] = agentDetails
	hierarchyInfo["template_count"] = len(templates)
	hierarchyInfo["agent_count"] = len(agentsList)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(hierarchyInfo)
}

func outputHierarchyText(templates []string, agentsList []string, registry *agents.TemplateRegistry) error {
	fmt.Printf("Agent Template Hierarchy:\n\n")

	// Sort templates for consistent output
	sort.Strings(templates)
	sort.Strings(agentsList)

	// Show templates
	fmt.Printf("Base Templates (%d):\n", len(templates))
	for _, templateName := range templates {
		if template, exists := registry.GetTemplate(templateName); exists {
			fmt.Printf("  📋 %s", templateName)
			if template.Metadata.Description != "" {
				fmt.Printf(" - %s", template.Metadata.Description)
			}
			fmt.Println()

			if template.Metadata.Model != "" {
				fmt.Printf("     Model: %s", template.Metadata.Model)
			}
			if template.Metadata.Color != "" {
				fmt.Printf(" | Color: %s", template.Metadata.Color)
			}
			if len(template.Metadata.Tools) > 0 {
				fmt.Printf(" | Tools: %d", len(template.Metadata.Tools))
			}
			fmt.Println()
		}
	}
	fmt.Println()

	// Show agents grouped by template
	fmt.Printf("Agents by Template (%d total):\n", len(agentsList))

	// Group agents by their parent template
	agentsByTemplate := make(map[string][]string)
	agentsWithoutTemplate := []string{}

	for _, agentName := range agentsList {
		if agent, exists := registry.GetAgent(agentName); exists {
			if agent.Metadata.Extends != "" {
				agentsByTemplate[agent.Metadata.Extends] = append(agentsByTemplate[agent.Metadata.Extends], agentName)
			} else {
				agentsWithoutTemplate = append(agentsWithoutTemplate, agentName)
			}
		}
	}

	// Show agents under their templates
	for _, templateName := range templates {
		if agents, hasAgents := agentsByTemplate[templateName]; hasAgents {
			sort.Strings(agents) // Sort agents for consistent output
			fmt.Printf("  📋 %s:\n", templateName)
			for _, agentName := range agents {
				if agent, exists := registry.GetAgent(agentName); exists {
					fmt.Printf("    └─ 🤖 %s", agentName)
					if agent.Metadata.Description != "" {
						fmt.Printf(" - %s", agent.Metadata.Description)
					}
					fmt.Println()
				}
			}
			fmt.Println()
		}
	}

	// Show agents without templates
	if len(agentsWithoutTemplate) > 0 {
		sort.Strings(agentsWithoutTemplate)
		fmt.Printf("  Standalone Agents:\n")
		for _, agentName := range agentsWithoutTemplate {
			if agent, exists := registry.GetAgent(agentName); exists {
				fmt.Printf("    🤖 %s", agentName)
				if agent.Metadata.Description != "" {
					fmt.Printf(" - %s", agent.Metadata.Description)
				}
				fmt.Println()
			}
		}
	}

	return nil
}
