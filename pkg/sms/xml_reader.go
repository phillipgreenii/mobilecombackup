package sms

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

// XMLSMSReader implements SMSReader interface for XML files
type XMLSMSReader struct {
	repoRoot string
}

// NewXMLSMSReader creates a new XML SMS reader
func NewXMLSMSReader(repoRoot string) *XMLSMSReader {
	return &XMLSMSReader{
		repoRoot: repoRoot,
	}
}

// getSMSFilePath returns the path to the SMS file for a given year
func (r *XMLSMSReader) getSMSFilePath(year int) string {
	return filepath.Join(r.repoRoot, "sms", fmt.Sprintf("sms-%d.xml", year))
}

// ReadMessages reads all messages from a specific year
func (r *XMLSMSReader) ReadMessages(year int) ([]Message, error) {
	var messages []Message
	err := r.StreamMessagesForYear(year, func(msg Message) error {
		messages = append(messages, msg)
		return nil
	})
	return messages, err
}

// StreamMessagesForYear streams messages for memory efficiency
func (r *XMLSMSReader) StreamMessagesForYear(year int, callback func(Message) error) error {
	filePath := r.getSMSFilePath(year)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open SMS file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	return r.streamMessagesFromReader(file, callback)
}

// streamMessagesFromReader streams messages from an io.Reader
func (r *XMLSMSReader) streamMessagesFromReader(reader io.Reader, callback func(Message) error) error {
	decoder := xml.NewDecoder(reader)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("XML parsing error: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "sms":
				sms, err := r.parseSMSElement(decoder, se)
				if err != nil {
					return fmt.Errorf("failed to parse SMS element: %w", err)
				}
				if err := callback(sms); err != nil {
					return err
				}
			case "mms":
				mms, err := r.parseMMSElement(decoder, se)
				if err != nil {
					return fmt.Errorf("failed to parse MMS element: %w", err)
				}
				if err := callback(mms); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// parseSMSElement parses a single SMS element
func (r *XMLSMSReader) parseSMSElement(decoder *xml.Decoder, startElement xml.StartElement) (SMS, error) {
	sms := SMS{}

	// Parse attributes
	for _, attr := range startElement.Attr {
		switch attr.Name.Local {
		case "protocol":
			sms.Protocol = attr.Value
		case "address":
			sms.Address = attr.Value
		case "date":
			if attr.Value != "" && attr.Value != "null" {
				timestamp, err := strconv.ParseInt(attr.Value, 10, 64)
				if err != nil {
					return sms, fmt.Errorf("invalid date format: %s", attr.Value)
				}
				sms.Date = time.Unix(timestamp/1000, (timestamp%1000)*1000000).UTC()
			}
		case "type":
			if attr.Value != "" && attr.Value != "null" {
				typeVal, err := strconv.Atoi(attr.Value)
				if err != nil {
					return sms, fmt.Errorf("invalid type format: %s", attr.Value)
				}
				sms.Type = MessageType(typeVal)
			}
		case "subject":
			if attr.Value != "null" {
				sms.Subject = attr.Value
			}
		case "body":
			sms.Body = attr.Value
		case "service_center":
			if attr.Value != "null" {
				sms.ServiceCenter = attr.Value
			}
		case "read":
			sms.Read = attr.Value == "1"
		case "status":
			if attr.Value != "" && attr.Value != "null" {
				status, err := strconv.Atoi(attr.Value)
				if err != nil {
					return sms, fmt.Errorf("invalid status format: %s", attr.Value)
				}
				sms.Status = status
			}
		case "locked":
			sms.Locked = attr.Value == "1"
		case "date_sent":
			if attr.Value != "" && attr.Value != "null" && attr.Value != "0" {
				timestamp, err := strconv.ParseInt(attr.Value, 10, 64)
				if err != nil {
					return sms, fmt.Errorf("invalid date_sent format: %s", attr.Value)
				}
				sms.DateSent = time.Unix(timestamp/1000, (timestamp%1000)*1000000).UTC()
			}
		case "readable_date":
			sms.ReadableDate = attr.Value
		case "contact_name":
			sms.ContactName = attr.Value
		}
	}

	// Skip to end element since SMS is self-closing
	if err := decoder.Skip(); err != nil {
		return sms, fmt.Errorf("failed to skip SMS element: %w", err)
	}

	return sms, nil
}

// parseMMSElement parses a single MMS element
func (r *XMLSMSReader) parseMMSElement(decoder *xml.Decoder, startElement xml.StartElement) (MMS, error) {
	mms := MMS{}

	// Parse MMS attributes
	for _, attr := range startElement.Attr {
		switch attr.Name.Local {
		case "date":
			if attr.Value != "" && attr.Value != "null" {
				timestamp, err := strconv.ParseInt(attr.Value, 10, 64)
				if err != nil {
					return mms, fmt.Errorf("invalid date format: %s", attr.Value)
				}
				mms.Date = time.Unix(timestamp/1000, (timestamp%1000)*1000000).UTC()
			}
		case "msg_box":
			if attr.Value != "" && attr.Value != "null" {
				msgBox, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid msg_box format: %s", attr.Value)
				}
				mms.MsgBox = msgBox
			}
		case "address":
			mms.Address = attr.Value
		case "m_type":
			if attr.Value != "" && attr.Value != "null" {
				mType, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid m_type format: %s", attr.Value)
				}
				mms.MType = mType
			}
		case "m_id":
			mms.MId = attr.Value
		case "thread_id":
			mms.ThreadId = attr.Value
		case "text_only":
			mms.TextOnly = attr.Value == "1"
		case "sub":
			if attr.Value != "null" {
				mms.Sub = attr.Value
			}
		case "readable_date":
			mms.ReadableDate = attr.Value
		case "contact_name":
			mms.ContactName = attr.Value
		case "read":
			mms.Read = attr.Value == "1"
		case "locked":
			mms.Locked = attr.Value == "1"
		case "date_sent":
			if attr.Value != "" && attr.Value != "null" && attr.Value != "0" {
				timestamp, err := strconv.ParseInt(attr.Value, 10, 64)
				if err != nil {
					return mms, fmt.Errorf("invalid date_sent format: %s", attr.Value)
				}
				mms.DateSent = time.Unix(timestamp/1000, (timestamp%1000)*1000000).UTC()
			}
		case "seen":
			mms.Seen = attr.Value == "1"
		case "deletable":
			mms.Deletable = attr.Value == "1"
		case "hidden":
			mms.Hidden = attr.Value == "1"
			// Add other MMS attributes as needed
		}
	}

	// Parse MMS content (parts and addresses)
	for {
		token, err := decoder.Token()
		if err != nil {
			return mms, fmt.Errorf("error reading MMS content: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "parts":
				parts, err := r.parseMMSParts(decoder)
				if err != nil {
					return mms, fmt.Errorf("failed to parse MMS parts: %w", err)
				}
				mms.Parts = parts
			case "addrs":
				addresses, err := r.parseMMSAddresses(decoder)
				if err != nil {
					return mms, fmt.Errorf("failed to parse MMS addresses: %w", err)
				}
				mms.Addresses = addresses
			default:
				// Skip unknown elements
				if err := decoder.Skip(); err != nil {
					return mms, fmt.Errorf("failed to skip unknown element %s: %w", se.Name.Local, err)
				}
			}
		case xml.EndElement:
			if se.Name.Local == "mms" {
				return mms, nil
			}
		}
	}
}

