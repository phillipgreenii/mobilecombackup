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

	"github.com/phillipgreen/mobilecombackup/pkg/types"
)

const (
	// XML attribute and element names
	attrAddress = "address"
	attrText    = "text"
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
	file, err := os.Open(filePath) // nolint:gosec // Core functionality requires file reading
	if err != nil {
		return fmt.Errorf("failed to open SMS file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	return r.StreamMessagesFromReader(file, callback)
}

// StreamMessagesFromReader streams messages from an io.Reader
func (r *XMLSMSReader) StreamMessagesFromReader(reader io.Reader, callback func(Message) error) error {
	decoder := xml.NewDecoder(reader)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("XML parsing error: %w", err)
		}

		if se, ok := token.(xml.StartElement); ok {
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

	// Parse SMS attributes
	if err := r.parseSMSAttributes(&sms, startElement.Attr); err != nil {
		return sms, err
	}

	// Skip to end element since SMS is self-closing
	if err := decoder.Skip(); err != nil {
		return sms, fmt.Errorf("failed to skip SMS element: %w", err)
	}

	return sms, nil
}

// parseSMSAttributes parses all SMS attributes from XML attributes
func (r *XMLSMSReader) parseSMSAttributes(sms *SMS, attrs []xml.Attr) error {
	for _, attr := range attrs {
		if err := r.parseSMSAttributeByCategory(sms, attr); err != nil {
			return err
		}
	}
	return nil
}

// parseSMSAttributeByCategory routes SMS attribute parsing to specialized handlers
func (r *XMLSMSReader) parseSMSAttributeByCategory(sms *SMS, attr xml.Attr) error {
	name := attr.Name.Local
	value := attr.Value

	// Core message attributes
	if err := r.parseSMSCoreAttributes(sms, name, value); err != nil {
		return err
	}

	// Timestamp attributes
	if err := r.parseSMSTimestampAttributes(sms, name, value); err != nil {
		return err
	}

	// Status and flag attributes
	r.parseSMSStatusAttributes(sms, name, value)

	// Optional string attributes
	r.parseSMSOptionalStringAttributes(sms, name, value)

	return nil
}

// parseSMSCoreAttributes handles core message identification and content
func (r *XMLSMSReader) parseSMSCoreAttributes(sms *SMS, name, value string) error {
	switch name {
	case "protocol":
		sms.Protocol = value
	case attrAddress:
		sms.Address = value
	case "body":
		sms.Body = value
	case "readable_date":
		sms.ReadableDate = value
	case "contact_name":
		sms.ContactName = value
	case "type":
		if value != "" && value != types.XMLNullValue {
			typeVal, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid type format: %s", value)
			}
			sms.Type = MessageType(typeVal)
		}
	case "status":
		if value != "" && value != types.XMLNullValue {
			status, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid status format: %s", value)
			}
			sms.Status = status
		}
	}
	return nil
}

// parseSMSTimestampAttributes handles timestamp parsing with validation
func (r *XMLSMSReader) parseSMSTimestampAttributes(sms *SMS, name, value string) error {
	switch name {
	case "date":
		if value != "" && value != types.XMLNullValue {
			timestamp, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid date format: %s", value)
			}
			sms.Date = timestamp
		}
	case "date_sent":
		if value != "" && value != types.XMLNullValue && value != "0" {
			timestamp, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid date_sent format: %s", value)
			}
			sms.DateSent = timestamp
		}
	}
	return nil
}

// parseSMSStatusAttributes handles boolean status flags
func (r *XMLSMSReader) parseSMSStatusAttributes(sms *SMS, name, value string) {
	switch name {
	case "read":
		if value == "1" {
			sms.Read = 1
		} else {
			sms.Read = 0
		}
	case "locked":
		if value == "1" {
			sms.Locked = 1
		} else {
			sms.Locked = 0
		}
	}
}

// parseSMSOptionalStringAttributes handles optional string fields with null checking
func (r *XMLSMSReader) parseSMSOptionalStringAttributes(sms *SMS, name, value string) {
	if value == types.XMLNullValue {
		return
	}

	switch name {
	case "subject":
		sms.Subject = value
	case "service_center":
		sms.ServiceCenter = value
	case "toa":
		sms.Toa = value
	case "sc_toa":
		sms.ScToa = value
	}
}

