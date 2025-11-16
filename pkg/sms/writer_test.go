package sms

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewXMLSMSWriter(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "sms")

	writer, err := NewXMLSMSWriter(repoPath)
	if err != nil {
		t.Fatalf("NewXMLSMSWriter failed: %v", err)
	}

	if writer == nil {
		t.Fatal("Writer is nil")
	}

	// Verify directory was created
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}
}

func TestXMLSMSWriter_WriteMessages(t *testing.T) {
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "sms")

	writer, err := NewXMLSMSWriter(repoPath)
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
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File was not created")
	}
}
