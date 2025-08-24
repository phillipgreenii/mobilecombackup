// Package agents provides types and functions for agent template inheritance and processing.
package agents

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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

// RBAC Implementation for FEAT-084 Task 1.2.2

// RBACController implements role-based access control
type RBACController struct {
	roles       map[string]types.SecurityRole
	permissions map[string]types.SecurityPermission
	userRoles   map[string][]string
	roleCache   map[string][]string // cached flattened permissions per role
	mu          sync.RWMutex
}

// NewRBACController creates a new RBAC controller with default roles
func NewRBACController() *RBACController {
	controller := &RBACController{
		roles:       make(map[string]types.SecurityRole),
		permissions: make(map[string]types.SecurityPermission),
		userRoles:   make(map[string][]string),
		roleCache:   make(map[string][]string),
	}

	// Initialize default permissions
	controller.initializeDefaultPermissions()
	// Initialize default roles
	controller.initializeDefaultRoles()

	return controller
}

// CheckPermission verifies if a security context has permission for a resource/action
func (r *RBACController) CheckPermission(ctx types.SecurityContext, resource, action string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if user is active
	if !ctx.User.Active {
		return false
	}

	// Check direct permissions in context
	requiredPermission := resource + ":" + action
	for _, permission := range ctx.Permissions {
		if permission == requiredPermission || permission == "*" {
			return true
		}
	}

	// Check role-based permissions
	for _, roleID := range ctx.User.Roles {
		if r.roleHasPermission(roleID, resource, action) {
			return true
		}
	}

	return false
}

// GetUserRoles retrieves all roles for a user including inherited roles
func (r *RBACController) GetUserRoles(userID string) ([]types.SecurityRole, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userRoleIDs, exists := r.userRoles[userID]
	if !exists {
		return []types.SecurityRole{}, nil
	}

	var roles []types.SecurityRole
	processedRoles := make(map[string]bool)

	// Get all roles including inherited ones
	for _, roleID := range userRoleIDs {
		r.collectRolesRecursively(roleID, &roles, processedRoles)
	}

	return roles, nil
}

// GetRolePermissions gets all permissions for a role including inherited permissions
func (r *RBACController) GetRolePermissions(roleID string) ([]types.SecurityPermission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	role, exists := r.roles[roleID]
	if !exists {
		return nil, fmt.Errorf("role not found: %s", roleID)
	}

	permissionMap := make(map[string]types.SecurityPermission)
	r.collectPermissionsRecursively(role, permissionMap)

	var permissions []types.SecurityPermission
	for _, permission := range permissionMap {
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// AssignRole assigns a role to a user
func (r *RBACController) AssignRole(userID, roleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Verify role exists
	if _, exists := r.roles[roleID]; !exists {
		return fmt.Errorf("role not found: %s", roleID)
	}

	// Get current user roles
	userRoles := r.userRoles[userID]

	// Check if role already assigned
	for _, existingRole := range userRoles {
		if existingRole == roleID {
			return nil // Already assigned
		}
	}

	// Add role
	r.userRoles[userID] = append(userRoles, roleID)

	return nil
}

// RevokeRole removes a role from a user
func (r *RBACController) RevokeRole(userID, roleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	userRoles := r.userRoles[userID]
	updatedRoles := make([]string, 0, len(userRoles))

	for _, role := range userRoles {
		if role != roleID {
			updatedRoles = append(updatedRoles, role)
		}
	}

	r.userRoles[userID] = updatedRoles

	return nil
}

// CreateRole creates a new role in the system
func (r *RBACController) CreateRole(role types.SecurityRole) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate role
	if err := r.validateRole(role); err != nil {
		return fmt.Errorf("invalid role: %w", err)
	}

	// Check if role already exists
	if _, exists := r.roles[role.ID]; exists {
		return fmt.Errorf("role already exists: %s", role.ID)
	}

	// Validate parent roles exist
	for _, parentID := range role.ParentRoles {
		if _, exists := r.roles[parentID]; !exists {
			return fmt.Errorf("parent role not found: %s", parentID)
		}
	}

	r.roles[role.ID] = role
	r.invalidateRoleCache(role.ID)

	return nil
}

// UpdateRole updates an existing role
func (r *RBACController) UpdateRole(role types.SecurityRole) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if role exists
	if _, exists := r.roles[role.ID]; !exists {
		return fmt.Errorf("role not found: %s", role.ID)
	}

	// Validate role
	if err := r.validateRole(role); err != nil {
		return fmt.Errorf("invalid role: %w", err)
	}

	r.roles[role.ID] = role
	r.invalidateRoleCache(role.ID)

	return nil
}

