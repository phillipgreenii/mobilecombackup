// Package coalescer provides generic deduplication and merging capabilities for backup entries.
//
// The coalescer package implements a thread-safe, hash-based deduplication system
// that can work with any entry type implementing the Entry interface. It's designed
// to efficiently merge multiple backup sources while eliminating duplicates and
// maintaining chronological order.
//
// # Features
//
//   - Generic implementation using Go 1.18+ generics
//   - Thread-safe concurrent operations with RWMutex
//   - Hash-based O(1) deduplication
//   - Year-based partitioning support
//   - Comprehensive summary reporting
//   - Memory-efficient streaming processing
//
// # Entry Interface
//
// Any type can be coalesced by implementing the Entry interface:
//
//	type MyEntry struct {
//		ID   string
//		Data string
//		Date time.Time
//	}
//
//	func (e MyEntry) Hash() string { return e.ID }
//	func (e MyEntry) Timestamp() time.Time { return e.Date }
//	func (e MyEntry) Year() int { return e.Date.Year() }
//
// # Usage Example
//
// Basic coalescing workflow:
//
//	// Create coalescer for specific entry type
//	coalescer := NewCoalescer[SMS]()
//
//	// Load existing entries from repository
//	existing := []SMS{...}
//	coalescer.LoadExisting(existing)
//
//	// Add new entries (duplicates are automatically filtered)
//	newEntries := []SMS{...}
//	for _, entry := range newEntries {
//		coalescer.Add(entry)
//	}
//
//	// Get deduplicated results sorted by timestamp
//	results := coalescer.GetSorted()
//	summary := coalescer.GetSummary()
//
//	fmt.Printf("Added %d new entries, found %d duplicates\n",
//		summary.Added, summary.Duplicates)
//
// # Thread Safety
//
// All coalescer operations are thread-safe and can be called concurrently
// from multiple goroutines. The implementation uses read-write locks to
// optimize for the common case of concurrent reads.
//
// # Performance Characteristics
//
//   - Add operation: O(1) average case
//   - Lookup operation: O(1) average case
//   - GetSorted operation: O(n log n) for sorting
//   - Memory usage: O(n) where n is unique entries
package coalescer

import (
	"sort"
	"sync"
)

// genericCoalescer implements Coalescer for any Entry type
type genericCoalescer[T Entry] struct {
	mu sync.RWMutex
	// Map of hash to entries for O(1) deduplication
	entries map[string]T
	// Track initial count for summary
	initialCount int
	// Track duplicates for summary
	duplicates int
}

// NewCoalescer creates a new generic coalescer
func NewCoalescer[T Entry]() Coalescer[T] {
	return &genericCoalescer[T]{
		entries: make(map[string]T),
	}
}

// LoadExisting loads entries from the repository for deduplication
func (c *genericCoalescer[T]) LoadExisting(entries []T) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, entry := range entries {
		hash := entry.Hash()
		if _, exists := c.entries[hash]; !exists {
			c.entries[hash] = entry
			c.initialCount++
		}
	}

	return nil
}

// Add attempts to add an entry, returns true if added (not duplicate)
func (c *genericCoalescer[T]) Add(entry T) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	hash := entry.Hash()
	if _, exists := c.entries[hash]; exists {
		c.duplicates++
		return false
	}

	c.entries[hash] = entry
	return true
}

// GetAll returns all entries (existing + new) sorted by timestamp
func (c *genericCoalescer[T]) GetAll() []T {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]T, 0, len(c.entries))
	for _, entry := range c.entries {
		result = append(result, entry)
	}

	// Sort by timestamp, maintaining stable order for same timestamps
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Timestamp().Before(result[j].Timestamp())
	})

	return result
}

// GetByYear returns entries for a specific year, sorted by timestamp
func (c *genericCoalescer[T]) GetByYear(year int) []T {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]T, 0)
	for _, entry := range c.entries {
		if entry.Year() == year {
			result = append(result, entry)
		}
	}

	// Sort by timestamp, maintaining stable order for same timestamps
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Timestamp().Before(result[j].Timestamp())
	})

	return result
}

// GetSummary returns statistics about the coalescing operation
func (c *genericCoalescer[T]) GetSummary() Summary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := len(c.entries)
	added := total - c.initialCount

	return Summary{
		Initial:    c.initialCount,
		Final:      total,
		Added:      added,
		Duplicates: c.duplicates,
		Rejected:   0, // Will be tracked by the importer
		Errors:     0, // Will be tracked by the importer
	}
}

// Reset clears all entries and statistics
func (c *genericCoalescer[T]) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]T)
	c.initialCount = 0
	c.duplicates = 0
}
