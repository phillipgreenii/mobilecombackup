package analyzer

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/types"
)

const symbolTypeInterface = "interface"

// MarkdownAnalyzer implements documentation analysis for Markdown files
// NOTE: Added imports for content fingerprinting
// Updated imports to include:
// - crypto/sha256 for content hashing
// - time for file modification tracking

type MarkdownAnalyzer struct {
	logger Logger
}

// NewMarkdownAnalyzer creates a new markdown analyzer
func NewMarkdownAnalyzer(logger Logger) *MarkdownAnalyzer {
	return &MarkdownAnalyzer{
		logger: logger,
	}
}

// ParseMarkdown parses markdown files and extracts sections
func (ma *MarkdownAnalyzer) ParseMarkdown(filePath string) types.Result[[]DocSection] { //nolint:funlen
	ma.logger.Debug("Parsing markdown file", "file", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return types.NewResultError[[]DocSection](fmt.Errorf("failed to open file %s: %w", filePath, err))
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			ma.logger.Warn("Failed to close file", "file", filePath, "error", closeErr)
		}
	}()

	// Get file modification time for all sections
	var fileModTime int64
	if fileInfo, err := os.Stat(filePath); err == nil {
		fileModTime = fileInfo.ModTime().Unix()
	}

	var sections []DocSection
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	currentSection := DocSection{
		File: filePath,
		Type: "content",
	}
	var contentLines []string

	// Regular expressions for markdown parsing
	headerRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	codeBlockRegex := regexp.MustCompile("^```")

	inCodeBlock := false

	// Helper function to finalize a section with fingerprinting
	finalizeSection := func(section *DocSection) {
		if section.Content != "" {
			// Calculate SHA-256 fingerprint of content
			hasher := sha256.New()
			hasher.Write([]byte(section.Content))
			section.Fingerprint = fmt.Sprintf("%x", hasher.Sum(nil))
			section.LastUpdated = fileModTime
		}
	}

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Handle code blocks
		if codeBlockRegex.MatchString(line) {
			inCodeBlock = !inCodeBlock
			contentLines = append(contentLines, line)
			continue
		}

		if inCodeBlock {
			contentLines = append(contentLines, line)
			continue
		}

		// Check for headers
		if matches := headerRegex.FindStringSubmatch(line); matches != nil {
			// Save previous section if it has content
			if len(contentLines) > 0 || currentSection.Title != "" {
				currentSection.Content = strings.Join(contentLines, "\n")
				finalizeSection(&currentSection)
				sections = append(sections, currentSection)
			}

			// Start new section
			level := len(matches[1])
			title := matches[2]
			anchor := ma.generateAnchor(title)

			currentSection = DocSection{
				File:   filePath,
				Line:   lineNumber,
				Title:  title,
				Level:  level,
				Anchor: anchor,
				Type:   "section",
			}
			contentLines = []string{}
		} else {
			contentLines = append(contentLines, line)
		}
	}

	// Add final section
	if len(contentLines) > 0 || currentSection.Title != "" {
		currentSection.Content = strings.Join(contentLines, "\n")
		finalizeSection(&currentSection)
		sections = append(sections, currentSection)
	}

	if err := scanner.Err(); err != nil {
		return types.NewResultError[[]DocSection](fmt.Errorf("error reading file %s: %w", filePath, err))
	}

	ma.logger.Debug("Parsed markdown sections with fingerprints", "file", filePath, "sections", len(sections))
	return types.NewResult(sections)
}

// ExtractCodeReferences finds code references in documentation
func (ma *MarkdownAnalyzer) ExtractCodeReferences(content string) types.Result[[]string] {
	ma.logger.Debug("Extracting code references from content")

	var references []string

	// Regular expressions for different types of code references
	patterns := []string{
		// Function calls: `functionName(`
		"`([a-zA-Z_][a-zA-Z0-9_]*)`",
		// Type references: `TypeName`
		"`([A-Z][a-zA-Z0-9_]*)`",
		// Package.Function: `pkg.Function`
		"`([a-z][a-zA-Z0-9_]*\\.[A-Z][a-zA-Z0-9_]*)`",
		// Code blocks with go
		"```go\\s+([^`]+)```",
		// Inline code with dots (likely qualified names)
		"`([a-zA-Z_][a-zA-Z0-9_]*\\.[a-zA-Z_][a-zA-Z0-9_]*)`",
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				ref := strings.TrimSpace(match[1])
				if ref != "" {
					references = append(references, ref)
				}
			}
		}
	}

	// Remove duplicates
	unique := make(map[string]bool)
	var uniqueRefs []string
	for _, ref := range references {
		if !unique[ref] {
			unique[ref] = true
			uniqueRefs = append(uniqueRefs, ref)
		}
	}

	ma.logger.Debug("Extracted code references", "count", len(uniqueRefs))
	return types.NewResult(uniqueRefs)
}

// ValidateLinks validates all links in documentation
func (ma *MarkdownAnalyzer) ValidateLinks(docFiles []string) types.Result[[]Inconsistency] {
	ma.logger.Debug("Validating links in documentation", "files", len(docFiles))

	var inconsistencies []Inconsistency
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	for _, docFile := range docFiles {
		file, err := os.Open(docFile)
		if err != nil {
			ma.logger.Warn("Failed to open file for link validation", "file", docFile, "error", err)
			continue
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				ma.logger.Warn("Failed to close file", "file", docFile, "error", closeErr)
			}
		}()

		scanner := bufio.NewScanner(file)
		lineNumber := 0

		for scanner.Scan() {
			lineNumber++
			line := scanner.Text()

			matches := linkRegex.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) >= 3 {
					linkText := match[1]
					linkURL := match[2]

					// Validate link
					if err := ma.validateLink(linkURL); err != nil {
						inconsistency := Inconsistency{
							ID:          fmt.Sprintf("broken-link-%s-%d", docFile, lineNumber),
							Type:        InconsistencyBrokenLink,
							Severity:    SeverityMedium,
							Title:       "Broken link detected",
							Description: fmt.Sprintf("Link '%s' is broken: %s", linkText, err.Error()),
							DocFile:     docFile,
							DocLine:     lineNumber,
							Actual:      linkURL,
							Suggestion:  "Fix the link URL or remove the broken link",
							Context: InconsistencyContext{
								DocSection: DocSection{
									File:    docFile,
									Line:    lineNumber,
									Content: line,
								},
							},
							Metadata: map[string]interface{}{
								"link_text": linkText,
								"link_url":  linkURL,
							},
						}
						inconsistencies = append(inconsistencies, inconsistency)
					}
				}
			}
		}

		if closeErr := file.Close(); closeErr != nil {
			ma.logger.Warn("Failed to close file", "file", docFile, "error", closeErr)
		}
	}

	ma.logger.Debug("Link validation completed", "inconsistencies", len(inconsistencies))
	return types.NewResult(inconsistencies)
}

// ExtractExamples extracts code examples from documentation
func (ma *MarkdownAnalyzer) ExtractExamples(content string) types.Result[[]string] {
	ma.logger.Debug("Extracting code examples from content")

	var examples []string

	// Extract code blocks (any language)
	codeBlockRegex := regexp.MustCompile("```[a-zA-Z]*\\n([^`]+)```")
	matches := codeBlockRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			code := strings.TrimSpace(match[1])
			if code != "" {
				examples = append(examples, code)
			}
		}
	}

	// Extract inline code that looks like examples (single-line only)
	inlineRegex := regexp.MustCompile("`([^`\n]+)`")
	inlineMatches := inlineRegex.FindAllStringSubmatch(content, -1)

	for _, match := range inlineMatches {
		if len(match) > 1 {
			code := strings.TrimSpace(match[1])
			// Only include if it looks like a function call or substantial code
			if strings.Contains(code, "(") && strings.Contains(code, ")") {
				examples = append(examples, code)
			}
		}
	}

	ma.logger.Debug("Extracted code examples", "count", len(examples))
	return types.NewResult(examples)
}

// Helper methods

func (ma *MarkdownAnalyzer) generateAnchor(title string) string {
	// Convert title to lowercase anchor
	anchor := strings.ToLower(title)
	// Replace spaces with hyphens
	anchor = regexp.MustCompile(`\s+`).ReplaceAllString(anchor, "-")
	// Remove special characters
	anchor = regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(anchor, "")
	// Remove multiple consecutive hyphens
	anchor = regexp.MustCompile(`-+`).ReplaceAllString(anchor, "-")
	// Trim hyphens from ends
	anchor = strings.Trim(anchor, "-")

	return anchor
}