// parseMMSElement parses a single MMS element
func (r *XMLSMSReader) parseMMSElement(decoder *xml.Decoder, startElement xml.StartElement) (MMS, error) {
	mms := MMS{}

	// Parse MMS attributes
	if err := r.parseMMSAttributes(&mms, startElement.Attr); err != nil {
		return mms, err
	}

	// Parse MMS content (parts and addresses)
	return r.parseMMSContent(decoder, &mms)
}

// parseMMSAttributes parses all MMS attributes from XML attributes
func (r *XMLSMSReader) parseMMSAttributes(mms *MMS, attrs []xml.Attr) error {
	for _, attr := range attrs {
		if err := r.parseMMSAttribute(mms, attr); err != nil {
			return err
		}
	}
	return nil
}

// parseMMSAttribute parses a single MMS attribute
func (r *XMLSMSReader) parseMMSAttribute(mms *MMS, attr xml.Attr) error {
	return r.parseMMSAttributeByCategory(mms, attr)
}

// parseMMSAttributeByCategory routes attribute parsing to specialized handlers
func (r *XMLSMSReader) parseMMSAttributeByCategory(mms *MMS, attr xml.Attr) error {
	name := attr.Name.Local
	value := attr.Value

	// Core message attributes
	if err := r.parseMMSCoreAttributes(mms, name, value); err != nil {
		return err
	}

	// Status and flag attributes
	if r.parseMMSStatusAttributes(mms, name, value) {
		return nil
	}

	// Network and protocol attributes
	if err := r.parseMMSNetworkAttributes(mms, name, value); err != nil {
		return err
	}

	// Optional string attributes
	r.parseMMSOptionalStringAttributes(mms, name, value)

	return nil
}

// parseMMSCoreAttributes handles core message identification and timing
func (r *XMLSMSReader) parseMMSCoreAttributes(mms *MMS, name, value string) error {
	switch name {
	case "date":
		return r.parseTimestampAttr(value, &mms.Date, "date")
	case "date_sent":
		return r.parseTimestampAttrWithZero(value, &mms.DateSent, "date_sent")
	case "d_tm":
		return r.parseTimestampAttrWithZero(value, &mms.DTm, "d_tm")
	case "exp":
		return r.parseTimestampAttrWithZero(value, &mms.Exp, "exp")
	case "msg_box":
		return r.parseIntAttr(value, &mms.MsgBox, "msg_box")
	case "m_type":
		return r.parseIntAttr(value, &mms.MType, "m_type")
	case attrAddress:
		mms.Address = value
	case "m_id":
		mms.MId = value
	case "thread_id":
		mms.ThreadID = value
	case "readable_date":
		mms.ReadableDate = value
	case "contact_name":
		mms.ContactName = value
	}
	return nil
}

// parseMMSStatusAttributes handles boolean status flags and returns true if handled
func (r *XMLSMSReader) parseMMSStatusAttributes(mms *MMS, name, value string) bool {
	switch name {
	case "text_only":
		mms.TextOnly = r.parseBooleanAttr(value)
	case "read":
		mms.Read = r.parseBooleanAttr(value)
	case "locked":
		mms.Locked = r.parseBooleanAttr(value)
	case "seen":
		mms.Seen = r.parseBooleanAttr(value)
	case "deletable":
		mms.Deletable = r.parseBooleanAttr(value)
	case "hidden":
		mms.Hidden = r.parseBooleanAttr(value)
	case "spam_report":
		mms.SpamReport = r.parseBooleanAttr(value)
	case "safe_message":
		mms.SafeMessage = r.parseBooleanAttr(value)
	case "callback_set":
		mms.CallbackSet = r.parseBooleanAttr(value)
	default:
		return false
	}
	return true
}

