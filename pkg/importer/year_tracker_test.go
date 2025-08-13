package importer

import (
	"testing"
)

func TestYearTracker_BasicFunctionality(t *testing.T) {
	tracker := NewYearTracker()

	// Test initial state
	if tracker.GetInitial(2023) != 0 {
		t.Errorf("Expected initial count 0, got %d", tracker.GetInitial(2023))
	}
	if tracker.GetAdded(2023) != 0 {
		t.Errorf("Expected added count 0, got %d", tracker.GetAdded(2023))
	}
	if tracker.GetDuplicates(2023) != 0 {
		t.Errorf("Expected duplicates count 0, got %d", tracker.GetDuplicates(2023))
	}

	// Track some initial entries
	tracker.TrackInitialEntry(2023)
	tracker.TrackInitialEntry(2023)
	tracker.TrackInitialEntry(2024)

	if tracker.GetInitial(2023) != 2 {
		t.Errorf("Expected initial count 2 for 2023, got %d", tracker.GetInitial(2023))
	}
	if tracker.GetInitial(2024) != 1 {
		t.Errorf("Expected initial count 1 for 2024, got %d", tracker.GetInitial(2024))
	}

	// Track some import entries
	tracker.TrackImportEntry(2023, true)  // added
	tracker.TrackImportEntry(2023, false) // duplicate
	tracker.TrackImportEntry(2024, true)  // added

	if tracker.GetAdded(2023) != 1 {
		t.Errorf("Expected added count 1 for 2023, got %d", tracker.GetAdded(2023))
	}
	if tracker.GetDuplicates(2023) != 1 {
		t.Errorf("Expected duplicates count 1 for 2023, got %d", tracker.GetDuplicates(2023))
	}
	if tracker.GetAdded(2024) != 1 {
		t.Errorf("Expected added count 1 for 2024, got %d", tracker.GetAdded(2024))
	}
}

func TestYearTracker_ValidateYearStatistics(t *testing.T) {
	tracker := NewYearTracker()

	// Set up year 2023: 5 initial + 3 added = 8 final (2 duplicates not counted)
	for i := 0; i < 5; i++ {
		tracker.TrackInitialEntry(2023)
	}
	for i := 0; i < 3; i++ {
		tracker.TrackImportEntry(2023, true) // added
	}
	for i := 0; i < 2; i++ {
		tracker.TrackImportEntry(2023, false) // duplicate
	}

	// Should validate successfully
	err := tracker.ValidateYearStatistics(2023, 8)
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}

	// Should fail with wrong final count
	err = tracker.ValidateYearStatistics(2023, 10)
	if err == nil {
		t.Errorf("Expected validation to fail with wrong final count")
	}

	// Should fail with wrong final count (too low)
	err = tracker.ValidateYearStatistics(2023, 6)
	if err == nil {
		t.Errorf("Expected validation to fail with wrong final count")
	}
}

func TestYearTracker_GetAllYears(t *testing.T) {
	tracker := NewYearTracker()

	// Initially should be empty
	years := tracker.GetAllYears()
	if len(years) != 0 {
		t.Errorf("Expected no years initially, got %v", years)
	}

	// Add activity for different years
	tracker.TrackInitialEntry(2020)
	tracker.TrackImportEntry(2021, true)
	tracker.TrackImportEntry(2022, false)

	years = tracker.GetAllYears()
	if len(years) != 3 {
		t.Errorf("Expected 3 years, got %d: %v", len(years), years)
	}

	// Check that all expected years are present
	yearMap := make(map[int]bool)
	for _, year := range years {
		yearMap[year] = true
	}

	expectedYears := []int{2020, 2021, 2022}
	for _, expected := range expectedYears {
		if !yearMap[expected] {
			t.Errorf("Expected year %d to be in results %v", expected, years)
		}
	}
}

func TestYearTracker_MultiYearScenario(t *testing.T) {
	tracker := NewYearTracker()

	// Simulate a realistic multi-year import scenario

	// Year 2022: 100 initial entries
	for i := 0; i < 100; i++ {
		tracker.TrackInitialEntry(2022)
	}

	// Year 2023: 50 initial, 25 new added, 10 duplicates
	for i := 0; i < 50; i++ {
		tracker.TrackInitialEntry(2023)
	}
	for i := 0; i < 25; i++ {
		tracker.TrackImportEntry(2023, true)
	}
	for i := 0; i < 10; i++ {
		tracker.TrackImportEntry(2023, false)
	}

	// Year 2024: 0 initial, 15 new added, 5 duplicates
	for i := 0; i < 15; i++ {
		tracker.TrackImportEntry(2024, true)
	}
	for i := 0; i < 5; i++ {
		tracker.TrackImportEntry(2024, false)
	}

	// Validate each year
	if err := tracker.ValidateYearStatistics(2022, 100); err != nil {
		t.Errorf("Year 2022 validation failed: %v", err)
	}

	if err := tracker.ValidateYearStatistics(2023, 75); err != nil {
		t.Errorf("Year 2023 validation failed: %v", err)
	}

	if err := tracker.ValidateYearStatistics(2024, 15); err != nil {
		t.Errorf("Year 2024 validation failed: %v", err)
	}

	// Verify individual counts
	if tracker.GetInitial(2022) != 100 {
		t.Errorf("2022 initial: expected 100, got %d", tracker.GetInitial(2022))
	}
	if tracker.GetAdded(2023) != 25 {
		t.Errorf("2023 added: expected 25, got %d", tracker.GetAdded(2023))
	}
	if tracker.GetDuplicates(2024) != 5 {
		t.Errorf("2024 duplicates: expected 5, got %d", tracker.GetDuplicates(2024))
	}
}
