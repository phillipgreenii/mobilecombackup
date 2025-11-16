package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/attachments"
	"github.com/phillipgreenii/mobilecombackup/pkg/calls"
	"github.com/phillipgreenii/mobilecombackup/pkg/contacts"
	"github.com/phillipgreenii/mobilecombackup/pkg/importer"
	"github.com/phillipgreenii/mobilecombackup/pkg/sms"
	"github.com/spf13/afero"
)

func TestDetermineExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		summary      *importer.ImportSummary
		allowRejects bool
		expectedCode int
		description  string
	}{
		{
			name: "no rejections returns 0",
			summary: &importer.ImportSummary{
				Calls: &importer.EntityStats{
					Total: &importer.YearStat{Rejected: 0},
				},
				SMS: &importer.EntityStats{
					Total: &importer.YearStat{Rejected: 0},
				},
			},
			allowRejects: false,
			expectedCode: 0,
			description:  "When there are no rejections, exit code should be 0",
		},
		{
			name: "calls rejections returns 1",
			summary: &importer.ImportSummary{
				Calls: &importer.EntityStats{
					Total: &importer.YearStat{Rejected: 5},
				},
				SMS: &importer.EntityStats{
					Total: &importer.YearStat{Rejected: 0},
				},
			},
			allowRejects: false,
			expectedCode: 1,
			description:  "When there are call rejections and allowRejects is false, exit code should be 1",
		},
		{
			name: "SMS rejections returns 1",
			summary: &importer.ImportSummary{
				Calls: &importer.EntityStats{
					Total: &importer.YearStat{Rejected: 0},
				},
				SMS: &importer.EntityStats{
					Total: &importer.YearStat{Rejected: 3},
				},
			},
			allowRejects: false,
			expectedCode: 1,
			description:  "When there are SMS rejections and allowRejects is false, exit code should be 1",
		},
		{
			name: "mixed rejections returns 1",
			summary: &importer.ImportSummary{
				Calls: &importer.EntityStats{
					Total: &importer.YearStat{Rejected: 2},
				},
				SMS: &importer.EntityStats{
					Total: &importer.YearStat{Rejected: 3},
				},
			},
			allowRejects: false,
			expectedCode: 1,
			description:  "When there are both call and SMS rejections and allowRejects is false, exit code should be 1",
		},
		{
			name: "rejections allowed returns 0",
			summary: &importer.ImportSummary{
				Calls: &importer.EntityStats{
					Total: &importer.YearStat{Rejected: 5},
				},
				SMS: &importer.EntityStats{
					Total: &importer.YearStat{Rejected: 3},
				},
			},
			allowRejects: true,
			expectedCode: 0,
			description:  "When there are rejections but allowRejects is true, exit code should be 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code := DetermineExitCode(tt.summary, tt.allowRejects)
			if code != tt.expectedCode {
				t.Errorf("%s: expected exit code %d, got %d", tt.description, tt.expectedCode, code)
			}
		})
	}
}

func TestCalculateTotalRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		summary     *importer.ImportSummary
		expected    int
		description string
	}{
		{
			name: "no rejections",
			summary: &importer.ImportSummary{
				Rejections: map[string]*importer.RejectionStats{},
			},
			expected:    0,
			description: "When there are no rejection stats, total should be 0",
		},
		{
			name: "single rejection type",
			summary: &importer.ImportSummary{
				Rejections: map[string]*importer.RejectionStats{
					"duplicate": {Count: 5},
				},
			},
			expected:    5,
			description: "When there is one rejection type, total should equal its count",
		},
		{
			name: "multiple rejection types",
			summary: &importer.ImportSummary{
				Rejections: map[string]*importer.RejectionStats{
					"duplicate":     {Count: 5},
					"invalid_phone": {Count: 3},
					"parse_error":   {Count: 2},
				},
			},
			expected:    10,
			description: "When there are multiple rejection types, total should be sum of all counts",
		},
		{
			name: "zero count rejections",
			summary: &importer.ImportSummary{
				Rejections: map[string]*importer.RejectionStats{
					"duplicate":     {Count: 5},
					"invalid_phone": {Count: 0},
					"parse_error":   {Count: 3},
				},
			},
			expected:    8,
			description: "Zero count rejections should not affect total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			total := CalculateTotalRejections(tt.summary)
			if total != tt.expected {
				t.Errorf("%s: expected %d, got %d", tt.description, tt.expected, total)
			}
		})
	}
}