// parseMMSNetworkAttributes handles network and protocol-related integer fields
func (r *XMLSMSReader) parseMMSNetworkAttributes(mms *MMS, name, value string) error {
	switch name {
	case "sub_id":
		return r.parseIntAttr(value, &mms.SubID, "sub_id")
	case "sim_slot":
		return r.parseIntAttr(value, &mms.SimSlot, "sim_slot")
	case "retr_st":
		return r.parseIntAttr(value, &mms.RetrSt, "retr_st")
	case "sub_cs":
		return r.parseIntAttr(value, &mms.SubCs, "sub_cs")
	case "st":
		return r.parseIntAttr(value, &mms.St, "st")
	case "read_status":
		return r.parseIntAttr(value, &mms.ReadStatus, "read_status")
	case "retr_txt_cs":
		return r.parseIntAttr(value, &mms.RetrTxtCs, "retr_txt_cs")
	case "d_rpt":
		return r.parseIntAttr(value, &mms.DRpt, "d_rpt")
	case "reserved":
		return r.parseIntAttr(value, &mms.Reserved, "reserved")
	case "v":
		return r.parseIntAttr(value, &mms.V, "v")
	case "pri":
		return r.parseIntAttr(value, &mms.Pri, "pri")
	case "msg_id":
		return r.parseIntAttr(value, &mms.MsgID, "msg_id")
	case "rr":
		return r.parseIntAttr(value, &mms.Rr, "rr")
	case "app_id":
		return r.parseIntAttr(value, &mms.AppID, "app_id")
	case "rpt_a":
		return r.parseIntAttr(value, &mms.RptA, "rpt_a")
	case "m_size":
		return r.parseIntAttr(value, &mms.MSize, "m_size")
	}
	return nil
}

// parseMMSOptionalStringAttributes handles optional string fields with null checking
func (r *XMLSMSReader) parseMMSOptionalStringAttributes(mms *MMS, name, value string) {
	if value == types.XMLNullValue {
		return
	}

	switch name {
	case "sub":
		mms.Sub = value
	case "sim_imsi":
		mms.SimImsi = value
	case "creator":
		mms.Creator = value
	case "ct_cls":
		mms.CtCls = value
	case "ct_l":
		mms.CtL = value
	case "tr_id":
		mms.TrID = value
	case "m_cls":
		mms.MCls = value
	case "ct_t":
		mms.CtT = value
	case "resp_txt":
		mms.RespTxt = value
	}
}

// parseMMSContent parses MMS child elements (parts and addresses)
func (r *XMLSMSReader) parseMMSContent(decoder *xml.Decoder, mms *MMS) (MMS, error) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return *mms, fmt.Errorf("error reading MMS content: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "parts":
				parts, err := r.parseMMSParts(decoder)
				if err != nil {
					return *mms, fmt.Errorf("failed to parse MMS parts: %w", err)
				}
				mms.Parts = parts
			case "addrs":
				addresses, err := r.parseMMSAddresses(decoder)
				if err != nil {
					return *mms, fmt.Errorf("failed to parse MMS addresses: %w", err)
				}
				mms.Addresses = addresses
			default:
				// Skip unknown elements
				if err := decoder.Skip(); err != nil {
					return *mms, fmt.Errorf("failed to skip unknown element %s: %w", se.Name.Local, err)
				}
			}
		case xml.EndElement:
			if se.Name.Local == "mms" {
				return *mms, nil
			}
		}
	}
}

// Helper functions for parsing different attribute types

// parseTimestampAttr parses a timestamp attribute
func (r *XMLSMSReader) parseTimestampAttr(value string, target *int64, attrName string) error {
	if value != "" && value != types.XMLNullValue {
		timestamp, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid %s format: %s", attrName, value)
		}
		*target = timestamp
	}
	return nil
}

// parseTimestampAttrWithZero parses a timestamp attribute, ignoring "0" values
func (r *XMLSMSReader) parseTimestampAttrWithZero(value string, target *int64, attrName string) error {
	if value != "" && value != types.XMLNullValue && value != "0" {
		timestamp, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid %s format: %s", attrName, value)
		}
		*target = timestamp
	}
	return nil
}