// DeleteRole removes a role from the system
func (r *RBACController) DeleteRole(roleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if role exists
	if _, exists := r.roles[roleID]; !exists {
		return fmt.Errorf("role not found: %s", roleID)
	}

	// Check if role is referenced by other roles
	for _, role := range r.roles {
		for _, parentID := range role.ParentRoles {
			if parentID == roleID {
				return fmt.Errorf("role is referenced by other roles, cannot delete: %s", roleID)
			}
		}
	}

	// Remove role from all users
	for userID, userRoles := range r.userRoles {
		updatedRoles := make([]string, 0, len(userRoles))
		for _, role := range userRoles {
			if role != roleID {
				updatedRoles = append(updatedRoles, role)
			}
		}
		r.userRoles[userID] = updatedRoles
	}

	delete(r.roles, roleID)
	delete(r.roleCache, roleID)

	return nil
}

// Helper methods

func (r *RBACController) roleHasPermission(roleID, resource, action string) bool {
	// Check cache first
	if cachedPerms, exists := r.roleCache[roleID]; exists {
		requiredPermission := resource + ":" + action
		for _, perm := range cachedPerms {
			if perm == requiredPermission || perm == "*" {
				return true
			}
		}
		return false
	}

	// Build cache for this role
	role, exists := r.roles[roleID]
	if !exists {
		return false
	}

	permissionMap := make(map[string]types.SecurityPermission)
	r.collectPermissionsRecursively(role, permissionMap)

	var permissions []string
	for _, permission := range permissionMap {
		permissions = append(permissions, permission.Resource+":"+permission.Action)
	}
	r.roleCache[roleID] = permissions

	// Check permission
	requiredPermission := resource + ":" + action
	for _, perm := range permissions {
		if perm == requiredPermission || perm == "*" {
			return true
		}
	}

	return false
}

func (r *RBACController) collectRolesRecursively(roleID string, roles *[]types.SecurityRole, processed map[string]bool) {
	if processed[roleID] {
		return // Avoid circular references
	}

	role, exists := r.roles[roleID]
	if !exists {
		return
	}

	processed[roleID] = true
	*roles = append(*roles, role)

	// Process parent roles
	for _, parentID := range role.ParentRoles {
		r.collectRolesRecursively(parentID, roles, processed)
	}
}

func (r *RBACController) collectPermissionsRecursively(role types.SecurityRole, permissions map[string]types.SecurityPermission) {
	// Add direct permissions
	for _, permID := range role.Permissions {
		if permission, exists := r.permissions[permID]; exists {
			permissions[permID] = permission
		}
	}

	// Add inherited permissions from parent roles
	for _, parentID := range role.ParentRoles {
		if parentRole, exists := r.roles[parentID]; exists {
			r.collectPermissionsRecursively(parentRole, permissions)
		}
	}
}

func (r *RBACController) validateRole(role types.SecurityRole) error {
	if role.ID == "" {
		return fmt.Errorf("role ID cannot be empty")
	}
	if role.Name == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	// Check for circular dependencies in parent roles
	if r.hasCircularDependency(role.ID, role.ParentRoles) {
		return fmt.Errorf("circular dependency detected in role hierarchy")
	}

	return nil
}

func (r *RBACController) hasCircularDependency(roleID string, parentRoles []string) bool {
	visited := make(map[string]bool)
	var checkCircular func(string, []string) bool

	checkCircular = func(currentID string, parents []string) bool {
		if visited[currentID] {
			return true
		}
		visited[currentID] = true

		for _, parentID := range parents {
			if parentID == roleID {
				return true
			}
			if parentRole, exists := r.roles[parentID]; exists {
				if checkCircular(parentID, parentRole.ParentRoles) {
					return true
				}
			}
		}

		visited[currentID] = false
		return false
	}

	return checkCircular(roleID, parentRoles)
}

func (r *RBACController) invalidateRoleCache(roleID string) {
	delete(r.roleCache, roleID)

	// Invalidate dependent roles
	for id, role := range r.roles {
		for _, parentID := range role.ParentRoles {
			if parentID == roleID {
				delete(r.roleCache, id)
				break
			}
		}
	}
}

