// Package agents provides types and functions for agent template inheritance and processing.
package agents

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/types"
	"gopkg.in/yaml.v3"
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

// DocSyncStateManager implements state management for documentation synchronization
type DocSyncStateManager struct {
	statePath    string
	currentState *DocSyncState
	mutex        sync.RWMutex
	logger       Logger
}

// Logger interface for documentation sync logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// DocSyncState represents the current state of documentation synchronization
type DocSyncState struct {
	Config       DocSyncConfig  `yaml:"config"`
	ActiveAgents []string       `yaml:"active_agents"`
	SyncStatus   DocSyncStatus  `yaml:"sync_status"`
	ErrorHistory []DocSyncError `yaml:"error_history"`
	Metrics      DocSyncMetrics `yaml:"metrics"`
	Metadata     StateMetadata  `yaml:"metadata"`
}

// DocSyncConfig contains configuration for documentation synchronization
type DocSyncConfig struct {
	Enabled         bool     `yaml:"enabled"`
	WatchMode       bool     `yaml:"watch_mode"`
	AutoFix         bool     `yaml:"auto_fix"`
	EnabledAgents   []string `yaml:"enabled_agents"`
	AgentTimeout    int      `yaml:"agent_timeout"`
	IncludePatterns []string `yaml:"include_patterns"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
	MaxConcurrency  int      `yaml:"max_concurrency"`
	BatchSize       int      `yaml:"batch_size"`
}

// DocSyncStatus represents the synchronization status
type DocSyncStatus string

const (
	DocSyncStatusIdle      DocSyncStatus = "idle"
	DocSyncStatusRunning   DocSyncStatus = "running"
	DocSyncStatusCompleted DocSyncStatus = "completed"
	DocSyncStatusFailed    DocSyncStatus = "failed"
	DocSyncStatusPartial   DocSyncStatus = "partial"
)

// DocSyncError represents a synchronization error
type DocSyncError struct {
	Timestamp int64  `yaml:"timestamp"`
	Message   string `yaml:"message"`
	Source    string `yaml:"source"`
	Severity  string `yaml:"severity"`
}

// DocSyncMetrics contains synchronization metrics
type DocSyncMetrics struct {
	DocumentationCoverage float64 `yaml:"documentation_coverage"`
	ConsistencyScore      float64 `yaml:"consistency_score"`
	QualityScore          float64 `yaml:"quality_score"`
	LastSyncTime          int64   `yaml:"last_sync_time"`
	TotalDocuments        int     `yaml:"total_documents"`
	SyncedDocuments       int     `yaml:"synced_documents"`
}

// StateMetadata contains metadata about the state
type StateMetadata struct {
	Status      StateStatus `yaml:"status"`
	LastUpdated int64       `yaml:"last_updated"`
	Checksum    string      `yaml:"checksum"`
	Version     string      `yaml:"version"`
}

// StateStatus represents the status of the state
type StateStatus string

const (
	StateStatusUninitialized StateStatus = "uninitialized"
	StateStatusLoading       StateStatus = "loading"
	StateStatusPersisting    StateStatus = "persisting"
	StateStatusReady         StateStatus = "ready"
)

// NewDocSyncStateManager creates a new documentation synchronization state manager
func NewDocSyncStateManager(statePath string, logger Logger) *DocSyncStateManager {
	return &DocSyncStateManager{
		statePath: statePath,
		currentState: &DocSyncState{
			Config:       DefaultDocSyncConfig(),
			ActiveAgents: make([]string, 0),
			SyncStatus:   DocSyncStatusIdle,
			ErrorHistory: make([]DocSyncError, 0),
			Metrics:      DocSyncMetrics{},
			Metadata: StateMetadata{
				Status:  StateStatusUninitialized,
				Version: "1.0.0",
			},
		},
		logger: logger,
	}
}

// Get retrieves the current documentation synchronization state
func (sm *DocSyncStateManager) Get() types.Result[*DocSyncState] {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if sm.currentState == nil {
		return types.NewResultError[*DocSyncState](errors.New("state not initialized"))
	}

	// Return a copy to prevent external mutations
	stateCopy := *sm.currentState
	return types.NewResult(&stateCopy)
}

// Set updates the documentation synchronization state with validation
func (sm *DocSyncStateManager) Set(state *DocSyncState) types.Result[*DocSyncState] {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Validate the state before setting
	if err := sm.validateState(state); err != nil {
		return types.NewResultError[*DocSyncState](err)
	}

	// Update metadata
	state.Metadata.LastUpdated = time.Now().UTC().UnixMilli()
	state.Metadata.Status = StateStatusReady

	// Calculate and set checksum
	checksum, err := sm.calculateChecksum(state)
	if err != nil {
		return types.NewResultError[*DocSyncState](fmt.Errorf("failed to calculate checksum: %w", err))
	}
	state.Metadata.Checksum = checksum

	sm.currentState = state

	sm.logger.Info("Documentation sync state updated successfully")
	return types.NewResult(sm.currentState)
}

// Reset restores documentation synchronization state to defaults
func (sm *DocSyncStateManager) Reset() types.Result[*DocSyncState] {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	defaultState := &DocSyncState{
		Config:       DefaultDocSyncConfig(),
		ActiveAgents: make([]string, 0),
		SyncStatus:   DocSyncStatusIdle,
		ErrorHistory: make([]DocSyncError, 0),
		Metrics:      DocSyncMetrics{},
		Metadata: StateMetadata{
			Status:      StateStatusReady,
			LastUpdated: time.Now().UTC().UnixMilli(),
			Version:     "1.0.0",
		},
	}

	sm.currentState = defaultState

	sm.logger.Info("Documentation sync state reset to defaults")
	return types.NewResult(sm.currentState)
}

// Persist saves the current state to storage
func (sm *DocSyncStateManager) Persist() types.Result[bool] {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if sm.currentState == nil {
		return types.NewResultError[bool](errors.New("no state to persist"))
	}

	// Update persistence metadata
	sm.currentState.Metadata.Status = StateStatusPersisting

	// Convert state to YAML for persistence
	data, err := yaml.Marshal(sm.currentState)
	if err != nil {
		return types.NewResultError[bool](fmt.Errorf("failed to marshal state: %w", err))
	}

	// Write to file with proper permissions
	err = os.WriteFile(sm.statePath, data, 0600)
	if err != nil {
		return types.NewResultError[bool](fmt.Errorf("failed to write state file: %w", err))
	}

	// Update metadata after successful persistence
	sm.currentState.Metadata.Status = StateStatusReady

	sm.logger.Info("Documentation sync state persisted successfully", "path", sm.statePath)
	return types.NewResult(true)
}

// Load restores state from storage
func (sm *DocSyncStateManager) Load() types.Result[*DocSyncState] {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Update metadata to loading status
	if sm.currentState != nil {
		sm.currentState.Metadata.Status = StateStatusLoading
	}

	// Check if state file exists
	if _, err := os.Stat(sm.statePath); os.IsNotExist(err) {
		// No state file exists, use defaults
		result := sm.reset() // internal method without mutex
		sm.logger.Info("No state file found, using defaults")
		return result
	}

	// Read state file
	data, err := os.ReadFile(sm.statePath)
	if err != nil {
		return types.NewResultError[*DocSyncState](fmt.Errorf("failed to read state file: %w", err))
	}

	// Parse YAML state
	var loadedState DocSyncState
	err = yaml.Unmarshal(data, &loadedState)
	if err != nil {
		return types.NewResultError[*DocSyncState](fmt.Errorf("failed to parse state file: %w", err))
	}

	// Validate loaded state
	if err := sm.validateState(&loadedState); err != nil {
		sm.logger.Warn("Loaded state is invalid, using defaults", "error", err.Error())
		return sm.reset() // internal method without mutex
	}

	// Verify checksum if present
	if loadedState.Metadata.Checksum != "" {
		expectedChecksum, err := sm.calculateChecksum(&loadedState)
		if err != nil {
			sm.logger.Warn("Failed to verify state checksum", "error", err.Error())
		} else if expectedChecksum != loadedState.Metadata.Checksum {
			sm.logger.Warn("State checksum mismatch, data may be corrupted")
		}
	}

	// Update metadata
	loadedState.Metadata.Status = StateStatusReady
	sm.currentState = &loadedState

	sm.logger.Info("Documentation sync state loaded successfully")
	return types.NewResult(sm.currentState)
}

// Validate checks state consistency
func (sm *DocSyncStateManager) Validate() types.Result[bool] {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if sm.currentState == nil {
		return types.NewResultError[bool](errors.New("no state to validate"))
	}

	err := sm.validateState(sm.currentState)
	if err != nil {
		return types.NewResultError[bool](err)
	}

	return types.NewResult(true)
}

// validateState performs comprehensive state validation
func (sm *DocSyncStateManager) validateState(state *DocSyncState) error {
	if state == nil {
		return errors.New("state cannot be nil")
	}

	// Validate configuration
	if err := sm.validateConfig(&state.Config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Validate sync status
	validStatuses := []DocSyncStatus{
		DocSyncStatusIdle,
		DocSyncStatusRunning,
		DocSyncStatusCompleted,
		DocSyncStatusFailed,
		DocSyncStatusPartial,
	}
	isValidStatus := false
	for _, status := range validStatuses {
		if state.SyncStatus == status {
			isValidStatus = true
			break
		}
	}
	if !isValidStatus {
		return fmt.Errorf("invalid sync status: %s", state.SyncStatus)
	}

	// Validate metrics
	if state.Metrics.DocumentationCoverage < 0 || state.Metrics.DocumentationCoverage > 1 {
		return errors.New("documentation coverage must be between 0 and 1")
	}
	if state.Metrics.ConsistencyScore < 0 || state.Metrics.ConsistencyScore > 1 {
		return errors.New("consistency score must be between 0 and 1")
	}
	if state.Metrics.QualityScore < 0 || state.Metrics.QualityScore > 1 {
		return errors.New("quality score must be between 0 and 1")
	}

	return nil
}

// validateConfig validates the documentation synchronization configuration
func (sm *DocSyncStateManager) validateConfig(config *DocSyncConfig) error {
	if config == nil {
		return errors.New("configuration cannot be nil")
	}

	// Validate timeout
	if config.AgentTimeout <= 0 {
		return errors.New("agent timeout must be positive")
	}

	// Validate concurrency settings
	if config.MaxConcurrency <= 0 {
		return errors.New("max concurrency must be positive")
	}
	if config.BatchSize <= 0 {
		return errors.New("batch size must be positive")
	}

	// Validate patterns
	for _, pattern := range config.IncludePatterns {
		if _, err := filepath.Match(pattern, "test"); err != nil {
			return fmt.Errorf("invalid include pattern '%s': %w", pattern, err)
		}
	}
	for _, pattern := range config.ExcludePatterns {
		if _, err := filepath.Match(pattern, "test"); err != nil {
			return fmt.Errorf("invalid exclude pattern '%s': %w", pattern, err)
		}
	}

	return nil
}

// calculateChecksum calculates SHA-256 checksum for state integrity verification
func (sm *DocSyncStateManager) calculateChecksum(state *DocSyncState) (string, error) {
	// Create a copy without metadata for checksum calculation
	stateCopy := *state
	stateCopy.Metadata.Checksum = ""
	stateCopy.Metadata.LastUpdated = 0

	data, err := json.Marshal(stateCopy)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// reset is an internal method to reset state without mutex (used when mutex is already held)
func (sm *DocSyncStateManager) reset() types.Result[*DocSyncState] {
	defaultState := &DocSyncState{
		Config:       DefaultDocSyncConfig(),
		ActiveAgents: make([]string, 0),
		SyncStatus:   DocSyncStatusIdle,
		ErrorHistory: make([]DocSyncError, 0),
		Metrics:      DocSyncMetrics{},
		Metadata: StateMetadata{
			Status:      StateStatusReady,
			LastUpdated: time.Now().UTC().UnixMilli(),
			Version:     "1.0.0",
		},
	}

	sm.currentState = defaultState
	return types.NewResult(sm.currentState)
}

// DefaultDocSyncConfig returns the default configuration for documentation synchronization
func DefaultDocSyncConfig() DocSyncConfig {
	return DocSyncConfig{
		Enabled:   true,
		WatchMode: false,
		AutoFix:   false,
		EnabledAgents: []string{
			"analyzer",
			"codesync",
			"quality",
		},
		AgentTimeout: 300, // 5 minutes
		IncludePatterns: []string{
			"*.md",
			"*.go",
			"*.yaml",
			"*.yml",
		},
		ExcludePatterns: []string{
			"vendor/**",
			"node_modules/**",
			".git/**",
			"tmp/**",
		},
		MaxConcurrency: 4,
		BatchSize:      10,
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