// parseIntAttr parses an integer attribute
func (r *XMLSMSReader) parseIntAttr(value string, target *int, attrName string) error {
	if value != "" && value != types.XMLNullValue {
		intVal, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid %s format: %s", attrName, value)
		}
		*target = intVal
	}
	return nil
}

// parseBooleanAttr parses a boolean attribute ("1" = 1, others = 0)
func (r *XMLSMSReader) parseBooleanAttr(value string) int {
	if value == "1" {
		return 1
	}
	return 0
}

// parseMMSParts parses the parts section of an MMS
func (r *XMLSMSReader) parseMMSParts(decoder *xml.Decoder) ([]MMSPart, error) {
	return parseXMLCollection(decoder, "parts", "part", r.parseMMSPart)
}

// parseMMSPart parses a single MMS part
func (r *XMLSMSReader) parseMMSPart(decoder *xml.Decoder, startElement xml.StartElement) (MMSPart, error) {
	part := MMSPart{}

	// Parse part attributes
	if err := r.parseMMSPartAttributes(&part, startElement.Attr); err != nil {
		return part, err
	}

	// Parse child elements
	return r.parseMMSPartContent(decoder, &part)
}

// parseMMSPartAttributes parses all MMS part attributes
func (r *XMLSMSReader) parseMMSPartAttributes(part *MMSPart, attrs []xml.Attr) error {
	for _, attr := range attrs {
		if err := r.parseMMSPartAttributeByCategory(part, attr); err != nil {
			return err
		}
	}
	return nil
}

// parseMMSPartAttributeByCategory routes part attribute parsing to specialized handlers
func (r *XMLSMSReader) parseMMSPartAttributeByCategory(part *MMSPart, attr xml.Attr) error {
	name := attr.Name.Local
	value := attr.Value

	// Core content attributes
	if err := r.parseMMSPartCoreAttributes(part, name, value); err != nil {
		return err
	}

	// Content metadata attributes
	r.parseMMSPartContentAttributes(part, name, value)

	// Numeric attributes
	if err := r.parseMMSPartNumericAttributes(part, name, value); err != nil {
		return err
	}

	return nil
}

// parseMMSPartCoreAttributes handles sequence, content type, and data
func (r *XMLSMSReader) parseMMSPartCoreAttributes(part *MMSPart, name, value string) error {
	switch name {
	case "seq":
		if value != "" && value != types.XMLNullValue {
			seq, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid seq format: %s", value)
			}
			part.Seq = seq
		}
	case "ct":
		part.ContentType = value
	case attrText:
		part.Text = value
	case "data":
		// Store the base64 data reference for attachment extraction
		if value != types.XMLNullValue && value != "" {
			part.Data = value
		}
	}
	return nil
}

// parseMMSPartContentAttributes handles content metadata with null checking
func (r *XMLSMSReader) parseMMSPartContentAttributes(part *MMSPart, name, value string) {
	if value == types.XMLNullValue {
		return
	}

	switch name {
	case "name":
		part.Name = value
	case "chset":
		part.Charset = value
	case "cd":
		part.ContentDisp = value
	case "fn":
		part.Filename = value
	case "cid":
		part.ContentID = value
	case "cl":
		part.ContentLoc = value
	}
}

// parseMMSPartNumericAttributes handles numeric content attributes
func (r *XMLSMSReader) parseMMSPartNumericAttributes(part *MMSPart, name, value string) error {
	if value == "" || value == types.XMLNullValue {
		return nil
	}

	switch name {
	case "ctt_s":
		cttS, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid ctt_s format: %s", value)
		}
		part.CttS = cttS
	case "ctt_t":
		cttT, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid ctt_t format: %s", value)
		}
		part.CttT = cttT
	}
	return nil
}

// parseMMSPartContent parses child elements of an MMS part
func (r *XMLSMSReader) parseMMSPartContent(decoder *xml.Decoder, part *MMSPart) (MMSPart, error) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return *part, fmt.Errorf("error reading part elements: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			if err := r.parseMMSPartChildElement(decoder, part, se); err != nil {
				return *part, err
			}
		case xml.EndElement:
			if se.Name.Local == "part" {
				return *part, nil
			}
		}
	}
}