func (ma *MarkdownAnalyzer) validateLink(linkURL string) error {
	// Simplified link validation
	// In a full implementation, this would check HTTP links, file existence, etc.

	// Check for obviously broken patterns
	if strings.TrimSpace(linkURL) == "" {
		return fmt.Errorf("empty link URL")
	}

	// Check for placeholder links
	placeholders := []string{"#", "#todo", "TODO", "FIXME", "example.com"}
	for _, placeholder := range placeholders {
		if strings.Contains(linkURL, placeholder) {
			return fmt.Errorf("placeholder link: %s", placeholder)
		}
	}

	// Check for internal file references
	if strings.HasPrefix(linkURL, "./") || strings.HasPrefix(linkURL, "../") || strings.HasSuffix(linkURL, ".md") {
		// For file links, check if file exists (simplified check)
		if !strings.HasPrefix(linkURL, "http") {
			// This would need more sophisticated file existence checking
			// For now, assume local files might exist
			return nil
		}
	}

	// Check for malformed URLs
	if strings.HasPrefix(linkURL, "http") {
		// Basic URL format validation
		if !strings.Contains(linkURL, "://") {
			return fmt.Errorf("malformed URL")
		}
	}

	return nil
}

// DocumentationStateManager manages the persistent state of documentation scanning
type DocumentationStateManager struct {
	statePath    string
	currentState *DocumentationScanState
	mutex        sync.RWMutex
	logger       Logger
}

// NewDocumentationStateManager creates a new state manager
func NewDocumentationStateManager(statePath string, logger Logger) *DocumentationStateManager {
	dsm := &DocumentationStateManager{
		statePath: statePath,
		logger:    logger,
	}
	dsm.createDefaultState()
	return dsm
}

// Load loads the documentation scan state from disk
func (dsm *DocumentationStateManager) Load() types.Result[*DocumentationScanState] {
	dsm.mutex.Lock()
	defer dsm.mutex.Unlock()

	// Update metadata to loading status
	if dsm.currentState != nil {
		dsm.currentState.Metadata.Status = "loading"
	}

	// Check if state file exists
	if _, err := os.Stat(dsm.statePath); os.IsNotExist(err) {
		// No state file exists, create default state
		dsm.logger.Info("No documentation state file found, creating new state")
		return dsm.createDefaultState()
	}

	// Read state file
	data, err := os.ReadFile(dsm.statePath)
	if err != nil {
		return types.NewResultError[*DocumentationScanState](fmt.Errorf("failed to read documentation state file: %w", err))
	}

	// Parse JSON state
	var loadedState DocumentationScanState
	err = json.Unmarshal(data, &loadedState)
	if err != nil {
		dsm.logger.Warn("Failed to parse documentation state file, creating new state", "error", err.Error())
		return dsm.createDefaultState()
	}

	// Validate loaded state
	if err := dsm.validateState(&loadedState); err != nil {
		dsm.logger.Warn("Loaded documentation state is invalid, creating new state", "error", err.Error())
		return dsm.createDefaultState()
	}

	// Update metadata
	loadedState.Metadata.Status = "ready"
	dsm.currentState = &loadedState

	dsm.logger.Info("Documentation scan state loaded successfully", "files", len(loadedState.FileStates))
	return types.NewResult(dsm.currentState)
}

// Persist saves the current documentation scan state to disk
func (dsm *DocumentationStateManager) Persist() types.Result[bool] {
	dsm.mutex.RLock()
	defer dsm.mutex.RUnlock()

	if dsm.currentState == nil {
		return types.NewResultError[bool](errors.New("no documentation state to persist"))
	}

	// Update persistence metadata
	now := time.Now().Unix()
	dsm.currentState.Metadata.Status = "persisting"
	dsm.currentState.Metadata.UpdatedAt = now

	// Calculate checksum
	checksum, err := dsm.calculateChecksum(dsm.currentState)
	if err != nil {
		dsm.logger.Warn("Failed to calculate documentation state checksum", "error", err.Error())
	} else {
		dsm.currentState.Metadata.Checksum = checksum
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dsm.statePath), 0750); err != nil {
		return types.NewResultError[bool](fmt.Errorf("failed to create state directory: %w", err))
	}

	// Convert state to JSON for persistence
	data, err := json.MarshalIndent(dsm.currentState, "", "  ")
	if err != nil {
		return types.NewResultError[bool](fmt.Errorf("failed to marshal documentation state: %w", err))
	}

	// Write to file with proper permissions
	err = os.WriteFile(dsm.statePath, data, 0600)
	if err != nil {
		return types.NewResultError[bool](fmt.Errorf("failed to write documentation state file: %w", err))
	}

	// Update metadata after successful persistence
	dsm.currentState.Metadata.Status = "ready"

	dsm.logger.Info("Documentation scan state persisted successfully", "path", dsm.statePath)
	return types.NewResult(true)
}

// UpdateFileState updates or creates the state for a single file
func (dsm *DocumentationStateManager) UpdateFileState(filePath string, sections []DocSection, lastModified int64) types.Result[bool] { //nolint:lll
	dsm.mutex.Lock()
	defer dsm.mutex.Unlock()

	if dsm.currentState == nil {
		// Create default state if none exists
		result := dsm.createDefaultState()
		if result.IsErr() {
			return types.NewResultError[bool](fmt.Errorf("failed to create default state: %w", result.Error))
		}
	}

	// Calculate file checksum from content
	var contentBuilder strings.Builder
	for _, section := range sections {
		contentBuilder.WriteString(section.Content)
	}
	hasher := sha256.New()
	hasher.Write([]byte(contentBuilder.String()))
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	// Update or create file state
	fileState := DocumentFileState{
		Path:         filePath,
		LastModified: lastModified,
		Checksum:     checksum,
		Sections:     sections,
		LastScanned:  time.Now().Unix(),
	}

	// Increment scan count if file already exists
	if existingState, exists := dsm.currentState.FileStates[filePath]; exists {
		fileState.ScanCount = existingState.ScanCount + 1
	} else {
		fileState.ScanCount = 1
	}

	dsm.currentState.FileStates[filePath] = fileState

	// Update metadata
	dsm.currentState.Metadata.TotalFiles = len(dsm.currentState.FileStates)
	totalSections := 0
	for _, fs := range dsm.currentState.FileStates {
		totalSections += len(fs.Sections)
	}
	dsm.currentState.Metadata.TotalSections = totalSections
	dsm.currentState.LastScanTime = time.Now().Unix()

	dsm.logger.Debug("Updated documentation file state", "file", filePath, "sections", len(sections))
	return types.NewResult(true)
}

// IsFileChanged checks if a file needs to be rescanned based on modification time and content
func (dsm *DocumentationStateManager) IsFileChanged(filePath string, lastModified int64) bool {
	dsm.mutex.RLock()
	defer dsm.mutex.RUnlock()

	if dsm.currentState == nil {
		return true // No state means file needs scanning
	}

	fileState, exists := dsm.currentState.FileStates[filePath]
	if !exists {
		return true // New file needs scanning
	}

	return fileState.LastModified != lastModified
}

// GetFileState returns the state for a specific file
func (dsm *DocumentationStateManager) GetFileState(filePath string) (*DocumentFileState, bool) {
	dsm.mutex.RLock()
	defer dsm.mutex.RUnlock()

	if dsm.currentState == nil {
		return nil, false
	}

	fileState, exists := dsm.currentState.FileStates[filePath]
	if !exists {
		return nil, false
	}

	return &fileState, true
}

// GetState returns the current documentation scan state
func (dsm *DocumentationStateManager) GetState() *DocumentationScanState {
	dsm.mutex.RLock()
	defer dsm.mutex.RUnlock()

	return dsm.currentState
}