func TestCalculateRepositoryHealth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		info        *RepositoryInfo
		expected    HealthStatus
		description string
	}{
		{
			name: "healthy repository with calls",
			info: &RepositoryInfo{
				Calls: map[string]YearInfo{
					"2023": {Count: 100},
				},
				SMS:    map[string]MessageInfo{},
				Errors: map[string]int{},
			},
			expected:    HealthOK,
			description: "Repository with calls and no errors should be healthy",
		},
		{
			name: "healthy repository with SMS",
			info: &RepositoryInfo{
				Calls: map[string]YearInfo{},
				SMS: map[string]MessageInfo{
					"2023": {TotalCount: 50},
				},
				Errors: map[string]int{},
			},
			expected:    HealthOK,
			description: "Repository with SMS and no errors should be healthy",
		},
		{
			name: "healthy repository with both",
			info: &RepositoryInfo{
				Calls: map[string]YearInfo{
					"2023": {Count: 100},
				},
				SMS: map[string]MessageInfo{
					"2023": {TotalCount: 50},
				},
				Errors: map[string]int{},
			},
			expected:    HealthOK,
			description: "Repository with both calls and SMS should be healthy",
		},
		{
			name: "empty repository",
			info: &RepositoryInfo{
				Calls:  map[string]YearInfo{},
				SMS:    map[string]MessageInfo{},
				Errors: map[string]int{},
			},
			expected:    HealthEmpty,
			description: "Repository with no data should be empty",
		},
		{
			name: "unhealthy repository with errors",
			info: &RepositoryInfo{
				Calls: map[string]YearInfo{
					"2023": {Count: 100},
				},
				SMS: map[string]MessageInfo{},
				Errors: map[string]int{
					"calls": 1,
				},
			},
			expected:    HealthUnhealthy,
			description: "Repository with errors should be unhealthy",
		},
		{
			name: "unhealthy empty repository with errors",
			info: &RepositoryInfo{
				Calls: map[string]YearInfo{},
				SMS:   map[string]MessageInfo{},
				Errors: map[string]int{
					"calls": 1,
				},
			},
			expected:    HealthUnhealthy,
			description: "Empty repository with errors should be unhealthy (errors take precedence)",
		},
		{
			name: "repository with zero count years",
			info: &RepositoryInfo{
				Calls: map[string]YearInfo{
					"2023": {Count: 0},
				},
				SMS: map[string]MessageInfo{
					"2023": {TotalCount: 0},
				},
				Errors: map[string]int{},
			},
			expected:    HealthEmpty,
			description: "Repository with years but zero counts should be empty",
		},
		{
			name: "repository with multiple years",
			info: &RepositoryInfo{
				Calls: map[string]YearInfo{
					"2022": {
						Count:    50,
						Earliest: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
						Latest:   time.Date(2022, 12, 31, 23, 59, 59, 0, time.UTC),
					},
					"2023": {
						Count:    150,
						Earliest: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
						Latest:   time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC),
					},
				},
				SMS:    map[string]MessageInfo{},
				Errors: map[string]int{},
			},
			expected:    HealthOK,
			description: "Repository with multiple years of data should be healthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			health := CalculateRepositoryHealth(tt.info)
			if health != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, health)
			}
		})
	}
}

