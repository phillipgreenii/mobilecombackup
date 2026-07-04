package analyzer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Integration tests for analyzer package components
// These tests use real file systems and test component interactions

func TestDocumentationStateManager_Integration(t *testing.T) {
	// Create temporary directory for test files
	testDir := t.TempDir()
	statePath := filepath.Join(testDir, "doc_state.json")

	logger := &IntegrationTestLogger{t: t}
	stateManager := NewDocumentationStateManager(statePath, logger)

	t.Run("NewStateManager_CreatesDefaultState", func(t *testing.T) {
		// Load should create a default state when no file exists
		result := stateManager.Load()
		if result.IsErr() {
			t.Fatalf("Failed to create default state: %v", result.Error)
		}

		state := result.Value
		if state.Version == "" {
			t.Error("Expected state to have version")
		}
		if state.FileStates == nil {
			t.Error("Expected state to have file states map")
		}
		if len(state.FileStates) != 0 {
			t.Errorf("Expected empty file states, got %d", len(state.FileStates))
		}
	})

	t.Run("UpdateFileState_PersistAndReload", func(t *testing.T) {
		// Create test markdown content
		testContent := `# Test Document

This is test content for integration testing.

## Section 1

Some content here.
`
		testFile := filepath.Join(testDir, "test.md")
		err := os.WriteFile(testFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test markdown file: %v", err)
		}

		// Get file modification time
		fileInfo, err := os.Stat(testFile)
		if err != nil {
			t.Fatalf("Failed to stat test file: %v", err)
		}
		modTime := fileInfo.ModTime().Unix()

		// Parse the markdown to get sections
		analyzer := NewMarkdownAnalyzer(logger)
		parseResult := analyzer.ParseMarkdown(testFile)
		if parseResult.IsErr() {
			t.Fatalf("Failed to parse markdown: %v", parseResult.Error)
		}
		sections := parseResult.Value

		// Update file state
		updateResult := stateManager.UpdateFileState(testFile, sections, modTime)
		if updateResult.IsErr() {
			t.Fatalf("Failed to update file state: %v", updateResult.Error)
		}

		// Persist state
		persistResult := stateManager.Persist()
		if persistResult.IsErr() {
			t.Fatalf("Failed to persist state: %v", persistResult.Error)
		}

		// Verify state file was created
		if _, err := os.Stat(statePath); os.IsNotExist(err) {
			t.Fatal("State file was not created")
		}

		// Create new state manager and load from file
		newStateManager := NewDocumentationStateManager(statePath, logger)
		loadResult := newStateManager.Load()
		if loadResult.IsErr() {
			t.Fatalf("Failed to load persisted state: %v", loadResult.Error)
		}

		loadedState := loadResult.Value
		if len(loadedState.FileStates) != 1 {
			t.Errorf("Expected 1 file state, got %d", len(loadedState.FileStates))
		}

		fileState, exists := loadedState.FileStates[testFile]
		if !exists {
			t.Error("Expected file state for test file")
		} else {
			if fileState.Path != testFile {
				t.Errorf("Expected path %s, got %s", testFile, fileState.Path)
			}
			if fileState.LastModified != modTime {
				t.Errorf("Expected mod time %d, got %d", modTime, fileState.LastModified)
			}
			if len(fileState.Sections) != len(sections) {
				t.Errorf("Expected %d sections, got %d", len(sections), len(fileState.Sections))
			}
		}
	})

	t.Run("IsFileChanged_DetectsChanges", func(t *testing.T) {
		testFile := filepath.Join(testDir, "change_test.md")

		// Create initial file
		initialContent := "# Initial Content"
		err := os.WriteFile(testFile, []byte(initialContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Get initial modification time
		fileInfo, err := os.Stat(testFile)
		if err != nil {
			t.Fatalf("Failed to stat test file: %v", err)
		}
		initialModTime := fileInfo.ModTime().Unix()

		// File should be changed since it's not in state
		if !stateManager.IsFileChanged(testFile, initialModTime) {
			t.Error("Expected file to be detected as changed (new file)")
		}

		// Add file to state
		analyzer := NewMarkdownAnalyzer(logger)
		parseResult := analyzer.ParseMarkdown(testFile)
		if parseResult.IsErr() {
			t.Fatalf("Failed to parse markdown: %v", parseResult.Error)
		}
		sections := parseResult.Value

		updateResult := stateManager.UpdateFileState(testFile, sections, initialModTime)
		if updateResult.IsErr() {
			t.Fatalf("Failed to update file state: %v", updateResult.Error)
		}

		// File should not be changed now
		if stateManager.IsFileChanged(testFile, initialModTime) {
			t.Error("Expected file to not be detected as changed")
		}

		// Wait over 1 second so mod time changes at Unix seconds precision
		time.Sleep(1100 * time.Millisecond)
		modifiedContent := "# Modified Content\n\nThis content has changed."
		err = os.WriteFile(testFile, []byte(modifiedContent), 0644)
		if err != nil {
			t.Fatalf("Failed to modify test file: %v", err)
		}

		// Get new modification time
		fileInfo, err = os.Stat(testFile)
		if err != nil {
			t.Fatalf("Failed to stat modified test file: %v", err)
		}
		newModTime := fileInfo.ModTime().Unix()

		// File should be changed now
		if !stateManager.IsFileChanged(testFile, newModTime) {
			t.Error("Expected file to be detected as changed after modification")
		}
	})
}

func TestConcurrentDocumentScanner_Integration(t *testing.T) {
	// Create test directory structure
	testDir := t.TempDir()
	docsDir := filepath.Join(testDir, "docs")
	err := os.MkdirAll(docsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test markdown files
	testFiles := map[string]string{
		"README.md": `# Project README

This is the main project documentation.

## Installation

Instructions for installing.
`,
		"docs/api.md": `# API Documentation  

API reference documentation.

## Endpoints

### GET /api/users

Returns list of users.
`,
		"docs/guide.md": `# User Guide

How to use the application.

## Getting Started

First steps.
`,
	}

	for relPath, content := range testFiles {
		fullPath := filepath.Join(testDir, relPath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory for %s: %v", relPath, err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", relPath, err)
		}
	}

	logger := &IntegrationTestLogger{t: t}
	stateManager := NewDocumentationStateManager(filepath.Join(testDir, "scan_state.json"), logger)
	analyzer := NewMarkdownAnalyzer(logger)
	scanner := NewConcurrentDocumentScanner(analyzer, stateManager, logger)

	t.Run("ScanDocuments_ProcessesAllFiles", func(t *testing.T) {
		// Configure scanner
		config := ScanConfig{
			MaxWorkers:     2,
			BatchSize:      10,
			ReportProgress: true,
		}
		scanner.SetConfig(config)

		// Build list of files to scan
		var filesToScan []string
		for relPath := range testFiles {
			filesToScan = append(filesToScan, filepath.Join(testDir, relPath))
		}

		// Scan documents
		result := scanner.ScanDocuments(filesToScan)
		if result.IsErr() {
			t.Fatalf("Scan failed: %v", result.Error)
		}

		scanResult := result.Value
		if scanResult.Progress.TotalFiles != len(testFiles) {
			t.Errorf("Expected %d total files, got %d", len(testFiles), scanResult.Progress.TotalFiles)
		}
		if scanResult.Progress.ProcessedFiles != len(testFiles) {
			t.Errorf("Expected %d processed files, got %d", len(testFiles), scanResult.Progress.ProcessedFiles)
		}
		if !scanResult.Success {
			t.Error("Expected scan to succeed")
		}

		// Verify state was updated
		state := stateManager.GetState()
		if state == nil {
			t.Fatal("Expected state to be available")
		}
		if len(state.FileStates) != len(testFiles) {
			t.Errorf("Expected %d file states, got %d", len(testFiles), len(state.FileStates))
		}

		// Verify each file was processed
		for relPath := range testFiles {
			fullPath := filepath.Join(testDir, relPath)
			fileState, exists := state.FileStates[fullPath]
			if !exists {
				t.Errorf("Expected file state for %s", relPath)
				continue
			}
			if len(fileState.Sections) == 0 {
				t.Errorf("Expected sections to be parsed for %s", relPath)
			}
		}
	})

	t.Run("ScanDocuments_HandlesErrors", func(t *testing.T) {
		// Test with nonexistent file
		badFiles := []string{
			filepath.Join(testDir, "nonexistent.md"),
			filepath.Join(testDir, "README.md"), // This one exists
		}

		result := scanner.ScanDocuments(badFiles)
		// Should still succeed with partial results
		if result.IsErr() {
			t.Fatalf("Expected scan to handle errors gracefully: %v", result.Error)
		}

		scanResult := result.Value
		// Should process the file that exists
		if scanResult.Progress.ProcessedFiles == 0 {
			t.Error("Expected at least one file to be processed")
		}
	})
}

func TestEventBus_Integration(t *testing.T) {
	logger := &IntegrationTestLogger{t: t}
	eventBus := NewDefaultEventBus(100, logger)

	t.Run("EventSubscriptionAndPublishing", func(t *testing.T) {
		var receivedEvents []Event
		handler := func(event Event) error {
			receivedEvents = append(receivedEvents, event)
			return nil
		}

		// Subscribe to events
		subscribeResult := eventBus.Subscribe("test.event", handler)
		if subscribeResult.IsErr() {
			t.Fatalf("Failed to subscribe: %v", subscribeResult.Error)
		}

		// Publish an event
		testEvent := Event{
			ID:        "test-1",
			Type:      "test.event",
			Source:    "integration-test",
			Data:      map[string]interface{}{"message": "hello"},
			Timestamp: time.Now().UnixMilli(),
			Priority:  EventPriorityNormal,
		}

		publishResult := eventBus.Publish(testEvent)
		if publishResult.IsErr() {
			t.Fatalf("Failed to publish event: %v", publishResult.Error)
		}

		// Give handler time to process
		time.Sleep(50 * time.Millisecond)

		// Verify event was received
		if len(receivedEvents) != 1 {
			t.Errorf("Expected 1 received event, got %d", len(receivedEvents))
		} else {
			received := receivedEvents[0]
			if received.ID != testEvent.ID {
				t.Errorf("Expected event ID %s, got %s", testEvent.ID, received.ID)
			}
			if received.Type != testEvent.Type {
				t.Errorf("Expected event type %s, got %s", testEvent.Type, received.Type)
			}
		}

		// Check event history
		historyResult := eventBus.GetEventHistory(10)
		if historyResult.IsErr() {
			t.Fatalf("Failed to get event history: %v", historyResult.Error)
		}

		history := historyResult.Value
		if len(history) != 1 {
			t.Errorf("Expected 1 event in history, got %d", len(history))
		}
	})
}

// IntegrationTestLogger implements the Logger interface for integration tests
type IntegrationTestLogger struct {
	t *testing.T
}

func (l *IntegrationTestLogger) Info(msg string, fields ...interface{}) {
	if testing.Verbose() {
		l.t.Logf("[INFO] %s", msg)
	}
}

func (l *IntegrationTestLogger) Warn(msg string, fields ...interface{}) {
	if testing.Verbose() {
		l.t.Logf("[WARN] %s", msg)
	}
}

func (l *IntegrationTestLogger) Error(msg string, fields ...interface{}) {
	l.t.Logf("[ERROR] %s", msg)
}

func (l *IntegrationTestLogger) Debug(msg string, fields ...interface{}) {
	if testing.Verbose() {
		l.t.Logf("[DEBUG] %s", msg)
	}
}