// parseMMSParts parses the parts section of an MMS
func (r *XMLSMSReader) parseMMSParts(decoder *xml.Decoder) ([]MMSPart, error) {
	var parts []MMSPart

	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("error reading MMS parts: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "part" {
				part, err := r.parseMMSPart(decoder, se)
				if err != nil {
					return nil, fmt.Errorf("failed to parse MMS part: %w", err)
				}
				parts = append(parts, part)
			} else {
				// Skip unknown elements
				if err := decoder.Skip(); err != nil {
					return nil, fmt.Errorf("failed to skip unknown element %s: %w", se.Name.Local, err)
				}
			}
		case xml.EndElement:
			if se.Name.Local == "parts" {
				return parts, nil
			}
		}
	}
}

// parseMMSPart parses a single MMS part
func (r *XMLSMSReader) parseMMSPart(decoder *xml.Decoder, startElement xml.StartElement) (MMSPart, error) {
	part := MMSPart{}

	// Parse part attributes
	for _, attr := range startElement.Attr {
		switch attr.Name.Local {
		case "seq":
			if attr.Value != "" && attr.Value != "null" {
				seq, err := strconv.Atoi(attr.Value)
				if err != nil {
					return part, fmt.Errorf("invalid seq format: %s", attr.Value)
				}
				part.Seq = seq
			}
		case "ct":
			part.ContentType = attr.Value
		case "name":
			if attr.Value != "null" {
				part.Name = attr.Value
			}
		case "chset":
			if attr.Value != "null" {
				part.Charset = attr.Value
			}
		case "cd":
			if attr.Value != "null" {
				part.ContentDisp = attr.Value
			}
		case "fn":
			if attr.Value != "null" {
				part.Filename = attr.Value
			}
		case "cid":
			if attr.Value != "null" {
				part.ContentId = attr.Value
			}
		case "cl":
			if attr.Value != "null" {
				part.ContentLoc = attr.Value
			}
		case "ctt_s":
			if attr.Value != "null" {
				part.CttS = attr.Value
			}
		case "ctt_t":
			if attr.Value != "null" {
				part.CttT = attr.Value
			}
		case "text":
			part.Text = attr.Value
		case "data":
			// Store the base64 data reference for attachment extraction
			if attr.Value != "null" && attr.Value != "" {
				part.Data = attr.Value
			}
		}
	}

	// Skip to end element since part is self-closing
	if err := decoder.Skip(); err != nil {
		return part, fmt.Errorf("failed to skip part element: %w", err)
	}

	return part, nil
}

