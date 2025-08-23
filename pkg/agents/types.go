// Package agents provides types and functions for agent template inheritance and processing.
package agents

import (
	"fmt"
	"strings"
	"time"
)

// AgentDefinition represents an agent with its metadata and content
type AgentDefinition struct {
	// Metadata from YAML frontmatter
	Metadata AgentMetadata `yaml:",inline"`

	// Content is the markdown content after the frontmatter
	Content string `yaml:"-"`

	// FilePath is the path to the agent definition file
	FilePath string `yaml:"-"`

	// Resolved indicates if template inheritance has been processed
	Resolved bool `yaml:"-"`
}

// AgentMetadata represents the YAML frontmatter of an agent definition
type AgentMetadata struct {
	Name            string         `yaml:"name"`
	Description     string         `yaml:"description,omitempty"`
	Model           string         `yaml:"model,omitempty"`
	Color           string         `yaml:"color,omitempty"`
	Tools           []string       `yaml:"tools,omitempty"`
	Type            string         `yaml:"type,omitempty"`    // "agent" or "template"
	Extends         string         `yaml:"extends,omitempty"` // Template to extend
	AdditionalTools []string       `yaml:"additional-tools,omitempty"`
	Overrides       AgentOverrides `yaml:"overrides,omitempty"`
}

// AgentOverrides allows overriding specific fields from parent template
type AgentOverrides struct {
	Model string   `yaml:"model,omitempty"`
	Color string   `yaml:"color,omitempty"`
	Tools []string `yaml:"tools,omitempty"`
}

// TemplateRegistry manages agent templates and their inheritance relationships
type TemplateRegistry struct {
	templates map[string]*AgentDefinition
	agents    map[string]*AgentDefinition
	basePath  string
}

// ValidationError represents an error in agent template validation
type ValidationError struct {
	AgentName string
	Message   string
	Field     string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("agent '%s' field '%s': %s", e.AgentName, e.Field, e.Message)
	}
	return fmt.Sprintf("agent '%s': %s", e.AgentName, e.Message)
}

// ValidationResult contains the results of agent template validation
type ValidationResult struct {
	IsValid  bool
	Errors   []ValidationError
	Warnings []string
}

// ProcessingStats contains statistics about template processing
type ProcessingStats struct {
	TemplatesLoaded   int
	AgentsProcessed   int
	InheritanceChains int
	ProcessingTime    time.Duration
	ValidationErrors  int
}

// String returns a formatted string representation of ProcessingStats
func (s ProcessingStats) String() string {
	return fmt.Sprintf("Templates: %d, Agents: %d, Chains: %d, Time: %v, Errors: %d",
		s.TemplatesLoaded, s.AgentsProcessed, s.InheritanceChains,
		s.ProcessingTime, s.ValidationErrors)
}

// AgentType represents the type of agent
type AgentType string

const (
	TypeAgent    AgentType = "agent"
	TypeTemplate AgentType = "template"
)

// String returns the string representation of AgentType
func (t AgentType) String() string {
	return string(t)
}

// IsTemplate returns true if the agent is a template
func (a *AgentDefinition) IsTemplate() bool {
	return strings.EqualFold(a.Metadata.Type, string(TypeTemplate))
}

// HasParent returns true if the agent extends another template
func (a *AgentDefinition) HasParent() bool {
	return a.Metadata.Extends != ""
}

// GetParentName returns the name of the parent template
func (a *AgentDefinition) GetParentName() string {
	return a.Metadata.Extends
}

// GetAllTools returns all tools including additional tools
func (a *AgentDefinition) GetAllTools() []string {
	allTools := make([]string, 0, len(a.Metadata.Tools)+len(a.Metadata.AdditionalTools))
	allTools = append(allTools, a.Metadata.Tools...)
	allTools = append(allTools, a.Metadata.AdditionalTools...)
	return allTools
}

// Validate performs basic validation on the agent definition
func (a *AgentDefinition) Validate() *ValidationResult {
	result := &ValidationResult{IsValid: true}

	// Required fields
	if a.Metadata.Name == "" {
		result.addError("", "name is required")
	}

	if a.Content == "" && !a.IsTemplate() {
		result.addError("content", "agent content cannot be empty")
	}

	// Model validation
	if a.Metadata.Model != "" {
		validModels := []string{"sonnet", "opus", "haiku"}
		if !contains(validModels, a.Metadata.Model) {
			result.addWarning(fmt.Sprintf("model '%s' is not in recommended list: %v",
				a.Metadata.Model, validModels))
		}
	}

	// Template validation
	if a.IsTemplate() && a.HasParent() {
		result.addError("extends", "templates cannot extend other templates (use composition instead)")
	}

	// Circular reference check (basic - parent can't be self)
	if a.Metadata.Extends == a.Metadata.Name {
		result.addError("extends", "agent cannot extend itself")
	}

	return result
}

// addError adds an error to the validation result
func (r *ValidationResult) addError(field, message string) {
	r.IsValid = false
	r.Errors = append(r.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// addWarning adds a warning to the validation result
func (r *ValidationResult) addWarning(message string) {
	r.Warnings = append(r.Warnings, message)
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// NewTemplateRegistry creates a new template registry
func NewTemplateRegistry(basePath string) *TemplateRegistry {
	return &TemplateRegistry{
		templates: make(map[string]*AgentDefinition),
		agents:    make(map[string]*AgentDefinition),
		basePath:  basePath,
	}
}

// GetTemplate returns a template by name
func (r *TemplateRegistry) GetTemplate(name string) (*AgentDefinition, bool) {
	template, exists := r.templates[name]
	return template, exists
}

// GetAgent returns an agent by name
func (r *TemplateRegistry) GetAgent(name string) (*AgentDefinition, bool) {
	agent, exists := r.agents[name]
	return agent, exists
}

// ListTemplates returns all template names
func (r *TemplateRegistry) ListTemplates() []string {
	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// ListAgents returns all agent names
func (r *TemplateRegistry) ListAgents() []string {
	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}
