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
	Debug(msg string, fields ...interface{})
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
		{ID: "doc.sync", Name: "Sync Documentation", Resource: "doc", Action: "sync",
			Description: "Execute documentation synchronization"},
		{ID: "config.read", Name: "Read Configuration", Resource: "config", Action: "read", Description: "Read system configuration"},
		{ID: "config.write", Name: "Write Configuration", Resource: "config", Action: "write",
			Description: "Modify system configuration"},
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
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Marshal event to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Append to log file with newline
	file, err := os.OpenFile(a.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
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

// DocAnalyzer defines the interface for documentation analysis operations
type DocAnalyzer interface {
	// ScanCodebase analyzes the codebase using Serena MCP tools
	ScanCodebase(rootPath string) (*CodebaseSnapshot, error)

	// ScanDocumentation analyzes documentation files for API references
	ScanDocumentation(docPaths []string) (*DocumentationSnapshot, error)

	// CompareCodeAndDocs compares code against documentation
	CompareCodeAndDocs(code *CodebaseSnapshot, docs *DocumentationSnapshot) (*AnalysisReport, error)

	// GenerateReport creates formatted reports from analysis results
	GenerateReport(report *AnalysisReport, format ReportFormat) ([]byte, error)
}

// CodebaseSnapshot represents extracted information from codebase
type CodebaseSnapshot struct {
	Timestamp  time.Time               `json:"timestamp"`
	RootPath   string                  `json:"root_path"`
	Packages   map[string]*PackageInfo `json:"packages"`
	PublicAPIs []*APIDefinition        `json:"public_apis"`
	Interfaces []*InterfaceDefinition  `json:"interfaces"`
	Types      []*TypeDefinition       `json:"types"`
	Functions  []*FunctionDefinition   `json:"functions"`
	Constants  []*ConstantDefinition   `json:"constants"`
	Variables  []*VariableDefinition   `json:"variables"`
	FileHashes map[string]string       `json:"file_hashes"` // SHA-256 of file contents
	Checksum   string                  `json:"checksum"`    // Overall snapshot checksum
}

// DocumentationSnapshot represents extracted information from documentation
type DocumentationSnapshot struct {
	Timestamp     time.Time              `json:"timestamp"`
	DocumentPaths []string               `json:"document_paths"`
	APIReferences []*APIReference        `json:"api_references"`
	CodeExamples  []*CodeExample         `json:"code_examples"`
	Interfaces    []*DocumentedInterface `json:"interfaces"`
	Functions     []*DocumentedFunction  `json:"functions"`
	Types         []*DocumentedType      `json:"types"`
	UsagePatterns []*UsagePattern        `json:"usage_patterns"`
	FileHashes    map[string]string      `json:"file_hashes"` // SHA-256 of file contents
	Checksum      string                 `json:"checksum"`    // Overall snapshot checksum
}

// AnalysisReport contains the results of comparing code against documentation
type AnalysisReport struct {
	Timestamp       time.Time                `json:"timestamp"`
	CodeSnapshot    *CodebaseSnapshot        `json:"code_snapshot"`
	DocsSnapshot    *DocumentationSnapshot   `json:"docs_snapshot"`
	Inconsistencies []*Inconsistency         `json:"inconsistencies"`
	MissingDocs     []*MissingDocumentation  `json:"missing_docs"`
	OrphanedDocs    []*OrphanedDocumentation `json:"orphaned_docs"`
	BreakingChanges []*BreakingChange        `json:"breaking_changes"`
	NewFeatures     []*NewFeature            `json:"new_features"`
	Summary         *AnalysisSummary         `json:"summary"`
	Checksum        string                   `json:"checksum"`
}

// PackageInfo contains information about a Go package
type PackageInfo struct {
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	ImportPath  string            `json:"import_path"`
	Files       []string          `json:"files"`
	IsPublic    bool              `json:"is_public"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

// APIDefinition represents a public API element from code
type APIDefinition struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"` // "function", "method", "type", "interface", "constant", "variable"
	Package       string            `json:"package"`
	FilePath      string            `json:"file_path"`
	LineNumber    int               `json:"line_number"`
	Signature     string            `json:"signature"`
	Documentation string            `json:"documentation"`
	IsExported    bool              `json:"is_exported"`
	IsDeprecated  bool              `json:"is_deprecated"`
	Tags          []string          `json:"tags"`
	Metadata      map[string]string `json:"metadata"`
}

// InterfaceDefinition represents an interface from code
type InterfaceDefinition struct {
	Name          string              `json:"name"`
	Package       string              `json:"package"`
	FilePath      string              `json:"file_path"`
	LineNumber    int                 `json:"line_number"`
	Methods       []*MethodDefinition `json:"methods"`
	Documentation string              `json:"documentation"`
	IsExported    bool                `json:"is_exported"`
	Metadata      map[string]string   `json:"metadata"`
}

// TypeDefinition represents a type from code
type TypeDefinition struct {
	Name          string              `json:"name"`
	Kind          string              `json:"kind"` // "struct", "alias", "primitive"
	Package       string              `json:"package"`
	FilePath      string              `json:"file_path"`
	LineNumber    int                 `json:"line_number"`
	Fields        []*FieldDefinition  `json:"fields"`
	Methods       []*MethodDefinition `json:"methods"`
	Documentation string              `json:"documentation"`
	IsExported    bool                `json:"is_exported"`
	Metadata      map[string]string   `json:"metadata"`
}

// FunctionDefinition represents a function from code
type FunctionDefinition struct {
	Name          string                 `json:"name"`
	Package       string                 `json:"package"`
	FilePath      string                 `json:"file_path"`
	LineNumber    int                    `json:"line_number"`
	Signature     string                 `json:"signature"`
	Parameters    []*ParameterDefinition `json:"parameters"`
	Returns       []*ReturnDefinition    `json:"returns"`
	Documentation string                 `json:"documentation"`
	IsExported    bool                   `json:"is_exported"`
	IsMethod      bool                   `json:"is_method"`
	Receiver      *ReceiverDefinition    `json:"receiver,omitempty"`
	Metadata      map[string]string      `json:"metadata"`
}

// MethodDefinition represents a method from code
type MethodDefinition struct {
	Name          string                 `json:"name"`
	Signature     string                 `json:"signature"`
	Parameters    []*ParameterDefinition `json:"parameters"`
	Returns       []*ReturnDefinition    `json:"returns"`
	Documentation string                 `json:"documentation"`
	IsExported    bool                   `json:"is_exported"`
	Metadata      map[string]string      `json:"metadata"`
}

// FieldDefinition represents a struct field from code
type FieldDefinition struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Tag           string            `json:"tag"`
	Documentation string            `json:"documentation"`
	IsExported    bool              `json:"is_exported"`
	Metadata      map[string]string `json:"metadata"`
}

// ParameterDefinition represents a function parameter
type ParameterDefinition struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Variadic bool   `json:"variadic"`
}

// ReturnDefinition represents a function return value
type ReturnDefinition struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
}

// ReceiverDefinition represents a method receiver
type ReceiverDefinition struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Pointer bool   `json:"pointer"`
}

