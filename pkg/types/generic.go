// Package types provides generic types and utilities for improved type safety
// and reduced code duplication across the mobilecombackup codebase.
package types

import (
	"strconv"
)

const (
	// XMLNullValue represents the null value in XML attributes
	XMLNullValue = "null"
)

// Result represents a generic operation result that can either contain a value or an error
type Result[T any] struct {
	Value T
	Error error
}

// NewResult creates a Result with a successful value
func NewResult[T any](value T) Result[T] {
	return Result[T]{Value: value}
}

// NewResultError creates a Result with an error
func NewResultError[T any](err error) Result[T] {
	var zero T
	return Result[T]{Value: zero, Error: err}
}

// IsOk returns true if the result contains a value (no error)
func (r Result[T]) IsOk() bool {
	return r.Error == nil
}

// IsErr returns true if the result contains an error
func (r Result[T]) IsErr() bool {
	return r.Error != nil
}

// Unwrap returns the value, panicking if there's an error
// Use this only when you're certain the result is Ok
func (r Result[T]) Unwrap() T {
	if r.Error != nil {
		panic("called Unwrap on Result with error: " + r.Error.Error())
	}
	return r.Value
}

// UnwrapOr returns the value if Ok, otherwise returns the provided default
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.Error != nil {
		return defaultValue
	}
	return r.Value
}

// Optional represents a value that may or may not be present
// This is useful for parsing XML attributes that might be "null" or empty
type Optional[T any] struct {
	value    T
	hasValue bool
}

// None creates an empty Optional
func None[T any]() Optional[T] {
	return Optional[T]{hasValue: false}
}

// Some creates an Optional with a value
func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, hasValue: true}
}

// IsSome returns true if the Optional contains a value
func (o Optional[T]) IsSome() bool {
	return o.hasValue
}

// IsNone returns true if the Optional is empty
func (o Optional[T]) IsNone() bool {
	return !o.hasValue
}

// Unwrap returns the value, panicking if empty
func (o Optional[T]) Unwrap() T {
	if !o.hasValue {
		panic("called Unwrap on empty Optional")
	}
	return o.value
}

// UnwrapOr returns the value if present, otherwise returns the provided default
func (o Optional[T]) UnwrapOr(defaultValue T) T {
	if !o.hasValue {
		return defaultValue
	}
	return o.value
}

// ParseOptionalInt parses a string to an optional integer
// Returns None if the string is empty, "null", or invalid
func ParseOptionalInt(s string) Optional[int] {
	if s == "" || s == XMLNullValue {
		return None[int]()
	}

	value, err := strconv.Atoi(s)
	if err != nil {
		return None[int]()
	}

	return Some(value)
}

// ParseOptionalInt64 parses a string to an optional int64
// Returns None if the string is empty, "null", or invalid
func ParseOptionalInt64(s string) Optional[int64] {
	if s == "" || s == XMLNullValue {
		return None[int64]()
	}

	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return None[int64]()
	}

	return Some(value)
}

// ParseOptionalString parses a string to an optional string
// Returns None if the string is empty or "null"
func ParseOptionalString(s string) Optional[string] {
	if s == "" || s == XMLNullValue {
		return None[string]()
	}

	return Some(s)
}

// TryParseInt attempts to parse a string as an integer, returning a Result
func TryParseInt(s string) Result[int] {
	value, err := strconv.Atoi(s)
	if err != nil {
		return NewResultError[int](err)
	}
	return NewResult(value)
}

// TryParseInt64 attempts to parse a string as an int64, returning a Result
func TryParseInt64(s string) Result[int64] {
	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return NewResultError[int64](err)
	}
	return NewResult(value)
}

// StateManager represents the core state management interface for docsync
type StateManager[T any] interface {
	// Get retrieves the current state
	Get() Result[T]

	// Set updates the state with validation
	Set(state T) Result[T]

	// Reset restores state to defaults
	Reset() Result[T]

	// Persist saves state to storage
	Persist() Result[bool]

	// Load restores state from storage
	Load() Result[T]

	// Validate checks state consistency
	Validate() Result[bool]
}

