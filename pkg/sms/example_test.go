package sms_test

import (
	"fmt"
	"log"

	"github.com/phillipgreen/mobilecombackup/pkg/sms"
)

// Example demonstrates basic usage of the SMSReader
func ExampleXMLSMSReader() {
	// Create a reader for a repository
	reader := sms.NewXMLSMSReader("/path/to/repository")

	// Get available years
	years, err := reader.GetAvailableYears()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Available years: %v\n", years)

	// Read all messages from a specific year
	if len(years) > 0 {
		messages, err := reader.ReadMessages(years[0])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Found %d messages from %d\n", len(messages), years[0])
	}
}

// Example demonstrates streaming messages for memory efficiency
func ExampleXMLSMSReader_StreamMessages() {
	reader := sms.NewXMLSMSReader("/path/to/repository")

	// Stream messages for memory-efficient processing
	messageCount := 0
	err := reader.StreamMessages(2014, func(msg sms.Message) error {
		messageCount++
		fmt.Printf("Message %d: %s -> %s (%d)\n",
			messageCount, msg.GetAddress(), msg.GetContactName(), msg.GetType())
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Processed %d messages\n", messageCount)
}

// Example demonstrates validation of SMS files
func ExampleXMLSMSReader_ValidateSMSFile() {
	reader := sms.NewXMLSMSReader("/path/to/repository")

	// Validate a specific year's SMS file
	err := reader.ValidateSMSFile(2014)
	if err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}
	fmt.Println("SMS file validation passed")
}

// Example demonstrates message type handling
func Example_messageInterface() {
	reader := sms.NewXMLSMSReader("/path/to/repository")

	err := reader.StreamMessages(2014, func(msg sms.Message) error {
		// Handle different message types
		switch m := msg.(type) {
		case sms.SMS:
			fmt.Printf("SMS from %s: %s\n", m.ContactName, m.Body)
		case sms.MMS:
			fmt.Printf("MMS from %s with %d parts\n", m.ContactName, len(m.Parts))
			for _, part := range m.Parts {
				if part.ContentType == "text/plain" {
					fmt.Printf("  Text: %s\n", part.Text)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

// Example demonstrates getting message counts efficiently
func ExampleXMLSMSReader_GetMessageCount() {
	reader := sms.NewXMLSMSReader("/path/to/repository")

	// Get count without loading all messages into memory
	count, err := reader.GetMessageCount(2014)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("2014 has %d messages\n", count)

	// Check multiple years efficiently
	years, _ := reader.GetAvailableYears()
	for _, year := range years {
		count, err := reader.GetMessageCount(year)
		if err != nil {
			continue
		}
		fmt.Printf("Year %d: %d messages\n", year, count)
	}
}

// Example demonstrates MMS part handling
func ExampleMMS_parts() {
	reader := sms.NewXMLSMSReader("/path/to/repository")

	err := reader.StreamMessages(2014, func(msg sms.Message) error {
		if mms, ok := msg.(sms.MMS); ok {
			fmt.Printf("MMS from %s:\n", mms.ContactName)

			for i, part := range mms.Parts {
				fmt.Printf("  Part %d: %s\n", i, part.ContentType)

				switch part.ContentType {
				case "text/plain":
					fmt.Printf("    Text: %s\n", part.Text)
				case "application/smil":
					fmt.Printf("    SMIL presentation data (length: %d)\n", len(part.Text))
				case "image/jpeg", "image/png":
					if part.AttachmentRef != "" {
						fmt.Printf("    Image attachment: %s\n", part.AttachmentRef)
					} else {
						fmt.Printf("    Image data (length: %d)\n", len(part.Data))
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}

// Example demonstrates attachment reference tracking
func ExampleXMLSMSReader_GetAttachmentRefs() {
	reader := sms.NewXMLSMSReader("/path/to/repository")

	// Get attachment references for a specific year
	refs, err := reader.GetAttachmentRefs(2014)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("2014 has %d attachment references\n", len(refs))

	// Get all attachment references across all years
	allRefs, err := reader.GetAllAttachmentRefs()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total attachment references: %d\n", len(allRefs))

	// List unique attachment files
	for ref := range allRefs {
		fmt.Printf("  %s\n", ref)
	}
}

// Example demonstrates message type constants
func ExampleMessageType() {
	// Create a sample SMS message
	smsMsg := sms.SMS{
		Address:     "+15555551234",
		Body:        "Hello World",
		Type:        sms.SentMessage,
		ContactName: "John Doe",
	}

	// Check message type
	switch smsMsg.Type {
	case sms.ReceivedMessage:
		fmt.Println("Received message from", smsMsg.ContactName)
	case sms.SentMessage:
		fmt.Println("Sent message to", smsMsg.ContactName)
	}

	// MMS message type is determined by MsgBox
	mmsMsg := sms.MMS{
		Address:     "+15555551234",
		MsgBox:      1, // 1 = received, 2 = sent
		ContactName: "Jane Smith",
	}

	switch mmsMsg.GetType() {
	case sms.ReceivedMessage:
		fmt.Println("Received MMS from", mmsMsg.ContactName)
	case sms.SentMessage:
		fmt.Println("Sent MMS to", mmsMsg.ContactName)
	}
}