// ConstantDefinition represents a constant from code
type ConstantDefinition struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Value         string            `json:"value"`
	Package       string            `json:"package"`
	FilePath      string            `json:"file_path"`
	LineNumber    int               `json:"line_number"`
	Documentation string            `json:"documentation"`
	IsExported    bool              `json:"is_exported"`
	Metadata      map[string]string `json:"metadata"`
}

// VariableDefinition represents a variable from code
type VariableDefinition struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Package       string            `json:"package"`
	FilePath      string            `json:"file_path"`
	LineNumber    int               `json:"line_number"`
	Documentation string            `json:"documentation"`
	IsExported    bool              `json:"is_exported"`
	IsGlobal      bool              `json:"is_global"`
	Metadata      map[string]string `json:"metadata"`
}

// APIReference represents an API reference found in documentation
type APIReference struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // "function", "method", "type", "interface"
	Package     string            `json:"package"`
	FilePath    string            `json:"file_path"`
	LineNumber  int               `json:"line_number"`
	Context     string            `json:"context"` // Surrounding text context
	Description string            `json:"description"`
	Examples    []*CodeExample    `json:"examples"`
	Metadata    map[string]string `json:"metadata"`
}

// CodeExample represents a code example found in documentation
type CodeExample struct {
	Code           string            `json:"code"`
	Language       string            `json:"language"`
	FilePath       string            `json:"file_path"`
	LineNumber     int               `json:"line_number"`
	Description    string            `json:"description"`
	IsRunnable     bool              `json:"is_runnable"`
	ExpectedOutput string            `json:"expected_output,omitempty"`
	Metadata       map[string]string `json:"metadata"`
}

// DocumentedInterface represents an interface documented in markdown
type DocumentedInterface struct {
	Name        string              `json:"name"`
	Package     string              `json:"package"`
	FilePath    string              `json:"file_path"`
	LineNumber  int                 `json:"line_number"`
	Description string              `json:"description"`
	Methods     []*DocumentedMethod `json:"methods"`
	Examples    []*CodeExample      `json:"examples"`
	Metadata    map[string]string   `json:"metadata"`
}

// DocumentedFunction represents a function documented in markdown
type DocumentedFunction struct {
	Name        string              `json:"name"`
	Package     string              `json:"package"`
	FilePath    string              `json:"file_path"`
	LineNumber  int                 `json:"line_number"`
	Signature   string              `json:"signature"`
	Description string              `json:"description"`
	Parameters  []*DocumentedParam  `json:"parameters"`
	Returns     []*DocumentedReturn `json:"returns"`
	Examples    []*CodeExample      `json:"examples"`
	Metadata    map[string]string   `json:"metadata"`
}

// DocumentedType represents a type documented in markdown
type DocumentedType struct {
	Name        string              `json:"name"`
	Package     string              `json:"package"`
	FilePath    string              `json:"file_path"`
	LineNumber  int                 `json:"line_number"`
	Description string              `json:"description"`
	Fields      []*DocumentedField  `json:"fields"`
	Methods     []*DocumentedMethod `json:"methods"`
	Examples    []*CodeExample      `json:"examples"`
	Metadata    map[string]string   `json:"metadata"`
}

// DocumentedMethod represents a method documented in markdown
type DocumentedMethod struct {
	Name        string              `json:"name"`
	Signature   string              `json:"signature"`
	Description string              `json:"description"`
	Parameters  []*DocumentedParam  `json:"parameters"`
	Returns     []*DocumentedReturn `json:"returns"`
	Examples    []*CodeExample      `json:"examples"`
	Metadata    map[string]string   `json:"metadata"`
}

// DocumentedField represents a field documented in markdown
type DocumentedField struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Required    bool              `json:"required"`
	Default     string            `json:"default"`
	Metadata    map[string]string `json:"metadata"`
}

// DocumentedParam represents a parameter documented in markdown
type DocumentedParam struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Required    bool              `json:"required"`
	Default     string            `json:"default"`
	Metadata    map[string]string `json:"metadata"`
}

// DocumentedReturn represents a return value documented in markdown
type DocumentedReturn struct {
	Name        string            `json:"name,omitempty"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

// UsagePattern represents a usage pattern found in documentation
type UsagePattern struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Pattern     string            `json:"pattern"`
	FilePath    string            `json:"file_path"`
	LineNumber  int               `json:"line_number"`
	Examples    []*CodeExample    `json:"examples"`
	Frequency   int               `json:"frequency"`
	Metadata    map[string]string `json:"metadata"`
}

// Inconsistency represents a mismatch between code and documentation
type Inconsistency struct {
	ID          string                `json:"id"`
	Type        InconsistencyType     `json:"type"`
	Severity    InconsistencySeverity `json:"severity"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	CodeRef     *SourceReference      `json:"code_ref,omitempty"`
	DocsRef     *SourceReference      `json:"docs_ref,omitempty"`
	Expected    string                `json:"expected"`
	Actual      string                `json:"actual"`
	Suggestions []string              `json:"suggestions"`
	Metadata    map[string]string     `json:"metadata"`
}

// MissingDocumentation represents code that lacks documentation
type MissingDocumentation struct {
	ID          string                `json:"id"`
	Type        string                `json:"type"` // "function", "type", "interface", etc.
	Name        string                `json:"name"`
	Package     string                `json:"package"`
	CodeRef     *SourceReference      `json:"code_ref"`
	Severity    InconsistencySeverity `json:"severity"`
	Description string                `json:"description"`
	Suggestions []string              `json:"suggestions"`
	Metadata    map[string]string     `json:"metadata"`
}

// OrphanedDocumentation represents documentation without corresponding code
type OrphanedDocumentation struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Type        string                `json:"type"`
	DocsRef     *SourceReference      `json:"docs_ref"`
	Severity    InconsistencySeverity `json:"severity"`
	Description string                `json:"description"`
	Suggestions []string              `json:"suggestions"`
	Metadata    map[string]string     `json:"metadata"`
}

// BreakingChange represents a breaking change detected in code
type BreakingChange struct {
	ID           string               `json:"id"`
	Type         string               `json:"type"` // "signature_change", "removal", "behavior_change"
	Name         string               `json:"name"`
	Package      string               `json:"package"`
	CodeRef      *SourceReference     `json:"code_ref"`
	DocsRef      *SourceReference     `json:"docs_ref,omitempty"`
	OldSignature string               `json:"old_signature"`
	NewSignature string               `json:"new_signature"`
	Impact       BreakingChangeImpact `json:"impact"`
	Description  string               `json:"description"`
	Suggestions  []string             `json:"suggestions"`
	Metadata     map[string]string    `json:"metadata"`
}

// NewFeature represents a new feature detected in code
type NewFeature struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // "function", "type", "interface", "method"
	Name        string            `json:"name"`
	Package     string            `json:"package"`
	CodeRef     *SourceReference  `json:"code_ref"`
	Signature   string            `json:"signature"`
	Description string            `json:"description"`
	IsPublic    bool              `json:"is_public"`
	Suggestions []string          `json:"suggestions"`
	Metadata    map[string]string `json:"metadata"`
}