// StateStatus represents the current status of state management
type StateStatus string

const (
	StateStatusUninitialized StateStatus = "uninitialized"
	StateStatusLoading       StateStatus = "loading"
	StateStatusReady         StateStatus = "ready"
	StateStatusPersisting    StateStatus = "persisting"
	StateStatusError         StateStatus = "error"
)

// StateMetadata contains metadata about state management
type StateMetadata struct {
	Status      StateStatus `json:"status" yaml:"status"`
	LastUpdated int64       `json:"last_updated" yaml:"last_updated"`
	Version     string      `json:"version" yaml:"version"`
	Checksum    string      `json:"checksum" yaml:"checksum"`
}

// DocSyncState represents the complete documentation synchronization state
type DocSyncState struct {
	// Configuration
	Config DocSyncConfig `json:"config" yaml:"config"`

	// Runtime state
	ActiveAgents []string       `json:"active_agents" yaml:"active_agents"`
	LastSyncTime int64          `json:"last_sync_time" yaml:"last_sync_time"`
	SyncStatus   DocSyncStatus  `json:"sync_status" yaml:"sync_status"`
	ErrorHistory []DocSyncError `json:"error_history" yaml:"error_history"`

	// Performance metrics
	Metrics DocSyncMetrics `json:"metrics" yaml:"metrics"`

	// Metadata
	Metadata StateMetadata `json:"metadata" yaml:"metadata"`
}

// DocSyncConfig contains configuration for documentation synchronization
type DocSyncConfig struct {
	// Basic settings
	Enabled   bool `json:"enabled" yaml:"enabled"`
	WatchMode bool `json:"watch_mode" yaml:"watch_mode"`
	AutoFix   bool `json:"auto_fix" yaml:"auto_fix"`

	// Agent configuration
	EnabledAgents []string `json:"enabled_agents" yaml:"enabled_agents"`
	AgentTimeout  int      `json:"agent_timeout" yaml:"agent_timeout"`

	// File patterns
	IncludePatterns []string `json:"include_patterns" yaml:"include_patterns"`
	ExcludePatterns []string `json:"exclude_patterns" yaml:"exclude_patterns"`

	// Performance settings
	MaxConcurrency int `json:"max_concurrency" yaml:"max_concurrency"`
	BatchSize      int `json:"batch_size" yaml:"batch_size"`
}

// DocSyncStatus represents the current synchronization status
type DocSyncStatus string

const (
	DocSyncStatusIdle      DocSyncStatus = "idle"
	DocSyncStatusRunning   DocSyncStatus = "running"
	DocSyncStatusCompleted DocSyncStatus = "completed"
	DocSyncStatusFailed    DocSyncStatus = "failed"
	DocSyncStatusPartial   DocSyncStatus = "partial"
)

// DocSyncError represents an error in documentation synchronization
type DocSyncError struct {
	Timestamp int64  `json:"timestamp" yaml:"timestamp"`
	Agent     string `json:"agent" yaml:"agent"`
	ErrorType string `json:"error_type" yaml:"error_type"`
	Message   string `json:"message" yaml:"message"`
	FilePath  string `json:"file_path" yaml:"file_path"`
	Severity  string `json:"severity" yaml:"severity"`
}

// DocSyncMetrics contains performance and quality metrics
type DocSyncMetrics struct {
	// Performance metrics
	TotalRunTime      int64 `json:"total_runtime" yaml:"total_runtime"`
	FilesProcessed    int64 `json:"files_processed" yaml:"files_processed"`
	FilesFixed        int64 `json:"files_fixed" yaml:"files_fixed"`
	ErrorsEncountered int64 `json:"errors_encountered" yaml:"errors_encountered"`

	// Quality metrics
	DocumentationCoverage float64 `json:"documentation_coverage" yaml:"documentation_coverage"`
	ConsistencyScore      float64 `json:"consistency_score" yaml:"consistency_score"`
	QualityScore          float64 `json:"quality_score" yaml:"quality_score"`
}

// Security Framework Types for FEAT-084

