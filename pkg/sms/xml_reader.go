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

	// Parse attributes
	for _, attr := range startElement.Attr {
		switch attr.Name.Local {
		case "protocol":
			sms.Protocol = attr.Value
		case attrAddress:
			sms.Address = attr.Value
		case "date":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				timestamp, err := strconv.ParseInt(attr.Value, 10, 64)
				if err != nil {
					return sms, fmt.Errorf("invalid date format: %s", attr.Value)
				}
				sms.Date = timestamp
			}
		case "type":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				typeVal, err := strconv.Atoi(attr.Value)
				if err != nil {
					return sms, fmt.Errorf("invalid type format: %s", attr.Value)
				}
				sms.Type = MessageType(typeVal)
			}
		case "subject":
			if attr.Value != types.XMLNullValue {
				sms.Subject = attr.Value
			}
		case "body":
			sms.Body = attr.Value
		case "service_center":
			if attr.Value != types.XMLNullValue {
				sms.ServiceCenter = attr.Value
			}
		case "read":
			if attr.Value == "1" {
				sms.Read = 1
			} else {
				sms.Read = 0
			}
		case "status":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				status, err := strconv.Atoi(attr.Value)
				if err != nil {
					return sms, fmt.Errorf("invalid status format: %s", attr.Value)
				}
				sms.Status = status
			}
		case "locked":
			if attr.Value == "1" {
				sms.Locked = 1
			} else {
				sms.Locked = 0
			}
		case "date_sent":
			if attr.Value != "" && attr.Value != types.XMLNullValue && attr.Value != "0" {
				timestamp, err := strconv.ParseInt(attr.Value, 10, 64)
				if err != nil {
					return sms, fmt.Errorf("invalid date_sent format: %s", attr.Value)
				}
				sms.DateSent = timestamp
			}
		case "readable_date":
			sms.ReadableDate = attr.Value
		case "contact_name":
			sms.ContactName = attr.Value
		case "toa":
			if attr.Value != types.XMLNullValue {
				sms.Toa = attr.Value
			}
		case "sc_toa":
			if attr.Value != types.XMLNullValue {
				sms.ScToa = attr.Value
			}
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
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				timestamp, err := strconv.ParseInt(attr.Value, 10, 64)
				if err != nil {
					return mms, fmt.Errorf("invalid date format: %s", attr.Value)
				}
				mms.Date = timestamp
			}
		case "msg_box":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				msgBox, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid msg_box format: %s", attr.Value)
				}
				mms.MsgBox = msgBox
			}
		case attrAddress:
			mms.Address = attr.Value
		case "m_type":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				mType, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid m_type format: %s", attr.Value)
				}
				mms.MType = mType
			}
		case "m_id":
			mms.MId = attr.Value
		case "thread_id":
			mms.ThreadID = attr.Value
		case "text_only":
			if attr.Value == "1" {
				mms.TextOnly = 1
			} else {
				mms.TextOnly = 0
			}
		case "sub":
			if attr.Value != types.XMLNullValue {
				mms.Sub = attr.Value
			}
		case "readable_date":
			mms.ReadableDate = attr.Value
		case "contact_name":
			mms.ContactName = attr.Value
		case "read":
			if attr.Value == "1" {
				mms.Read = 1
			} else {
				mms.Read = 0
			}
		case "locked":
			if attr.Value == "1" {
				mms.Locked = 1
			} else {
				mms.Locked = 0
			}
		case "date_sent":
			if attr.Value != "" && attr.Value != types.XMLNullValue && attr.Value != "0" {
				timestamp, err := strconv.ParseInt(attr.Value, 10, 64)
				if err != nil {
					return mms, fmt.Errorf("invalid date_sent format: %s", attr.Value)
				}
				mms.DateSent = timestamp
			}
		case "seen":
			if attr.Value == "1" {
				mms.Seen = 1
			} else {
				mms.Seen = 0
			}
		case "deletable":
			if attr.Value == "1" {
				mms.Deletable = 1
			} else {
				mms.Deletable = 0
			}
		case "hidden":
			if attr.Value == "1" {
				mms.Hidden = 1
			} else {
				mms.Hidden = 0
			}
		case "sim_imsi":
			if attr.Value != types.XMLNullValue {
				mms.SimImsi = attr.Value
			}
		case "creator":
			if attr.Value != types.XMLNullValue {
				mms.Creator = attr.Value
			}
		case "sub_id":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				subID, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid sub_id format: %s", attr.Value)
				}
				mms.SubID = subID
			}
		case "sim_slot":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				simSlot, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid sim_slot format: %s", attr.Value)
				}
				mms.SimSlot = simSlot
			}
		case "spam_report":
			if attr.Value == "1" {
				mms.SpamReport = 1
			} else {
				mms.SpamReport = 0
			}
		case "safe_message":
			if attr.Value == "1" {
				mms.SafeMessage = 1
			} else {
				mms.SafeMessage = 0
			}
			// Handle other attributes that appear in test data
		case "callback_set":
			if attr.Value == "1" {
				mms.CallbackSet = 1
			} else {
				mms.CallbackSet = 0
			}
		case "retr_st":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				retrSt, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid retr_st format: %s", attr.Value)
				}
				mms.RetrSt = retrSt
			}
		case "ct_cls":
			if attr.Value != types.XMLNullValue {
				mms.CtCls = attr.Value
			}
		case "sub_cs":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				subCs, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid sub_cs format: %s", attr.Value)
				}
				mms.SubCs = subCs
			}
		case "ct_l":
			if attr.Value != types.XMLNullValue {
				mms.CtL = attr.Value
			}
		case "tr_id":
			if attr.Value != types.XMLNullValue {
				mms.TrID = attr.Value
			}
		case "st":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				st, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid st format: %s", attr.Value)
				}
				mms.St = st
			}
		case "m_cls":
			if attr.Value != types.XMLNullValue {
				mms.MCls = attr.Value
			}
		case "d_tm":
			if attr.Value != "" && attr.Value != types.XMLNullValue && attr.Value != "0" {
				timestamp, err := strconv.ParseInt(attr.Value, 10, 64)
				if err != nil {
					return mms, fmt.Errorf("invalid d_tm format: %s", attr.Value)
				}
				mms.DTm = timestamp
			}
		case "read_status":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				readStatus, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid read_status format: %s", attr.Value)
				}
				mms.ReadStatus = readStatus
			}
		case "ct_t":
			if attr.Value != types.XMLNullValue {
				mms.CtT = attr.Value
			}
		case "retr_txt_cs":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				retrTxtCs, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid retr_txt_cs format: %s", attr.Value)
				}
				mms.RetrTxtCs = retrTxtCs
			}
		case "d_rpt":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				drpt, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid d_rpt format: %s", attr.Value)
				}
				mms.DRpt = drpt
			}
		case "reserved":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				reserved, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid reserved format: %s", attr.Value)
				}
				mms.Reserved = reserved
			}
		case "v":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				v, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid v format: %s", attr.Value)
				}
				mms.V = v
			}
		case "exp":
			if attr.Value != "" && attr.Value != types.XMLNullValue && attr.Value != "0" {
				timestamp, err := strconv.ParseInt(attr.Value, 10, 64)
				if err != nil {
					return mms, fmt.Errorf("invalid exp format: %s", attr.Value)
				}
				mms.Exp = timestamp
			}
		case "pri":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				pri, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid pri format: %s", attr.Value)
				}
				mms.Pri = pri
			}
		case "msg_id":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				msgID, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid msg_id format: %s", attr.Value)
				}
				mms.MsgID = msgID
			}
		case "rr":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				rr, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid rr format: %s", attr.Value)
				}
				mms.Rr = rr
			}
		case "app_id":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				appID, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid app_id format: %s", attr.Value)
				}
				mms.AppID = appID
			}
		case "resp_txt":
			if attr.Value != types.XMLNullValue {
				mms.RespTxt = attr.Value
			}
		case "rpt_a":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				rptA, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid rpt_a format: %s", attr.Value)
				}
				mms.RptA = rptA
			}
		case "m_size":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				mSize, err := strconv.Atoi(attr.Value)
				if err != nil {
					return mms, fmt.Errorf("invalid m_size format: %s", attr.Value)
				}
				mms.MSize = mSize
			}
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
	return parseXMLCollection(decoder, "parts", "part", r.parseMMSPart)
}