func (r *RBACController) initializeDefaultPermissions() {
	defaultPermissions := []types.SecurityPermission{
		{ID: "doc.read", Name: "Read Documentation", Resource: "doc", Action: "read", Description: "Read documentation files"},
		{ID: "doc.write", Name: "Write Documentation", Resource: "doc", Action: "write", Description: "Modify documentation files"},
		{ID: "doc.sync", Name: "Sync Documentation", Resource: "doc", Action: "sync", Description: "Execute documentation synchronization"},
		{ID: "config.read", Name: "Read Configuration", Resource: "config", Action: "read", Description: "Read system configuration"},
		{ID: "config.write", Name: "Write Configuration", Resource: "config", Action: "write", Description: "Modify system configuration"},
		{ID: "audit.read", Name: "Read Audit Logs", Resource: "audit", Action: "read", Description: "Access audit log information"},
		{ID: "user.manage", Name: "Manage Users", Resource: "user", Action: "manage", Description: "Create, update, and delete users"},
		{ID: "role.manage", Name: "Manage Roles", Resource: "role", Action: "manage", Description: "Create, update, and delete roles"},
	}

	for _, permission := range defaultPermissions {
		r.permissions[permission.ID] = permission
	}
}

func (r *RBACController) initializeDefaultRoles() {
	defaultRoles := []types.SecurityRole{
		{
			ID:          "reader",
			Name:        "Reader",
			Description: "Can read documentation and configuration",
			Permissions: []string{"doc.read", "config.read"},
			ParentRoles: []string{},
		},
		{
			ID:          "writer",
			Name:        "Writer",
			Description: "Can read and write documentation",
			Permissions: []string{"doc.write"},
			ParentRoles: []string{"reader"},
		},
		{
			ID:          "maintainer",
			Name:        "Maintainer",
			Description: "Can sync documentation and manage configuration",
			Permissions: []string{"doc.sync", "config.write"},
			ParentRoles: []string{"writer"},
		},
		{
			ID:          "admin",
			Name:        "Administrator",
			Description: "Full system access including user and role management",
			Permissions: []string{"audit.read", "user.manage", "role.manage"},
			ParentRoles: []string{"maintainer"},
		},
	}

	for _, role := range defaultRoles {
		r.roles[role.ID] = role
	}
}

// Audit Logging System Implementation for FEAT-084 Task 1.2.3

// AuditLoggerImpl implements comprehensive audit logging for security events
type AuditLoggerImpl struct {
	events    []types.AuditEvent
	maxEvents int
	mu        sync.RWMutex
	logFile   string
}

// NewAuditLogger creates a new audit logger with specified maximum events
func NewAuditLogger(maxEvents int, logFile string) *AuditLoggerImpl {
	return &AuditLoggerImpl{
		events:    make([]types.AuditEvent, 0, maxEvents),
		maxEvents: maxEvents,
		logFile:   logFile,
	}
}

// LogEvent records a security event to the audit trail
func (a *AuditLoggerImpl) LogEvent(event types.AuditEvent) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Validate event
	if err := a.validateEvent(event); err != nil {
		return fmt.Errorf("invalid audit event: %w", err)
	}

	// Set timestamp if not provided
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UTC().UnixMilli()
	}

	// Generate ID if not provided
	if event.ID == "" {
		event.ID = a.generateEventID(event)
	}

	// Add to in-memory store
	a.events = append(a.events, event)

	// Maintain max events limit (FIFO)
	if len(a.events) > a.maxEvents {
		a.events = a.events[1:]
	}

	// Persist to file if configured
	if a.logFile != "" {
		return a.persistEvent(event)
	}

	return nil
}

// GetEvents retrieves audit events for a specific user within time range
func (a *AuditLoggerImpl) GetEvents(userID string, fromTime, toTime int64) ([]types.AuditEvent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var matchingEvents []types.AuditEvent

	for _, event := range a.events {
		if event.UserID == userID &&
			event.Timestamp >= fromTime &&
			event.Timestamp <= toTime {
			matchingEvents = append(matchingEvents, event)
		}
	}

	return matchingEvents, nil
}

// GetEventsByAction retrieves audit events by action type within time range
func (a *AuditLoggerImpl) GetEventsByAction(action string, fromTime, toTime int64) ([]types.AuditEvent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var matchingEvents []types.AuditEvent

	for _, event := range a.events {
		if event.Action == action &&
			event.Timestamp >= fromTime &&
			event.Timestamp <= toTime {
			matchingEvents = append(matchingEvents, event)
		}
	}

	return matchingEvents, nil
}

