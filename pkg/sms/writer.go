package sms

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// XMLSMSWriter writes SMS/MMS messages to XML files in the repository
type XMLSMSWriter struct {
	repoPath string
}

// NewXMLSMSWriter creates a new XML SMS writer
func NewXMLSMSWriter(repoPath string) (*XMLSMSWriter, error) {
	// Ensure the sms directory exists
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sms directory: %w", err)
	}

	return &XMLSMSWriter{
		repoPath: repoPath,
	}, nil
}

// WriteMessages writes messages to an XML file
func (w *XMLSMSWriter) WriteMessages(filename string, messages []Message) error {
	// Recalculate readable_date for all messages using EST
	loc, _ := time.LoadLocation("America/New_York")
	for _, msg := range messages {
		t := time.Unix(msg.GetDate()/1000, (msg.GetDate()%1000)*int64(time.Millisecond))
		readableDate := t.In(loc).Format("Jan 2, 2006 3:04:05 PM")

		// Set readable_date based on message type
		switch m := msg.(type) {
		case *SMS:
			m.ReadableDate = readableDate
		case *MMS:
			m.ReadableDate = readableDate
		}
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
	// We need to convert messages to their concrete types for XML marshaling
	var smsMessages []*SMS
	var mmsMessages []*MMS

	for _, msg := range messages {
		switch m := msg.(type) {
		case *SMS:
			smsMessages = append(smsMessages, m)
		case SMS:
			smsMessages = append(smsMessages, &m)
		case *MMS:
			mmsMessages = append(mmsMessages, m)
		case MMS:
			mmsMessages = append(mmsMessages, &m)
		}
	}

	// For now, we'll handle SMS and MMS separately
	// In the actual implementation, they're mixed in the same file
	root := struct {
		XMLName xml.Name `xml:"smses"`
		Count   int      `xml:"count,attr"`
		SMS     []*SMS   `xml:"sms"`
		MMS     []*MMS   `xml:"mms"`
	}{
		Count: len(messages),
		SMS:   smsMessages,
		MMS:   mmsMessages,
	}

	// Encode the document
	if err := encoder.Encode(root); err != nil {
		return fmt.Errorf("failed to encode XML: %w", err)
	}

	// Add final newline
	file.WriteString("\n")

	return nil
}
