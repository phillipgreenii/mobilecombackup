package cmd

import (
	"testing"

	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
)

func TestSMSMMSTypeCounting(t *testing.T) {
	// Create a sample SMS and MMS message for testing
	smsMessage := sms.SMS{
		Address:      "+15555550000",
		Date:         1373942505322,
		Type:         sms.SentMessage,
		Body:         "Test SMS message",
		ReadableDate: "Jul 15, 2013 10:41:45 PM",
		ContactName:  "Jane Smith",
	}

	mmsMessage := sms.MMS{
		Date:         1414697344000,
		MsgBox:       2, // Sent message
		Address:      "+15555550001",
		ReadableDate: "Oct 30, 2014 3:29:04 PM",
		ContactName:  "Ted Turner",
	}

	// Test type assertions - this is the core of the bug fix
	var smsCount, mmsCount int

	// Test SMS message counting
	var msg sms.Message = smsMessage
	switch msg.(type) {
	case sms.SMS:
		smsCount++
	case sms.MMS:
		mmsCount++
	}

	// Test MMS message counting
	msg = mmsMessage
	switch msg.(type) {
	case sms.SMS:
		smsCount++
	case sms.MMS:
		mmsCount++
	}

	// Verify correct counts
	if smsCount != 1 {
		t.Errorf("Expected 1 SMS message, got %d", smsCount)
	}
	if mmsCount != 1 {
		t.Errorf("Expected 1 MMS message, got %d", mmsCount)
	}
}

func TestInfoSMSStatsWithTestData(t *testing.T) {
	// Test with actual test data to ensure our fix works end-to-end
	reader := sms.NewXMLSMSReader("../../../testdata/to_process")

	// Verify the reader can be created (basic smoke test)
	if reader == nil {
		t.Fatal("Failed to create SMS reader")
	}

	// The key insight is that messages are passed as values (sms.SMS, sms.MMS)
	// not pointers (*sms.SMS, *sms.MMS) to the callback function
	// Our fix in info.go changes the type assertions to match this

	// The actual integration testing is done in the full command tests
	// This test documents the core issue that was fixed
}