// AnalysisSummary provides a high-level summary of the analysis results
type AnalysisSummary struct {
	TotalInconsistencies    int                    `json:"total_inconsistencies"`
	CriticalInconsistencies int                    `json:"critical_inconsistencies"`
	HighInconsistencies     int                    `json:"high_inconsistencies"`
	MediumInconsistencies   int                    `json:"medium_inconsistencies"`
	LowInconsistencies      int                    `json:"low_inconsistencies"`
	MissingDocsCount        int                    `json:"missing_docs_count"`
	OrphanedDocsCount       int                    `json:"orphaned_docs_count"`
	BreakingChangesCount    int                    `json:"breaking_changes_count"`
	NewFeaturesCount        int                    `json:"new_features_count"`
	CodebaseSize            *CodebaseSize          `json:"codebase_size"`
	DocumentationCoverage   *DocumentationCoverage `json:"documentation_coverage"`
	QualityScore            float64                `json:"quality_score"` // 0.0-1.0
	Recommendations         []string               `json:"recommendations"`
}

// CodebaseSize provides metrics about the codebase
type CodebaseSize struct {
	TotalFiles  int `json:"total_files"`
	TotalLines  int `json:"total_lines"`
	PublicAPIs  int `json:"public_apis"`
	PrivateAPIs int `json:"private_apis"`
	Interfaces  int `json:"interfaces"`
	Types       int `json:"types"`
	Functions   int `json:"functions"`
	Methods     int `json:"methods"`
}

// DocumentationCoverage provides metrics about documentation coverage
type DocumentationCoverage struct {
	TotalDocFiles      int     `json:"total_doc_files"`
	DocumentedAPIs     int     `json:"documented_apis"`
	UndocumentedAPIs   int     `json:"undocumented_apis"`
	CoveragePercentage float64 `json:"coverage_percentage"`
	ExampleCoverage    float64 `json:"example_coverage"`
	UpToDatePercentage float64 `json:"up_to_date_percentage"`
}

// SourceReference represents a reference to a location in source code or documentation
type SourceReference struct {
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	ColumnStart int    `json:"column_start,omitempty"`
	ColumnEnd   int    `json:"column_end,omitempty"`
	Context     string `json:"context,omitempty"` // Surrounding text for context
}

// InconsistencyType represents the type of inconsistency
type InconsistencyType string

const (
	InconsistencyTypeSignatureMismatch     InconsistencyType = "signature_mismatch"
	InconsistencyTypeParameterMismatch     InconsistencyType = "parameter_mismatch"
	InconsistencyTypeReturnMismatch        InconsistencyType = "return_mismatch"
	InconsistencyTypeBehaviorMismatch      InconsistencyType = "behavior_mismatch"
	InconsistencyTypeExampleOutdated       InconsistencyType = "example_outdated"
	InconsistencyTypeDocumentationMissing  InconsistencyType = "documentation_missing"
	InconsistencyTypeDocumentationOrphaned InconsistencyType = "documentation_orphaned"
)

// InconsistencySeverity represents the severity level of an inconsistency
type InconsistencySeverity string

const (
	InconsistencySeverityCritical InconsistencySeverity = "critical"
	InconsistencySeverityHigh     InconsistencySeverity = "high"
	InconsistencySeverityMedium   InconsistencySeverity = "medium"
	InconsistencySeverityLow      InconsistencySeverity = "low"
)

// BreakingChangeImpact represents the impact level of a breaking change
type BreakingChangeImpact string

const (
	BreakingChangeImpactMajor BreakingChangeImpact = "major"
	BreakingChangeImpactMinor BreakingChangeImpact = "minor"
	BreakingChangeImpactPatch BreakingChangeImpact = "patch"
)

// ReportFormat represents the output format for reports
type ReportFormat string

const (
	ReportFormatJSON     ReportFormat = "json"
	ReportFormatText     ReportFormat = "text"
	ReportFormatMarkdown ReportFormat = "markdown"
	ReportFormatHTML     ReportFormat = "html"
)

// DocAnalyzerImpl implements the DocAnalyzer interface
type DocAnalyzerImpl struct {
	logger         Logger
	stateManager   *DocSyncStateManager
	rbacController *RBACController
	auditLogger    *AuditLoggerImpl
	config         *DocAnalyzerConfig
	mu             sync.RWMutex
}

// DocAnalyzerConfig contains configuration for the documentation analyzer
type DocAnalyzerConfig struct {
	MaxFileSize               int64    `yaml:"max_file_size" json:"max_file_size"`                             // Maximum file size to analyze (bytes)
	SupportedExtensions       []string `yaml:"supported_extensions" json:"supported_extensions"`               // File extensions to analyze
	ExcludePatterns           []string `yaml:"exclude_patterns" json:"exclude_patterns"`                       // Patterns to exclude from analysis
	IncludePatterns           []string `yaml:"include_patterns" json:"include_patterns"`                       // Patterns to include in analysis
	DocumentationPaths        []string `yaml:"documentation_paths" json:"documentation_paths"`                 // Paths to search for documentation
	IgnorePrivateAPIs         bool     `yaml:"ignore_private_apis" json:"ignore_private_apis"`                 // Whether to ignore non-exported APIs
	EnableIncrementalAnalysis bool     `yaml:"enable_incremental_analysis" json:"enable_incremental_analysis"` // Enable incremental analysis
	CacheEnabled              bool     `yaml:"cache_enabled" json:"cache_enabled"`                             // Enable caching of analysis results
	CacheTTL                  int      `yaml:"cache_ttl" json:"cache_ttl"`                                     // Cache TTL in seconds
	MaxConcurrency            int      `yaml:"max_concurrency" json:"max_concurrency"`                         // Maximum concurrent analysis operations
	TimeoutSeconds            int      `yaml:"timeout_seconds" json:"timeout_seconds"`                         // Analysis timeout in seconds
	AnalysisDepth             int      `yaml:"analysis_depth" json:"analysis_depth"`                           // Depth of analysis (1=shallow, 3=deep)
}

// NewDocAnalyzer creates a new documentation analyzer instance
func NewDocAnalyzer(
	logger Logger,
	stateManager *DocSyncStateManager,
	rbacController *RBACController,
	auditLogger *AuditLoggerImpl,
	config *DocAnalyzerConfig,
) *DocAnalyzerImpl {
	return &DocAnalyzerImpl{
		logger:         logger,
		stateManager:   stateManager,
		rbacController: rbacController,
		auditLogger:    auditLogger,
		config:         config,
	}
}

// DefaultDocAnalyzerConfig returns default configuration for the documentation analyzer
func DefaultDocAnalyzerConfig() *DocAnalyzerConfig {
	return &DocAnalyzerConfig{
		MaxFileSize:               10 * 1024 * 1024, // 10MB
		SupportedExtensions:       []string{".go", ".md", ".rst", ".txt"},
		ExcludePatterns:           []string{"vendor/", "node_modules/", ".git/", "*.test", "*_test.go"},
		IncludePatterns:           []string{"*.go", "*.md"},
		DocumentationPaths:        []string{"docs/", "README.md", "*.md"},
		IgnorePrivateAPIs:         true,
		EnableIncrementalAnalysis: true,
		CacheEnabled:              true,
		CacheTTL:                  3600, // 1 hour
		MaxConcurrency:            4,
		TimeoutSeconds:            300, // 5 minutes
		AnalysisDepth:             2,   // Medium depth
	}
}