// createDefaultState creates a new default documentation scan state
func (dsm *DocumentationStateManager) createDefaultState() types.Result[*DocumentationScanState] {
	now := time.Now().Unix()
	dsm.currentState = &DocumentationScanState{
		Version:      "1.0",
		LastScanTime: now,
		FileStates:   make(map[string]DocumentFileState),
		Metadata: DocumentationStateMetadata{
			Status:        "ready",
			TotalFiles:    0,
			TotalSections: 0,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}

	return types.NewResult(dsm.currentState)
}

// validateState validates the loaded documentation scan state
func (dsm *DocumentationStateManager) validateState(state *DocumentationScanState) error {
	if state == nil {
		return errors.New("state is nil")
	}

	if state.Version == "" {
		return errors.New("state version is empty")
	}

	if state.FileStates == nil {
		return errors.New("file states map is nil")
	}

	// Validate file states
	for path, fileState := range state.FileStates {
		if fileState.Path != path {
			return fmt.Errorf("file state path mismatch: key=%s, path=%s", path, fileState.Path)
		}
		if fileState.LastModified < 0 {
			return fmt.Errorf("invalid last modified time for file %s", path)
		}
	}

	return nil
}

// calculateChecksum calculates a checksum for the documentation scan state
func (dsm *DocumentationStateManager) calculateChecksum(state *DocumentationScanState) (string, error) {
	// Create a copy without metadata for checksum calculation
	stateCopy := *state
	stateCopy.Metadata.Checksum = ""
	stateCopy.Metadata.UpdatedAt = 0

	data, err := json.Marshal(stateCopy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state for checksum: %w", err)
	}

	hasher := sha256.New()
	hasher.Write(data)
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// ConcurrentDocumentScanner provides parallel processing for large documentation sets
type ConcurrentDocumentScanner struct {
	analyzer     *MarkdownAnalyzer
	stateManager *DocumentationStateManager
	logger       Logger
	config       ScanConfig
}

// ScanConfig contains configuration for concurrent document scanning
type ScanConfig struct {
	MaxWorkers     int  `json:"max_workers" yaml:"max_workers"`
	BatchSize      int  `json:"batch_size" yaml:"batch_size"`
	ReportProgress bool `json:"report_progress" yaml:"report_progress"`
}

// ScanProgress tracks the progress of concurrent document scanning
type ScanProgress struct {
	TotalFiles      int     `json:"total_files"`
	ProcessedFiles  int     `json:"processed_files"`
	SuccessfulFiles int     `json:"successful_files"`
	FailedFiles     int     `json:"failed_files"`
	TotalSections   int     `json:"total_sections"`
	StartTime       int64   `json:"start_time"`
	ElapsedSeconds  float64 `json:"elapsed_seconds"`
	FilesPerSecond  float64 `json:"files_per_second"`
	Errors          []error `json:"-"` // Not serialized to JSON
}

// ScanResult contains the results of a concurrent scan operation
type ScanResult struct {
	Progress     ScanProgress                 `json:"progress"`
	FileStates   map[string]DocumentFileState `json:"file_states"`
	Success      bool                         `json:"success"`
	ErrorSummary string                       `json:"error_summary,omitempty"`
}

// WorkerJob represents a single file to be processed
type WorkerJob struct {
	FilePath     string
	LastModified int64
}

// WorkerResult contains the result of processing a single file
type WorkerResult struct {
	Job      WorkerJob
	Sections []DocSection
	Error    error
}

// NewConcurrentDocumentScanner creates a new concurrent document scanner
func NewConcurrentDocumentScanner(analyzer *MarkdownAnalyzer, stateManager *DocumentationStateManager, logger Logger) *ConcurrentDocumentScanner { //nolint:lll
	return &ConcurrentDocumentScanner{
		analyzer:     analyzer,
		stateManager: stateManager,
		logger:       logger,
		config: ScanConfig{
			MaxWorkers:     4,  // Default to 4 workers
			BatchSize:      50, // Process 50 files per batch
			ReportProgress: true,
		},
	}
}

// SetConfig updates the scanner configuration
func (cds *ConcurrentDocumentScanner) SetConfig(config ScanConfig) {
	cds.config = config
}

// ScanDocuments performs concurrent scanning of multiple documentation files
func (cds *ConcurrentDocumentScanner) ScanDocuments(filePaths []string) types.Result[*ScanResult] { //nolint:funlen,nestif
	startTime := time.Now()

	cds.logger.Info("Starting concurrent document scan", "files", len(filePaths), "workers", cds.config.MaxWorkers)

	// Initialize progress tracking
	progress := ScanProgress{
		TotalFiles: len(filePaths),
		StartTime:  startTime.Unix(),
		Errors:     make([]error, 0),
	}

	// Filter files that need scanning (incremental scanning)
	var jobsToProcess []WorkerJob
	for _, filePath := range filePaths {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			progress.Errors = append(progress.Errors, fmt.Errorf("failed to stat file %s: %w", filePath, err))
			progress.FailedFiles++
			continue
		}

		lastModified := fileInfo.ModTime().Unix()

		// Check if file needs scanning
		if cds.stateManager.IsFileChanged(filePath, lastModified) {
			jobsToProcess = append(jobsToProcess, WorkerJob{
				FilePath:     filePath,
				LastModified: lastModified,
			})
		} else {
			cds.logger.Debug("Skipping unchanged file", "file", filePath)
			progress.SuccessfulFiles++
		}
	}

	cds.logger.Info("Files requiring processing", "total", len(filePaths), "to_process", len(jobsToProcess), "skipped", len(filePaths)-len(jobsToProcess)) //nolint:lll

	// If no files to process, return success
	if len(jobsToProcess) == 0 {
		progress.ProcessedFiles = len(filePaths)
		progress.SuccessfulFiles = len(filePaths)
		progress.ElapsedSeconds = time.Since(startTime).Seconds()

		return types.NewResult(&ScanResult{
			Progress:   progress,
			FileStates: make(map[string]DocumentFileState),
			Success:    true,
		})
	}

	// Process files in batches if needed
	var allResults []WorkerResult
	batchSize := cds.config.BatchSize

	for i := 0; i < len(jobsToProcess); i += batchSize {
		end := i + batchSize
		if end > len(jobsToProcess) {
			end = len(jobsToProcess)
		}

		batch := jobsToProcess[i:end]
		cds.logger.Debug("Processing batch", "batch", i/batchSize+1, "files", len(batch))

		batchResults := cds.processBatch(batch)
		allResults = append(allResults, batchResults...)

		// Update progress
		progress.ProcessedFiles = i + len(batch)
		if cds.config.ReportProgress && len(jobsToProcess) > 10 {
			elapsed := time.Since(startTime).Seconds()
			progress.ElapsedSeconds = elapsed
			progress.FilesPerSecond = float64(progress.ProcessedFiles) / elapsed
			cds.logger.Info("Scan progress", "processed", progress.ProcessedFiles, "total", len(jobsToProcess), "rate", fmt.Sprintf("%.1f files/sec", progress.FilesPerSecond)) //nolint:lll
		}
	}

	// Process results and update state
	fileStates := make(map[string]DocumentFileState)
	for _, result := range allResults {
		if result.Error != nil { //nolint:nestif
			progress.Errors = append(progress.Errors, fmt.Errorf("failed to process %s: %w", result.Job.FilePath, result.Error))
			progress.FailedFiles++
		} else {
			// Update state manager
			updateResult := cds.stateManager.UpdateFileState(result.Job.FilePath, result.Sections, result.Job.LastModified)
			if updateResult.IsErr() {
				progress.Errors = append(progress.Errors, fmt.Errorf("failed to update state for %s: %w", result.Job.FilePath, updateResult.Error)) //nolint:lll
				progress.FailedFiles++
			} else {
				progress.SuccessfulFiles++
				progress.TotalSections += len(result.Sections)

				// Add to results
				fileState, exists := cds.stateManager.GetFileState(result.Job.FilePath)
				if exists {
					fileStates[result.Job.FilePath] = *fileState
				}
			}
		}
	}

	// Finalize progress
	progress.ProcessedFiles = len(jobsToProcess)
	progress.ElapsedSeconds = time.Since(startTime).Seconds()
	if progress.ElapsedSeconds > 0 {
		progress.FilesPerSecond = float64(progress.ProcessedFiles) / progress.ElapsedSeconds
	}

	// Persist state
	if persistResult := cds.stateManager.Persist(); persistResult.IsErr() {
		progress.Errors = append(progress.Errors, fmt.Errorf("failed to persist state: %w", persistResult.Error))
	}

	// Determine success
	success := progress.FailedFiles == 0 && len(progress.Errors) == 0
	var errorSummary string
	if len(progress.Errors) > 0 {
		errorSummary = fmt.Sprintf("%d errors occurred during scanning", len(progress.Errors))
	}

	cds.logger.Info("Concurrent document scan completed",
		"total_files", progress.TotalFiles,
		"processed", progress.ProcessedFiles,
		"successful", progress.SuccessfulFiles,
		"failed", progress.FailedFiles,
		"sections", progress.TotalSections,
		"elapsed", fmt.Sprintf("%.2fs", progress.ElapsedSeconds),
		"rate", fmt.Sprintf("%.1f files/sec", progress.FilesPerSecond))

	return types.NewResult(&ScanResult{
		Progress:     progress,
		FileStates:   fileStates,
		Success:      success,
		ErrorSummary: errorSummary,
	})
}

// processBatch processes a batch of files using worker pool
func (cds *ConcurrentDocumentScanner) processBatch(jobs []WorkerJob) []WorkerResult {
	jobChan := make(chan WorkerJob, len(jobs))
	resultChan := make(chan WorkerResult, len(jobs))

	// Start workers
	numWorkers := cds.config.MaxWorkers
	if numWorkers > len(jobs) {
		numWorkers = len(jobs)
	}

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go cds.worker(&wg, jobChan, resultChan)
	}

	// Send jobs to workers
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	// Wait for workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var results []WorkerResult //nolint:prealloc
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// worker processes individual files from the job channel
func (cds *ConcurrentDocumentScanner) worker(wg *sync.WaitGroup, jobChan <-chan WorkerJob, resultChan chan<- WorkerResult) {
	defer wg.Done()

	for job := range jobChan {
		cds.logger.Debug("Processing file", "file", job.FilePath)

		result := WorkerResult{Job: job}

		// Parse the markdown file
		parseResult := cds.analyzer.ParseMarkdown(job.FilePath)
		if parseResult.IsErr() {
			result.Error = parseResult.Error
		} else {
			result.Sections = parseResult.Value
		}

		resultChan <- result
	}
}

// GetScanProgress returns the current scan progress (if available)
func (cds *ConcurrentDocumentScanner) GetScanProgress() *ScanProgress {
	// In a more advanced implementation, this could return real-time progress
	// For now, it's mainly used for the final result
	return nil
}

// EnhancedDocumentAnalyzer provides advanced analysis capabilities for documentation
type EnhancedDocumentAnalyzer struct {
	analyzer *MarkdownAnalyzer
	logger   Logger
}

// CodeBlockInfo contains detailed information about code blocks
type CodeBlockInfo struct {
	Language  string   `json:"language"`
	Content   string   `json:"content"`
	LineStart int      `json:"line_start"`
	LineEnd   int      `json:"line_end"`
	Imports   []string `json:"imports"`
	Functions []string `json:"functions"`
	Types     []string `json:"types"`
	Variables []string `json:"variables"`
}

// CrossReference represents a reference between documentation sections
type CrossReference struct {
	FromSection   string `json:"from_section"`
	ToSection     string `json:"to_section"`
	ReferenceType string `json:"reference_type"` // "link", "code_reference", "heading_reference"
	LineNumber    int    `json:"line_number"`
	Context       string `json:"context"`
}

// DocumentMetadata contains enhanced metadata extracted from documentation
type DocumentMetadata struct {
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Tags         []string          `json:"tags"`
	Authors      []string          `json:"authors"`
	LastModified string            `json:"last_modified"`
	Version      string            `json:"version"`
	Dependencies []string          `json:"dependencies"`
	CodeBlocks   []CodeBlockInfo   `json:"code_blocks"`
	CrossRefs    []CrossReference  `json:"cross_references"`
	CustomFields map[string]string `json:"custom_fields"`
}

// EnhancedAnalysisResult contains the results of enhanced document analysis
type EnhancedAnalysisResult struct {
	Sections    []DocSection     `json:"sections"`
	Metadata    DocumentMetadata `json:"metadata"`
	CodeBlocks  []CodeBlockInfo  `json:"code_blocks"`
	CrossRefs   []CrossReference `json:"cross_references"`
	WordCount   int              `json:"word_count"`
	ReadingTime int              `json:"reading_time_minutes"`
	Complexity  string           `json:"complexity"` // "low", "medium", "high"
}

// NewEnhancedDocumentAnalyzer creates a new enhanced document analyzer
func NewEnhancedDocumentAnalyzer(analyzer *MarkdownAnalyzer, logger Logger) *EnhancedDocumentAnalyzer {
	return &EnhancedDocumentAnalyzer{
		analyzer: analyzer,
		logger:   logger,
	}
}

// AnalyzeDocument performs comprehensive analysis of a documentation file
func (eda *EnhancedDocumentAnalyzer) AnalyzeDocument(filePath string) types.Result[*EnhancedAnalysisResult] {
	eda.logger.Debug("Starting enhanced analysis", "file", filePath)

	// First, use the standard parser to get sections
	parseResult := eda.analyzer.ParseMarkdown(filePath)
	if parseResult.IsErr() {
		return types.NewResultError[*EnhancedAnalysisResult](fmt.Errorf("failed to parse markdown: %w", parseResult.Error))
	}

	sections := parseResult.Value

	// Read the full file content for advanced analysis
	content, err := os.ReadFile(filePath) //nolint:gosec
	if err != nil {
		return types.NewResultError[*EnhancedAnalysisResult](fmt.Errorf("failed to read file for enhanced analysis: %w", err))
	}

	fileContent := string(content)

	// Extract enhanced code blocks
	codeBlocks := eda.extractEnhancedCodeBlocks(fileContent)

	// Find cross-references
	crossRefs := eda.findCrossReferences(sections, fileContent)

	// Extract metadata
	metadata := eda.extractMetadata(sections, fileContent, codeBlocks, crossRefs)

	// Calculate reading metrics
	wordCount := eda.calculateWordCount(fileContent)
	readingTime := eda.estimateReadingTime(wordCount)
	complexity := eda.assessComplexity(sections, codeBlocks)

	result := &EnhancedAnalysisResult{
		Sections:    sections,
		Metadata:    metadata,
		CodeBlocks:  codeBlocks,
		CrossRefs:   crossRefs,
		WordCount:   wordCount,
		ReadingTime: readingTime,
		Complexity:  complexity,
	}

	eda.logger.Debug("Enhanced analysis completed",
		"file", filePath,
		"sections", len(sections),
		"code_blocks", len(codeBlocks),
		"cross_refs", len(crossRefs),
		"word_count", wordCount,
		"complexity", complexity)

	return types.NewResult(result)
}

// extractEnhancedCodeBlocks extracts code blocks with language detection and analysis
func (eda *EnhancedDocumentAnalyzer) extractEnhancedCodeBlocks(content string) []CodeBlockInfo {
	var codeBlocks []CodeBlockInfo

	// Enhanced regex for code blocks with language detection
	codeBlockRegex := regexp.MustCompile("```([a-zA-Z0-9]*)\n([^`]+)```")
	matches := codeBlockRegex.FindAllStringSubmatch(content, -1)

	lines := strings.Split(content, "\n")

	for _, match := range matches {
		if len(match) >= 3 {
			language := strings.TrimSpace(match[1])
			codeContent := strings.TrimSpace(match[2])

			// Find line numbers
			lineStart, lineEnd := eda.findCodeBlockLines(lines, codeContent)

			// Extract language-specific elements
			codeBlock := CodeBlockInfo{
				Language:  language,
				Content:   codeContent,
				LineStart: lineStart,
				LineEnd:   lineEnd,
			}

			// Language-specific analysis
			switch strings.ToLower(language) {
			case "go", "golang":
				codeBlock.Imports = eda.extractGoImports(codeContent)
				codeBlock.Functions = eda.extractGoFunctions(codeContent)
				codeBlock.Types = eda.extractGoTypes(codeContent)
				codeBlock.Variables = eda.extractGoVariables(codeContent)
			case "javascript", "js", "typescript", "ts":
				codeBlock.Functions = eda.extractJSFunctions(codeContent)
				codeBlock.Variables = eda.extractJSVariables(codeContent)
			case "python", "py":
				codeBlock.Imports = eda.extractPythonImports(codeContent)
				codeBlock.Functions = eda.extractPythonFunctions(codeContent)
			case "java":
				codeBlock.Imports = eda.extractJavaImports(codeContent)
				codeBlock.Functions = eda.extractJavaFunctions(codeContent)
				codeBlock.Types = eda.extractJavaTypes(codeContent)
			default:
				// Generic analysis for unknown languages
				codeBlock.Functions = eda.extractGenericFunctions(codeContent)
			}

			codeBlocks = append(codeBlocks, codeBlock)
		}
	}

	return codeBlocks
}

// findCrossReferences identifies references between documentation sections
func (eda *EnhancedDocumentAnalyzer) findCrossReferences(sections []DocSection, content string) []CrossReference {
	var crossRefs []CrossReference

	// Build section anchor map
	sectionMap := make(map[string]DocSection)
	for _, section := range sections {
		if section.Anchor != "" {
			sectionMap[section.Anchor] = section
		}
	}

	lines := strings.Split(content, "\n")

	// Find markdown links
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(#([^)]+)\)`)
	for i, line := range lines {
		matches := linkRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				linkText := match[1]
				anchor := match[2]

				if targetSection, exists := sectionMap[anchor]; exists {
					crossRef := CrossReference{
						FromSection:   eda.findContainingSection(sections, i+1),
						ToSection:     targetSection.Title,
						ReferenceType: "link",
						LineNumber:    i + 1,
						Context:       linkText,
					}
					crossRefs = append(crossRefs, crossRef)
				}
			}
		}
	}

	// Find heading references (e.g., "See the Installation section")
	headingRefRegex := regexp.MustCompile(`(?i)(see|refer to|check|visit)\s+(?:the\s+)?([A-Z][a-zA-Z\s]+)\s+(?:section|chapter|part)`) //nolint:lll
	for i, line := range lines {
		matches := headingRefRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				referencedTitle := strings.TrimSpace(match[2])

				// Try to find matching section
				for _, section := range sections {
					if strings.Contains(strings.ToLower(section.Title), strings.ToLower(referencedTitle)) {
						crossRef := CrossReference{
							FromSection:   eda.findContainingSection(sections, i+1),
							ToSection:     section.Title,
							ReferenceType: "heading_reference",
							LineNumber:    i + 1,
							Context:       match[0],
						}
						crossRefs = append(crossRefs, crossRef)
						break
					}
				}
			}
		}
	}

	return crossRefs
}