// GetEventsByResource retrieves audit events by resource within time range
func (a *AuditLoggerImpl) GetEventsByResource(resource string, fromTime, toTime int64) ([]types.AuditEvent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var matchingEvents []types.AuditEvent

	for _, event := range a.events {
		if event.Resource == resource &&
			event.Timestamp >= fromTime &&
			event.Timestamp <= toTime {
			matchingEvents = append(matchingEvents, event)
		}
	}

	return matchingEvents, nil
}

// SearchEvents performs a text search across audit events within time range
func (a *AuditLoggerImpl) SearchEvents(query string, fromTime, toTime int64) ([]types.AuditEvent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var matchingEvents []types.AuditEvent
	queryLower := strings.ToLower(query)

	for _, event := range a.events {
		if event.Timestamp >= fromTime && event.Timestamp <= toTime {
			if a.eventMatchesQuery(event, queryLower) {
				matchingEvents = append(matchingEvents, event)
			}
		}
	}

	return matchingEvents, nil
}

// GetEventStats returns statistics about audit events
func (a *AuditLoggerImpl) GetEventStats(fromTime, toTime int64) (map[string]interface{}, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := map[string]interface{}{
		"total_events": 0,
		"by_action":    make(map[string]int),
		"by_resource":  make(map[string]int),
		"by_result":    make(map[string]int),
		"by_user":      make(map[string]int),
		"time_range": map[string]int64{
			"from": fromTime,
			"to":   toTime,
		},
	}

	actionStats := stats["by_action"].(map[string]int)
	resourceStats := stats["by_resource"].(map[string]int)
	resultStats := stats["by_result"].(map[string]int)
	userStats := stats["by_user"].(map[string]int)

	totalEvents := 0
	for _, event := range a.events {
		if event.Timestamp >= fromTime && event.Timestamp <= toTime {
			totalEvents++
			actionStats[event.Action]++
			resourceStats[event.Resource]++
			resultStats[event.Result]++
			userStats[event.UserID]++
		}
	}

	stats["total_events"] = totalEvents
	return stats, nil
}

// Helper methods for AuditLoggerImpl

func (a *AuditLoggerImpl) validateEvent(event types.AuditEvent) error {
	if event.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}
	if event.Action == "" {
		return fmt.Errorf("action cannot be empty")
	}
	if event.Resource == "" {
		return fmt.Errorf("resource cannot be empty")
	}
	if event.Result == "" {
		return fmt.Errorf("result cannot be empty")
	}

	// Validate result is one of expected values
	validResults := map[string]bool{
		"success": true,
		"failure": true,
		"denied":  true,
		"error":   true,
	}
	if !validResults[event.Result] {
		return fmt.Errorf("invalid result value: %s", event.Result)
	}

	return nil
}

func (a *AuditLoggerImpl) generateEventID(event types.AuditEvent) string {
	// Generate unique ID based on event content and timestamp
	content := fmt.Sprintf("%s:%s:%s:%s:%d",
		event.UserID, event.Action, event.Resource, event.Result, event.Timestamp)
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes as hex
}

func (a *AuditLoggerImpl) eventMatchesQuery(event types.AuditEvent, queryLower string) bool {
	// Search in various fields
	searchFields := []string{
		strings.ToLower(event.Action),
		strings.ToLower(event.Resource),
		strings.ToLower(event.Result),
		strings.ToLower(event.UserID),
		strings.ToLower(event.IPAddress),
		strings.ToLower(event.UserAgent),
	}

	for _, field := range searchFields {
		if strings.Contains(field, queryLower) {
			return true
		}
	}

	// Search in details map
	if event.Details != nil {
		for key, value := range event.Details {
			keyLower := strings.ToLower(key)
			valueLower := strings.ToLower(fmt.Sprintf("%v", value))
			if strings.Contains(keyLower, queryLower) || strings.Contains(valueLower, queryLower) {
				return true
			}
		}
	}

	return false
}