// ScanCodebase analyzes the codebase using Serena MCP tools
func (d *DocAnalyzerImpl) ScanCodebase(rootPath string) (*CodebaseSnapshot, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Audit log the analysis operation
	if d.auditLogger != nil {
		d.auditLogger.LogEvent(types.AuditEvent{
			UserID:   "doc_analyzer",
			Action:   "scan_codebase",
			Resource: rootPath,
			Result:   "initiated",
			Details: map[string]interface{}{
				"root_path": rootPath,
				"timestamp": time.Now().UTC(),
			},
		})
	}

	snapshot := &CodebaseSnapshot{
		Timestamp:  time.Now().UTC(),
		RootPath:   rootPath,
		Packages:   make(map[string]*PackageInfo),
		PublicAPIs: make([]*APIDefinition, 0),
		Interfaces: make([]*InterfaceDefinition, 0),
		Types:      make([]*TypeDefinition, 0),
		Functions:  make([]*FunctionDefinition, 0),
		Constants:  make([]*ConstantDefinition, 0),
		Variables:  make([]*VariableDefinition, 0),
		FileHashes: make(map[string]string),
	}

	// Use Serena MCP tools for semantic analysis
	d.logger.Info("Starting codebase analysis", map[string]interface{}{
		"root_path": rootPath,
		"config":    d.config,
	})

	// Step 1: Discover Go packages using Serena MCP
	err := d.discoverPackages(rootPath, snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to discover packages: %w", err)
	}

	// Step 2: Analyze each package for symbols
	for _, pkg := range snapshot.Packages {
		if err := d.analyzePackage(pkg, snapshot); err != nil {
			d.logger.Error("Failed to analyze package", map[string]interface{}{
				"package": pkg.Name,
				"error":   err,
			})
			// Continue with other packages rather than failing completely
			continue
		}
	}

	// Step 3: Calculate file hashes for change detection
	if err := d.calculateFileHashes(rootPath, snapshot); err != nil {
		d.logger.Warn("Failed to calculate file hashes", map[string]interface{}{
			"error": err,
		})
	}

	// Calculate overall checksum for the snapshot
	checksum, err := d.calculateSnapshotChecksum(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate snapshot checksum: %w", err)
	}
	snapshot.Checksum = checksum

	d.logger.Info("Codebase analysis complete", map[string]interface{}{
		"packages":    len(snapshot.Packages),
		"public_apis": len(snapshot.PublicAPIs),
		"interfaces":  len(snapshot.Interfaces),
		"types":       len(snapshot.Types),
		"functions":   len(snapshot.Functions),
	})

	return snapshot, nil
}

// ScanDocumentation analyzes documentation files for API references
func (d *DocAnalyzerImpl) ScanDocumentation(docPaths []string) (*DocumentationSnapshot, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Audit log the analysis operation
	if d.auditLogger != nil {
		d.auditLogger.LogEvent(types.AuditEvent{
			UserID:   "doc_analyzer",
			Action:   "scan_documentation",
			Resource: "multiple_paths",
			Result:   "initiated",
			Details: map[string]interface{}{
				"doc_paths": docPaths,
				"timestamp": time.Now().UTC(),
			},
		})
	}

	snapshot := &DocumentationSnapshot{
		Timestamp:     time.Now().UTC(),
		DocumentPaths: docPaths,
		APIReferences: make([]*APIReference, 0),
		CodeExamples:  make([]*CodeExample, 0),
		Interfaces:    make([]*DocumentedInterface, 0),
		Functions:     make([]*DocumentedFunction, 0),
		Types:         make([]*DocumentedType, 0),
		UsagePatterns: make([]*UsagePattern, 0),
		FileHashes:    make(map[string]string),
	}

	// TODO: This method will parse markdown files and extract API references
	// Implementation will be added in the next step

	d.logger.Info("ScanDocumentation called", map[string]interface{}{
		"doc_paths": docPaths,
		"config":    d.config,
	})

	// Calculate checksum for the snapshot
	checksum, err := d.calculateDocSnapshotChecksum(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate documentation snapshot checksum: %w", err)
	}
	snapshot.Checksum = checksum

	return snapshot, nil
}

// CompareCodeAndDocs compares code against documentation
func (d *DocAnalyzerImpl) CompareCodeAndDocs(code *CodebaseSnapshot, docs *DocumentationSnapshot) (*AnalysisReport, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Audit log the comparison operation
	if d.auditLogger != nil {
		d.auditLogger.LogEvent(types.AuditEvent{
			UserID:   "doc_analyzer",
			Action:   "compare_code_and_docs",
			Resource: "analysis",
			Result:   "initiated",
			Details: map[string]interface{}{
				"code_checksum": code.Checksum,
				"docs_checksum": docs.Checksum,
				"timestamp":     time.Now().UTC(),
			},
		})
	}

	report := &AnalysisReport{
		Timestamp:       time.Now().UTC(),
		CodeSnapshot:    code,
		DocsSnapshot:    docs,
		Inconsistencies: make([]*Inconsistency, 0),
		MissingDocs:     make([]*MissingDocumentation, 0),
		OrphanedDocs:    make([]*OrphanedDocumentation, 0),
		BreakingChanges: make([]*BreakingChange, 0),
		NewFeatures:     make([]*NewFeature, 0),
	}

	// TODO: This method will implement the comparison algorithms
	// Implementation will be added in the next step

	d.logger.Info("CompareCodeAndDocs called", map[string]interface{}{
		"code_apis": len(code.PublicAPIs),
		"doc_refs":  len(docs.APIReferences),
	})

	// Generate summary
	summary := d.generateAnalysisSummary(report)
	report.Summary = summary

	// Calculate checksum for the report
	checksum, err := d.calculateReportChecksum(report)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate report checksum: %w", err)
	}
	report.Checksum = checksum

	return report, nil
}

// GenerateReport creates formatted reports from analysis results
func (d *DocAnalyzerImpl) GenerateReport(report *AnalysisReport, format ReportFormat) ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Audit log the report generation
	if d.auditLogger != nil {
		d.auditLogger.LogEvent(types.AuditEvent{
			UserID:   "doc_analyzer",
			Action:   "generate_report",
			Resource: string(format),
			Result:   "initiated",
			Details: map[string]interface{}{
				"report_checksum": report.Checksum,
				"format":          string(format),
				"timestamp":       time.Now().UTC(),
			},
		})
	}

	switch format {
	case ReportFormatJSON:
		return d.generateJSONReport(report)
	case ReportFormatText:
		return d.generateTextReport(report)
	case ReportFormatMarkdown:
		return d.generateMarkdownReport(report)
	case ReportFormatHTML:
		return d.generateHTMLReport(report)
	default:
		return nil, fmt.Errorf("unsupported report format: %s", format)
	}
}

