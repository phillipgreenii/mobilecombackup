package validation

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PerformanceOptions controls validation performance behavior
type PerformanceOptions struct {
	// ParallelValidation enables concurrent validation of different components
	ParallelValidation bool

	// EarlyTermination stops validation on first critical error
	EarlyTermination bool

	// MaxConcurrency limits the number of concurrent validation operations
	MaxConcurrency int

	// ProgressCallback receives progress updates during validation
	ProgressCallback func(stage string, progress float64)

	// Timeout sets maximum time for validation to complete
	Timeout time.Duration
}

// DefaultPerformanceOptions returns recommended performance settings
func DefaultPerformanceOptions() *PerformanceOptions {
	return &PerformanceOptions{
		ParallelValidation: true,
		EarlyTermination:   false,
		MaxConcurrency:     4,
		ProgressCallback:   nil,
		Timeout:            30 * time.Minute,
	}
}

// OptimizedRepositoryValidator extends RepositoryValidator with performance optimizations
type OptimizedRepositoryValidator interface {
	RepositoryValidator

	// ValidateRepositoryWithOptions performs validation with performance controls
	ValidateRepositoryWithOptions(ctx context.Context, options *PerformanceOptions) (*ValidationReport, error)

	// ValidateAsync performs validation asynchronously with progress reporting
	ValidateAsync(options *PerformanceOptions) (<-chan *ValidationReport, <-chan error)

	// GetMetrics returns current validation performance metrics
	GetMetrics() *ValidationMetrics

	// ClearCache clears the validation cache
	ClearCache()
}

// OptimizedRepositoryValidatorImpl implements optimized validation
type OptimizedRepositoryValidatorImpl struct {
	*RepositoryValidatorImpl
	cache   *validationCache
	metrics *ValidationMetrics
}

// ValidationMetrics tracks validation performance statistics
type ValidationMetrics struct {
	mu                  sync.RWMutex
	TotalDuration       time.Duration
	StructureDuration   time.Duration
	ManifestDuration    time.Duration
	ContentDuration     time.Duration
	ConsistencyDuration time.Duration
	FilesProcessed      int
	CacheHits           int
	CacheMisses         int
}

// validationCache provides caching for frequently accessed validation data
type validationCache struct {
	mu           sync.RWMutex
	checksums    map[string]string    // file path -> checksum
	fileStats    map[string]FileInfo  // file path -> file info
	lastModified map[string]time.Time // file path -> last modified time
	enabled      bool
	maxSize      int
}

// FileInfo holds cached file information
type FileInfo struct {
	Size    int64
	ModTime time.Time
	Exists  bool
}

// NewOptimizedRepositoryValidator creates an optimized repository validator
func NewOptimizedRepositoryValidator(base RepositoryValidator) OptimizedRepositoryValidator {
	baseImpl, ok := base.(*RepositoryValidatorImpl)
	if !ok {
		// Fallback: create a new base validator if cast fails
		// This would normally require the constructor parameters
		panic("cannot optimize non-standard RepositoryValidator implementation")
	}

	return &OptimizedRepositoryValidatorImpl{
		RepositoryValidatorImpl: baseImpl,
		cache: &validationCache{
			checksums:    make(map[string]string),
			fileStats:    make(map[string]FileInfo),
			lastModified: make(map[string]time.Time),
			enabled:      true,
			maxSize:      1000,
		},
		metrics: &ValidationMetrics{},
	}
}

// ValidateRepositoryWithOptions performs validation with performance controls
func (v *OptimizedRepositoryValidatorImpl) ValidateRepositoryWithOptions(ctx context.Context, options *PerformanceOptions) (*ValidationReport, error) {
	if options == nil {
		options = DefaultPerformanceOptions()
	}

	// Set up timeout context if specified
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	startTime := time.Now()
	defer func() {
		v.metrics.mu.Lock()
		v.metrics.TotalDuration = time.Since(startTime)
		v.metrics.mu.Unlock()
	}()

	report := &ValidationReport{
		Timestamp:      time.Now().UTC(),
		RepositoryPath: v.repositoryRoot,
		Status:         Valid,
		Violations:     []ValidationViolation{},
	}

	if options.ParallelValidation {
		return v.validateParallel(ctx, report, options)
	} else {
		return v.validateSequential(ctx, report, options)
	}
}