// extractMetadata extracts comprehensive metadata from the document
func (eda *EnhancedDocumentAnalyzer) extractMetadata(sections []DocSection, content string, codeBlocks []CodeBlockInfo, crossRefs []CrossReference) DocumentMetadata { //nolint:lll
	metadata := DocumentMetadata{
		CodeBlocks:   codeBlocks,
		CrossRefs:    crossRefs,
		CustomFields: make(map[string]string),
	}

	// Extract title (first H1 heading)
	for _, section := range sections {
		if section.Level == 1 && metadata.Title == "" {
			metadata.Title = section.Title
			break
		}
	}

	// Extract description (first paragraph after title)
	if len(sections) > 1 {
		metadata.Description = eda.extractFirstParagraph(sections[1].Content)
	}

	// Extract tags from content
	metadata.Tags = eda.extractTags(content)

	// Extract dependencies from code blocks
	depSet := make(map[string]bool)
	for _, codeBlock := range codeBlocks {
		for _, imp := range codeBlock.Imports {
			depSet[imp] = true
		}
	}
	for dep := range depSet {
		metadata.Dependencies = append(metadata.Dependencies, dep)
	}

	// Extract YAML frontmatter if present
	metadata.CustomFields = eda.extractFrontmatter(content)

	return metadata
}