// parseMMSAddresses parses the addresses section of an MMS
func (r *XMLSMSReader) parseMMSAddresses(decoder *xml.Decoder) ([]MMSAddress, error) {
	var addresses []MMSAddress

	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("error reading MMS addresses: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "addr" {
				addr, err := r.parseMMSAddress(decoder, se)
				if err != nil {
					return nil, fmt.Errorf("failed to parse MMS address: %w", err)
				}
				addresses = append(addresses, addr)
			} else {
				// Skip unknown elements
				if err := decoder.Skip(); err != nil {
					return nil, fmt.Errorf("failed to skip unknown element %s: %w", se.Name.Local, err)
				}
			}
		case xml.EndElement:
			if se.Name.Local == "addrs" {
				return addresses, nil
			}
		}
	}
}

// parseMMSAddress parses a single MMS address
func (r *XMLSMSReader) parseMMSAddress(decoder *xml.Decoder, startElement xml.StartElement) (MMSAddress, error) {
	addr := MMSAddress{}

	// Parse address attributes
	for _, attr := range startElement.Attr {
		switch attr.Name.Local {
		case "address":
			addr.Address = attr.Value
		case "type":
			if attr.Value != "" && attr.Value != "null" {
				addrType, err := strconv.Atoi(attr.Value)
				if err != nil {
					return addr, fmt.Errorf("invalid address type format: %s", attr.Value)
				}
				addr.Type = addrType
			}
		case "charset":
			if attr.Value != "" && attr.Value != "null" {
				charset, err := strconv.Atoi(attr.Value)
				if err != nil {
					return addr, fmt.Errorf("invalid charset format: %s", attr.Value)
				}
				addr.Charset = charset
			}
		}
	}

	// Skip to end element since addr is self-closing
	if err := decoder.Skip(); err != nil {
		return addr, fmt.Errorf("failed to skip addr element: %w", err)
	}

	return addr, nil
}

