package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/phillipgreenii/mobilecombackup/pkg/importer"
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
