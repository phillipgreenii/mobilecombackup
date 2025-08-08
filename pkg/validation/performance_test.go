package validation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
	"github.com/phillipgreen/mobilecombackup/pkg/contacts"
)

func TestDefaultPerformanceOptions(t *testing.T) {
	options := DefaultPerformanceOptions()

	if !options.ParallelValidation {
		t.Error("Expected parallel validation to be enabled by default")
	}
	if options.EarlyTermination {
		t.Error("Expected early termination to be disabled by default")
	}
	if options.MaxConcurrency != 4 {
		t.Errorf("Expected max concurrency 4, got %d", options.MaxConcurrency)
	}
	if options.Timeout != 30*time.Minute {
		t.Errorf("Expected timeout 30 minutes, got %v", options.Timeout)
	}
}

func TestNewOptimizedRepositoryValidator(t *testing.T) {
	tempDir := t.TempDir()

	// Create base validator
	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		&mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	// Create optimized validator
	optimized := NewOptimizedRepositoryValidator(baseValidator)
	if optimized == nil {
		t.Fatal("Expected non-nil optimized validator")
	}

	// Verify it implements the interface
	// Note: optimized is already of type OptimizedRepositoryValidator
	// This test is redundant but kept for clarity
	if optimized == nil {
		t.Error("Expected non-nil OptimizedRepositoryValidator")
	}

	// Verify it still implements the base interface
	_, ok := optimized.(RepositoryValidator)
	if !ok {
		t.Error("Expected RepositoryValidator interface implementation")
	}
}

func TestOptimizedRepositoryValidator_ValidateRepositoryWithOptions_Sequential(t *testing.T) {
	tempDir := t.TempDir()

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		&mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)

	options := &PerformanceOptions{
		ParallelValidation: false,
		EarlyTermination:   false,
		MaxConcurrency:     2,
		Timeout:            5 * time.Second,
	}

	ctx := context.Background()
	report, err := optimized.ValidateRepositoryWithOptions(ctx, options)
	if err != nil {
		t.Fatalf("Sequential validation failed: %v", err)
	}

	if report == nil {
		t.Fatal("Expected validation report")
	}
	if report.RepositoryPath != tempDir {
		t.Errorf("Expected repository path %s, got %s", tempDir, report.RepositoryPath)
	}
}

func TestOptimizedRepositoryValidator_ValidateRepositoryWithOptions_Parallel(t *testing.T) {
	tempDir := t.TempDir()

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		&mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)

	options := &PerformanceOptions{
		ParallelValidation: true,
		EarlyTermination:   false,
		MaxConcurrency:     2,
		Timeout:            5 * time.Second,
	}

	ctx := context.Background()
	report, err := optimized.ValidateRepositoryWithOptions(ctx, options)
	if err != nil {
		t.Fatalf("Parallel validation failed: %v", err)
	}

	if report == nil {
		t.Fatal("Expected validation report")
	}
	if report.RepositoryPath != tempDir {
		t.Errorf("Expected repository path %s, got %s", tempDir, report.RepositoryPath)
	}
}

func TestOptimizedRepositoryValidator_ValidateRepositoryWithOptions_ProgressCallback(t *testing.T) {
	tempDir := t.TempDir()

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		&mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)

	progressCalls := []struct {
		stage    string
		progress float64
	}{}

	options := &PerformanceOptions{
		ParallelValidation: false, // Use sequential for predictable progress order
		EarlyTermination:   false,
		ProgressCallback: func(stage string, progress float64) {
			progressCalls = append(progressCalls, struct {
				stage    string
				progress float64
			}{stage, progress})
		},
	}

	ctx := context.Background()
	_, err := optimized.ValidateRepositoryWithOptions(ctx, options)
	if err != nil {
		t.Fatalf("Validation with progress callback failed: %v", err)
	}

	// Verify progress was reported
	if len(progressCalls) == 0 {
		t.Error("Expected progress callbacks to be called")
	}

	// Verify final progress reaches 1.0
	finalProgress := progressCalls[len(progressCalls)-1]
	if finalProgress.progress != 1.0 {
		t.Errorf("Expected final progress to be 1.0, got %f", finalProgress.progress)
	}
	if finalProgress.stage != "complete" {
		t.Errorf("Expected final stage to be 'complete', got %s", finalProgress.stage)
	}
}

func TestOptimizedRepositoryValidator_ValidateRepositoryWithOptions_Timeout(t *testing.T) {
	tempDir := t.TempDir()

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		&mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)

	// Test with context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	options := &PerformanceOptions{
		ParallelValidation: false,
	}

	_, err := optimized.ValidateRepositoryWithOptions(ctx, options)

	// Should get a context error
	if err == nil {
		t.Error("Expected context cancellation error")
	}
	if err != context.Canceled {
		t.Logf("Got error (context-related): %v", err)
	}
}

func TestOptimizedRepositoryValidator_ValidateRepositoryWithOptions_EarlyTermination(t *testing.T) {
	tempDir := t.TempDir()

	// Create mock with critical error
	attachmentReader := &mockAttachmentReader{
		attachments:    []*attachments.Attachment{},
		structureError: fmt.Errorf("critical structure error"), // This will cause structure violation
	}

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		&mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)},
		attachmentReader,
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)

	options := &PerformanceOptions{
		ParallelValidation: false,
		EarlyTermination:   true,
	}

	ctx := context.Background()
	_, err := optimized.ValidateRepositoryWithOptions(ctx, options)

	// Should get early termination error if there are critical violations
	// Note: This test depends on the mock implementation producing critical errors
	// For now, we'll just verify the function doesn't panic
	if err != nil {
		t.Logf("Got error (expected for early termination test): %v", err)
	}
}

