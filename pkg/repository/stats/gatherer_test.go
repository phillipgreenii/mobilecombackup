package stats

import (
	"context"
	"testing"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/contacts"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
)

// Mock readers for testing

type mockCallsReader struct {
	years       []int
	counts      map[int]int
	callsByYear map[int][]calls.Call
	err         error
}

func (m *mockCallsReader) GetAvailableYears(_ context.Context) ([]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.years, nil
}

func (m *mockCallsReader) GetCallsCount(_ context.Context, year int) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.counts[year], nil
}

func (m *mockCallsReader) StreamCallsForYear(_ context.Context, year int, callback func(calls.Call) error) error {
	if m.err != nil {
		return m.err
	}
	for _, call := range m.callsByYear[year] {
		if err := callback(call); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockCallsReader) ReadCalls(_ context.Context, _ int) ([]calls.Call, error) {
	return nil, nil
}

func (m *mockCallsReader) ValidateCallsFile(_ context.Context, _ int) error {
	return nil
}

type mockSMSReader struct {
	years          []int
	counts         map[int]int
	messagesByYear map[int][]sms.Message
	attachmentRefs map[string]bool
	err            error
}

func (m *mockSMSReader) GetAvailableYears(_ context.Context) ([]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.years, nil
}

func (m *mockSMSReader) GetMessageCount(_ context.Context, year int) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.counts[year], nil
}