// parseMMSPartChildElement handles individual child elements
func (r *XMLSMSReader) parseMMSPartChildElement(decoder *xml.Decoder, part *MMSPart, startElement xml.StartElement) error {
	switch startElement.Name.Local {
	case "AttachmentRef":
		// Read the text content of the AttachmentRef element
		var attachmentRef string
		if err := decoder.DecodeElement(&attachmentRef, &startElement); err != nil {
			return fmt.Errorf("failed to decode AttachmentRef: %w", err)
		}
		part.AttachmentRef = attachmentRef
	default:
		// Skip unknown elements
		if err := decoder.Skip(); err != nil {
			return fmt.Errorf("failed to skip unknown element %s: %w", startElement.Name.Local, err)
		}
	}
	return nil
}

// parseMMSAddresses parses the addresses section of an MMS
func (r *XMLSMSReader) parseMMSAddresses(decoder *xml.Decoder) ([]MMSAddress, error) {
	return parseXMLCollection(decoder, "addrs", "addr", r.parseMMSAddress)
}

// parseMMSAddress parses a single MMS address
func (r *XMLSMSReader) parseMMSAddress(decoder *xml.Decoder, startElement xml.StartElement) (MMSAddress, error) {
	addr := MMSAddress{}

	// Parse address attributes
	for _, attr := range startElement.Attr {
		switch attr.Name.Local {
		case attrAddress:
			addr.Address = attr.Value
		case "type":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				addrType, err := strconv.Atoi(attr.Value)
				if err != nil {
					return addr, fmt.Errorf("invalid address type format: %s", attr.Value)
				}
				addr.Type = addrType
			}
		case "charset":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
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
	file, err := os.Open(filePath) // nolint:gosec // Core functionality requires file reading
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
	err = r.StreamMessagesForYear(year, func(msg Message) error {
		actualCount++

		// Validate year consistency
		timestamp := msg.GetDate()
		msgTime := time.Unix(timestamp/1000, (timestamp%1000)*int64(time.Millisecond))
		msgYear := msgTime.Year()
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
	err := r.StreamMessagesForYear(year, func(msg Message) error {
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

// extractHashFromAttachmentPath extracts the hash from an attachment reference path
// Expected format: attachments/xx/hash/filename or attachments/xx/hash
// Returns the hash or empty string if path is invalid
func extractHashFromAttachmentPath(attachmentPath string) string {
	if attachmentPath == "" {
		return ""
	}

	// Split path by separators
	parts := strings.Split(attachmentPath, "/")

	// Expected format: [attachments, prefix, hash, filename] or [attachments, prefix, hash]
	if len(parts) < 3 || parts[0] != "attachments" {
		return ""
	}

	// Extract potential hash (should be 64-char lowercase hex)
	potentialHash := parts[2]
	if len(potentialHash) != 64 {
		return ""
	}

	// Validate it's a proper hex string
	for _, char := range potentialHash {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return ""
		}
	}

	return potentialHash
}

// GetAllAttachmentRefs returns all attachment references across all years as a map of hashes
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
			// Extract hash from attachment reference path
			hash := extractHashFromAttachmentPath(ref)
			if hash != "" {
				allRefs[hash] = true
			}
		}
	}

	return allRefs, nil
}

// parseXMLCollection parses a collection of XML elements with a common pattern
func parseXMLCollection[T any](
	decoder *xml.Decoder,
	containerName, elementName string,
	parseFunc func(*xml.Decoder, xml.StartElement) (T, error),
) ([]T, error) {
	var collection []T

	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("error reading XML collection: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == elementName {
				item, err := parseFunc(decoder, se)
				if err != nil {
					return nil, fmt.Errorf("failed to parse %s: %w", elementName, err)
				}
				collection = append(collection, item)
			} else {
				// Skip unknown elements
				if err := decoder.Skip(); err != nil {
					return nil, fmt.Errorf("failed to skip unknown element %s: %w", se.Name.Local, err)
				}
			}
		case xml.EndElement:
			if se.Name.Local == containerName {
				return collection, nil
			}
		}
	}
}
