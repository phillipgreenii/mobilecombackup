package calls

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// XML element names
	callsElementName = "calls"
	callElementName  = "call"
)

// XMLCallsReader implements CallsReader for XML-based repository
type XMLCallsReader struct {
	repoRoot string
}

// NewXMLCallsReader creates a new XML-based calls reader
func NewXMLCallsReader(repoRoot string) *XMLCallsReader {
	return &XMLCallsReader{
		repoRoot: repoRoot,
	}
}

// xmlCall represents a single call in XML format
type xmlCall struct {
	Number       string `xml:"number,attr"`
	Duration     string `xml:"duration,attr"`
	Date         string `xml:"date,attr"`
	Type         string `xml:"type,attr"`
	ReadableDate string `xml:"readable_date,attr"`
	ContactName  string `xml:"contact_name,attr"`
}

// parseCallElement parses a single call element from XML attributes
func (r *XMLCallsReader) parseCallElement(se xml.StartElement) (Call, error) {
	var xmlCall xmlCall
	for _, attr := range se.Attr {
		switch attr.Name.Local {
		case "number":
			xmlCall.Number = attr.Value
		case "duration":
			xmlCall.Duration = attr.Value
		case "date":
			xmlCall.Date = attr.Value
		case "type":
			xmlCall.Type = attr.Value
		case "readable_date":
			xmlCall.ReadableDate = attr.Value
		case "contact_name":
			xmlCall.ContactName = attr.Value
		}
	}

	// Convert to Call struct
	duration, err := strconv.Atoi(xmlCall.Duration)
	if err != nil {
		return Call{}, fmt.Errorf("invalid duration '%s': %w", xmlCall.Duration, err)
	}

	dateMillis, err := strconv.ParseInt(xmlCall.Date, 10, 64)
	if err != nil {
		return Call{}, fmt.Errorf("invalid date '%s': %w", xmlCall.Date, err)
	}

	typeInt, err := strconv.Atoi(xmlCall.Type)
	if err != nil {
		return Call{}, fmt.Errorf("invalid type '%s': %w", xmlCall.Type, err)
	}

	return Call{
		Number:       xmlCall.Number,
		Duration:     duration,
		Date:         dateMillis,
		Type:         CallType(typeInt),
		ReadableDate: xmlCall.ReadableDate,
		ContactName:  xmlCall.ContactName,
	}, nil
}

// StreamCalls streams all calls from the repository across all years
func (r *XMLCallsReader) StreamCalls(callback func(*Call) error) error {
	years, err := r.GetAvailableYears()
	if err != nil {
		return err
	}

	// If no years found, return nil (empty repository)
	if len(years) == 0 {
		return nil
	}

	for _, year := range years {
		if err := r.StreamCallsForYear(year, func(call Call) error {
			return callback(&call)
		}); err != nil {
			return err
		}
	}

	return nil
}

// getCallsFilePath returns the path to the calls file for a given year
func (r *XMLCallsReader) getCallsFilePath(year int) string {
	return filepath.Join(r.repoRoot, "calls", fmt.Sprintf("calls-%d.xml", year))
}

// Context-aware method implementations

