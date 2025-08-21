// Package calls provides functionality for processing call log data from mobile phone backups.
// It includes features for reading, coalescing, and validating call entries from XML format.
package calls

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/coalescer"
)

// CallEntry wraps a Call to implement the coalescer.Entry interface
type CallEntry struct {
	*Call
}

// Hash returns a unique identifier for deduplication
// Excludes readable_date and contact_name fields as per requirement
func (c CallEntry) Hash() string {
	h := sha256.New()

	// Include all fields except readable_date and contact_name
	_, _ = fmt.Fprintf(h, "number:%s|", c.Number)
	_, _ = fmt.Fprintf(h, "duration:%d|", c.Duration)
	_, _ = fmt.Fprintf(h, "date:%d|", c.Date)
	_, _ = fmt.Fprintf(h, "type:%d|", c.Type)

	return fmt.Sprintf("%x", h.Sum(nil))
}

// Timestamp returns the entry's timestamp for sorting
func (c CallEntry) Timestamp() time.Time {
	return c.Call.Timestamp()
}

// Year returns the year for partitioning
func (c CallEntry) Year() int {
	return c.Timestamp().UTC().Year()
}

// NewCallCoalescer creates a new coalescer for calls
func NewCallCoalescer() coalescer.Coalescer[CallEntry] {
	return coalescer.NewCoalescer[CallEntry]()
}