// Helper methods for checksum calculation
func (d *DocAnalyzerImpl) calculateSnapshotChecksum(snapshot *CodebaseSnapshot) (string, error) {
	// Create a deterministic representation for checksum
	data := fmt.Sprintf("%s|%d|%d|%d|%d|%d|%d",
		snapshot.RootPath,
		len(snapshot.PublicAPIs),
		len(snapshot.Interfaces),
		len(snapshot.Types),
		len(snapshot.Functions),
		len(snapshot.Constants),
		len(snapshot.Variables))

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash), nil
}

func (d *DocAnalyzerImpl) calculateDocSnapshotChecksum(snapshot *DocumentationSnapshot) (string, error) {
	// Create a deterministic representation for checksum
	data := fmt.Sprintf("%d|%d|%d|%d|%d|%d",
		len(snapshot.APIReferences),
		len(snapshot.CodeExamples),
		len(snapshot.Interfaces),
		len(snapshot.Functions),
		len(snapshot.Types),
		len(snapshot.UsagePatterns))

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash), nil
}

func (d *DocAnalyzerImpl) calculateReportChecksum(report *AnalysisReport) (string, error) {
	// Create a deterministic representation for checksum
	data := fmt.Sprintf("%s|%s|%d|%d|%d|%d|%d",
		report.CodeSnapshot.Checksum,
		report.DocsSnapshot.Checksum,
		len(report.Inconsistencies),
		len(report.MissingDocs),
		len(report.OrphanedDocs),
		len(report.BreakingChanges),
		len(report.NewFeatures))

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash), nil
}

// Helper method to generate analysis summary
func (d *DocAnalyzerImpl) generateAnalysisSummary(report *AnalysisReport) *AnalysisSummary {
	summary := &AnalysisSummary{
		MissingDocsCount:     len(report.MissingDocs),
		OrphanedDocsCount:    len(report.OrphanedDocs),
		BreakingChangesCount: len(report.BreakingChanges),
		NewFeaturesCount:     len(report.NewFeatures),
		CodebaseSize: &CodebaseSize{
			PublicAPIs: len(report.CodeSnapshot.PublicAPIs),
			Interfaces: len(report.CodeSnapshot.Interfaces),
			Types:      len(report.CodeSnapshot.Types),
			Functions:  len(report.CodeSnapshot.Functions),
		},
		DocumentationCoverage: &DocumentationCoverage{
			TotalDocFiles:  len(report.DocsSnapshot.DocumentPaths),
			DocumentedAPIs: len(report.DocsSnapshot.APIReferences),
		},
		Recommendations: make([]string, 0),
	}

	// Count inconsistencies by severity
	for _, inconsistency := range report.Inconsistencies {
		summary.TotalInconsistencies++
		switch inconsistency.Severity {
		case InconsistencySeverityCritical:
			summary.CriticalInconsistencies++
		case InconsistencySeverityHigh:
			summary.HighInconsistencies++
		case InconsistencySeverityMedium:
			summary.MediumInconsistencies++
		case InconsistencySeverityLow:
			summary.LowInconsistencies++
		}
	}

	// Calculate quality score (simple algorithm for now)
	totalAPIs := len(report.CodeSnapshot.PublicAPIs)
	documentedAPIs := len(report.DocsSnapshot.APIReferences)
	if totalAPIs > 0 {
		summary.DocumentationCoverage.CoveragePercentage = float64(documentedAPIs) / float64(totalAPIs) * 100.0
	}

	// Simple quality score calculation
	if totalAPIs > 0 {
		coverage := summary.DocumentationCoverage.CoveragePercentage / 100.0
		inconsistencyPenalty := float64(summary.TotalInconsistencies) / float64(totalAPIs)
		summary.QualityScore = coverage - (inconsistencyPenalty * 0.5)
		if summary.QualityScore < 0 {
			summary.QualityScore = 0
		}
		if summary.QualityScore > 1 {
			summary.QualityScore = 1
		}
	}

	// Generate basic recommendations
	if summary.CriticalInconsistencies > 0 {
		summary.Recommendations = append(summary.Recommendations, "Address critical documentation inconsistencies immediately")
	}
	if summary.DocumentationCoverage.CoveragePercentage < 50.0 {
		summary.Recommendations = append(summary.Recommendations, "Improve documentation coverage for public APIs")
	}
	if summary.OrphanedDocsCount > 0 {
		summary.Recommendations = append(summary.Recommendations, "Remove or update orphaned documentation")
	}

	return summary
}