func (a *AuditLoggerImpl) persistEvent(event types.AuditEvent) error {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(a.logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Marshal event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Append to log file with newline
	file, err := os.OpenFile(a.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(append(eventJSON, '\n')); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// Security Context Manager Implementation for FEAT-084

// SecurityContextManagerImpl manages security contexts and sessions
type SecurityContextManagerImpl struct {
	sessions       map[string]types.SecurityContext
	rbacController *RBACController
	sessionTimeout int64 // timeout in milliseconds
	mu             sync.RWMutex
}

// NewSecurityContextManager creates a new security context manager
func NewSecurityContextManager(rbacController *RBACController, sessionTimeoutMinutes int) *SecurityContextManagerImpl {
	return &SecurityContextManagerImpl{
		sessions:       make(map[string]types.SecurityContext),
		rbacController: rbacController,
		sessionTimeout: int64(sessionTimeoutMinutes * 60 * 1000), // Convert to milliseconds
	}
}

// CreateContext creates a new security context for a user
func (s *SecurityContextManagerImpl) CreateContext(user types.SecurityUser, sessionID string) (types.SecurityContext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !user.Active {
		return types.SecurityContext{}, fmt.Errorf("user is not active: %s", user.ID)
	}

	// Get user permissions from RBAC
	permissions, err := s.getUserPermissions(user.ID)
	if err != nil {
		return types.SecurityContext{}, fmt.Errorf("failed to get user permissions: %w", err)
	}

	now := time.Now().UTC().UnixMilli()
	context := types.SecurityContext{
		User:        user,
		SessionID:   sessionID,
		Permissions: permissions,
		Metadata:    make(map[string]string),
		CreatedAt:   now,
		ExpiresAt:   now + s.sessionTimeout,
	}

	s.sessions[sessionID] = context
	return context, nil
}

// ValidateContext validates a security context and returns validation result
func (s *SecurityContextManagerImpl) ValidateContext(ctx types.SecurityContext) types.SecurityValidationResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := types.SecurityValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
		Context:  "context_validation",
	}

	// Check if session exists
	storedContext, exists := s.sessions[ctx.SessionID]
	if !exists {
		result.Valid = false
		result.Errors = append(result.Errors, "session not found")
		return result
	}

	// Check if session has expired
	now := time.Now().UTC().UnixMilli()
	if storedContext.ExpiresAt < now {
		result.Valid = false
		result.Errors = append(result.Errors, "session has expired")
		return result
	}

	// Check if user is still active
	if !ctx.User.Active {
		result.Valid = false
		result.Errors = append(result.Errors, "user is not active")
		return result
	}

	// Warn if session will expire soon (within 5 minutes)
	if storedContext.ExpiresAt-now < 5*60*1000 {
		result.Warnings = append(result.Warnings, "session will expire soon")
	}

	return result
}

// RefreshContext extends the expiration time of a security context
func (s *SecurityContextManagerImpl) RefreshContext(ctx types.SecurityContext) (types.SecurityContext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	storedContext, exists := s.sessions[ctx.SessionID]
	if !exists {
		return types.SecurityContext{}, fmt.Errorf("session not found")
	}

	// Update expiration time
	now := time.Now().UTC().UnixMilli()
	storedContext.ExpiresAt = now + s.sessionTimeout

	// Refresh permissions in case roles changed
	permissions, err := s.getUserPermissions(storedContext.User.ID)
	if err != nil {
		return types.SecurityContext{}, fmt.Errorf("failed to refresh permissions: %w", err)
	}
	storedContext.Permissions = permissions

	s.sessions[ctx.SessionID] = storedContext
	return storedContext, nil
}

// RevokeContext removes a security context (logout)
func (s *SecurityContextManagerImpl) RevokeContext(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

// GetUserPermissions retrieves all permissions for a user
func (s *SecurityContextManagerImpl) GetUserPermissions(userID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getUserPermissions(userID)
}

// Helper method (called with lock held)
func (s *SecurityContextManagerImpl) getUserPermissions(userID string) ([]string, error) {
	// Get roles for user
	roles, err := s.rbacController.GetUserRoles(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Collect all permissions from roles
	permissionSet := make(map[string]bool)
	for _, role := range roles {
		for _, permissionID := range role.Permissions {
			permissionSet[permissionID] = true
		}
	}

	// Convert to slice
	var permissions []string
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// CleanupExpiredSessions removes expired sessions (should be called periodically)
func (s *SecurityContextManagerImpl) CleanupExpiredSessions() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().UnixMilli()
	expiredCount := 0

	for sessionID, context := range s.sessions {
		if context.ExpiresAt < now {
			delete(s.sessions, sessionID)
			expiredCount++
		}
	}

	return expiredCount
}

// Security Validation Framework Implementation for FEAT-084 Task 1.2.4

// SecurityValidatorImpl provides comprehensive security validation and sanitization
type SecurityValidatorImpl struct {
	// Configuration
	maxInputLength    int
	allowedHTMLTags   map[string]bool
	allowedAttributes map[string]bool
	mu                sync.RWMutex
}

// NewSecurityValidator creates a new security validator with default settings
func NewSecurityValidator() *SecurityValidatorImpl {
	return &SecurityValidatorImpl{
		maxInputLength: 10000, // 10KB max input
		allowedHTMLTags: map[string]bool{
			"p": true, "br": true, "strong": true, "em": true,
			"ul": true, "ol": true, "li": true, "h1": true,
			"h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		},
		allowedAttributes: map[string]bool{
			"class": true, "id": true, "title": true, "alt": true,
		},
	}
}

// ValidateInput validates input based on type and security policies
func (v *SecurityValidatorImpl) ValidateInput(input string, inputType string) types.SecurityValidationResult {
	v.mu.RLock()
	defer v.mu.RUnlock()

	result := types.SecurityValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
		Context:  fmt.Sprintf("input_validation_%s", inputType),
	}

	// Check input length
	if len(input) > v.maxInputLength {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("input too long: %d > %d", len(input), v.maxInputLength))
		return result
	}

	// Type-specific validation
	switch strings.ToLower(inputType) {
	case "email":
		v.validateEmail(input, &result)
	case "username":
		v.validateUsername(input, &result)
	case "password":
		v.validatePassword(input, &result)
	case "html":
		v.validateHTML(input, &result)
	case "sql":
		v.validateSQL(input, &result)
	case "shell":
		v.validateShell(input, &result)
	case "path":
		v.validatePath(input, &result)
	case "url":
		v.validateURL(input, &result)
	case "json":
		v.validateJSON(input, &result)
	case "yaml":
		v.validateYAML(input, &result)
	default:
		v.validateGeneric(input, &result)
	}

	return result
}

// SanitizeInput cleans input based on type while preserving functionality
func (v *SecurityValidatorImpl) SanitizeInput(input string, inputType string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	switch strings.ToLower(inputType) {
	case "html":
		return v.sanitizeHTML(input)
	case "sql":
		return v.sanitizeSQL(input)
	case "shell":
		return v.sanitizeShell(input)
	case "path":
		return v.sanitizePath(input)
	case "filename":
		return v.sanitizeFilename(input)
	case "username":
		return v.sanitizeUsername(input)
	default:
		return v.sanitizeGeneric(input)
	}
}

// ValidatePermissions validates a list of permissions
func (v *SecurityValidatorImpl) ValidatePermissions(permissions []string) types.SecurityValidationResult {
	result := types.SecurityValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
		Context:  "permissions_validation",
	}

	permissionPattern := `^[a-z_]+\.[a-z_]+$`
	re := regexp.MustCompile(permissionPattern)

	for _, permission := range permissions {
		if permission == "*" {
			result.Warnings = append(result.Warnings, "wildcard permission '*' grants full access")
			continue
		}

		if !re.MatchString(permission) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("invalid permission format: %s", permission))
		}

		// Check for common dangerous permissions
		dangerousPerms := []string{
			"system.execute", "file.delete", "user.delete", "config.override",
		}
		for _, dangerous := range dangerousPerms {
			if permission == dangerous {
				result.Warnings = append(result.Warnings, fmt.Sprintf("dangerous permission detected: %s", permission))
			}
		}
	}

	return result
}

