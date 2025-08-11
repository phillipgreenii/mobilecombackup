package calls_test

import (
	"fmt"
	"log"

	"github.com/phillipgreen/mobilecombackup/pkg/calls"
)

// Example demonstrates basic usage of the CallsReader
func ExampleXMLCallsReader() {
	// Create a reader for a repository
	reader := calls.NewXMLCallsReader("/path/to/repository")

	// Get available years
	years, err := reader.GetAvailableYears()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Available years: %v\n", years)

	// Read all calls from a specific year
	if len(years) > 0 {
		callsFromYear, err := reader.ReadCalls(years[0])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Found %d calls from %d\n", len(callsFromYear), years[0])
	}
}

// Example demonstrates streaming calls for memory efficiency
func ExampleXMLCallsReader_StreamCalls() {
	reader := calls.NewXMLCallsReader("/path/to/repository")

	// Stream calls for memory-efficient processing
	callCount := 0
	err := reader.StreamCallsForYear(2014, func(call calls.Call) error {
		callCount++
		fmt.Printf("Call %d: %s -> %s (%d)\n",
			callCount, call.Number, call.ContactName, call.Type)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Processed %d calls\n", callCount)
}

// Example demonstrates validation of call files
func ExampleXMLCallsReader_ValidateCallsFile() {
	reader := calls.NewXMLCallsReader("/path/to/repository")

	// Validate a specific year's call file
	err := reader.ValidateCallsFile(2014)
	if err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}
	fmt.Println("Call file validation passed")
}

// Example demonstrates call type usage
func ExampleCallType() {
	// Create a sample call
	call := calls.Call{
		Number:      "+15555551234",
		Duration:    120,
		Type:        calls.Incoming,
		ContactName: "John Doe",
	}

	// Check call type
	switch call.Type {
	case calls.Incoming:
		fmt.Println("Received call from", call.ContactName)
	case calls.Outgoing:
		fmt.Println("Called", call.ContactName)
	case calls.Missed:
		fmt.Println("Missed call from", call.ContactName)
	case calls.Voicemail:
		fmt.Println("Voicemail from", call.ContactName)
	}
}

// Example demonstrates getting call counts efficiently
func ExampleXMLCallsReader_GetCallsCount() {
	reader := calls.NewXMLCallsReader("/path/to/repository")

	// Get count without loading all calls into memory
	count, err := reader.GetCallsCount(2014)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("2014 has %d calls\n", count)

	// Check multiple years efficiently
	years, _ := reader.GetAvailableYears()
	for _, year := range years {
		count, err := reader.GetCallsCount(year)
		if err != nil {
			continue
		}
		fmt.Printf("Year %d: %d calls\n", year, count)
	}
}
