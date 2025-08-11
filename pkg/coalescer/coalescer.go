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