// Report generation methods
func (d *DocAnalyzerImpl) generateJSONReport(report *AnalysisReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

func (d *DocAnalyzerImpl) generateTextReport(report *AnalysisReport) ([]byte, error) {
	var buf strings.Builder

	buf.WriteString("Documentation Analysis Report\n")
	buf.WriteString("=============================\n\n")
	buf.WriteString(fmt.Sprintf("Generated: %s\n", report.Timestamp.Format(time.RFC3339)))
	buf.WriteString(fmt.Sprintf("Code Snapshot: %s\n", report.CodeSnapshot.Checksum[:8]))
	buf.WriteString(fmt.Sprintf("Docs Snapshot: %s\n\n", report.DocsSnapshot.Checksum[:8]))

	// Summary section
	buf.WriteString("Summary\n")
	buf.WriteString("-------\n")
	if report.Summary != nil {
		buf.WriteString(fmt.Sprintf("Quality Score: %.2f/1.00\n", report.Summary.QualityScore))
		buf.WriteString(fmt.Sprintf("Documentation Coverage: %.1f%%\n", report.Summary.DocumentationCoverage.CoveragePercentage))
		buf.WriteString(fmt.Sprintf("Total Inconsistencies: %d\n", report.Summary.TotalInconsistencies))
		buf.WriteString(fmt.Sprintf("Missing Documentation: %d\n", report.Summary.MissingDocsCount))
		buf.WriteString(fmt.Sprintf("Orphaned Documentation: %d\n", report.Summary.OrphanedDocsCount))
		buf.WriteString(fmt.Sprintf("Breaking Changes: %d\n", report.Summary.BreakingChangesCount))
		buf.WriteString(fmt.Sprintf("New Features: %d\n\n", report.Summary.NewFeaturesCount))

		if len(report.Summary.Recommendations) > 0 {
			buf.WriteString("Recommendations\n")
			buf.WriteString("---------------\n")
			for _, rec := range report.Summary.Recommendations {
				buf.WriteString(fmt.Sprintf("- %s\n", rec))
			}
			buf.WriteString("\n")
		}
	}

	// Detailed sections would be added here in a full implementation
	buf.WriteString("Report generated by Documentation Synchronization System\n")

	return []byte(buf.String()), nil
}

func (d *DocAnalyzerImpl) generateMarkdownReport(report *AnalysisReport) ([]byte, error) {
	var buf strings.Builder

	buf.WriteString("# Documentation Analysis Report\n\n")
	buf.WriteString(fmt.Sprintf("**Generated:** %s  \n", report.Timestamp.Format(time.RFC3339)))
	buf.WriteString(fmt.Sprintf("**Code Snapshot:** `%s`  \n", report.CodeSnapshot.Checksum[:8]))
	buf.WriteString(fmt.Sprintf("**Docs Snapshot:** `%s`  \n\n", report.DocsSnapshot.Checksum[:8]))

	// Summary section
	buf.WriteString("## Summary\n\n")
	if report.Summary != nil {
		buf.WriteString(fmt.Sprintf("- **Quality Score:** %.2f/1.00\n", report.Summary.QualityScore))
		buf.WriteString(fmt.Sprintf("- **Documentation Coverage:** %.1f%%\n", report.Summary.DocumentationCoverage.CoveragePercentage))
		buf.WriteString(fmt.Sprintf("- **Total Inconsistencies:** %d\n", report.Summary.TotalInconsistencies))
		buf.WriteString(fmt.Sprintf("- **Missing Documentation:** %d\n", report.Summary.MissingDocsCount))
		buf.WriteString(fmt.Sprintf("- **Orphaned Documentation:** %d\n", report.Summary.OrphanedDocsCount))
		buf.WriteString(fmt.Sprintf("- **Breaking Changes:** %d\n", report.Summary.BreakingChangesCount))
		buf.WriteString(fmt.Sprintf("- **New Features:** %d\n\n", report.Summary.NewFeaturesCount))

		if len(report.Summary.Recommendations) > 0 {
			buf.WriteString("## Recommendations\n\n")
			for _, rec := range report.Summary.Recommendations {
				buf.WriteString(fmt.Sprintf("- %s\n", rec))
			}
			buf.WriteString("\n")
		}
	}

	buf.WriteString("---\n")
	buf.WriteString("*Report generated by Documentation Synchronization System*\n")

	return []byte(buf.String()), nil
}

func (d *DocAnalyzerImpl) generateHTMLReport(report *AnalysisReport) ([]byte, error) {
	var buf strings.Builder

	buf.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	buf.WriteString("<title>Documentation Analysis Report</title>\n")
	buf.WriteString("<style>body{font-family:Arial,sans-serif;margin:2em;}h1{color:#333;}.summary{background:#f5f5f5;padding:1em;border-radius:5px;}</style>\n")
	buf.WriteString("</head>\n<body>\n")

	buf.WriteString("<h1>Documentation Analysis Report</h1>\n")
	buf.WriteString(fmt.Sprintf("<p><strong>Generated:</strong> %s</p>\n", report.Timestamp.Format(time.RFC3339)))
	buf.WriteString(fmt.Sprintf("<p><strong>Code Snapshot:</strong> <code>%s</code></p>\n", report.CodeSnapshot.Checksum[:8]))
	buf.WriteString(fmt.Sprintf("<p><strong>Docs Snapshot:</strong> <code>%s</code></p>\n", report.DocsSnapshot.Checksum[:8]))

	if report.Summary != nil {
		buf.WriteString("<div class=\"summary\">\n<h2>Summary</h2>\n<ul>\n")
		buf.WriteString(fmt.Sprintf("<li><strong>Quality Score:</strong> %.2f/1.00</li>\n", report.Summary.QualityScore))
		buf.WriteString(fmt.Sprintf("<li><strong>Documentation Coverage:</strong> %.1f%%</li>\n", report.Summary.DocumentationCoverage.CoveragePercentage))
		buf.WriteString(fmt.Sprintf("<li><strong>Total Inconsistencies:</strong> %d</li>\n", report.Summary.TotalInconsistencies))
		buf.WriteString(fmt.Sprintf("<li><strong>Missing Documentation:</strong> %d</li>\n", report.Summary.MissingDocsCount))
		buf.WriteString(fmt.Sprintf("<li><strong>Orphaned Documentation:</strong> %d</li>\n", report.Summary.OrphanedDocsCount))
		buf.WriteString(fmt.Sprintf("<li><strong>Breaking Changes:</strong> %d</li>\n", report.Summary.BreakingChangesCount))
		buf.WriteString(fmt.Sprintf("<li><strong>New Features:</strong> %d</li>\n", report.Summary.NewFeaturesCount))
		buf.WriteString("</ul>\n</div>\n")
	}

	buf.WriteString("<hr>\n<p><em>Report generated by Documentation Synchronization System</em></p>\n")
	buf.WriteString("</body>\n</html>\n")

	return []byte(buf.String()), nil
}

// discoverPackages discovers Go packages in the rootPath using directory traversal
func (d *DocAnalyzerImpl) discoverPackages(rootPath string, snapshot *CodebaseSnapshot) error {
	// This is a simplified implementation - in a full system, we might use more sophisticated
	// package discovery. For now, we'll scan for Go files and group by directory.

	// TODO: In a complete implementation, we would use Serena MCP tools to discover packages
	// For now, we'll implement a basic directory traversal approach

	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files if configured
		if d.config.IgnorePrivateAPIs && strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Skip files matching exclude patterns
		for _, pattern := range d.config.ExcludePatterns {
			if matched, _ := filepath.Match(pattern, path); matched {
				return nil
			}
		}

		// Get package directory
		pkgDir := filepath.Dir(path)
		relPkgDir, err := filepath.Rel(rootPath, pkgDir)
		if err != nil {
			return err
		}

		// Create or update package info
		pkgKey := relPkgDir
		if pkgKey == "." {
			pkgKey = "main"
		}

		pkg, exists := snapshot.Packages[pkgKey]
		if !exists {
			// Extract package name from the first Go file in the directory
			pkgName, err := d.extractPackageName(path)
			if err != nil {
				d.logger.Warn("Failed to extract package name", map[string]interface{}{
					"file":  path,
					"error": err,
				})
				pkgName = "unknown"
			}

			pkg = &PackageInfo{
				Name:        pkgName,
				Path:        pkgDir,
				ImportPath:  d.buildImportPath(rootPath, pkgDir),
				Files:       make([]string, 0),
				IsPublic:    !strings.Contains(relPkgDir, "internal"),
				Description: "",
				Metadata:    make(map[string]string),
			}
			snapshot.Packages[pkgKey] = pkg
		}

		pkg.Files = append(pkg.Files, path)
		return nil
	})
}

// analyzePackage analyzes a single package using Serena MCP tools
func (d *DocAnalyzerImpl) analyzePackage(pkg *PackageInfo, snapshot *CodebaseSnapshot) error {
	d.logger.Debug("Analyzing package", map[string]interface{}{
		"package": pkg.Name,
		"files":   len(pkg.Files),
	})

	// Analyze each file in the package
	for _, filePath := range pkg.Files {
		if err := d.analyzeFile(filePath, pkg, snapshot); err != nil {
			d.logger.Error("Failed to analyze file", map[string]interface{}{
				"file":  filePath,
				"error": err,
			})
			// Continue with other files
			continue
		}
	}

	return nil
}