func TestTestExitHandler(t *testing.T) {
	t.Parallel()

	t.Run("records exit calls", func(t *testing.T) {
		t.Parallel()
		handler := &TestExitHandler{}

		// Initially not called
		if handler.Called {
			t.Error("expected handler.Called to be false initially")
		}

		// Call exit
		handler.Exit(42)

		// Should record the call
		if !handler.Called {
			t.Error("expected handler.Called to be true after Exit()")
		}
		if handler.Code != 42 {
			t.Errorf("expected exit code 42, got %d", handler.Code)
		}
	})

	t.Run("records multiple exit calls", func(t *testing.T) {
		t.Parallel()
		handler := &TestExitHandler{}

		handler.Exit(1)
		handler.Exit(2)

		// Should record the last call
		if handler.Code != 2 {
			t.Errorf("expected exit code 2 (last call), got %d", handler.Code)
		}
		if !handler.Called {
			t.Error("expected handler.Called to be true")
		}
	})
}

func TestImportContextHandleImportError(t *testing.T) {
	t.Parallel()

	t.Run("calls exit handler with code 2", func(t *testing.T) {
		t.Parallel()
		exitHandler := &TestExitHandler{}
		var buf bytes.Buffer

		ctx := &ImportContext{
			Quiet:       false,
			ExitHandler: exitHandler,
			Output:      &buf,
		}

		ctx.HandleImportError(errors.New("test error"))

		if !exitHandler.Called {
			t.Error("expected exit handler to be called")
		}
		if exitHandler.Code != 2 {
			t.Errorf("expected exit code 2, got %d", exitHandler.Code)
		}
		if !strings.Contains(buf.String(), "test error") {
			t.Errorf("expected error message in output, got: %s", buf.String())
		}
	})

	t.Run("quiet mode suppresses output", func(t *testing.T) {
		t.Parallel()
		exitHandler := &TestExitHandler{}
		var buf bytes.Buffer

		ctx := &ImportContext{
			Quiet:       true,
			ExitHandler: exitHandler,
			Output:      &buf,
		}

		ctx.HandleImportError(errors.New("test error"))

		if !exitHandler.Called {
			t.Error("expected exit handler to be called")
		}
		if exitHandler.Code != 2 {
			t.Errorf("expected exit code 2, got %d", exitHandler.Code)
		}
		if buf.Len() > 0 {
			t.Errorf("expected no output in quiet mode, got: %s", buf.String())
		}
	})
}

func TestImportContextHandleExitCode(t *testing.T) {
	t.Parallel()

	t.Run("exit code 0 when no rejections", func(t *testing.T) {
		t.Parallel()
		exitHandler := &TestExitHandler{}

		ctx := &ImportContext{
			ExitHandler:      exitHandler,
			NoErrorOnRejects: false,
		}

		summary := &importer.ImportSummary{
			Calls: &importer.EntityStats{
				Total: &importer.YearStat{Rejected: 0},
			},
			SMS: &importer.EntityStats{
				Total: &importer.YearStat{Rejected: 0},
			},
		}

		ctx.HandleExitCode(summary)

		if exitHandler.Called {
			t.Errorf("expected no exit call, but got exit with code %d", exitHandler.Code)
		}
	})

	t.Run("exit code 1 when rejections exist", func(t *testing.T) {
		t.Parallel()
		exitHandler := &TestExitHandler{}

		ctx := &ImportContext{
			ExitHandler:      exitHandler,
			NoErrorOnRejects: false,
		}

		summary := &importer.ImportSummary{
			Calls: &importer.EntityStats{
				Total: &importer.YearStat{Rejected: 5},
			},
			SMS: &importer.EntityStats{
				Total: &importer.YearStat{Rejected: 0},
			},
		}

		ctx.HandleExitCode(summary)

		if !exitHandler.Called {
			t.Error("expected exit handler to be called")
		}
		if exitHandler.Code != 1 {
			t.Errorf("expected exit code 1, got %d", exitHandler.Code)
		}
	})

	t.Run("no exit when rejections allowed", func(t *testing.T) {
		t.Parallel()
		exitHandler := &TestExitHandler{}

		ctx := &ImportContext{
			ExitHandler:      exitHandler,
			NoErrorOnRejects: true,
		}

		summary := &importer.ImportSummary{
			Calls: &importer.EntityStats{
				Total: &importer.YearStat{Rejected: 5},
			},
			SMS: &importer.EntityStats{
				Total: &importer.YearStat{Rejected: 3},
			},
		}

		ctx.HandleExitCode(summary)

		if exitHandler.Called {
			t.Errorf("expected no exit call when NoErrorOnRejects=true, but got exit with code %d", exitHandler.Code)
		}
	})
}

