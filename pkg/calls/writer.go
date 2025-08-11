package calls

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// XMLCallsWriter writes calls to XML files in the repository
type XMLCallsWriter struct {
	repoPath string
}

// NewXMLCallsWriter creates a new XML calls writer
func NewXMLCallsWriter(repoPath string) (*XMLCallsWriter, error) {
	// Ensure the calls directory exists
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create calls directory: %w", err)
	}

	return &XMLCallsWriter{
		repoPath: repoPath,
	}, nil
}

// WriteCalls writes calls to an XML file
func (w *XMLCallsWriter) WriteCalls(filename string, calls []*Call) error {
	// Recalculate readable_date for all calls using EST
	loc, _ := time.LoadLocation("America/New_York")
	for _, call := range calls {
		t := time.Unix(call.Date/1000, (call.Date%1000)*int64(time.Millisecond))
		call.ReadableDate = t.In(loc).Format("Jan 2, 2006 3:04:05 PM")
	}

	// Create the file
	filepath := filepath.Join(w.repoPath, filename)
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write XML header
	file.WriteString(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>` + "\n")

	// Create encoder with proper formatting
	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")

	// Create root element
	root := struct {
		XMLName xml.Name `xml:"calls"`
		Count   int      `xml:"count,attr"`
		Calls   []*Call  `xml:"call"`
	}{
		Count: len(calls),
		Calls: calls,
	}

	// Encode the document
	if err := encoder.Encode(root); err != nil {
		return fmt.Errorf("failed to encode XML: %w", err)
	}

	// Add final newline
	file.WriteString("\n")

	return nil
}