// analyzeFile analyzes a single Go file using Serena MCP tools
func (d *DocAnalyzerImpl) analyzeFile(filePath string, pkg *PackageInfo, snapshot *CodebaseSnapshot) error {
	d.logger.Debug("Analyzing file", map[string]interface{}{
		"file":    filePath,
		"package": pkg.Name,
	})

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse the Go file using the standard AST parser
	// Note: This is a simplified implementation. In a full system, we would use
	// go/parser, go/ast, and go/doc for complete semantic analysis

	relPath, _ := filepath.Rel(snapshot.RootPath, filePath)

	// Extract functions using regex patterns (simplified approach)
	if err := d.extractFunctions(content, filePath, relPath, pkg, snapshot); err != nil {
		d.logger.Warn("Failed to extract functions", map[string]interface{}{
			"file":  filePath,
			"error": err,
		})
	}

	// Extract types using regex patterns (simplified approach)
	if err := d.extractTypes(content, filePath, relPath, pkg, snapshot); err != nil {
		d.logger.Warn("Failed to extract types", map[string]interface{}{
			"file":  filePath,
			"error": err,
		})
	}

	// Extract interfaces using regex patterns (simplified approach)
	if err := d.extractInterfaces(content, filePath, relPath, pkg, snapshot); err != nil {
		d.logger.Warn("Failed to extract interfaces", map[string]interface{}{
			"file":  filePath,
			"error": err,
		})
	}

	// Extract constants and variables using regex patterns (simplified approach)
	if err := d.extractConstantsAndVariables(content, filePath, relPath, pkg, snapshot); err != nil {
		d.logger.Warn("Failed to extract constants and variables", map[string]interface{}{
			"file":  filePath,
			"error": err,
		})
	}

	return nil
}

// calculateFileHashes calculates SHA-256 hashes for all relevant files
func (d *DocAnalyzerImpl) calculateFileHashes(rootPath string, snapshot *CodebaseSnapshot) error {
	for _, pkg := range snapshot.Packages {
		for _, filePath := range pkg.Files {
			content, err := os.ReadFile(filePath)
			if err != nil {
				d.logger.Warn("Failed to read file for hashing", map[string]interface{}{
					"file":  filePath,
					"error": err,
				})
				continue
			}

			hash := sha256.Sum256(content)
			relPath, _ := filepath.Rel(rootPath, filePath)
			snapshot.FileHashes[relPath] = fmt.Sprintf("%x", hash)
		}
	}
	return nil
}

// extractPackageName extracts the package name from a Go file
func (d *DocAnalyzerImpl) extractPackageName(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Simple regex to extract package name
	packageRegex := regexp.MustCompile(`(?m)^package\s+(\w+)`)
	matches := packageRegex.FindStringSubmatch(string(content))
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find package declaration in %s", filePath)
	}

	return matches[1], nil
}

// buildImportPath constructs the import path for a package
func (d *DocAnalyzerImpl) buildImportPath(rootPath, pkgDir string) string {
	// This is a simplified implementation
	// In practice, this would need to determine the module path and construct proper import paths
	relPath, _ := filepath.Rel(rootPath, pkgDir)
	if relPath == "." {
		return "main"
	}
	return relPath
}

// extractFunctions extracts function definitions from Go source code
func (d *DocAnalyzerImpl) extractFunctions(content []byte, filePath, relPath string, pkg *PackageInfo, snapshot *CodebaseSnapshot) error {
	// Simplified regex-based extraction for demonstration
	// In a production system, this would use go/parser and go/ast

	funcRegex := regexp.MustCompile(`(?m)^func\s+(\*?\w+\s+)?(\w+)\s*\([^)]*\)\s*(\([^)]*\))?\s*(\w+)?\s*{`)
	matches := funcRegex.FindAllStringSubmatch(string(content), -1)

	lines := strings.Split(string(content), "\n")

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		receiver := strings.TrimSpace(match[1])
		name := match[2]
		// returns := match[3]  // Could be used for return type extraction

		// Skip if not exported and we're ignoring private APIs
		if d.config.IgnorePrivateAPIs && !isExported(name) {
			continue
		}

		// Find line number
		lineNum := d.findLineNumber(string(content), match[0])

		// Extract documentation (simple approach - look for comment block before function)
		documentation := d.extractDocumentation(lines, lineNum-1)

		funcDef := &FunctionDefinition{
			Name:          name,
			Package:       pkg.Name,
			FilePath:      relPath,
			LineNumber:    lineNum,
			Signature:     match[0],                 // Simplified - would be more precise in full implementation
			Parameters:    []*ParameterDefinition{}, // Would parse parameters properly
			Returns:       []*ReturnDefinition{},    // Would parse returns properly
			Documentation: documentation,
			IsExported:    isExported(name),
			IsMethod:      receiver != "",
			Metadata: map[string]string{
				"extraction_method": "regex",
				"receiver":          receiver,
			},
		}

		if receiver != "" {
			funcDef.Receiver = &ReceiverDefinition{
				Name:    strings.TrimSpace(strings.Trim(receiver, "*")),
				Type:    strings.TrimSpace(receiver),
				Pointer: strings.Contains(receiver, "*"),
			}
		}

		snapshot.Functions = append(snapshot.Functions, funcDef)

		// Add to public APIs if exported
		if funcDef.IsExported {
			api := &APIDefinition{
				Name:          funcDef.Name,
				Type:          "function",
				Package:       pkg.Name,
				FilePath:      relPath,
				LineNumber:    funcDef.LineNumber,
				Signature:     funcDef.Signature,
				Documentation: funcDef.Documentation,
				IsExported:    true,
				IsDeprecated:  strings.Contains(strings.ToLower(documentation), "deprecated"),
				Tags:          []string{"function"},
				Metadata:      funcDef.Metadata,
			}
			if funcDef.IsMethod {
				api.Tags = append(api.Tags, "method")
			}
			snapshot.PublicAPIs = append(snapshot.PublicAPIs, api)
		}
	}

	return nil
}

// extractTypes extracts type definitions from Go source code
func (d *DocAnalyzerImpl) extractTypes(content []byte, filePath, relPath string, pkg *PackageInfo, snapshot *CodebaseSnapshot) error {
	// Simplified regex-based extraction
	typeRegex := regexp.MustCompile(`(?m)^type\s+(\w+)\s+(struct|interface|[^{]*)\s*{?`)
	matches := typeRegex.FindAllStringSubmatch(string(content), -1)

	lines := strings.Split(string(content), "\n")

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		name := match[1]
		kind := match[2]

		// Skip if not exported and we're ignoring private APIs
		if d.config.IgnorePrivateAPIs && !isExported(name) {
			continue
		}

		// Find line number
		lineNum := d.findLineNumber(string(content), match[0])

		// Extract documentation
		documentation := d.extractDocumentation(lines, lineNum-1)

		typeDef := &TypeDefinition{
			Name:          name,
			Kind:          kind,
			Package:       pkg.Name,
			FilePath:      relPath,
			LineNumber:    lineNum,
			Fields:        []*FieldDefinition{},  // Would be populated in full implementation
			Methods:       []*MethodDefinition{}, // Would be populated in full implementation
			Documentation: documentation,
			IsExported:    isExported(name),
			Metadata: map[string]string{
				"extraction_method": "regex",
				"raw_definition":    match[0],
			},
		}

		snapshot.Types = append(snapshot.Types, typeDef)

		// Add to public APIs if exported
		if typeDef.IsExported {
			api := &APIDefinition{
				Name:          typeDef.Name,
				Type:          "type",
				Package:       pkg.Name,
				FilePath:      relPath,
				LineNumber:    typeDef.LineNumber,
				Signature:     match[0],
				Documentation: typeDef.Documentation,
				IsExported:    true,
				IsDeprecated:  strings.Contains(strings.ToLower(documentation), "deprecated"),
				Tags:          []string{"type", kind},
				Metadata:      typeDef.Metadata,
			}
			snapshot.PublicAPIs = append(snapshot.PublicAPIs, api)
		}
	}

	return nil
}