// Helper methods for language-specific analysis

func (eda *EnhancedDocumentAnalyzer) extractGoImports(code string) []string {
	var imports []string
	importRegex := regexp.MustCompile(`import\s+\"([^\"]+)\"`)
	matches := importRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, match[1])
		}
	}
	return imports
}

func (eda *EnhancedDocumentAnalyzer) extractGoFunctions(code string) []string {
	var functions []string
	funcRegex := regexp.MustCompile(`func\s+(\w+)\s*\(`)
	matches := funcRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			functions = append(functions, match[1])
		}
	}
	return functions
}

func (eda *EnhancedDocumentAnalyzer) extractGoTypes(code string) []string {
	var types []string
	typeRegex := regexp.MustCompile(`type\s+(\w+)\s+(?:struct|interface)`)
	matches := typeRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			types = append(types, match[1])
		}
	}
	return types
}

func (eda *EnhancedDocumentAnalyzer) extractGoVariables(code string) []string {
	var variables []string
	varRegex := regexp.MustCompile(`(?:var|:=)\s+(\w+)`)
	matches := varRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			variables = append(variables, match[1])
		}
	}
	return variables
}

func (eda *EnhancedDocumentAnalyzer) extractJSFunctions(code string) []string {
	var functions []string
	patterns := []string{
		`function\s+(\w+)\s*\(`,
		`(\w+)\s*:\s*function\s*\(`,
		`const\s+(\w+)\s*=\s*\(.*\)\s*=>\s*{`,
	}
	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllStringSubmatch(code, -1)
		for _, match := range matches {
			if len(match) > 1 {
				functions = append(functions, match[1])
			}
		}
	}
	return functions
}

func (eda *EnhancedDocumentAnalyzer) extractJSVariables(code string) []string {
	var variables []string
	varRegex := regexp.MustCompile(`(?:var|let|const)\s+(\w+)`)
	matches := varRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			variables = append(variables, match[1])
		}
	}
	return variables
}

func (eda *EnhancedDocumentAnalyzer) extractPythonImports(code string) []string {
	var imports []string
	patterns := []string{
		`import\s+(\w+)`,
		`from\s+(\w+)\s+import`,
	}
	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllStringSubmatch(code, -1)
		for _, match := range matches {
			if len(match) > 1 {
				imports = append(imports, match[1])
			}
		}
	}
	return imports
}

func (eda *EnhancedDocumentAnalyzer) extractPythonFunctions(code string) []string {
	var functions []string
	funcRegex := regexp.MustCompile(`def\s+(\w+)\s*\(`)
	matches := funcRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			functions = append(functions, match[1])
		}
	}
	return functions
}

func (eda *EnhancedDocumentAnalyzer) extractJavaImports(code string) []string {
	var imports []string
	importRegex := regexp.MustCompile(`import\s+([a-zA-Z0-9_.]+);`)
	matches := importRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, match[1])
		}
	}
	return imports
}

func (eda *EnhancedDocumentAnalyzer) extractJavaFunctions(code string) []string {
	var functions []string
	methodRegex := regexp.MustCompile(`(?:public|private|protected|static)?\s*(?:\w+\s+)*(\w+)\s*\([^)]*\)\s*{`)
	matches := methodRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 && match[1] != "class" && match[1] != symbolTypeInterface {
			functions = append(functions, match[1])
		}
	}
	return functions
}

func (eda *EnhancedDocumentAnalyzer) extractJavaTypes(code string) []string {
	var types []string
	classRegex := regexp.MustCompile(`(?:class|interface)\s+(\w+)`)
	matches := classRegex.FindAllStringSubmatch(code, -1)
	for _, match := range matches {
		if len(match) > 1 {
			types = append(types, match[1])
		}
	}
	return types
}

func (eda *EnhancedDocumentAnalyzer) extractGenericFunctions(code string) []string {
	var functions []string
	// Generic function pattern - look for word followed by parentheses
	funcRegex := regexp.MustCompile(`(\w+)\s*\(`)
	matches := funcRegex.FindAllStringSubmatch(code, -1)
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			funcName := match[1]
			// Filter out common non-function words
			if !seen[funcName] && !eda.isCommonKeyword(funcName) {
				functions = append(functions, funcName)
				seen[funcName] = true
			}
		}
	}
	return functions
}

// Helper methods for document analysis

func (eda *EnhancedDocumentAnalyzer) findCodeBlockLines(lines []string, codeContent string) (int, int) {
	codeLines := strings.Split(codeContent, "\n")
	if len(codeLines) == 0 {
		return 0, 0
	}

	firstCodeLine := strings.TrimSpace(codeLines[0])
	for i, line := range lines {
		if strings.TrimSpace(line) == firstCodeLine {
			return i + 1, i + len(codeLines)
		}
	}
	return 0, 0
}

func (eda *EnhancedDocumentAnalyzer) findContainingSection(sections []DocSection, lineNumber int) string {
	for i := len(sections) - 1; i >= 0; i-- {
		if sections[i].Line <= lineNumber {
			return sections[i].Title
		}
	}
	return "Unknown"
}

func (eda *EnhancedDocumentAnalyzer) extractFirstParagraph(content string) string {
	lines := strings.Split(content, "\n")
	var paragraph []string //nolint:prealloc
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		paragraph = append(paragraph, line)
	}
	return strings.Join(paragraph, " ")
}

func (eda *EnhancedDocumentAnalyzer) extractTags(content string) []string {
	var tags []string
	// Look for hashtags or tag patterns
	tagRegex := regexp.MustCompile(`#(\w+)`)
	matches := tagRegex.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			tags = append(tags, match[1])
			seen[match[1]] = true
		}
	}
	return tags
}

func (eda *EnhancedDocumentAnalyzer) extractFrontmatter(content string) map[string]string { //nolint:nestif
	fields := make(map[string]string)

	// Look for YAML frontmatter
	if strings.HasPrefix(content, "---\n") { //nolint:nestif
		endIndex := strings.Index(content[4:], "\n---\n")
		if endIndex != -1 {
			frontmatter := content[4 : endIndex+4]
			lines := strings.Split(frontmatter, "\n")
			for _, line := range lines {
				if strings.Contains(line, ":") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])
						fields[key] = value
					}
				}
			}
		}
	}

	return fields
}

func (eda *EnhancedDocumentAnalyzer) calculateWordCount(content string) int {
	// Remove markdown syntax and count words
	cleaned := regexp.MustCompile(`[#\*\x60\[\]()_]+`).ReplaceAllString(content, "")
	words := strings.Fields(cleaned)
	return len(words)
}

func (eda *EnhancedDocumentAnalyzer) estimateReadingTime(wordCount int) int {
	// Average reading speed: 200-250 words per minute, use 225
	return (wordCount + 224) / 225 // Round up
}

func (eda *EnhancedDocumentAnalyzer) assessComplexity(sections []DocSection, codeBlocks []CodeBlockInfo) string {
	// Simple complexity assessment based on structure and content
	complexityScore := 0

	// More sections = higher complexity
	if len(sections) > 10 {
		complexityScore += 2
	} else if len(sections) > 5 {
		complexityScore += 1
	}

	// Code blocks increase complexity
	complexityScore += len(codeBlocks)

	// Nested sections increase complexity
	maxLevel := 0
	for _, section := range sections {
		if section.Level > maxLevel {
			maxLevel = section.Level
		}
	}
	if maxLevel > 3 {
		complexityScore += 2
	} else if maxLevel > 2 {
		complexityScore += 1
	}

	// Classify complexity
	switch {
	case complexityScore <= 3:
		return string(SeverityLow)
	case complexityScore <= 7:
		return string(SeverityMedium)
	default:
		return string(SeverityHigh)
	}
}

func (eda *EnhancedDocumentAnalyzer) isCommonKeyword(word string) bool {
	keywords := map[string]bool{
		"if": true, "else": true, "for": true, "while": true, "do": true,
		"return": true, "var": true, "let": true, "const": true, "function": true,
		"class": true, "interface": true, "struct": true, "enum": true,
		"public": true, "private": true, "protected": true, "static": true,
		"import": true, "export": true, "from": true, "as": true,
		"try": true, "catch": true, "finally": true, "throw": true,
	}
	return keywords[strings.ToLower(word)]
}

// DefaultEventBus implements the EventBus interface
type DefaultEventBus struct {
	subscribers map[string][]EventHandler
	history     []Event
	maxHistory  int
	mutex       sync.RWMutex
	logger      Logger
}

// NewDefaultEventBus creates a new event bus
func NewDefaultEventBus(maxHistory int, logger Logger) *DefaultEventBus {
	return &DefaultEventBus{
		subscribers: make(map[string][]EventHandler),
		history:     make([]Event, 0),
		maxHistory:  maxHistory,
		logger:      logger,
	}
}