// Mock implementations for testing InfoContext

// MockCallsReader implements calls.Reader for testing
type MockCallsReader struct {
	Years      []int
	YearCounts map[int]int
	CallsData  map[int][]calls.Call
	Err        error
}

func (m *MockCallsReader) GetAvailableYears(_ context.Context) ([]int, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Years, nil
}

func (m *MockCallsReader) GetCallsCount(_ context.Context, year int) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.YearCounts[year], nil
}

func (m *MockCallsReader) StreamCallsForYear(_ context.Context, year int, processor func(calls.Call) error) error {
	if m.Err != nil {
		return m.Err
	}
	for _, call := range m.CallsData[year] {
		if err := processor(call); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockCallsReader) ReadCalls(_ context.Context, year int) ([]calls.Call, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.CallsData[year], nil
}

func (m *MockCallsReader) ValidateCallsFile(_ context.Context, year int) error {
	return m.Err
}

// MockSMSReader implements sms.Reader for testing
type MockSMSReader struct {
	Years          []int
	MessageCounts  map[int]int
	MessagesData   map[int][]sms.Message
	AttachmentRefs map[string]bool
	Err            error
}

func (m *MockSMSReader) GetAvailableYears(_ context.Context) ([]int, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Years, nil
}

func (m *MockSMSReader) GetMessageCount(_ context.Context, year int) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	return m.MessageCounts[year], nil
}

func (m *MockSMSReader) StreamMessagesForYear(_ context.Context, year int, processor func(sms.Message) error) error {
	if m.Err != nil {
		return m.Err
	}
	for _, msg := range m.MessagesData[year] {
		if err := processor(msg); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockSMSReader) GetAllAttachmentRefs(_ context.Context) (map[string]bool, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.AttachmentRefs, nil
}

func (m *MockSMSReader) ReadMessages(_ context.Context, year int) ([]sms.Message, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.MessagesData[year], nil
}

func (m *MockSMSReader) GetAttachmentRefs(_ context.Context, year int) ([]string, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var refs []string
	for ref := range m.AttachmentRefs {
		refs = append(refs, ref)
	}
	return refs, nil
}

func (m *MockSMSReader) ValidateSMSFile(_ context.Context, year int) error {
	return m.Err
}

// MockContactsReader implements contacts.Reader for testing
type MockContactsReader struct {
	Count int
	Err   error
}

func (m *MockContactsReader) LoadContacts(_ context.Context) error {
	return m.Err
}

func (m *MockContactsReader) GetContactsCount() int {
	return m.Count
}

func (m *MockContactsReader) GetContactByNumber(number string) (string, bool) {
	return "", false
}

func (m *MockContactsReader) GetNumbersByContact(name string) ([]string, bool) {
	return nil, false
}

func (m *MockContactsReader) GetAllContacts(_ context.Context) ([]*contacts.Contact, error) {
	return nil, nil
}

func (m *MockContactsReader) ContactExists(name string) bool {
	return false
}

func (m *MockContactsReader) IsKnownNumber(number string) bool {
	return false
}

func (m *MockContactsReader) AddUnprocessedContacts(_ context.Context, addresses, contactNames string) error {
	return nil
}

func (m *MockContactsReader) GetUnprocessedEntries() []contacts.UnprocessedEntry {
	return nil
}

// TestInfoContextReadRepositoryMetadata tests the readRepositoryMetadata method
func TestInfoContextReadRepositoryMetadata(t *testing.T) {
	t.Parallel()

	t.Run("reads valid metadata", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		_ = afero.WriteFile(fs, "/test/.mobilecombackup.yaml", []byte(`
repository_structure_version: "1.0.0"
created_at: "2024-01-15T10:30:00Z"
created_by: "mobilecombackup"
`), 0644)

		ctx := &InfoContext{
			RepoPath: "/test",
			Fs:       fs,
		}

		info := &RepositoryInfo{}
		err := ctx.readRepositoryMetadata(info)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Version != "1.0.0" {
			t.Errorf("expected version 1.0.0, got %s", info.Version)
		}
		if info.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set")
		}
	})

	t.Run("handles missing file", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()

		ctx := &InfoContext{
			RepoPath: "/test",
			Fs:       fs,
		}

		info := &RepositoryInfo{}
		err := ctx.readRepositoryMetadata(info)

		if err == nil {
			t.Error("expected error for missing file")
		}
	})

	t.Run("handles invalid yaml", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		_ = afero.WriteFile(fs, "/test/.mobilecombackup.yaml", []byte(`invalid: yaml: content:`), 0644)

		ctx := &InfoContext{
			RepoPath: "/test",
			Fs:       fs,
		}

		info := &RepositoryInfo{}
		err := ctx.readRepositoryMetadata(info)

		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})
}