// SecurityRole represents a role in the RBAC system
type SecurityRole struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Permissions []string `json:"permissions" yaml:"permissions"`
	ParentRoles []string `json:"parent_roles" yaml:"parent_roles"`
}

// SecurityPermission represents a specific permission
type SecurityPermission struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Resource    string `json:"resource" yaml:"resource"`
	Action      string `json:"action" yaml:"action"`
	Description string `json:"description" yaml:"description"`
}

// SecurityUser represents a user in the system
type SecurityUser struct {
	ID       string   `json:"id" yaml:"id"`
	Username string   `json:"username" yaml:"username"`
	Roles    []string `json:"roles" yaml:"roles"`
	Active   bool     `json:"active" yaml:"active"`
}

// SecurityContext provides security context for operations
type SecurityContext struct {
	User        SecurityUser      `json:"user" yaml:"user"`
	SessionID   string            `json:"session_id" yaml:"session_id"`
	Permissions []string          `json:"permissions" yaml:"permissions"`
	Metadata    map[string]string `json:"metadata" yaml:"metadata"`
	CreatedAt   int64             `json:"created_at" yaml:"created_at"`
	ExpiresAt   int64             `json:"expires_at" yaml:"expires_at"`
}

// AuditEvent represents a security event for audit logging
type AuditEvent struct {
	ID        string                 `json:"id" yaml:"id"`
	Timestamp int64                  `json:"timestamp" yaml:"timestamp"`
	UserID    string                 `json:"user_id" yaml:"user_id"`
	Action    string                 `json:"action" yaml:"action"`
	Resource  string                 `json:"resource" yaml:"resource"`
	Result    string                 `json:"result" yaml:"result"` // success, failure, denied
	Details   map[string]interface{} `json:"details" yaml:"details"`
	IPAddress string                 `json:"ip_address" yaml:"ip_address"`
	UserAgent string                 `json:"user_agent" yaml:"user_agent"`
}

// SecurityValidationResult represents the result of a security validation
type SecurityValidationResult struct {
	Valid    bool     `json:"valid" yaml:"valid"`
	Errors   []string `json:"errors" yaml:"errors"`
	Warnings []string `json:"warnings" yaml:"warnings"`
	Context  string   `json:"context" yaml:"context"`
}

// Security Interface Definitions

// SecurityContextManager manages security contexts and sessions
type SecurityContextManager interface {
	CreateContext(user SecurityUser, sessionID string) (SecurityContext, error)
	ValidateContext(ctx SecurityContext) SecurityValidationResult
	RefreshContext(ctx SecurityContext) (SecurityContext, error)
	RevokeContext(sessionID string) error
	GetUserPermissions(userID string) ([]string, error)
}

// AuditLogger handles security event logging and retrieval
type AuditLogger interface {
	LogEvent(event AuditEvent) error
	GetEvents(userID string, fromTime, toTime int64) ([]AuditEvent, error)
	GetEventsByAction(action string, fromTime, toTime int64) ([]AuditEvent, error)
	GetEventsByResource(resource string, fromTime, toTime int64) ([]AuditEvent, error)
	SearchEvents(query string, fromTime, toTime int64) ([]AuditEvent, error)
}

// AccessController manages role-based access control
type AccessController interface {
	CheckPermission(ctx SecurityContext, resource, action string) bool
	GetUserRoles(userID string) ([]SecurityRole, error)
	GetRolePermissions(roleID string) ([]SecurityPermission, error)
	AssignRole(userID, roleID string) error
	RevokeRole(userID, roleID string) error
	CreateRole(role SecurityRole) error
	UpdateRole(role SecurityRole) error
	DeleteRole(roleID string) error
}

// SecurityValidator provides security validation utilities
type SecurityValidator interface {
	ValidateInput(input string, inputType string) SecurityValidationResult
	SanitizeInput(input string, inputType string) string
	ValidatePermissions(permissions []string) SecurityValidationResult
	ValidateRole(role SecurityRole) SecurityValidationResult
	ValidateUser(user SecurityUser) SecurityValidationResult
}