// parseMMSPart parses a single MMS part
func (r *XMLSMSReader) parseMMSPart(decoder *xml.Decoder, startElement xml.StartElement) (MMSPart, error) {
	part := MMSPart{}

	// Parse part attributes
	for _, attr := range startElement.Attr {
		switch attr.Name.Local {
		case "seq":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				seq, err := strconv.Atoi(attr.Value)
				if err != nil {
					return part, fmt.Errorf("invalid seq format: %s", attr.Value)
				}
				part.Seq = seq
			}
		case "ct":
			part.ContentType = attr.Value
		case "name":
			if attr.Value != types.XMLNullValue {
				part.Name = attr.Value
			}
		case "chset":
			if attr.Value != types.XMLNullValue {
				part.Charset = attr.Value
			}
		case "cd":
			if attr.Value != types.XMLNullValue {
				part.ContentDisp = attr.Value
			}
		case "fn":
			if attr.Value != types.XMLNullValue {
				part.Filename = attr.Value
			}
		case "cid":
			if attr.Value != types.XMLNullValue {
				part.ContentID = attr.Value
			}
		case "cl":
			if attr.Value != types.XMLNullValue {
				part.ContentLoc = attr.Value
			}
		case "ctt_s":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				cttS, err := strconv.Atoi(attr.Value)
				if err != nil {
					return part, fmt.Errorf("invalid ctt_s format: %s", attr.Value)
				}
				part.CttS = cttS
			}
		case "ctt_t":
			if attr.Value != "" && attr.Value != types.XMLNullValue {
				cttT, err := strconv.Atoi(attr.Value)
				if err != nil {
					return part, fmt.Errorf("invalid ctt_t format: %s", attr.Value)
				}
				part.CttT = cttT
			}
		case attrText:
			part.Text = attr.Value
		case "data":
			// Store the base64 data reference for attachment extraction
			if attr.Value != types.XMLNullValue && attr.Value != "" {
				part.Data = attr.Value
			}
		}
	}

	// Parse child elements like AttachmentRef
	for {
		token, err := decoder.Token()
		if err != nil {
			return part, fmt.Errorf("error reading part elements: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "AttachmentRef":
				// Read the text content of the AttachmentRef element
				var attachmentRef string
				if err := decoder.DecodeElement(&attachmentRef, &se); err != nil {
					return part, fmt.Errorf("failed to decode AttachmentRef: %w", err)
				}
				part.AttachmentRef = attachmentRef
			default:
				// Skip unknown elements
				if err := decoder.Skip(); err != nil {
					return part, fmt.Errorf("failed to skip unknown element %s: %w", se.Name.Local, err)
				}
			}
		case xml.EndElement:
			if se.Name.Local == "part" {
				return part, nil
			}
		}
	}
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