// ValidateRole validates a security role
func (v *SecurityValidatorImpl) ValidateRole(role types.SecurityRole) types.SecurityValidationResult {
	result := types.SecurityValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
		Context:  "role_validation",
	}

	// Validate ID
	if role.ID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "role ID cannot be empty")
	} else if !v.isValidIdentifier(role.ID) {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid role ID format: %s", role.ID))
	}

	// Validate Name
	if role.Name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "role name cannot be empty")
	}

	// Validate permissions
	permResult := v.ValidatePermissions(role.Permissions)
	result.Errors = append(result.Errors, permResult.Errors...)
	result.Warnings = append(result.Warnings, permResult.Warnings...)
	if !permResult.Valid {
		result.Valid = false
	}

	// Check for privilege escalation in parent roles
	if len(role.ParentRoles) > 5 {
		result.Warnings = append(result.Warnings, "role has many parent roles, check for privilege escalation")
	}

	return result
}

// ValidateUser validates a security user
func (v *SecurityValidatorImpl) ValidateUser(user types.SecurityUser) types.SecurityValidationResult {
	result := types.SecurityValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
		Context:  "user_validation",
	}

	// Validate ID
	if user.ID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "user ID cannot be empty")
	} else if !v.isValidIdentifier(user.ID) {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid user ID format: %s", user.ID))
	}

	// Validate Username
	usernameResult := v.ValidateInput(user.Username, "username")
	result.Errors = append(result.Errors, usernameResult.Errors...)
	result.Warnings = append(result.Warnings, usernameResult.Warnings...)
	if !usernameResult.Valid {
		result.Valid = false
	}

	// Check role count
	if len(user.Roles) > 10 {
		result.Warnings = append(result.Warnings, "user has many roles, check for privilege escalation")
	}

	// Validate role IDs
	for _, roleID := range user.Roles {
		if !v.isValidIdentifier(roleID) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("invalid role ID in user roles: %s", roleID))
		}
	}

	return result
}