// Subscribe registers an event handler for specific event types
func (eb *DefaultEventBus) Subscribe(eventType string, handler EventHandler) types.Result[interface{}] {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if eb.subscribers[eventType] == nil {
		eb.subscribers[eventType] = make([]EventHandler, 0)
	}

	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)

	if eb.logger != nil {
		eb.logger.Debug("EventBus: Subscribed to event type", map[string]interface{}{
			"event_type":    eventType,
			"handler_count": len(eb.subscribers[eventType]),
		})
	}

	return types.NewResult[interface{}](nil)
}

// Publish sends an event to all registered handlers
func (eb *DefaultEventBus) Publish(event Event) types.Result[interface{}] {
	eb.mutex.Lock()
	// Add to history
	eb.history = append(eb.history, event)
	if len(eb.history) > eb.maxHistory {
		eb.history = eb.history[1:]
	}

	// Get handlers for this event type
	handlers := make([]EventHandler, len(eb.subscribers[event.Type]))
	copy(handlers, eb.subscribers[event.Type])
	eb.mutex.Unlock()

	if eb.logger != nil {
		eb.logger.Debug("EventBus: Publishing event", map[string]interface{}{
			"event_id":      event.ID,
			"event_type":    event.Type,
			"source":        event.Source,
			"target":        event.Target,
			"handler_count": len(handlers),
		})
	}

	// Execute handlers without holding the lock
	var errors []string
	for _, handler := range handlers {
		if err := handler(event); err != nil {
			errors = append(errors, err.Error())
			if eb.logger != nil {
				eb.logger.Error("EventBus: Handler error", err, map[string]interface{}{
					"event_id":   event.ID,
					"event_type": event.Type,
				})
			}
		}
	}

	if len(errors) > 0 {
		return types.NewResultError[interface{}](fmt.Errorf("event handler errors: %v", errors))
	}

	return types.NewResult[interface{}](nil)
}

// Unsubscribe removes an event handler (simplified implementation)
func (eb *DefaultEventBus) Unsubscribe(eventType string, handler EventHandler) types.Result[interface{}] {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	// Note: This is a simplified implementation that removes all handlers for the event type
	// A complete implementation would need to compare function pointers or use IDs
	delete(eb.subscribers, eventType)

	if eb.logger != nil {
		eb.logger.Debug("EventBus: Unsubscribed from event type", map[string]interface{}{
			"event_type": eventType,
		})
	}

	return types.NewResult[interface{}](nil)
}

// GetEventHistory returns recent events for debugging
func (eb *DefaultEventBus) GetEventHistory(limit int) types.Result[[]Event] {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	if limit <= 0 || limit > len(eb.history) {
		limit = len(eb.history)
	}

	history := make([]Event, limit)
	copy(history, eb.history[len(eb.history)-limit:])

	return types.NewResult(history)
}

// DefaultStateSynchronizer implements the StateSynchronizer interface
type DefaultStateSynchronizer struct {
	docScanner            *ConcurrentDocumentScanner
	codeAnalyzer          CodeAnalyzer
	inconsistencyDetector ComparisonEngine
	stateManager          *DocumentationStateManager
	eventBus              EventBus
	config                *SyncConfig
	logger                Logger

	// Runtime state
	mutex          sync.RWMutex
	isRunning      bool
	currentMode    SyncMode
	watchSession   *WatchSession
	agentStates    map[string]*AgentStatus
	lastSyncResult *SyncResult
	startTime      int64
	processedFiles int
}

// NewDefaultStateSynchronizer creates a new state synchronizer
func NewDefaultStateSynchronizer(
	docScanner *ConcurrentDocumentScanner,
	codeAnalyzer CodeAnalyzer,
	inconsistencyDetector ComparisonEngine,
	stateManager *DocumentationStateManager,
	logger Logger,
) *DefaultStateSynchronizer {
	eventBus := NewDefaultEventBus(1000, logger) // Keep last 1000 events

	return &DefaultStateSynchronizer{
		docScanner:            docScanner,
		codeAnalyzer:          codeAnalyzer,
		inconsistencyDetector: inconsistencyDetector,
		stateManager:          stateManager,
		eventBus:              eventBus,
		logger:                logger,
		agentStates:           make(map[string]*AgentStatus),
		isRunning:             false,
	}
}

// SynchronizeProject performs full project synchronization with all agents
func (ss *DefaultStateSynchronizer) SynchronizeProject(config *SyncConfig) types.Result[*SyncResult] { //nolint:funlen
	ss.mutex.Lock()
	if ss.isRunning {
		ss.mutex.Unlock()
		return types.NewResultError[*SyncResult](fmt.Errorf("synchronization already in progress"))
	}
	ss.isRunning = true
	ss.config = config
	ss.currentMode = config.SyncMode
	ss.startTime = time.Now().UnixMilli()
	ss.mutex.Unlock()

	defer func() {
		ss.mutex.Lock()
		ss.isRunning = false
		ss.mutex.Unlock()
	}()

	if ss.logger != nil {
		ss.logger.Info("Starting full project synchronization", map[string]interface{}{
			"project_root":   config.ProjectRoot,
			"enabled_agents": config.EnabledAgents,
			"sync_mode":      config.SyncMode,
		})
	}

	// Initialize result
	result := &SyncResult{
		StartTime:      ss.startTime,
		SyncMode:       config.SyncMode,
		ExecutedAgents: make([]string, 0),
		SkippedAgents:  make([]string, 0),
		FailedAgents:   make([]string, 0),
		AgentResults:   make(map[string]interface{}),
		Errors:         make([]SyncError, 0),
	}

	// Publish synchronization start event
	startEvent := Event{
		ID:        generateEventID(),
		Type:      "sync.started",
		Source:    "state-synchronizer",
		Data:      map[string]interface{}{"mode": config.SyncMode, "project_root": config.ProjectRoot},
		Timestamp: time.Now().UnixMilli(),
		Priority:  EventPriorityNormal,
	}
	ss.eventBus.Publish(startEvent)

	// Phase 1: Document Scanning
	if ss.isAgentEnabled("doc-scanner", config.EnabledAgents) {
		if docResult := ss.runDocumentScanner(config); !docResult.IsOk() {
			ss.addSyncError(result, "doc-scanner", "scan_failed", docResult.Error.Error(), "", 0)
			result.FailedAgents = append(result.FailedAgents, "doc-scanner")
		} else {
			result.ExecutedAgents = append(result.ExecutedAgents, "doc-scanner")
			result.DocumentationState = docResult.Value
			result.AgentResults["doc-scanner"] = docResult.Value
		}
	} else {
		result.SkippedAgents = append(result.SkippedAgents, "doc-scanner")
	}

	// Phase 2: Code Analysis
	var codeSymbols []CodeSymbol
	if ss.isAgentEnabled("code-analyzer", config.EnabledAgents) {
		if codeResult := ss.runCodeAnalyzer(config); !codeResult.IsOk() {
			ss.addSyncError(result, "code-analyzer", "analysis_failed", codeResult.Error.Error(), "", 0)
			result.FailedAgents = append(result.FailedAgents, "code-analyzer")
		} else {
			result.ExecutedAgents = append(result.ExecutedAgents, "code-analyzer")
			codeSymbols = codeResult.Value
			result.CodeSymbols = codeSymbols
			result.AgentResults["code-analyzer"] = codeSymbols
		}
	} else {
		result.SkippedAgents = append(result.SkippedAgents, "code-analyzer")
	}

	// Phase 3: Inconsistency Detection
	if ss.isAgentEnabled("inconsistency-detector", config.EnabledAgents) {
		if inconsistencyResult := ss.runInconsistencyDetector(config, codeSymbols, result.DocumentationState); !inconsistencyResult.IsOk() { //nolint:lll
			ss.addSyncError(result, "inconsistency-detector", "detection_failed", inconsistencyResult.Error.Error(), "", 0)
			result.FailedAgents = append(result.FailedAgents, "inconsistency-detector")
		} else {
			result.ExecutedAgents = append(result.ExecutedAgents, "inconsistency-detector")
			result.Inconsistencies = inconsistencyResult.Value
			result.AgentResults["inconsistency-detector"] = inconsistencyResult.Value
		}
	} else {
		result.SkippedAgents = append(result.SkippedAgents, "inconsistency-detector")
	}

	// Phase 4: Calculate Quality Metrics
	result.QualityMetrics = ss.calculateQualityMetrics(result.Inconsistencies, codeSymbols)

	// Phase 5: Generate Performance Metrics
	result.EndTime = time.Now().UnixMilli()
	result.Duration = result.EndTime - result.StartTime
	result.PerformanceMetrics = ss.calculatePerformanceMetrics(result)

	// Determine overall success
	result.Success = len(result.FailedAgents) == 0

	// Store result
	ss.mutex.Lock()
	ss.lastSyncResult = result
	ss.mutex.Unlock()

	// Publish completion event
	completionEvent := Event{
		ID:        generateEventID(),
		Type:      "sync.completed",
		Source:    "state-synchronizer",
		Data:      map[string]interface{}{"success": result.Success, "duration": result.Duration},
		Timestamp: time.Now().UnixMilli(),
		Priority:  EventPriorityNormal,
	}
	ss.eventBus.Publish(completionEvent)

	if ss.logger != nil {
		ss.logger.Info("Project synchronization completed", map[string]interface{}{
			"success":         result.Success,
			"duration_ms":     result.Duration,
			"executed_agents": len(result.ExecutedAgents),
			"failed_agents":   len(result.FailedAgents),
			"inconsistencies": len(result.Inconsistencies),
		})
	}

	return types.NewResult(result)
}

