package sms

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestNewXMLSMSWriter(t *testing.T) {
	fs := afero.NewMemMapFs()
	repoPath := filepath.Join("/", "sms")

	writer, err := NewXMLSMSWriter(repoPath, fs)
	if err != nil {
		t.Fatalf("NewXMLSMSWriter failed: %v", err)
	}

	if writer == nil {
		t.Fatal("Writer is nil")
	}

	// Verify directory was created
	if _, err := fs.Stat(repoPath); err != nil {
		t.Error("Directory was not created")
	}
}

func TestXMLSMSWriter_WriteMessages(t *testing.T) {
	fs := afero.NewMemMapFs()
	repoPath := filepath.Join("/", "sms")

	writer, err := NewXMLSMSWriter(repoPath, fs)
	if err != nil {
		t.Fatalf("NewXMLSMSWriter failed: %v", err)
	}

	messages := []Message{
		&SMS{
			Address: "1234567890",
			Date:    1234567890000,
			Type:    1,
			Body:    "Test message",
		},
	}

	filename := "test.xml"
	err = writer.WriteMessages(filename, messages)
	if err != nil {
		t.Fatalf("WriteMessages failed: %v", err)
	}

	// Verify file was created
	filePath := filepath.Join(repoPath, filename)
	if _, err := fs.Stat(filePath); err != nil {
		t.Error("File was not created")
	}
}
