package contacts

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/security"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
)

// ContactExtractor extracts contact information from backup files
type ContactExtractor struct {
	manager *Manager
}

// NewContactExtractor creates a new contact extractor
func NewContactExtractor(manager *Manager) *ContactExtractor {
	return &ContactExtractor{
		manager: manager,
	}
}

// ExtractFromCallsFile extracts contacts from a calls XML file
func (e *ContactExtractor) ExtractFromCallsFile(filePath string) (int, error) {
	file, err := os.Open(filePath) // #nosec G304
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Parse the XML file using the same structure as the importer
	decoder := security.NewSecureXMLDecoder(file)

	// Find the root element
	var root struct {
		XMLName xml.Name
		Count   string `xml:"count,attr"`
		Calls   []struct {
			XMLName      xml.Name
			Number       string `xml:"number,attr"`
			Duration     int    `xml:"duration,attr"`
			Date         int64  `xml:"date,attr"`
			Type         int    `xml:"type,attr"`
			ReadableDate string `xml:"readable_date,attr"`
			ContactName  string `xml:"contact_name,attr"`
		} `xml:"call"`
	}

	if err := decoder.Decode(&root); err != nil {
		return 0, fmt.Errorf("failed to parse XML: %w", err)
	}

	// Extract contacts from each call
	for _, xmlCall := range root.Calls {
		call := calls.Call{
			Number:       xmlCall.Number,
			Duration:     xmlCall.Duration,
			Date:         xmlCall.Date,
			Type:         calls.CallType(xmlCall.Type),
			ReadableDate: xmlCall.ReadableDate,
			ContactName:  xmlCall.ContactName,
		}
		e.ExtractFromCall(&call)
	}

	return len(root.Calls), nil
}

// ExtractFromSMSFile extracts contacts from an SMS XML file
func (e *ContactExtractor) ExtractFromSMSFile(filePath string) (int, error) {
	// Create an SMS reader to parse the XML file
	reader := sms.NewXMLSMSReader("")

	// Read all messages from the file
	file, err := os.Open(filePath) // #nosec G304
	if err != nil {
		return 0, fmt.Errorf("failed to open SMS file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Count messages processed
	messageCount := 0

	// Stream messages and extract contacts from each
	err = reader.StreamMessagesFromReader(file, func(msg sms.Message) error {
		messageCount++
		switch message := msg.(type) {
		case sms.SMS:
			e.ExtractFromSMS(message)
		case sms.MMS:
			e.ExtractFromMMS(message)
		}
		return nil
	})

	if err != nil {
		return messageCount, fmt.Errorf("failed to stream messages from file: %w", err)
	}

	return messageCount, nil
}

// ExtractFromCall extracts contact information from a call record
func (e *ContactExtractor) ExtractFromCall(call *calls.Call) {
	// Extract contact names if both number and contact name are present
	if call.Number != "" && call.ContactName != "" {
		// Split contact names by comma and process each separately
		contactNames := strings.Split(call.ContactName, ",")
		for _, name := range contactNames {
			name = strings.TrimSpace(name)
			if name != "" && !isUnknownContact(name) {
				e.manager.AddUnprocessedContact(call.Number, name)
			}
		}
	}
}

// ExtractFromSMS extracts contact information from an SMS message
func (e *ContactExtractor) ExtractFromSMS(smsMsg sms.SMS) {
	// Extract primary address contact, splitting multiple contact names
	if smsMsg.Address != "" && smsMsg.ContactName != "" {
		// Split contact names by comma and process each separately
		contactNames := strings.Split(smsMsg.ContactName, ",")
		for _, name := range contactNames {
			name = strings.TrimSpace(name)
			if name != "" && !isUnknownContact(name) {
				e.manager.AddUnprocessedContact(smsMsg.Address, name)
			}
		}
	}
}

// ExtractFromMMS extracts contact information from an MMS message
func (e *ContactExtractor) ExtractFromMMS(mmsMsg sms.MMS) {
	// Extract primary address contact, splitting multiple contact names
	if mmsMsg.Address != "" && mmsMsg.ContactName != "" {
		// Split contact names by comma and process each separately
		contactNames := strings.Split(mmsMsg.ContactName, ",")
		for _, name := range contactNames {
			name = strings.TrimSpace(name)
			if name != "" && !isUnknownContact(name) {
				e.manager.AddUnprocessedContact(mmsMsg.Address, name)
			}
		}
	}
}