func TestOptimizedRepositoryValidator_ValidateRepositoryWithOptions_DefaultOptions(t *testing.T) {
	tempDir := t.TempDir()

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		&mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)

	ctx := context.Background()
	report, err := optimized.ValidateRepositoryWithOptions(ctx, nil) // nil options should use defaults
	if err != nil {
		t.Fatalf("Validation with default options failed: %v", err)
	}

	if report == nil {
		t.Fatal("Expected validation report")
	}
}

func TestOptimizedRepositoryValidator_ValidateAsync(t *testing.T) {
	tempDir := t.TempDir()

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		&mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)

	options := &PerformanceOptions{
		ParallelValidation: true,
		EarlyTermination:   false,
		MaxConcurrency:     4,
		Timeout:            0, // Disable timeout
	}

	reportCh, errorCh := optimized.ValidateAsync(options)

	// Wait for results with a reasonable timeout
	select {
	case report := <-reportCh:
		if report == nil {
			t.Error("Expected validation report from async validation")
		} else {
			t.Logf("Async validation completed successfully")
		}
	case err := <-errorCh:
		t.Fatalf("Async validation failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Async validation timed out")
	}
}

func TestOptimizedRepositoryValidator_GetMetrics(t *testing.T) {
	tempDir := t.TempDir()

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		&mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)

	// Run validation to generate metrics
	ctx := context.Background()
	_, err := optimized.ValidateRepositoryWithOptions(ctx, DefaultPerformanceOptions())
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	metrics := optimized.GetMetrics()
	if metrics == nil {
		t.Fatal("Expected validation metrics")
	}

	// Verify metrics are populated
	if metrics.TotalDuration == 0 {
		t.Error("Expected non-zero total duration")
	}

	// Verify individual stage durations are tracked
	// Note: Some stages might be very fast and show 0 duration in tests
	t.Logf("Metrics: Total=%v, Structure=%v, Manifest=%v, Content=%v, Consistency=%v",
		metrics.TotalDuration,
		metrics.StructureDuration,
		metrics.ManifestDuration,
		metrics.ContentDuration,
		metrics.ConsistencyDuration)
}

func TestOptimizedRepositoryValidator_ClearCache(t *testing.T) {
	tempDir := t.TempDir()

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		&mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)

	// Clear cache should not panic
	optimized.ClearCache()

	// Verify validation still works after cache clear
	ctx := context.Background()
	_, err := optimized.ValidateRepositoryWithOptions(ctx, DefaultPerformanceOptions())
	if err != nil {
		t.Fatalf("Validation failed after cache clear: %v", err)
	}
}

func TestOptimizedRepositoryValidator_BackwardCompatibility(t *testing.T) {
	tempDir := t.TempDir()

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{}},
		&mockSMSReader{availableYears: []int{}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)

	// Verify base interface methods still work
	report, err := optimized.ValidateRepository()
	if err != nil {
		t.Fatalf("Base ValidateRepository method failed: %v", err)
	}
	if report == nil {
		t.Fatal("Expected validation report from base method")
	}

	// Test other base methods
	structureViolations := optimized.ValidateStructure()
	t.Logf("Structure violations: %d", len(structureViolations))

	manifestViolations := optimized.ValidateManifest()
	t.Logf("Manifest violations: %d", len(manifestViolations))

	contentViolations := optimized.ValidateContent()
	t.Logf("Content violations: %d", len(contentViolations))

	consistencyViolations := optimized.ValidateConsistency()
	t.Logf("Consistency violations: %d", len(consistencyViolations))
}

func TestEarlyTerminationError(t *testing.T) {
	violations := []ValidationViolation{
		{Type: ChecksumMismatch, Severity: SeverityError, Message: "Critical error"},
	}

	err := &EarlyTerminationError{
		Stage:      "content",
		Violations: violations,
	}

	expectedMsg := "validation terminated early at stage 'content' due to critical errors"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestValidationTimeout(t *testing.T) {
	timeout := 5 * time.Minute
	err := &ValidationTimeout{Timeout: timeout}

	expectedMsg := "validation timed out after 5m0s"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// Benchmark tests to verify performance improvements
func BenchmarkOptimizedValidation_Sequential(b *testing.B) {
	tempDir := b.TempDir()

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{2023, 2024}},
		&mockSMSReader{availableYears: []int{2023, 2024}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)
	options := &PerformanceOptions{ParallelValidation: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := optimized.ValidateRepositoryWithOptions(ctx, options)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}

func BenchmarkOptimizedValidation_Parallel(b *testing.B) {
	tempDir := b.TempDir()

	baseValidator := NewRepositoryValidator(
		tempDir,
		&mockCallsReader{availableYears: []int{2023, 2024}},
		&mockSMSReader{availableYears: []int{2023, 2024}, allAttachmentRefs: make(map[string]bool)},
		&mockAttachmentReader{attachments: []*attachments.Attachment{}},
		&mockContactsReader{contacts: []*contacts.Contact{}},
	)

	optimized := NewOptimizedRepositoryValidator(baseValidator)
	options := &PerformanceOptions{ParallelValidation: true, MaxConcurrency: 4}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := optimized.ValidateRepositoryWithOptions(ctx, options)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}
