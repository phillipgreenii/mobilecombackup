package validation

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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
	ValidateRepositoryWithOptions(ctx context.Context, options *PerformanceOptions) (*Report, error)

	// ValidateAsync performs validation asynchronously with progress reporting
	ValidateAsync(options *PerformanceOptions) (<-chan *Report, <-chan error)

	// GetMetrics returns current validation performance metrics
	GetMetrics() *Metrics

	// ClearCache clears the validation cache
	ClearCache()
}

// OptimizedRepositoryValidatorImpl implements optimized validation
type OptimizedRepositoryValidatorImpl struct {
	*RepositoryValidatorImpl
	cache   *validationCache
	metrics *Metrics
}

// Metrics tracks validation performance statistics
type Metrics struct {
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
	// TODO: Update this after context refactoring is complete
	// For now, create a placeholder implementation
	baseImpl, ok := base.(*RepositoryValidatorImpl)
	if !ok {
		// For now, return a basic wrapper that delegates everything to base
		// This is a temporary solution during context refactoring
		panic("OptimizedRepositoryValidator temporarily disabled during context refactoring")
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
		metrics: &Metrics{},
	}
}

// ValidateRepositoryWithOptions performs validation with performance controls
func (v *OptimizedRepositoryValidatorImpl) ValidateRepositoryWithOptions(
	ctx context.Context,
	options *PerformanceOptions,
) (*Report, error) {
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

	report := &Report{
		Timestamp:      time.Now().UTC(),
		RepositoryPath: v.repositoryRoot,
		Status:         Valid,
		Violations:     []Violation{},
	}

	if options.ParallelValidation {
		return v.validateParallel(ctx, report, options)
	}
	return v.validateSequential(ctx, report, options)
}

// ValidateAsync performs validation asynchronously with progress reporting
func (v *OptimizedRepositoryValidatorImpl) ValidateAsync(options *PerformanceOptions) (<-chan *Report, <-chan error) {
	reportCh := make(chan *Report, 1)
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
func (v *OptimizedRepositoryValidatorImpl) validateParallel(
	ctx context.Context,
	report *Report,
	options *PerformanceOptions,
) (*Report, error) {
	// Setup concurrency control and channels
	semaphore, violationsCh, errorCh, wg := v.setupParallelValidation(options)

	// Setup progress tracking
	updateProgress := v.createProgressCallback(options)

	// Launch validation workers
	v.launchValidationWorkers(ctx, semaphore, violationsCh, errorCh, wg, options, updateProgress)

	// Collect results
	allViolations, err := v.collectValidationResults(ctx, violationsCh, errorCh, wg)
	if err != nil {
		return nil, err
	}

	// Finalize report
	report.Violations = allViolations
	v.determineStatus(report)

	return report, nil
}

// setupParallelValidation initializes concurrency control and communication channels
func (v *OptimizedRepositoryValidatorImpl) setupParallelValidation(
	options *PerformanceOptions,
) (chan struct{}, chan []Violation, chan error, *sync.WaitGroup) {
	semaphore := make(chan struct{}, options.MaxConcurrency)
	violationsCh := make(chan []Violation, 4)
	errorCh := make(chan error, 4)
	var wg sync.WaitGroup
	return semaphore, violationsCh, errorCh, &wg
}

// createProgressCallback creates a thread-safe progress update function
func (v *OptimizedRepositoryValidatorImpl) createProgressCallback(
	options *PerformanceOptions,
) func(string) {
	var completedStages int32
	const totalStages = 4 // Number of validation stages
	var progressMu sync.Mutex

	return func(stage string) {
		completed := atomic.AddInt32(&completedStages, 1)
		if options.ProgressCallback != nil {
			// Protect the callback invocation to ensure ordered/consistent calls
			progressMu.Lock()
			progress := float64(completed) / float64(totalStages)
			options.ProgressCallback(stage, progress)
			progressMu.Unlock()
		}
	}
}

// launchValidationWorkers starts concurrent validation tasks
func (v *OptimizedRepositoryValidatorImpl) launchValidationWorkers(
	ctx context.Context,
	semaphore chan struct{},
	violationsCh chan []Violation,
	errorCh chan error,
	wg *sync.WaitGroup,
	options *PerformanceOptions,
	updateProgress func(string),
) {
	validationTasks := []struct {
		name string
		fn   func() []Violation
	}{
		{"structure", v.ValidateStructure},
		{"manifest", v.ValidateManifest},
		{"content", v.ValidateContent},
		{"consistency", v.ValidateConsistency},
	}

	for _, task := range validationTasks {
		wg.Add(1)
		go func(taskName string, taskFn func() []Violation) {
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
}

// collectValidationResults gathers results from all validation workers
func (v *OptimizedRepositoryValidatorImpl) collectValidationResults(
	ctx context.Context,
	violationsCh chan []Violation,
	errorCh chan error,
	_ *sync.WaitGroup,
) ([]Violation, error) {
	allViolations := []Violation{}
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

	return allViolations, nil
}

// validateSequential performs validation sequentially with optimizations
func (v *OptimizedRepositoryValidatorImpl) validateSequential(
	ctx context.Context,
	report *Report,
	options *PerformanceOptions,
) (*Report, error) {
	tasks := []struct {
		name string
		fn   func() []Violation
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
func (v *OptimizedRepositoryValidatorImpl) hasCriticalErrors(violations []Violation) bool {
	for _, violation := range violations {
		if violation.Severity == SeverityError && v.isCriticalViolation(violation) {
			return true
		}
	}
	return false
}

// isCriticalViolation determines if a violation is critical for early termination
func (v *OptimizedRepositoryValidatorImpl) isCriticalViolation(violation Violation) bool {
	// Consider certain types as critical
	criticalTypes := map[ViolationType]bool{
		ChecksumMismatch:   true,
		StructureViolation: true,
	}
	return criticalTypes[violation.Type]
}

// determineStatus sets the overall validation status
func (v *OptimizedRepositoryValidatorImpl) determineStatus(report *Report) {
	hasErrors := false
	for _, violation := range report.Violations {
		if violation.Severity == SeverityError {
			hasErrors = true
			break
		}
	}

	switch {
	case hasErrors:
		report.Status = Invalid
	default:
		report.Status = Valid // Valid with or without warnings
	}
}

// GetMetrics returns current validation performance metrics
func (v *OptimizedRepositoryValidatorImpl) GetMetrics() *Metrics {
	v.metrics.mu.RLock()
	defer v.metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &Metrics{
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
	Violations []Violation
}

func (e *EarlyTerminationError) Error() string {
	return fmt.Sprintf("validation terminated early at stage '%s' due to critical errors", e.Stage)
}

// Timeout indicates validation exceeded the specified timeout
type Timeout struct {
	Timeout time.Duration
}

func (e *Timeout) Error() string {
	return fmt.Sprintf("validation timed out after %v", e.Timeout)
}