// Type-specific validation methods

func (v *SecurityValidatorImpl) validateEmail(input string, result *types.SecurityValidationResult) {
	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailPattern)
	if !re.MatchString(input) {
		result.Valid = false
		result.Errors = append(result.Errors, "invalid email format")
	}
}

func (v *SecurityValidatorImpl) validateUsername(input string, result *types.SecurityValidationResult) {
	if len(input) < 3 {
		result.Valid = false
		result.Errors = append(result.Errors, "username too short (minimum 3 characters)")
		return
	}
	if len(input) > 32 {
		result.Valid = false
		result.Errors = append(result.Errors, "username too long (maximum 32 characters)")
		return
	}

	usernamePattern := `^[a-zA-Z0-9_-]+$`
	re := regexp.MustCompile(usernamePattern)
	if !re.MatchString(input) {
		result.Valid = false
		result.Errors = append(result.Errors, "username contains invalid characters")
	}
}

func (v *SecurityValidatorImpl) validatePassword(input string, result *types.SecurityValidationResult) {
	if len(input) < 8 {
		result.Valid = false
		result.Errors = append(result.Errors, "password too short (minimum 8 characters)")
		return
	}

	checks := []struct {
		pattern string
		message string
		isError bool
	}{
		{`[a-z]`, "password should contain lowercase letters", false},
		{`[A-Z]`, "password should contain uppercase letters", false},
		{`[0-9]`, "password should contain numbers", false},
		{`[^a-zA-Z0-9]`, "password should contain special characters", false},
	}

	for _, check := range checks {
		re := regexp.MustCompile(check.pattern)
		if !re.MatchString(input) {
			if check.isError {
				result.Valid = false
				result.Errors = append(result.Errors, check.message)
			} else {
				result.Warnings = append(result.Warnings, check.message)
			}
		}
	}
}

func (v *SecurityValidatorImpl) validateHTML(input string, result *types.SecurityValidationResult) {
	// Check for script tags
	scriptPattern := `<script[^>]*>.*?</script>`
	re := regexp.MustCompile(`(?i)` + scriptPattern)
	if re.MatchString(input) {
		result.Valid = false
		result.Errors = append(result.Errors, "HTML contains script tags")
	}

	// Check for dangerous attributes
	dangerousAttrs := []string{"onload", "onclick", "onerror", "javascript:"}
	for _, attr := range dangerousAttrs {
		if strings.Contains(strings.ToLower(input), attr) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("HTML contains dangerous attribute: %s", attr))
		}
	}
}

func (v *SecurityValidatorImpl) validateSQL(input string, result *types.SecurityValidationResult) {
	// Check for SQL injection patterns
	sqlPatterns := []string{
		`(?i)\bUNION\b.*\bSELECT\b`,
		`(?i)\bDROP\b.*\bTABLE\b`,
		`(?i)\bDELETE\b.*\bFROM\b`,
		`(?i)--`,
		`(?i)/\*.*\*/`,
		`(?i)\bOR\b.*=.*\bOR\b`,
		`(?i)\bAND\b.*=.*\bAND\b`,
		`'.*'.*=.*'.*'`,
	}

	for _, pattern := range sqlPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(input) {
			result.Valid = false
			result.Errors = append(result.Errors, "SQL injection pattern detected")
			break
		}
	}
}

func (v *SecurityValidatorImpl) validateShell(input string, result *types.SecurityValidationResult) {
	// Check for shell injection patterns
	shellPatterns := []string{`[;&|]`, "`", `\$\(`, `>\s*/`, `<\s*/`, `\|\s*sh`, `\|\s*bash`}

	for _, pattern := range shellPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(input) {
			result.Valid = false
			result.Errors = append(result.Errors, "shell injection pattern detected")
			break
		}
	}
}