// StreamMessages streams all messages from the repository across all years
func (r *XMLSMSReader) StreamMessages(callback func(Message) error) error {
	years, err := r.GetAvailableYears()
	if err != nil {
		return err
	}
	
	// If no years found, return nil (empty repository)
	if len(years) == 0 {
		return nil
	}
	
	for _, year := range years {
		if err := r.StreamMessagesForYear(year, callback); err != nil {
			return err
		}
	}
	
	return nil
}

// GetAvailableYears returns list of years with SMS data
func (r *XMLSMSReader) GetAvailableYears() ([]int, error) {
	smsDir := filepath.Join(r.repoRoot, "sms")
	entries, err := os.ReadDir(smsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []int{}, nil
		}
		return nil, fmt.Errorf("failed to read SMS directory: %w", err)
	}

	var years []int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, "sms-") && strings.HasSuffix(name, ".xml") {
			yearStr := strings.TrimPrefix(name, "sms-")
			yearStr = strings.TrimSuffix(yearStr, ".xml")

			year, err := strconv.Atoi(yearStr)
			if err != nil {
				continue // Skip files with invalid year format
			}
			years = append(years, year)
		}
	}

	sort.Ints(years)
	return years, nil
}

// GetMessageCount returns total number of messages for a year
func (r *XMLSMSReader) GetMessageCount(year int) (int, error) {
	filePath := r.getSMSFilePath(year)
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open SMS file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	decoder := xml.NewDecoder(file)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return 0, fmt.Errorf("no smses element found in file")
		}
		if err != nil {
			return 0, fmt.Errorf("XML parsing error: %w", err)
		}

		if se, ok := token.(xml.StartElement); ok && se.Name.Local == "smses" {
			for _, attr := range se.Attr {
				if attr.Name.Local == "count" {
					count, err := strconv.Atoi(attr.Value)
					if err != nil {
						return 0, fmt.Errorf("invalid count format: %s", attr.Value)
					}
					return count, nil
				}
			}
		}
	}
}

// ValidateSMSFile validates XML structure and year consistency
func (r *XMLSMSReader) ValidateSMSFile(year int) error {
	// Get declared count
	declaredCount, err := r.GetMessageCount(year)
	if err != nil {
		return fmt.Errorf("failed to get declared count: %w", err)
	}

	// Count actual messages
	actualCount := 0
	err = r.StreamMessages(year, func(msg Message) error {
		actualCount++

		// Validate year consistency
		msgYear := msg.GetDate().Year()
		if msgYear != year {
			return fmt.Errorf("message dated %d found in %d file", msgYear, year)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("validation failed during streaming: %w", err)
	}

	// Check count consistency
	if actualCount != declaredCount {
		return fmt.Errorf("count mismatch: declared %d, actual %d", declaredCount, actualCount)
	}

	return nil
}

// GetAttachmentRefs returns all attachment references in a year
func (r *XMLSMSReader) GetAttachmentRefs(year int) ([]string, error) {
	var refs []string
	err := r.StreamMessages(year, func(msg Message) error {
		if mms, ok := msg.(MMS); ok {
			for _, part := range mms.Parts {
				if part.AttachmentRef != "" {
					refs = append(refs, part.AttachmentRef)
				}
			}
		}
		return nil
	})
	return refs, err
}

// GetAllAttachmentRefs returns all attachment references across all years
func (r *XMLSMSReader) GetAllAttachmentRefs() (map[string]bool, error) {
	years, err := r.GetAvailableYears()
	if err != nil {
		return nil, fmt.Errorf("failed to get available years: %w", err)
	}

	allRefs := make(map[string]bool)
	for _, year := range years {
		refs, err := r.GetAttachmentRefs(year)
		if err != nil {
			return nil, fmt.Errorf("failed to get attachment refs for year %d: %w", year, err)
		}
		for _, ref := range refs {
			allRefs[ref] = true
		}
	}

	return allRefs, nil
}