// TestInfoContextCountRejections tests the countRejections method
func TestInfoContextCountRejections(t *testing.T) {
	t.Parallel()

	t.Run("counts rejection files correctly", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		_ = fs.MkdirAll("/test/rejected", 0755)
		_ = afero.WriteFile(fs, "/test/rejected/calls_rejected_1.xml", []byte("data"), 0644)
		_ = afero.WriteFile(fs, "/test/rejected/calls_rejected_2.xml", []byte("data"), 0644)
		_ = afero.WriteFile(fs, "/test/rejected/sms_rejected_1.xml", []byte("data"), 0644)
		_ = afero.WriteFile(fs, "/test/rejected/other.txt", []byte("data"), 0644)

		ctx := &InfoContext{
			RepoPath: "/test",
			Fs:       fs,
		}

		info := &RepositoryInfo{
			Rejections: make(map[string]int),
		}
		ctx.countRejections(info)

		if info.Rejections["calls"] != 2 {
			t.Errorf("expected 2 call rejections, got %d", info.Rejections["calls"])
		}
		if info.Rejections["sms"] != 1 {
			t.Errorf("expected 1 SMS rejection, got %d", info.Rejections["sms"])
		}
	})

	t.Run("handles missing rejected directory", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()

		ctx := &InfoContext{
			RepoPath: "/test",
			Fs:       fs,
		}

		info := &RepositoryInfo{
			Rejections: make(map[string]int),
		}
		ctx.countRejections(info)

		if len(info.Rejections) != 0 {
			t.Errorf("expected no rejections for missing directory, got %v", info.Rejections)
		}
	})
}