func (m *mockSMSReader) StreamMessagesForYear(_ context.Context, year int, callback func(sms.Message) error) error {
	if m.err != nil {
		return m.err
	}
	for _, msg := range m.messagesByYear[year] {
		if err := callback(msg); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockSMSReader) GetAllAttachmentRefs(_ context.Context) (map[string]bool, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.attachmentRefs, nil
}

func (m *mockSMSReader) ReadMessages(_ context.Context, _ int) ([]sms.Message, error) {
	return nil, nil
}

func (m *mockSMSReader) ValidateSMSFile(_ context.Context, _ int) error {
	return nil
}

func (m *mockSMSReader) GetAttachmentRefs(_ context.Context, _ int) ([]string, error) {
	return nil, nil
}

type mockAttachmentReader struct {
	attachments         []*attachments.Attachment
	orphanedAttachments []*attachments.Attachment
	err                 error
}

func (m *mockAttachmentReader) ListAttachments() ([]*attachments.Attachment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.attachments, nil
}

func (m *mockAttachmentReader) FindOrphanedAttachments(_ map[string]bool) ([]*attachments.Attachment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.orphanedAttachments, nil
}

func (m *mockAttachmentReader) GetAttachment(_ string) (*attachments.Attachment, error) {
	return nil, nil
}

func (m *mockAttachmentReader) ReadAttachment(_ string) ([]byte, error) {
	return nil, nil
}

func (m *mockAttachmentReader) AttachmentExists(_ string) (bool, error) {
	return false, nil
}

func (m *mockAttachmentReader) StreamAttachments(_ func(*attachments.Attachment) error) error {
	return nil
}

func (m *mockAttachmentReader) VerifyAttachment(_ string) (bool, error) {
	return false, nil
}

func (m *mockAttachmentReader) GetAttachmentPath(_ string) string {
	return ""
}

func (m *mockAttachmentReader) ValidateAttachmentStructure() error {
	return nil
}

// Context-aware methods (delegate to legacy methods)

func (m *mockAttachmentReader) GetAttachmentContext(_ context.Context, hash string) (*attachments.Attachment, error) {
	return m.GetAttachment(hash)
}

func (m *mockAttachmentReader) ReadAttachmentContext(_ context.Context, hash string) ([]byte, error) {
	return m.ReadAttachment(hash)
}

func (m *mockAttachmentReader) AttachmentExistsContext(_ context.Context, hash string) (bool, error) {
	return m.AttachmentExists(hash)
}

func (m *mockAttachmentReader) ListAttachmentsContext(_ context.Context) ([]*attachments.Attachment, error) {
	return m.ListAttachments()
}

func (m *mockAttachmentReader) StreamAttachmentsContext(_ context.Context, callback func(*attachments.Attachment) error) error {
	return m.StreamAttachments(callback)
}

func (m *mockAttachmentReader) VerifyAttachmentContext(_ context.Context, hash string) (bool, error) {
	return m.VerifyAttachment(hash)
}

func (m *mockAttachmentReader) FindOrphanedAttachmentsContext(_ context.Context, referencedHashes map[string]bool) ([]*attachments.Attachment, error) {
	return m.FindOrphanedAttachments(referencedHashes)
}

func (m *mockAttachmentReader) ValidateAttachmentStructureContext(_ context.Context) error {
	return m.ValidateAttachmentStructure()
}

type mockContactsReader struct {
	contactsList []*contacts.Contact
	count        int
	err          error
}

func (m *mockContactsReader) LoadContacts(_ context.Context) error {
	return m.err
}

func (m *mockContactsReader) GetContactsCount() int {
	return m.count
}

func (m *mockContactsReader) GetAllContacts(_ context.Context) ([]*contacts.Contact, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.contactsList, nil
}

func (m *mockContactsReader) GetContactByNumber(_ string) (string, bool) {
	return "", false
}

func (m *mockContactsReader) GetNumbersByContact(_ string) ([]string, bool) {
	return nil, false
}

func (m *mockContactsReader) ContactExists(_ string) bool {
	return false
}

func (m *mockContactsReader) IsKnownNumber(_ string) bool {
	return false
}

func (m *mockContactsReader) AddUnprocessedContacts(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockContactsReader) GetUnprocessedEntries() []contacts.UnprocessedEntry {
	return nil
}

func TestStatsGatherer_GatherCallsStats(t *testing.T) {
	t.Parallel()

	t.Run("success_with_multiple_years", func(t *testing.T) {
		mockReader := &mockCallsReader{
			years: []int{2014, 2015},
			counts: map[int]int{
				2014: 10,
				2015: 15,
			},
			callsByYear: map[int][]calls.Call{
				2014: {
					{Date: 1410881505425}, // Sep 16, 2014
					{Date: 1420000000000}, // Dec 31, 2014
				},
				2015: {
					{Date: 1420070400000}, // Jan 1, 2015
					{Date: 1451606400000}, // Dec 31, 2015
				},
			},
		}

		gatherer := NewStatsGatherer("/test", mockReader, nil, nil, nil)
		stats, err := gatherer.GatherCallsStats(context.Background())

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(stats) != 2 {
			t.Errorf("Expected 2 years, got %d", len(stats))
		}

		if stats["2014"].Count != 10 {
			t.Errorf("Expected count 10 for 2014, got %d", stats["2014"].Count)
		}

		if stats["2015"].Count != 15 {
			t.Errorf("Expected count 15 for 2015, got %d", stats["2015"].Count)
		}

		// Verify date ranges are set
		if stats["2014"].Earliest.IsZero() {
			t.Error("Expected earliest date to be set for 2014")
		}
		if stats["2014"].Latest.IsZero() {
			t.Error("Expected latest date to be set for 2014")
		}
	})

	t.Run("empty_repository", func(t *testing.T) {
		mockReader := &mockCallsReader{
			years:  []int{},
			counts: map[int]int{},
		}

		gatherer := NewStatsGatherer("/test", mockReader, nil, nil, nil)
		stats, err := gatherer.GatherCallsStats(context.Background())

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(stats) != 0 {
			t.Errorf("Expected 0 years, got %d", len(stats))
		}
	})
}

func TestStatsGatherer_GatherSMSStats(t *testing.T) {
	t.Parallel()

	t.Run("success_with_sms_and_mms", func(t *testing.T) {
		mockReader := &mockSMSReader{
			years: []int{2014},
			counts: map[int]int{
				2014: 3,
			},
			messagesByYear: map[int][]sms.Message{
				2014: {
					sms.SMS{Date: 1410881505425}, // SMS
					sms.SMS{Date: 1411000000000}, // SMS
					sms.MMS{Date: 1412000000000}, // MMS
				},
			},
		}

		gatherer := NewStatsGatherer("/test", nil, mockReader, nil, nil)
		stats, err := gatherer.GatherSMSStats(context.Background())

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(stats) != 1 {
			t.Errorf("Expected 1 year, got %d", len(stats))
		}

		yearStats := stats["2014"]
		if yearStats.TotalCount != 3 {
			t.Errorf("Expected total count 3, got %d", yearStats.TotalCount)
		}

		if yearStats.SMSCount != 2 {
			t.Errorf("Expected SMS count 2, got %d", yearStats.SMSCount)
		}

		if yearStats.MMSCount != 1 {
			t.Errorf("Expected MMS count 1, got %d", yearStats.MMSCount)
		}
	})
}

func TestStatsGatherer_GatherAttachmentStats(t *testing.T) {
	t.Parallel()

	t.Run("success_with_attachments", func(t *testing.T) {
		mockAttachmentReader := &mockAttachmentReader{
			attachments: []*attachments.Attachment{
				{
					Hash: "hash1",
					Size: 1024,
					Path: "attachments/ha/hash1/photo.jpg",
				},
				{
					Hash: "hash2",
					Size: 2048,
					Path: "attachments/ha/hash2/screenshot.png",
				},
				{
					Hash: "hash3",
					Size: 512,
					Path: "attachments/ha/hash3/video.mp4",
				},
			},
			orphanedAttachments: []*attachments.Attachment{
				{
					Hash: "hash3",
					Size: 512,
					Path: "attachments/ha/hash3/video.mp4",
				},
			},
		}

		mockSMSReader := &mockSMSReader{
			attachmentRefs: map[string]bool{
				"hash1": true,
				"hash2": true,
			},
		}

		gatherer := NewStatsGatherer("/test", nil, mockSMSReader, mockAttachmentReader, nil)
		stats, err := gatherer.GatherAttachmentStats(context.Background())

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if stats.Count != 3 {
			t.Errorf("Expected count 3, got %d", stats.Count)
		}

		expectedSize := int64(1024 + 2048 + 512)
		if stats.TotalSize != expectedSize {
			t.Errorf("Expected total size %d, got %d", expectedSize, stats.TotalSize)
		}

		if stats.Referenced != 2 {
			t.Errorf("Expected 2 referenced attachments, got %d", stats.Referenced)
		}

		if stats.Orphaned != 1 {
			t.Errorf("Expected 1 orphaned attachment, got %d", stats.Orphaned)
		}

		if stats.ByType["image/jpeg"] != 1 {
			t.Errorf("Expected 1 JPEG, got %d", stats.ByType["image/jpeg"])
		}

		if stats.ByType["video/mp4"] != 1 {
			t.Errorf("Expected 1 MP4, got %d", stats.ByType["video/mp4"])
		}
	})
}

func TestStatsGatherer_GatherContactsStats(t *testing.T) {
	t.Parallel()

	t.Run("success_with_contacts", func(t *testing.T) {
		mockContactsReader := &mockContactsReader{
			count: 4,
			contactsList: []*contacts.Contact{
				{Name: "John Doe", Numbers: []string{"5551234567"}},
				{Name: "Jane Smith", Numbers: []string{"5559876543"}},
				{Name: "(Unknown)", Numbers: []string{"5555555555"}},
				{Name: "", Numbers: []string{"5554443333"}},
			},
		}

		gatherer := NewStatsGatherer("/test", nil, nil, nil, mockContactsReader)
		stats, err := gatherer.GatherContactsStats(context.Background())

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if stats.Count != 4 {
			t.Errorf("Expected count 4, got %d", stats.Count)
		}

		if stats.WithNames != 2 {
			t.Errorf("Expected 2 with names, got %d", stats.WithNames)
		}

		if stats.PhoneOnly != 2 {
			t.Errorf("Expected 2 phone-only, got %d", stats.PhoneOnly)
		}
	})
}

func TestStatsGatherer_GatherStats(t *testing.T) {
	t.Parallel()

	t.Run("success_full_stats", func(t *testing.T) {
		mockCallsReader := &mockCallsReader{
			years:  []int{2014},
			counts: map[int]int{2014: 5},
			callsByYear: map[int][]calls.Call{
				2014: {{Date: 1410881505425}},
			},
		}

		mockSMSReader := &mockSMSReader{
			years:  []int{2014},
			counts: map[int]int{2014: 3},
			messagesByYear: map[int][]sms.Message{
				2014: {sms.SMS{Date: 1410881505425}},
			},
			attachmentRefs: map[string]bool{},
		}

		mockAttachmentReader := &mockAttachmentReader{
			attachments:         []*attachments.Attachment{},
			orphanedAttachments: []*attachments.Attachment{},
		}

		mockContactsReader := &mockContactsReader{
			count:        0,
			contactsList: []*contacts.Contact{},
		}

		gatherer := NewStatsGatherer(
			"/test",
			mockCallsReader,
			mockSMSReader,
			mockAttachmentReader,
			mockContactsReader,
		)

		stats, err := gatherer.GatherStats(context.Background())

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !stats.ValidationOK {
			t.Error("Expected validation to be OK")
		}

		if len(stats.Errors) != 0 {
			t.Errorf("Expected no errors, got %d", len(stats.Errors))
		}
	})
}

func TestUpdateDateRange(t *testing.T) {
	t.Parallel()

	var earliest, latest time.Time
	ts1 := time.Date(2014, 9, 16, 0, 0, 0, 0, time.UTC)
	ts2 := time.Date(2014, 12, 31, 0, 0, 0, 0, time.UTC)
	ts3 := time.Date(2014, 6, 1, 0, 0, 0, 0, time.UTC)

	earliest, latest = updateDateRange(earliest, latest, ts1)
	if !earliest.Equal(ts1) || !latest.Equal(ts1) {
		t.Error("First timestamp should set both earliest and latest")
	}

	earliest, latest = updateDateRange(earliest, latest, ts2)
	if !earliest.Equal(ts1) || !latest.Equal(ts2) {
		t.Error("Later timestamp should update latest only")
	}

	earliest, latest = updateDateRange(earliest, latest, ts3)
	if !earliest.Equal(ts3) || !latest.Equal(ts2) {
		t.Error("Earlier timestamp should update earliest only")
	}
}

func TestDetermineMimeTypeFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"jpg_extension", "photo.jpg", "image/jpeg"},
		{"jpeg_extension", "photo.jpeg", "image/jpeg"},
		{"png_extension", "screenshot.png", "image/png"},
		{"gif_extension", "animation.gif", "image/gif"},
		{"mp4_extension", "video.mp4", "video/mp4"},
		{"3gp_extension", "video.3gp", "video/3gpp"},
		{"amr_extension", "voice.amr", "audio/amr"},
		{"mp3_extension", "song.mp3", "audio/mp3"},
		{"unknown_extension", "file.xyz", "application/octet-stream"},
		{"no_extension", "file", "application/octet-stream"},
		{"full_path", "attachments/ab/abcdef/photo.jpg", "image/jpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineMimeTypeFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
