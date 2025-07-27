package calls

import (
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

// xmlCalls represents the root XML element
type xmlCalls struct {
	XMLName xml.Name   `xml:"calls"`
	Count   int        `xml:"count,attr"`
	Calls   []xmlCall  `xml:"call"`
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

// ReadCalls reads all calls from a specific year file
func (r *XMLCallsReader) ReadCalls(year int) ([]Call, error) {
	var calls []Call
	err := r.StreamCalls(year, func(call Call) error {
		calls = append(calls, call)
		return nil
	})
	return calls, err
}

// StreamCalls streams calls from a year file for memory efficiency
func (r *XMLCallsReader) StreamCalls(year int, callback func(Call) error) error {
	filename := r.getCallsFilePath(year)
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open calls file for year %d: %w", year, err)
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	
	// Read the opening calls element
	for {
		tok, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("failed to parse XML: %w", err)
		}
		
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "calls" {
			// We found the calls element, now stream the call elements
			return r.streamCallElements(decoder, callback)
		}
	}
}

// streamCallElements streams individual call elements
func (r *XMLCallsReader) streamCallElements(decoder *xml.Decoder, callback func(Call) error) error {
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to parse XML token: %w", err)
		}

		switch se := tok.(type) {
		case xml.StartElement:
			if se.Name.Local == "call" {
				call, err := r.parseCallElement(se)
				if err != nil {
					return fmt.Errorf("failed to parse call element: %w", err)
				}
				if err := callback(call); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if se.Name.Local == "calls" {
				return nil
			}
		}
	}
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
	date := time.Unix(0, dateMillis*int64(time.Millisecond))

	typeInt, err := strconv.Atoi(xmlCall.Type)
	if err != nil {
		return Call{}, fmt.Errorf("invalid type '%s': %w", xmlCall.Type, err)
	}

	return Call{
		Number:       xmlCall.Number,
		Duration:     duration,
		Date:         date,
		Type:         CallType(typeInt),
		ReadableDate: xmlCall.ReadableDate,
		ContactName:  xmlCall.ContactName,
	}, nil
}

// GetAvailableYears returns list of years with call data
func (r *XMLCallsReader) GetAvailableYears() ([]int, error) {
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
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "calls-") && strings.HasSuffix(name, ".xml") {
			yearStr := strings.TrimPrefix(name, "calls-")
			yearStr = strings.TrimSuffix(yearStr, ".xml")
			year, err := strconv.Atoi(yearStr)
			if err == nil {
				years = append(years, year)
			}
		}
	}

	sort.Ints(years)
	return years, nil
}

// GetCallsCount returns total number of calls for a year
func (r *XMLCallsReader) GetCallsCount(year int) (int, error) {
	filename := r.getCallsFilePath(year)
	file, err := os.Open(filename)
	if err != nil {
		return 0, fmt.Errorf("failed to open calls file for year %d: %w", year, err)
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	
	// Read the opening calls element to get count attribute
	for {
		tok, err := decoder.Token()
		if err != nil {
			return 0, fmt.Errorf("failed to parse XML: %w", err)
		}
		
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "calls" {
			for _, attr := range se.Attr {
				if attr.Name.Local == "count" {
					count, err := strconv.Atoi(attr.Value)
					if err != nil {
						return 0, fmt.Errorf("invalid count attribute '%s': %w", attr.Value, err)
					}
					return count, nil
				}
			}
			return 0, fmt.Errorf("count attribute not found in calls element")
		}
	}
}

// ValidateCallsFile validates XML structure and year consistency
func (r *XMLCallsReader) ValidateCallsFile(year int) error {
	filename := r.getCallsFilePath(year)
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open calls file for year %d: %w", year, err)
	}
	defer file.Close()

	// Parse the entire file to validate structure
	var xmlData xmlCalls
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&xmlData); err != nil {
		return fmt.Errorf("invalid XML structure: %w", err)
	}

	// Validate count matches actual entries
	if xmlData.Count != len(xmlData.Calls) {
		return fmt.Errorf("count mismatch: attribute says %d but found %d calls", xmlData.Count, len(xmlData.Calls))
	}

	// Validate all calls belong to the specified year (based on UTC)
	for i, xmlCall := range xmlData.Calls {
		dateMillis, err := strconv.ParseInt(xmlCall.Date, 10, 64)
		if err != nil {
			return fmt.Errorf("call %d: invalid date '%s': %w", i, xmlCall.Date, err)
		}
		date := time.Unix(0, dateMillis*int64(time.Millisecond)).UTC()
		if date.Year() != year {
			return fmt.Errorf("call %d: date %s belongs to year %d, not %d", i, date.Format(time.RFC3339), date.Year(), year)
		}
	}

	return nil
}

// getCallsFilePath returns the path to the calls file for a given year
func (r *XMLCallsReader) getCallsFilePath(year int) string {
	return filepath.Join(r.repoRoot, "calls", fmt.Sprintf("calls-%d.xml", year))
}