package security

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// ErrInvalidSizeFormat indicates an invalid size format
	ErrInvalidSizeFormat = errors.New("invalid size format")
	// ErrNegativeSize indicates a negative size value
	ErrNegativeSize = errors.New("size cannot be negative")
)

// SizeUnit represents different size units
type SizeUnit int64

const (
	Byte SizeUnit = 1
	KB   SizeUnit = 1024
	MB   SizeUnit = 1024 * KB
	GB   SizeUnit = 1024 * MB
	TB   SizeUnit = 1024 * GB
)

// Size limits for validation
const (
	MaxReasonableSize = 100 * TB // 100TB maximum reasonable size
)

// ParseSize parses human-readable size strings into bytes
// Supports formats like: "500MB", "500M", "1GB", "1G", "1024", "1024B"
func ParseSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, fmt.Errorf("%w: empty size string", ErrInvalidSizeFormat)
	}

	// Regular expression to parse size string
	// Matches: optional negative sign + number + optional unit (B, K, KB, M, MB, G, GB, T, TB)
	sizeRegex := regexp.MustCompile(`^(-?\d+(?:\.\d+)?)\s*([KMGTB]?B?|[KMGT])?$`)

	// Clean and normalize the input
	normalized := strings.ToUpper(strings.TrimSpace(sizeStr))
	matches := sizeRegex.FindStringSubmatch(normalized)

	if len(matches) != 3 {
		return 0, fmt.Errorf("%w: invalid format %q", ErrInvalidSizeFormat, sizeStr)
	}

	// Parse the numeric part
	numStr := matches[1]
	unitStr := matches[2]

	// Parse number (supports decimal)
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid number %q", ErrInvalidSizeFormat, numStr)
	}

	if num < 0 {
		return 0, fmt.Errorf("%w: %f", ErrNegativeSize, num)
	}

	// Determine multiplier based on unit
	var multiplier SizeUnit

	switch unitStr {
	case "", "B":
		multiplier = Byte
	case "K", "KB":
		multiplier = KB
	case "M", "MB":
		multiplier = MB
	case "G", "GB":
		multiplier = GB
	case "T", "TB":
		multiplier = TB
	default:
		return 0, fmt.Errorf("%w: unknown unit %q", ErrInvalidSizeFormat, unitStr)
	}

	// Calculate final size
	size := int64(num * float64(multiplier))

	// Sanity check for reasonable sizes
	if size > int64(MaxReasonableSize) {
		return 0, fmt.Errorf("%w: size %d exceeds maximum reasonable size %d", ErrInvalidSizeFormat, size, MaxReasonableSize)
	}

	return size, nil
}

// FormatSize formats a size in bytes to a human-readable string
func FormatSize(bytes int64) string {
	if bytes == 0 {
		return "0B"
	}

	units := []struct {
		unit SizeUnit
		name string
	}{
		{TB, "TB"},
		{GB, "GB"},
		{MB, "MB"},
		{KB, "KB"},
		{Byte, "B"},
	}

	for _, u := range units {
		if bytes >= int64(u.unit) {
			value := float64(bytes) / float64(u.unit)
			if value == float64(int64(value)) {
				return fmt.Sprintf("%.0f%s", value, u.name)
			}
			return fmt.Sprintf("%.1f%s", value, u.name)
		}
	}

	return fmt.Sprintf("%dB", bytes)
}

// FileSizeLimitExceededError represents an error when a file size limit is exceeded
type FileSizeLimitExceededError struct {
	Filename  string
	Limit     int64
	Attempted int64
	Context   string
}

func (e *FileSizeLimitExceededError) Error() string {
	if e.Context != "" {
		if e.Attempted > 0 {
			return fmt.Sprintf("%s: file size limit exceeded for %s: limit %s, attempted %s",
				e.Context, e.Filename, FormatSize(e.Limit), FormatSize(e.Attempted))
		}
		return fmt.Sprintf("%s: file size limit exceeded for %s: limit %s",
			e.Context, e.Filename, FormatSize(e.Limit))
	}

	if e.Attempted > 0 {
		return fmt.Sprintf("file size limit exceeded for %s: limit %s, attempted %s",
			e.Filename, FormatSize(e.Limit), FormatSize(e.Attempted))
	}
	return fmt.Sprintf("file size limit exceeded for %s: limit %s",
		e.Filename, FormatSize(e.Limit))
}

// NewFileSizeLimitExceededError creates a new FileSizeLimitExceededError
func NewFileSizeLimitExceededError(filename string, limit, attempted int64, context string) *FileSizeLimitExceededError {
	return &FileSizeLimitExceededError{
		Filename:  filename,
		Limit:     limit,
		Attempted: attempted,
		Context:   context,
	}
}