// TestInfoContextGatherRepositoryInfo tests the GatherRepositoryInfo method
func TestInfoContextGatherRepositoryInfo(t *testing.T) {
	t.Parallel()

	t.Run("gathers info successfully", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		_ = fs.MkdirAll("/test/attachments", 0755)

		ctx := &InfoContext{
			RepoPath: "/test",
			Fs:       fs,
			CallsReader: &MockCallsReader{
				Years:      []int{},
				YearCounts: make(map[int]int),
				CallsData:  make(map[int][]calls.Call),
			},
			SMSReader: &MockSMSReader{
				Years:          []int{},
				MessageCounts:  make(map[int]int),
				MessagesData:   make(map[int][]sms.Message),
				AttachmentRefs: make(map[string]bool),
			},
			AttachmentReader: attachments.NewAttachmentManager("/test", fs),
			ContactsReader: &MockContactsReader{
				Count: 0,
			},
		}

		info, err := ctx.GatherRepositoryInfo()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info == nil {
			t.Fatal("expected info to be returned")
		}
		if !info.ValidationOK {
			t.Error("expected validation to pass with no errors")
		}
	})

	t.Run("tracks errors from readers", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		_ = fs.MkdirAll("/test/attachments", 0755)

		ctx := &InfoContext{
			RepoPath: "/test",
			Fs:       fs,
			CallsReader: &MockCallsReader{
				Err: errors.New("calls error"),
			},
			SMSReader: &MockSMSReader{
				Err: errors.New("sms error"),
			},
			AttachmentReader: attachments.NewAttachmentManager("/test", fs),
			ContactsReader: &MockContactsReader{
				Err: errors.New("contacts error"),
			},
		}

		info, err := ctx.GatherRepositoryInfo()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Errors["calls"] != 1 {
			t.Error("expected calls error to be tracked")
		}
		if info.Errors["sms"] != 1 {
			t.Error("expected sms error to be tracked")
		}
		if info.Errors["contacts"] != 1 {
			t.Error("expected contacts error to be tracked")
		}
		if info.ValidationOK {
			t.Error("expected validation to fail with errors")
		}
	})
}

// TestNewInfoContext tests the NewInfoContext constructor
func TestNewInfoContext(t *testing.T) {
	t.Parallel()

	ctx := NewInfoContext("/test/repo", true, false)

	if ctx == nil {
		t.Fatal("expected context to be created")
	}
	if ctx.RepoPath != "/test/repo" {
		t.Errorf("expected RepoPath /test/repo, got %s", ctx.RepoPath)
	}
	if !ctx.OutputJSON {
		t.Error("expected OutputJSON to be true")
	}
	if ctx.Quiet {
		t.Error("expected Quiet to be false")
	}
	if ctx.CallsReader == nil {
		t.Error("expected CallsReader to be initialized")
	}
	if ctx.SMSReader == nil {
		t.Error("expected SMSReader to be initialized")
	}
	if ctx.ContactsReader == nil {
		t.Error("expected ContactsReader to be initialized")
	}
	if ctx.ExitHandler == nil {
		t.Error("expected ExitHandler to be initialized")
	}
}

// TestNewImportContext tests the NewImportContext constructor
func TestNewImportContext(t *testing.T) {
	t.Parallel()

	t.Run("creates context with default settings", func(t *testing.T) {
		t.Parallel()
		options := &importer.ImportOptions{
			Quiet: false,
		}

		ctx := NewImportContext(options, false, false, false)

		if ctx == nil {
			t.Fatal("expected context to be created")
		}
		if ctx.Options != options {
			t.Error("expected Options to be set")
		}
		if ctx.Quiet {
			t.Error("expected Quiet to be false")
		}
		if ctx.OutputJSON {
			t.Error("expected OutputJSON to be false")
		}
		if ctx.DryRun {
			t.Error("expected DryRun to be false")
		}
		if ctx.NoErrorOnRejects {
			t.Error("expected NoErrorOnRejects to be false")
		}
		if ctx.ExitHandler == nil {
			t.Error("expected ExitHandler to be initialized")
		}
	})

	t.Run("JSON mode forces quiet", func(t *testing.T) {
		t.Parallel()
		options := &importer.ImportOptions{
			Quiet: false,
		}

		ctx := NewImportContext(options, true, false, false)

		if !ctx.Quiet {
			t.Error("expected Quiet to be true when OutputJSON is true")
		}
		if !ctx.OutputJSON {
			t.Error("expected OutputJSON to be true")
		}
	})

	t.Run("options quiet is respected", func(t *testing.T) {
		t.Parallel()
		options := &importer.ImportOptions{
			Quiet: true,
		}

		ctx := NewImportContext(options, false, false, false)

		if !ctx.Quiet {
			t.Error("expected Quiet to be true when options.Quiet is true")
		}
	})
}