// SynchronizeIncremental performs incremental synchronization of changed files
func (ss *DefaultStateSynchronizer) SynchronizeIncremental(config *SyncConfig, changedFiles []string) types.Result[*SyncResult] {
	if ss.logger != nil {
		ss.logger.Info("Starting incremental synchronization", map[string]interface{}{
			"changed_files": len(changedFiles),
			"files":         changedFiles,
		})
	}

	// For incremental sync, we can optimize by only processing changed files
	// This is a simplified implementation - a production version would be more sophisticated
	incrementalConfig := *config
	incrementalConfig.SyncMode = SyncModeIncremental

	// TODO: Add logic to filter processing based on changedFiles
	// For now, we perform a full sync but mark it as incremental
	return ss.SynchronizeProject(&incrementalConfig)
}

// GetSyncStatus returns current synchronization status and progress
func (ss *DefaultStateSynchronizer) GetSyncStatus() types.Result[*SyncStatus] {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	status := &SyncStatus{
		IsActive:        ss.isRunning,
		CurrentMode:     ss.currentMode,
		RunningAgents:   make([]string, 0),
		CompletedAgents: make([]string, 0),
		FailedAgents:    make([]string, 0),
		AgentProgress:   make(map[string]float64),
		WatchSession:    ss.watchSession,
	}

	if ss.lastSyncResult != nil {
		status.LastSync = ss.lastSyncResult.EndTime
		status.CompletedAgents = ss.lastSyncResult.ExecutedAgents
		status.FailedAgents = ss.lastSyncResult.FailedAgents
	}

	// Calculate overall progress
	totalAgents := len(ss.agentStates)
	if totalAgents > 0 {
		var totalProgress float64
		for agentName, agentStatus := range ss.agentStates {
			status.AgentProgress[agentName] = agentStatus.Progress
			totalProgress += agentStatus.Progress

			switch agentStatus.State {
			case AgentStateRunning:
				status.RunningAgents = append(status.RunningAgents, agentName)
			case AgentStateCompleted:
				// Already added to CompletedAgents from lastSyncResult
			case AgentStateFailed:
				// Already added to FailedAgents from lastSyncResult
			}
		}
		status.OverallProgress = totalProgress / float64(totalAgents)
	}

	if ss.isRunning {
		// Estimate completion time based on progress and elapsed time
		elapsed := time.Now().UnixMilli() - ss.startTime
		if status.OverallProgress > 0 {
			estimated := float64(elapsed) / status.OverallProgress
			status.EstimatedCompletion = ss.startTime + int64(estimated)
		}
	}

	return types.NewResult(status)
}

// StartWatchMode starts continuous monitoring and synchronization
func (ss *DefaultStateSynchronizer) StartWatchMode(config *SyncConfig) types.Result[*WatchSession] {
	ss.mutex.Lock()
	if ss.watchSession != nil && ss.watchSession.IsActive {
		ss.mutex.Unlock()
		return types.NewResultError[*WatchSession](fmt.Errorf("watch mode already active"))
	}

	session := &WatchSession{
		ID:           generateSessionID(),
		StartTime:    time.Now().UnixMilli(),
		IsActive:     true,
		WatchedPaths: config.IncludePatterns,
		LastActivity: time.Now().UnixMilli(),
	}
	ss.watchSession = session
	ss.mutex.Unlock()

	if ss.logger != nil {
		ss.logger.Info("Started watch mode", map[string]interface{}{
			"session_id":    session.ID,
			"watched_paths": session.WatchedPaths,
		})
	}

	// Publish watch mode start event
	watchEvent := Event{
		ID:        generateEventID(),
		Type:      "watch.started",
		Source:    "state-synchronizer",
		Data:      map[string]interface{}{"session_id": session.ID, "paths": session.WatchedPaths},
		Timestamp: time.Now().UnixMilli(),
		Priority:  EventPriorityNormal,
	}
	ss.eventBus.Publish(watchEvent)

	return types.NewResult(session)
}

// StopWatchMode stops continuous monitoring
func (ss *DefaultStateSynchronizer) StopWatchMode() types.Result[interface{}] {
	ss.mutex.Lock()
	if ss.watchSession == nil || !ss.watchSession.IsActive {
		ss.mutex.Unlock()
		return types.NewResultError[interface{}](fmt.Errorf("watch mode not active"))
	}

	ss.watchSession.IsActive = false
	sessionID := ss.watchSession.ID
	ss.mutex.Unlock()

	if ss.logger != nil {
		ss.logger.Info("Stopped watch mode", map[string]interface{}{
			"session_id": sessionID,
		})
	}

	// Publish watch mode stop event
	stopEvent := Event{
		ID:        generateEventID(),
		Type:      "watch.stopped",
		Source:    "state-synchronizer",
		Data:      map[string]interface{}{"session_id": sessionID},
		Timestamp: time.Now().UnixMilli(),
		Priority:  EventPriorityNormal,
	}
	ss.eventBus.Publish(stopEvent)

	return types.NewResult[interface{}](nil)
}

// GetAgentStatus returns status of individual agents
func (ss *DefaultStateSynchronizer) GetAgentStatus() types.Result[map[string]*AgentStatus] {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()

	// Create a deep copy of agent states
	result := make(map[string]*AgentStatus)
	for name, status := range ss.agentStates {
		statusCopy := *status
		result[name] = &statusCopy
	}

	return types.NewResult(result)
}

// Helper methods

func (ss *DefaultStateSynchronizer) isAgentEnabled(agentName string, enabledAgents []string) bool {
	if len(enabledAgents) == 0 {
		return true // If no specific agents listed, enable all
	}

	for _, enabled := range enabledAgents {
		if enabled == agentName {
			return true
		}
	}
	return false
}

func (ss *DefaultStateSynchronizer) runDocumentScanner(config *SyncConfig) types.Result[*DocumentationScanState] {
	ss.updateAgentStatus("doc-scanner", AgentStateRunning, "Scanning documentation files", 0.0)

	startTime := time.Now().UnixMilli()

	// Configure document scanner
	scanConfig := &ScanConfig{
		MaxWorkers:     config.MaxConcurrency,
		BatchSize:      config.BatchSize,
		ReportProgress: config.Verbose,
	}

	ss.docScanner.SetConfig(*scanConfig)

	// Publish agent start event
	agentEvent := Event{
		ID:        generateEventID(),
		Type:      "agent.started",
		Source:    "doc-scanner",
		Data:      map[string]interface{}{"config": scanConfig},
		Timestamp: time.Now().UnixMilli(),
		Priority:  EventPriorityNormal,
	}
	ss.eventBus.Publish(agentEvent)

	// Find documentation files to scan
	docFiles := ss.findDocumentationFiles(config)

	// Run document scan
	scanResult := ss.docScanner.ScanDocuments(docFiles)
	if !scanResult.IsOk() {
		ss.updateAgentStatus("doc-scanner", AgentStateFailed, "Scan failed", 0.0)
		return types.NewResultError[*DocumentationScanState](scanResult.Error)
	}

	duration := time.Now().UnixMilli() - startTime
	ss.updateAgentStatus("doc-scanner", AgentStateCompleted, "Scan completed", 100.0)
	ss.setAgentDuration("doc-scanner", duration)

	// Load current state from state manager
	currentState := ss.stateManager.GetState()

	// Publish agent completion event
	completionEvent := Event{
		ID:        generateEventID(),
		Type:      "agent.completed",
		Source:    "doc-scanner",
		Data:      map[string]interface{}{"duration": duration, "files_processed": len(currentState.FileStates)},
		Timestamp: time.Now().UnixMilli(),
		Priority:  EventPriorityNormal,
	}
	ss.eventBus.Publish(completionEvent)

	return types.NewResult(currentState)
}