// ReadCallsContext reads all calls from a specific year file with context support
func (r *XMLCallsReader) ReadCallsContext(ctx context.Context, year int) ([]Call, error) {
	// Check context before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var calls []Call
	err := r.StreamCallsForYearContext(ctx, year, func(call Call) error {
		calls = append(calls, call)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return calls, nil
}

// StreamCallsForYearContext streams calls from a year file for memory efficiency with context support
func (r *XMLCallsReader) StreamCallsForYearContext(ctx context.Context, year int, callback func(Call) error) error {
	// Check context before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	filePath := r.getCallsFilePath(year)
	file, err := os.Open(filePath) // #nosec G304 - validated path within repository
	if err != nil {
		return fmt.Errorf("failed to open calls file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	decoder := xml.NewDecoder(file)
	return r.streamCallElementsWithContext(ctx, decoder, callback)
}

// streamCallElementsWithContext streams call elements with context support
func (r *XMLCallsReader) streamCallElementsWithContext(
	ctx context.Context, decoder *xml.Decoder, callback func(Call) error,
) error {
	callCount := 0
	const checkInterval = 10 // Check context every 10 calls

	for {
		// Check context periodically
		if callCount%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading XML token: %w", err)
		}

		if se, ok := token.(xml.StartElement); ok && se.Name.Local == callElementName {
			call, err := r.parseCallElement(se)
			if err != nil {
				return fmt.Errorf("error parsing call element: %w", err)
			}

			if err := callback(call); err != nil {
				return err
			}
			callCount++
		}
	}

	return nil
}

// GetAvailableYearsContext returns list of years with call data with context support
func (r *XMLCallsReader) GetAvailableYearsContext(ctx context.Context) ([]int, error) {
	// Check context before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Read the calls directory
	callsDir := filepath.Join(r.repoRoot, "calls")
	entries, err := os.ReadDir(callsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []int{}, nil
		}
		return nil, fmt.Errorf("failed to read calls directory: %w", err)
	}

	var years []int
	for _, entry := range entries {
		// Check context periodically
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, "calls-") && strings.HasSuffix(name, ".xml") {
			yearStr := strings.TrimPrefix(strings.TrimSuffix(name, ".xml"), "calls-")
			if year, err := strconv.Atoi(yearStr); err == nil {
				years = append(years, year)
			}
		}
	}

	// Sort years in ascending order
	sort.Ints(years)
	return years, nil
}

// GetCallsCountContext returns total number of calls for a year with context support
func (r *XMLCallsReader) GetCallsCountContext(ctx context.Context, year int) (int, error) {
	// Check context before starting
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	filePath := r.getCallsFilePath(year)
	file, err := os.Open(filePath) // #nosec G304 - validated path within repository
	if err != nil {
		return 0, fmt.Errorf("failed to open calls file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	decoder := xml.NewDecoder(file)

	for {
		// Check context periodically during XML parsing
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("error reading XML token: %w", err)
		}

		if se, ok := token.(xml.StartElement); ok && se.Name.Local == callsElementName {
			for _, attr := range se.Attr {
				if attr.Name.Local == "count" {
					count, err := strconv.Atoi(attr.Value)
					if err != nil {
						return 0, fmt.Errorf("invalid count attribute '%s': %w", attr.Value, err)
					}
					return count, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("calls element with count attribute not found")
}

// ValidateCallsFileContext validates XML structure and year consistency with context support
func (r *XMLCallsReader) ValidateCallsFileContext(ctx context.Context, year int) error {
	// Check context before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	filePath := r.getCallsFilePath(year)
	file, err := os.Open(filePath) // #nosec G304 - validated path within repository
	if err != nil {
		return fmt.Errorf("failed to open calls file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	decoder := xml.NewDecoder(file)
	var rootFound bool
	var declaredCount int
	callIndex := 0
	const checkInterval = 100 // Check context every 100 calls

	for {
		// Check context periodically
		if callIndex%checkInterval == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("XML parsing error: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == callsElementName && !rootFound {
			rootFound = true
			// Extract count attribute
			count, err := r.extractCountAttribute(se)
			if err != nil {
				return err
			}
			declaredCount = count
		} else if se.Name.Local == callElementName {
			if err := r.validateCallElement(se, year, callIndex); err != nil {
				return err
			}
			callIndex++
		}
	}

	if !rootFound {
		return fmt.Errorf("root element 'calls' not found")
	}

	// Validate count matches
	if declaredCount != callIndex {
		return fmt.Errorf("count mismatch: declared count=%d, actual count=%d", declaredCount, callIndex)
	}

	return nil
}

// Legacy method implementations that delegate to context versions

// ReadCalls delegates to ReadCallsContext with background context
func (r *XMLCallsReader) ReadCalls(year int) ([]Call, error) {
	return r.ReadCallsContext(context.Background(), year)
}

// StreamCallsForYear delegates to StreamCallsForYearContext with background context
func (r *XMLCallsReader) StreamCallsForYear(year int, callback func(Call) error) error {
	return r.StreamCallsForYearContext(context.Background(), year, callback)
}

// GetAvailableYears delegates to GetAvailableYearsContext with background context
func (r *XMLCallsReader) GetAvailableYears() ([]int, error) {
	return r.GetAvailableYearsContext(context.Background())
}

// GetCallsCount delegates to GetCallsCountContext with background context
func (r *XMLCallsReader) GetCallsCount(year int) (int, error) {
	return r.GetCallsCountContext(context.Background(), year)
}

// ValidateCallsFile delegates to ValidateCallsFileContext with background context
func (r *XMLCallsReader) ValidateCallsFile(year int) error {
	return r.ValidateCallsFileContext(context.Background(), year)
}

// extractCountAttribute extracts the count attribute from the calls element
func (r *XMLCallsReader) extractCountAttribute(se xml.StartElement) (int, error) {
	for _, attr := range se.Attr {
		if attr.Name.Local == "count" {
			count, err := strconv.Atoi(attr.Value)
			if err != nil {
				return 0, fmt.Errorf("invalid count attribute '%s': %w", attr.Value, err)
			}
			return count, nil
		}
	}
	return 0, nil
}

// validateCallElement validates a single call element
func (r *XMLCallsReader) validateCallElement(se xml.StartElement, year int, callIndex int) error {
	call, err := r.parseCallElement(se)
	if err != nil {
		return fmt.Errorf("call %d: %w", callIndex, err)
	}

	// Validate that call date matches the expected year
	timestamp := time.Unix(0, call.Date*int64(time.Millisecond)).UTC()
	if timestamp.Year() != year {
		return fmt.Errorf("call %d: date %s belongs to year %d, not %d",
			callIndex, timestamp.Format(time.RFC3339), timestamp.Year(), year)
	}
	return nil
}
