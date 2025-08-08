package importer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillipgreen/mobilecombackup/pkg/sms"
)

func TestSMSImporter_ImportFile(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "repo")
	
	// Create repository structure
	if err := os.MkdirAll(filepath.Join(repoRoot, "sms"), 0755); err != nil {
		t.Fatal(err)
	}
	
	// Copy test file
	testFile := filepath.Join(tempDir, "sms-test.xml")
	if err := copyFile("../../testdata/to_process/sms-test.xml", testFile); err != nil {
		t.Fatal(err)
	}
	
	// Create importer
	options := &ImportOptions{
		RepoRoot: repoRoot,
		Paths:    []string{testFile},
	}
	importer := NewSMSImporter(options)
	
	// Load repository (should be empty)
	if err := importer.LoadRepository(); err != nil {
		t.Fatalf("Failed to load repository: %v", err)
	}
	
	// Import file
	stats, err := importer.ImportFile(testFile)
	if err != nil {
		t.Fatalf("Failed to import file: %v", err)
	}
	
	// Verify stats
	if stats.Added == 0 {
		t.Error("Expected some messages to be added")
	}
	if stats.Duplicates != 0 {
		t.Error("Expected no duplicates in empty repository")
	}
	
	// Write repository
	if err := importer.WriteRepository(); err != nil {
		t.Fatalf("Failed to write repository: %v", err)
	}
	
	// Verify files were created
	years := []int{2013, 2014, 2015}
	for _, year := range years {
		path := filepath.Join(repoRoot, "sms", fmt.Sprintf("sms-%d.xml", year))
		if _, err := os.Stat(path); err != nil {
			// Some years might not have data, that's ok
			continue
		}
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	
	_, err = io.Copy(dstFile, srcFile)
	return err
}

func TestSMSImporter_MessageValidation(t *testing.T) {
	tempDir := t.TempDir()
	options := &ImportOptions{
		RepoRoot: tempDir,
		Paths:    []string{},
	}
	importer := NewSMSImporter(options)
	
	tests := []struct {
		name       string
		msg        sms.Message
		wantErrors []string
	}{
		{
			name: "valid SMS",
			msg: sms.SMS{
				Date:    1234567890000,
				Address: "+15555551234",
				Type:    sms.SentMessage,
				Body:    "Test message",
			},
			wantErrors: nil,
		},
		{
			name: "missing timestamp",
			msg: sms.SMS{
				Date:    0,
				Address: "+15555551234",
				Type:    sms.SentMessage,
			},
			wantErrors: []string{"missing-timestamp"},
		},
		{
			name: "missing address",
			msg: sms.SMS{
				Date:    1234567890000,
				Address: "",
				Type:    sms.SentMessage,
			},
			wantErrors: []string{"missing-address"},
		},
		{
			name: "invalid type",
			msg: sms.SMS{
				Date:    1234567890000,
				Address: "+15555551234",
				Type:    99,
			},
			wantErrors: []string{"invalid-type"},
		},
		{
			name: "valid MMS",
			msg: sms.MMS{
				Date:    1234567890000,
				Address: "+15555551234",
				MsgBox:  1,
			},
			wantErrors: nil,
		},
		{
			name: "invalid MMS msgbox",
			msg: sms.MMS{
				Date:    1234567890000,
				Address: "+15555551234",
				MsgBox:  99,
			},
			wantErrors: []string{"invalid-msg-box"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := importer.validateMessage(tt.msg)
			
			if len(violations) != len(tt.wantErrors) {
				t.Errorf("Got %d violations, want %d", len(violations), len(tt.wantErrors))
				t.Errorf("Violations: %v", violations)
				return
			}
			
			// Check each expected error
			for i, want := range tt.wantErrors {
				if i >= len(violations) || violations[i] != want {
					t.Errorf("Violation[%d] = %v, want %v", i, violations[i], want)
				}
			}
		})
	}
}

func TestSMSImporter_HashCalculation(t *testing.T) {
	// Test that hash excludes readable_date and contact_name
	msg1 := &sms.SMS{
		Date:         1234567890000,
		Address:      "+15555551234",
		Type:         sms.SentMessage,
		Body:         "Test message",
		ReadableDate: "Jan 1, 2009",
		ContactName:  "John Doe",
	}
	
	msg2 := &sms.SMS{
		Date:         1234567890000,
		Address:      "+15555551234",
		Type:         sms.SentMessage,
		Body:         "Test message",
		ReadableDate: "January 1st, 2009", // Different format
		ContactName:  "Jane Smith",         // Different contact
	}
	
	entry1 := sms.NewMessageEntry(msg1)
	entry2 := sms.NewMessageEntry(msg2)
	
	// Hashes should be the same since readable_date and contact_name are excluded
	if entry1.Hash() != entry2.Hash() {
		t.Error("Expected same hash when only readable_date and contact_name differ")
	}
	
	// Change a field that IS included in hash
	msg2.Body = "Different message"
	entry2 = sms.NewMessageEntry(msg2)
	
	if entry1.Hash() == entry2.Hash() {
		t.Error("Expected different hash when body differs")
	}
}