func (ss *DefaultStateSynchronizer) runCodeAnalyzer(config *SyncConfig) types.Result[[]CodeSymbol] {
	ss.updateAgentStatus("code-analyzer", AgentStateRunning, "Analyzing code symbols", 0.0)

	startTime := time.Now().UnixMilli()

	// Publish agent start event
	agentEvent := Event{
		ID:        generateEventID(),
		Type:      "agent.started",
		Source:    "code-analyzer",
		Data:      map[string]interface{}{"project_root": config.ProjectRoot},
		Timestamp: time.Now().UnixMilli(),
		Priority:  EventPriorityNormal,
	}
	ss.eventBus.Publish(agentEvent)

	// For now, analyze the main package - in a complete implementation,
	// we would discover and analyze all packages
	analysisResult := ss.codeAnalyzer.AnalyzePackage(config.ProjectRoot)
	if !analysisResult.IsOk() {
		ss.updateAgentStatus("code-analyzer", AgentStateFailed, "Analysis failed", 0.0)
		return types.NewResultError[[]CodeSymbol](analysisResult.Error)
	}

	duration := time.Now().UnixMilli() - startTime
	symbols := analysisResult.Value
	ss.updateAgentStatus("code-analyzer", AgentStateCompleted, "Analysis completed", 100.0)
	ss.setAgentDuration("code-analyzer", duration)

	// Publish agent completion event
	completionEvent := Event{
		ID:        generateEventID(),
		Type:      "agent.completed",
		Source:    "code-analyzer",
		Data:      map[string]interface{}{"duration": duration, "symbols_found": len(symbols)},
		Timestamp: time.Now().UnixMilli(),
		Priority:  EventPriorityNormal,
	}
	ss.eventBus.Publish(completionEvent)

	return types.NewResult(symbols)
}

func (ss *DefaultStateSynchronizer) runInconsistencyDetector(config *SyncConfig, codeSymbols []CodeSymbol, docState *DocumentationScanState) types.Result[[]Inconsistency] { //nolint:lll,unparam
	ss.updateAgentStatus("inconsistency-detector", AgentStateRunning, "Detecting inconsistencies", 0.0)

	startTime := time.Now().UnixMilli()

	// Publish agent start event
	agentEvent := Event{
		ID:        generateEventID(),
		Type:      "agent.started",
		Source:    "inconsistency-detector",
		Data:      map[string]interface{}{"code_symbols": len(codeSymbols), "doc_files": len(docState.FileStates)},
		Timestamp: time.Now().UnixMilli(),
		Priority:  EventPriorityNormal,
	}
	ss.eventBus.Publish(agentEvent)

	// Extract doc sections from documentation state
	docSections := ss.extractDocSections(docState)

	// Run inconsistency detection
	detectionResult := ss.inconsistencyDetector.DetectInconsistencies(codeSymbols, docSections)
	if !detectionResult.IsOk() {
		ss.updateAgentStatus("inconsistency-detector", AgentStateFailed, "Detection failed", 0.0)
		return types.NewResultError[[]Inconsistency](detectionResult.Error)
	}

	duration := time.Now().UnixMilli() - startTime
	inconsistencies := detectionResult.Value
	ss.updateAgentStatus("inconsistency-detector", AgentStateCompleted, "Detection completed", 100.0)
	ss.setAgentDuration("inconsistency-detector", duration)

	// Publish agent completion event
	completionEvent := Event{
		ID:        generateEventID(),
		Type:      "agent.completed",
		Source:    "inconsistency-detector",
		Data:      map[string]interface{}{"duration": duration, "inconsistencies_found": len(inconsistencies)},
		Timestamp: time.Now().UnixMilli(),
		Priority:  EventPriorityNormal,
	}
	ss.eventBus.Publish(completionEvent)

	return types.NewResult(inconsistencies)
}

// findDocumentationFiles discovers documentation files based on config patterns
func (ss *DefaultStateSynchronizer) findDocumentationFiles(config *SyncConfig) []string {
	var files []string

	// Simple implementation - in practice would use filepath.Walk or similar
	// For now, return some common documentation files
	commonDocs := []string{
		filepath.Join(config.ProjectRoot, "README.md"),
		filepath.Join(config.ProjectRoot, "CLAUDE.md"),
		filepath.Join(config.ProjectRoot, "docs", "ARCHITECTURE.md"),
		filepath.Join(config.ProjectRoot, "issues", "specification.md"),
	}

	// Check which files exist
	for _, file := range commonDocs {
		if _, err := os.Stat(file); err == nil {
			files = append(files, file)
		}
	}

	return files
}

func (ss *DefaultStateSynchronizer) extractDocSections(docState *DocumentationScanState) []DocSection {
	var docSections []DocSection //nolint:prealloc

	// This is a simplified implementation - in practice, we'd need to parse
	// the documentation content more thoroughly
	for filePath, fileState := range docState.FileStates {
		section := DocSection{
			Title:       filepath.Base(filePath),
			File:        filePath,
			Content:     "", // Would need to read file content
			Level:       1,
			Line:        1,
			Fingerprint: fileState.Checksum,
		}
		docSections = append(docSections, section)
	}

	return docSections
}

func (ss *DefaultStateSynchronizer) calculateQualityMetrics(inconsistencies []Inconsistency, codeSymbols []CodeSymbol) *QualityMetrics { //nolint:lll
	metrics := &QualityMetrics{
		TotalInconsistencies: len(inconsistencies),
		ScoreBreakdown:       make(map[string]float64),
	}

	// Count by severity
	for _, inc := range inconsistencies {
		switch inc.Severity {
		case SeverityCritical:
			metrics.CriticalIssues++
		case SeverityHigh:
			metrics.HighPriorityIssues++
		case SeverityMedium:
			metrics.MediumPriorityIssues++
		case SeverityLow:
			metrics.LowPriorityIssues++
		}
	}

	// Calculate scores (simplified scoring algorithm)
	totalSymbols := float64(len(codeSymbols))
	if totalSymbols > 0 {
		// Documentation coverage score
		documentedSymbols := totalSymbols - float64(metrics.CriticalIssues) - float64(metrics.HighPriorityIssues)
		metrics.CoverageScore = math.Max(0, documentedSymbols/totalSymbols*100)

		// Consistency score based on inconsistencies
		totalIssues := float64(metrics.TotalInconsistencies)
		if totalIssues == 0 {
			metrics.ConsistencyScore = 100
		} else {
			metrics.ConsistencyScore = math.Max(0, 100-totalIssues/totalSymbols*50)
		}

		// Overall score (weighted average)
		metrics.DocumentationScore = (metrics.CoverageScore + metrics.ConsistencyScore) / 2
		metrics.CodeQualityScore = metrics.ConsistencyScore // Simplified
		metrics.OverallScore = (metrics.DocumentationScore + metrics.CodeQualityScore) / 2

		metrics.ScoreBreakdown["coverage"] = metrics.CoverageScore
		metrics.ScoreBreakdown["consistency"] = metrics.ConsistencyScore
		metrics.ScoreBreakdown["documentation"] = metrics.DocumentationScore
		metrics.ScoreBreakdown["code_quality"] = metrics.CodeQualityScore
	}

	return metrics
}

func (ss *DefaultStateSynchronizer) calculatePerformanceMetrics(result *SyncResult) *PerformanceMetrics {
	metrics := &PerformanceMetrics{
		TotalDuration:     result.Duration,
		AgentDurations:    make(map[string]int64),
		ConcurrentWorkers: ss.config.MaxConcurrency,
		FilesProcessed:    ss.processedFiles,
	}

	// Calculate files per second
	if result.Duration > 0 {
		metrics.FilesPerSecond = float64(metrics.FilesProcessed) / (float64(result.Duration) / 1000.0)
	}

	// Get agent durations from agent states
	ss.mutex.RLock()
	for name, status := range ss.agentStates {
		metrics.AgentDurations[name] = status.LastDuration
	}
	ss.mutex.RUnlock()

	// TODO: Add actual resource usage monitoring
	// For now, provide placeholder values
	metrics.MemoryPeakMB = 0
	metrics.MemoryAverageMB = 0
	metrics.CPUAveragePercent = 0
	metrics.CacheHitRatio = 0

	return metrics
}

func (ss *DefaultStateSynchronizer) updateAgentStatus(agentName string, state AgentState, task string, progress float64) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if ss.agentStates[agentName] == nil {
		ss.agentStates[agentName] = &AgentStatus{
			Name:         agentName,
			State:        AgentStateIdle,
			LastRun:      0,
			LastDuration: 0,
			SuccessRate:  100.0,
			ErrorCount:   0,
			Progress:     0.0,
			Metadata:     make(map[string]interface{}),
		}
	}

	status := ss.agentStates[agentName]
	status.State = state
	status.CurrentTask = task
	status.Progress = progress

	if state == AgentStateRunning && status.LastRun == 0 {
		status.LastRun = time.Now().UnixMilli()
	}
}

func (ss *DefaultStateSynchronizer) setAgentDuration(agentName string, duration int64) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()

	if status := ss.agentStates[agentName]; status != nil {
		status.LastDuration = duration
	}
}

func (ss *DefaultStateSynchronizer) addSyncError(result *SyncResult, agent, errorType, message, file string, line int) {
	syncError := SyncError{
		Agent:       agent,
		Type:        errorType,
		Message:     message,
		File:        file,
		Line:        line,
		Context:     make(map[string]interface{}),
		Timestamp:   time.Now().UnixMilli(),
		Severity:    SeverityHigh,
		Recoverable: true,
	}
	result.Errors = append(result.Errors, syncError)
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixMilli(), rand.Intn(10000)) //nolint:gosec
}

func generateSessionID() string {
	return fmt.Sprintf("session_%d_%d", time.Now().UnixMilli(), rand.Intn(10000)) //nolint:gosec
}
