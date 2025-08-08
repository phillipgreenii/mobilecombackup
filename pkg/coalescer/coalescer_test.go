package coalescer

import (
	"testing"
	"time"
)

// mockEntry implements Entry for testing
type mockEntry struct {
	hash      string
	timestamp time.Time
	year      int
}

func (m mockEntry) Hash() string         { return m.hash }
func (m mockEntry) Timestamp() time.Time { return m.timestamp }
func (m mockEntry) Year() int            { return m.year }

func TestGenericCoalescer_AddAndDuplicates(t *testing.T) {
	coalescer := NewCoalescer[mockEntry]()
	
	entry1 := mockEntry{
		hash:      "hash1",
		timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		year:      2023,
	}
	
	entry2 := mockEntry{
		hash:      "hash2",
		timestamp: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
		year:      2023,
	}
	
	duplicate := mockEntry{
		hash:      "hash1", // Same hash as entry1
		timestamp: time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC),
		year:      2023,
	}
	
	// First entry should be added
	if !coalescer.Add(entry1) {
		t.Error("Expected first entry to be added")
	}
	
	// Second entry with different hash should be added
	if !coalescer.Add(entry2) {
		t.Error("Expected second entry to be added")
	}
	
	// Duplicate should not be added
	if coalescer.Add(duplicate) {
		t.Error("Expected duplicate to be rejected")
	}
	
	// Check summary
	summary := coalescer.GetSummary()
	if summary.Initial != 0 {
		t.Errorf("Expected Initial=0, got %d", summary.Initial)
	}
	if summary.Final != 2 {
		t.Errorf("Expected Final=2, got %d", summary.Final)
	}
	if summary.Added != 2 {
		t.Errorf("Expected Added=2, got %d", summary.Added)
	}
	if summary.Duplicates != 1 {
		t.Errorf("Expected Duplicates=1, got %d", summary.Duplicates)
	}
}

func TestGenericCoalescer_LoadExisting(t *testing.T) {
	coalescer := NewCoalescer[mockEntry]()
	
	existing := []mockEntry{
		{
			hash:      "existing1",
			timestamp: time.Date(2022, 1, 1, 12, 0, 0, 0, time.UTC),
			year:      2022,
		},
		{
			hash:      "existing2",
			timestamp: time.Date(2022, 1, 2, 12, 0, 0, 0, time.UTC),
			year:      2022,
		},
	}
	
	// Load existing entries
	err := coalescer.LoadExisting(existing)
	if err != nil {
		t.Fatalf("Failed to load existing: %v", err)
	}
	
	// Check summary
	summary := coalescer.GetSummary()
	if summary.Initial != 2 {
		t.Errorf("Expected Initial=2, got %d", summary.Initial)
	}
	if summary.Final != 2 {
		t.Errorf("Expected Final=2, got %d", summary.Final)
	}
	
	// Try to add duplicate of existing
	duplicate := mockEntry{
		hash:      "existing1",
		timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		year:      2023,
	}
	
	if coalescer.Add(duplicate) {
		t.Error("Expected duplicate of existing to be rejected")
	}
}

func TestGenericCoalescer_GetAll_Sorted(t *testing.T) {
	coalescer := NewCoalescer[mockEntry]()
	
	// Add entries out of order
	entries := []mockEntry{
		{
			hash:      "hash3",
			timestamp: time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC),
			year:      2023,
		},
		{
			hash:      "hash1",
			timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			year:      2023,
		},
		{
			hash:      "hash2",
			timestamp: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
			year:      2023,
		},
	}
	
	for _, entry := range entries {
		coalescer.Add(entry)
	}
	
	// Get all should return sorted by timestamp
	all := coalescer.GetAll()
	if len(all) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(all))
	}
	
	// Check order
	for i := 1; i < len(all); i++ {
		if all[i-1].Timestamp().After(all[i].Timestamp()) {
			t.Errorf("Entries not sorted: %v > %v", 
				all[i-1].Timestamp(), all[i].Timestamp())
		}
	}
}

func TestGenericCoalescer_GetByYear(t *testing.T) {
	coalescer := NewCoalescer[mockEntry]()
	
	entries := []mockEntry{
		{
			hash:      "hash1",
			timestamp: time.Date(2022, 12, 31, 12, 0, 0, 0, time.UTC),
			year:      2022,
		},
		{
			hash:      "hash2",
			timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			year:      2023,
		},
		{
			hash:      "hash3",
			timestamp: time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC),
			year:      2023,
		},
		{
			hash:      "hash4",
			timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			year:      2024,
		},
	}
	
	for _, entry := range entries {
		coalescer.Add(entry)
	}
	
	// Test each year
	year2022 := coalescer.GetByYear(2022)
	if len(year2022) != 1 {
		t.Errorf("Expected 1 entry for 2022, got %d", len(year2022))
	}
	
	year2023 := coalescer.GetByYear(2023)
	if len(year2023) != 2 {
		t.Errorf("Expected 2 entries for 2023, got %d", len(year2023))
	}
	
	year2024 := coalescer.GetByYear(2024)
	if len(year2024) != 1 {
		t.Errorf("Expected 1 entry for 2024, got %d", len(year2024))
	}
	
	// Test non-existent year
	year2025 := coalescer.GetByYear(2025)
	if len(year2025) != 0 {
		t.Errorf("Expected 0 entries for 2025, got %d", len(year2025))
	}
}

func TestGenericCoalescer_Reset(t *testing.T) {
	coalescer := NewCoalescer[mockEntry]()
	
	// Add some entries
	entry := mockEntry{
		hash:      "hash1",
		timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		year:      2023,
	}
	coalescer.Add(entry)
	
	// Reset
	coalescer.Reset()
	
	// Check that everything is cleared
	summary := coalescer.GetSummary()
	if summary.Initial != 0 || summary.Final != 0 || summary.Added != 0 || summary.Duplicates != 0 {
		t.Error("Expected all counts to be 0 after reset")
	}
	
	// Same entry should be accepted again
	if !coalescer.Add(entry) {
		t.Error("Expected entry to be added after reset")
	}
}

func TestGenericCoalescer_ThreadSafety(t *testing.T) {
	coalescer := NewCoalescer[mockEntry]()
	
	// Run concurrent operations
	done := make(chan bool)
	
	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			entry := mockEntry{
				hash:      string(rune(i)),
				timestamp: time.Now(),
				year:      2023,
			}
			coalescer.Add(entry)
		}
		done <- true
	}()
	
	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = coalescer.GetAll()
			_ = coalescer.GetByYear(2023)
			_ = coalescer.GetSummary()
		}
		done <- true
	}()
	
	// Wait for both to complete
	<-done
	<-done
	
	// If we get here without panic/race, thread safety is working
}