// extractInterfaces extracts interface definitions from Go source code
func (d *DocAnalyzerImpl) extractInterfaces(content []byte, filePath, relPath string, pkg *PackageInfo, snapshot *CodebaseSnapshot) error {
	// Simplified regex-based extraction for interfaces
	interfaceRegex := regexp.MustCompile(`(?m)^type\s+(\w+)\s+interface\s*{`)
	matches := interfaceRegex.FindAllStringSubmatch(string(content), -1)

	lines := strings.Split(string(content), "\n")

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		name := match[1]

		// Skip if not exported and we're ignoring private APIs
		if d.config.IgnorePrivateAPIs && !isExported(name) {
			continue
		}

		// Find line number
		lineNum := d.findLineNumber(string(content), match[0])

		// Extract documentation
		documentation := d.extractDocumentation(lines, lineNum-1)

		interfaceDef := &InterfaceDefinition{
			Name:          name,
			Package:       pkg.Name,
			FilePath:      relPath,
			LineNumber:    lineNum,
			Methods:       []*MethodDefinition{}, // Would be populated in full implementation
			Documentation: documentation,
			IsExported:    isExported(name),
			Metadata: map[string]string{
				"extraction_method": "regex",
				"raw_definition":    match[0],
			},
		}

		snapshot.Interfaces = append(snapshot.Interfaces, interfaceDef)

		// Add to public APIs if exported
		if interfaceDef.IsExported {
			api := &APIDefinition{
				Name:          interfaceDef.Name,
				Type:          "interface",
				Package:       pkg.Name,
				FilePath:      relPath,
				LineNumber:    interfaceDef.LineNumber,
				Signature:     match[0],
				Documentation: interfaceDef.Documentation,
				IsExported:    true,
				IsDeprecated:  strings.Contains(strings.ToLower(documentation), "deprecated"),
				Tags:          []string{"interface"},
				Metadata:      interfaceDef.Metadata,
			}
			snapshot.PublicAPIs = append(snapshot.PublicAPIs, api)
		}
	}

	return nil
}

// extractConstantsAndVariables extracts const and var declarations
func (d *DocAnalyzerImpl) extractConstantsAndVariables(content []byte, filePath, relPath string, pkg *PackageInfo, snapshot *CodebaseSnapshot) error {
	lines := strings.Split(string(content), "\n")

	// Extract constants
	constRegex := regexp.MustCompile(`(?m)^(const\s+(\w+)\s*=\s*([^/\n]+))`)
	constMatches := constRegex.FindAllStringSubmatch(string(content), -1)

	for _, match := range constMatches {
		if len(match) < 4 {
			continue
		}

		name := match[2]
		value := strings.TrimSpace(match[3])

		// Skip if not exported and we're ignoring private APIs
		if d.config.IgnorePrivateAPIs && !isExported(name) {
			continue
		}

		// Find line number
		lineNum := d.findLineNumber(string(content), match[0])

		// Extract documentation
		documentation := d.extractDocumentation(lines, lineNum-1)

		constDef := &ConstantDefinition{
			Name:          name,
			Type:          "unknown", // Would be determined in full implementation
			Value:         value,
			Package:       pkg.Name,
			FilePath:      relPath,
			LineNumber:    lineNum,
			Documentation: documentation,
			IsExported:    isExported(name),
			Metadata: map[string]string{
				"extraction_method": "regex",
				"raw_definition":    match[1],
			},
		}

		snapshot.Constants = append(snapshot.Constants, constDef)

		// Add to public APIs if exported
		if constDef.IsExported {
			api := &APIDefinition{
				Name:          constDef.Name,
				Type:          "constant",
				Package:       pkg.Name,
				FilePath:      relPath,
				LineNumber:    constDef.LineNumber,
				Signature:     match[1],
				Documentation: constDef.Documentation,
				IsExported:    true,
				IsDeprecated:  strings.Contains(strings.ToLower(documentation), "deprecated"),
				Tags:          []string{"constant"},
				Metadata:      constDef.Metadata,
			}
			snapshot.PublicAPIs = append(snapshot.PublicAPIs, api)
		}
	}

	// Extract variables
	varRegex := regexp.MustCompile(`(?m)^(var\s+(\w+)\s*([^=\n]*)?(\s*=\s*[^/\n]+)?)`)
	varMatches := varRegex.FindAllStringSubmatch(string(content), -1)

	for _, match := range varMatches {
		if len(match) < 3 {
			continue
		}

		name := match[2]

		// Skip if not exported and we're ignoring private APIs
		if d.config.IgnorePrivateAPIs && !isExported(name) {
			continue
		}

		// Find line number
		lineNum := d.findLineNumber(string(content), match[0])

		// Extract documentation
		documentation := d.extractDocumentation(lines, lineNum-1)

		varDef := &VariableDefinition{
			Name:          name,
			Type:          "unknown", // Would be determined in full implementation
			Package:       pkg.Name,
			FilePath:      relPath,
			LineNumber:    lineNum,
			Documentation: documentation,
			IsExported:    isExported(name),
			IsGlobal:      true, // Package-level variables are global
			Metadata: map[string]string{
				"extraction_method": "regex",
				"raw_definition":    match[1],
			},
		}

		snapshot.Variables = append(snapshot.Variables, varDef)

		// Add to public APIs if exported
		if varDef.IsExported {
			api := &APIDefinition{
				Name:          varDef.Name,
				Type:          "variable",
				Package:       pkg.Name,
				FilePath:      relPath,
				LineNumber:    varDef.LineNumber,
				Signature:     match[1],
				Documentation: varDef.Documentation,
				IsExported:    true,
				IsDeprecated:  strings.Contains(strings.ToLower(documentation), "deprecated"),
				Tags:          []string{"variable"},
				Metadata:      varDef.Metadata,
			}
			snapshot.PublicAPIs = append(snapshot.PublicAPIs, api)
		}
	}

	return nil
}

// Helper functions
func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	first := rune(name[0])
	return first >= 'A' && first <= 'Z'
}

func (d *DocAnalyzerImpl) findLineNumber(content, match string) int {
	index := strings.Index(content, match)
	if index == -1 {
		return 1
	}
	return strings.Count(content[:index], "\n") + 1
}

func (d *DocAnalyzerImpl) extractDocumentation(lines []string, lineIndex int) string {
	var docLines []string

	// Look backwards from the function/type declaration for comment blocks
	for i := lineIndex - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		// Empty line breaks the documentation block
		if line == "" {
			break
		}

		// Documentation comment
		if strings.HasPrefix(line, "//") {
			comment := strings.TrimSpace(strings.TrimPrefix(line, "//"))
			docLines = append([]string{comment}, docLines...) // Prepend to maintain order
		} else if strings.HasPrefix(line, "/*") && strings.HasSuffix(line, "*/") {
			// Single line block comment
			comment := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "/*"), "*/"))
			docLines = append([]string{comment}, docLines...)
		} else {
			// Non-comment line breaks the documentation block
			break
		}
	}

	return strings.Join(docLines, " ")
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