// ValidateAsync performs validation asynchronously with progress reporting
func (v *OptimizedRepositoryValidatorImpl) ValidateAsync(options *PerformanceOptions) (<-chan *ValidationReport, <-chan error) {
	reportCh := make(chan *ValidationReport, 1)
	errorCh := make(chan error, 1)

	go func() {
		defer close(reportCh)
		defer close(errorCh)

		ctx := context.Background()
		report, err := v.ValidateRepositoryWithOptions(ctx, options)
		if err != nil {
			errorCh <- err
			return
		}
		reportCh <- report
	}()

	return reportCh, errorCh
}

// validateParallel performs validation with concurrent component validation
func (v *OptimizedRepositoryValidatorImpl) validateParallel(ctx context.Context, report *ValidationReport, options *PerformanceOptions) (*ValidationReport, error) {
	// Use worker pool to limit concurrency
	semaphore := make(chan struct{}, options.MaxConcurrency)
	var wg sync.WaitGroup
	violationsCh := make(chan []ValidationViolation, 4)
	errorCh := make(chan error, 4)

	// Progress tracking
	stages := []string{"structure", "manifest", "content", "consistency"}
	completedStages := 0
	totalStages := len(stages)

	updateProgress := func(stage string) {
		completedStages++
		if options.ProgressCallback != nil {
			progress := float64(completedStages) / float64(totalStages)
			options.ProgressCallback(stage, progress)
		}
	}

	// Launch validation workers
	validationTasks := []struct {
		name string
		fn   func() []ValidationViolation
	}{
		{"structure", v.ValidateStructure},
		{"manifest", v.ValidateManifest},
		{"content", v.ValidateContent},
		{"consistency", v.ValidateConsistency},
	}

	for _, task := range validationTasks {
		wg.Add(1)
		go func(taskName string, taskFn func() []ValidationViolation) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check for context cancellation
			select {
			case <-ctx.Done():
				errorCh <- ctx.Err()
				return
			default:
			}

			startTime := time.Now()
			violations := taskFn()
			duration := time.Since(startTime)

			// Update metrics
			v.updateTaskMetrics(taskName, duration)

			// Check for early termination
			if options.EarlyTermination && v.hasCriticalErrors(violations) {
				errorCh <- &EarlyTerminationError{Stage: taskName, Violations: violations}
				return
			}

			violationsCh <- violations
			updateProgress(taskName)
		}(task.name, task.fn)
	}

	// Wait for all tasks to complete
	go func() {
		wg.Wait()
		close(violationsCh)
		close(errorCh)
	}()

	// Collect results
	allViolations := []ValidationViolation{}
	for {
		select {
		case violations, ok := <-violationsCh:
			if !ok {
				violationsCh = nil
			} else {
				allViolations = append(allViolations, violations...)
			}
		case err, ok := <-errorCh:
			if !ok {
				errorCh = nil
			} else if err != nil {
				return nil, err
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		if violationsCh == nil && errorCh == nil {
			break
		}
	}

	// Finalize report
	report.Violations = allViolations
	v.determineStatus(report)

	return report, nil
}

// validateSequential performs validation sequentially with optimizations
func (v *OptimizedRepositoryValidatorImpl) validateSequential(ctx context.Context, report *ValidationReport, options *PerformanceOptions) (*ValidationReport, error) {
	tasks := []struct {
		name string
		fn   func() []ValidationViolation
	}{
		{"structure", v.ValidateStructure},
		{"manifest", v.ValidateManifest},
		{"content", v.ValidateContent},
		{"consistency", v.ValidateConsistency},
	}

	totalTasks := len(tasks)
	for i, task := range tasks {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Progress callback
		if options.ProgressCallback != nil {
			progress := float64(i) / float64(totalTasks)
			options.ProgressCallback(task.name, progress)
		}

		startTime := time.Now()
		violations := task.fn()
		duration := time.Since(startTime)

		// Update metrics
		v.updateTaskMetrics(task.name, duration)

		// Add violations to report
		report.Violations = append(report.Violations, violations...)

		// Check for early termination
		if options.EarlyTermination && v.hasCriticalErrors(violations) {
			v.determineStatus(report)
			return report, &EarlyTerminationError{Stage: task.name, Violations: violations}
		}
	}

	// Final progress update
	if options.ProgressCallback != nil {
		options.ProgressCallback("complete", 1.0)
	}

	v.determineStatus(report)
	return report, nil
}

// updateTaskMetrics updates performance metrics for a specific task
func (v *OptimizedRepositoryValidatorImpl) updateTaskMetrics(taskName string, duration time.Duration) {
	v.metrics.mu.Lock()
	defer v.metrics.mu.Unlock()

	switch taskName {
	case "structure":
		v.metrics.StructureDuration = duration
	case "manifest":
		v.metrics.ManifestDuration = duration
	case "content":
		v.metrics.ContentDuration = duration
	case "consistency":
		v.metrics.ConsistencyDuration = duration
	}
}

// hasCriticalErrors checks if violations contain critical errors
func (v *OptimizedRepositoryValidatorImpl) hasCriticalErrors(violations []ValidationViolation) bool {
	for _, violation := range violations {
		if violation.Severity == SeverityError && v.isCriticalViolation(violation) {
			return true
		}
	}
	return false
}

// isCriticalViolation determines if a violation is critical for early termination
func (v *OptimizedRepositoryValidatorImpl) isCriticalViolation(violation ValidationViolation) bool {
	// Consider certain types as critical
	criticalTypes := map[ViolationType]bool{
		ChecksumMismatch:   true,
		StructureViolation: true,
	}
	return criticalTypes[violation.Type]
}

// determineStatus sets the overall validation status
func (v *OptimizedRepositoryValidatorImpl) determineStatus(report *ValidationReport) {
	hasErrors := false
	for _, violation := range report.Violations {
		if violation.Severity == SeverityError {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		report.Status = Invalid
	} else if len(report.Violations) > 0 {
		report.Status = Valid // Valid with warnings
	} else {
		report.Status = Valid
	}
}

// GetMetrics returns current validation performance metrics
func (v *OptimizedRepositoryValidatorImpl) GetMetrics() *ValidationMetrics {
	v.metrics.mu.RLock()
	defer v.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &ValidationMetrics{
		TotalDuration:       v.metrics.TotalDuration,
		StructureDuration:   v.metrics.StructureDuration,
		ManifestDuration:    v.metrics.ManifestDuration,
		ContentDuration:     v.metrics.ContentDuration,
		ConsistencyDuration: v.metrics.ConsistencyDuration,
		FilesProcessed:      v.metrics.FilesProcessed,
		CacheHits:           v.metrics.CacheHits,
		CacheMisses:         v.metrics.CacheMisses,
	}
}

// ClearCache clears the validation cache
func (v *OptimizedRepositoryValidatorImpl) ClearCache() {
	v.cache.mu.Lock()
	defer v.cache.mu.Unlock()

	v.cache.checksums = make(map[string]string)
	v.cache.fileStats = make(map[string]FileInfo)
	v.cache.lastModified = make(map[string]time.Time)
}

// EarlyTerminationError indicates validation was terminated early due to critical errors
type EarlyTerminationError struct {
	Stage      string
	Violations []ValidationViolation
}

func (e *EarlyTerminationError) Error() string {
	return fmt.Sprintf("validation terminated early at stage '%s' due to critical errors", e.Stage)
}

// ValidationTimeout indicates validation exceeded the specified timeout
type ValidationTimeout struct {
	Timeout time.Duration
}

func (e *ValidationTimeout) Error() string {
	return fmt.Sprintf("validation timed out after %v", e.Timeout)
}