func (v *SecurityValidatorImpl) validatePath(input string, result *types.SecurityValidationResult) {
	// Check for path traversal
	if strings.Contains(input, "..") {
		result.Valid = false
		result.Errors = append(result.Errors, "path traversal attempt detected")
	}

	// Check for absolute paths in dangerous locations
	dangerousPaths := []string{"/etc/", "/proc/", "/sys/", "/dev/"}
	for _, dangerous := range dangerousPaths {
		if strings.HasPrefix(input, dangerous) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("access to system directory: %s", dangerous))
		}
	}
}

func (v *SecurityValidatorImpl) validateURL(input string, result *types.SecurityValidationResult) {
	_, err := url.Parse(input)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, "invalid URL format")
		return
	}

	// Check for dangerous protocols
	dangerousProtocols := []string{"javascript:", "data:", "file:"}
	for _, protocol := range dangerousProtocols {
		if strings.HasPrefix(strings.ToLower(input), protocol) {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("dangerous protocol: %s", protocol))
		}
	}
}

func (v *SecurityValidatorImpl) validateJSON(input string, result *types.SecurityValidationResult) {
	var js interface{}
	if err := json.Unmarshal([]byte(input), &js); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid JSON: %s", err.Error()))
	}
}

func (v *SecurityValidatorImpl) validateYAML(input string, result *types.SecurityValidationResult) {
	var yamlData interface{}
	if err := yaml.Unmarshal([]byte(input), &yamlData); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid YAML: %s", err.Error()))
	}
}

func (v *SecurityValidatorImpl) validateGeneric(input string, result *types.SecurityValidationResult) {
	// Check for null bytes
	if strings.Contains(input, "\x00") {
		result.Valid = false
		result.Errors = append(result.Errors, "input contains null bytes")
	}

	// Check for control characters
	for _, r := range input {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			result.Warnings = append(result.Warnings, "input contains control characters")
			break
		}
	}
}

// Sanitization methods

func (v *SecurityValidatorImpl) sanitizeHTML(input string) string {
	// Remove script tags
	re := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	input = re.ReplaceAllString(input, "")

	// Remove dangerous attributes
	dangerousAttrs := []string{"onload", "onclick", "onerror", "onmouseover"}
	for _, attr := range dangerousAttrs {
		re := regexp.MustCompile(`(?i)\s*` + attr + `\s*=\s*[^>\s]*`)
		input = re.ReplaceAllString(input, "")
	}

	return input
}

func (v *SecurityValidatorImpl) sanitizeSQL(input string) string {
	// Escape single quotes
	input = strings.ReplaceAll(input, "'", "''")

	// Remove SQL comments
	re := regexp.MustCompile(`--.*$`)
	input = re.ReplaceAllString(input, "")
	re = regexp.MustCompile(`/\*.*?\*/`)
	input = re.ReplaceAllString(input, "")

	return strings.TrimSpace(input)
}

func (v *SecurityValidatorImpl) sanitizeShell(input string) string {
	// Remove shell metacharacters
	shellMeta := []string{";", "&", "|", ">", "<", "`", "$", "(", ")", "[", "]", "{", "}"}
	for _, meta := range shellMeta {
		input = strings.ReplaceAll(input, meta, "")
	}
	return strings.TrimSpace(input)
}

func (v *SecurityValidatorImpl) sanitizePath(input string) string {
	// Remove path traversal sequences
	input = strings.ReplaceAll(input, "../", "")
	input = strings.ReplaceAll(input, "..\\", "")

	// Clean path
	return filepath.Clean(input)
}

func (v *SecurityValidatorImpl) sanitizeFilename(input string) string {
	// Remove dangerous characters for filenames
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	return re.ReplaceAllString(input, "_")
}

func (v *SecurityValidatorImpl) sanitizeUsername(input string) string {
	// Keep only alphanumeric, underscore, and hyphen
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	return re.ReplaceAllString(input, "")
}

func (v *SecurityValidatorImpl) sanitizeGeneric(input string) string {
	// Remove null bytes and control characters
	re := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	return re.ReplaceAllString(input, "")
}

// Helper methods

func (v *SecurityValidatorImpl) isValidIdentifier(input string) bool {
	idPattern := `^[a-zA-Z][a-zA-Z0-9_-]*$`
	re := regexp.MustCompile(idPattern)
	return re.MatchString(input) && len(input) <= 64
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
