package importer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	
	"gopkg.in/yaml.v3"
)

// XMLRejectionWriter writes rejected entries to XML files
type XMLRejectionWriter struct {
	repoRoot string
	dirOnce  sync.Once
	dirErr   error
}

// NewXMLRejectionWriter creates a new XML rejection writer
func NewXMLRejectionWriter(repoRoot string) *XMLRejectionWriter {
	// Don't create directory here - only create it when needed
	return &XMLRejectionWriter{
		repoRoot: repoRoot,
	}
}

// ensureRejectedDirectory creates the rejected directory structure on first use
func (w *XMLRejectionWriter) ensureRejectedDirectory() error {
	w.dirOnce.Do(func() {
		// Create main rejected directory
		rejectedDir := filepath.Join(w.repoRoot, "rejected")
		if err := os.MkdirAll(rejectedDir, 0755); err != nil {
			w.dirErr = fmt.Errorf("failed to create rejected directory: %w", err)
			return
		}
		
		// Create subdirectories for calls and sms
		for _, subdir := range []string{"calls", "sms"} {
			path := filepath.Join(rejectedDir, subdir)
			if err := os.MkdirAll(path, 0755); err != nil {
				w.dirErr = fmt.Errorf("failed to create rejected/%s directory: %w", subdir, err)
				return
			}
		}
	})
	return w.dirErr
}

// WriteRejections writes rejected entries to a file
func (w *XMLRejectionWriter) WriteRejections(originalFile string, rejections []RejectedEntry) (string, error) {
	// Only create directories if we have rejections to write
	if len(rejections) == 0 {
		return "", nil
	}
	
	// Ensure rejected directory exists
	if err := w.ensureRejectedDirectory(); err != nil {
		return "", err
	}
	
	// Calculate hash of original file
	hash, err := w.hashFile(originalFile)
	if err != nil {
		return "", fmt.Errorf("failed to hash original file: %w", err)
	}
	
	// Generate timestamp
	timestamp := time.Now().Format("20060102-150405")
	
	// Extract base name without extension
	baseName := filepath.Base(originalFile)
	ext := filepath.Ext(baseName)
	nameNoExt := baseName[:len(baseName)-len(ext)]
	
	// Determine type subdirectory
	var typeDir string
	if strings.Contains(nameNoExt, "call") {
		typeDir = "calls"
	} else if strings.Contains(nameNoExt, "sms") {
		typeDir = "sms"
	} else {
		typeDir = "" // Root rejected directory
	}
	
	// Create rejection file name
	rejectedDir := filepath.Join(w.repoRoot, "rejected")
	if typeDir != "" {
		rejectedDir = filepath.Join(rejectedDir, typeDir)
	}
	rejectFile := fmt.Sprintf("%s-%s-%s.xml", nameNoExt, hash[:8], timestamp)
	rejectPath := filepath.Join(rejectedDir, rejectFile)
	
	// Write XML rejection file
	if err := w.writeXMLRejections(rejectPath, rejections, typeDir); err != nil {
		return "", err
	}
	
	return rejectPath, nil
}

// hashFile calculates SHA-256 hash of a file
func (w *XMLRejectionWriter) hashFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// writeXMLRejections writes rejected entries as XML
func (w *XMLRejectionWriter) writeXMLRejections(filename string, rejections []RejectedEntry, entryType string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create rejection file: %w", err)
	}
	defer file.Close()
	
	// Write XML header
	file.WriteString(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>` + "\n")
	
	// Determine root element name based on type
	var rootName string
	switch entryType {
	case "calls":
		rootName = "calls"
	case "sms":
		rootName = "smses"
	default:
		rootName = "entries" // Generic fallback
	}
	
	// Write opening tag
	fmt.Fprintf(file, "<%s count=\"%d\">\n", rootName, len(rejections))
	
	// Write each rejected entry
	for _, rej := range rejections {
		file.WriteString("  " + rej.Data + "\n")
	}
	
	// Write closing tag
	fmt.Fprintf(file, "</%s>\n", rootName)
	
	return nil
}

// writeViolations writes violation details as YAML
func (w *XMLRejectionWriter) writeViolations(filename string, rejections []RejectedEntry) error {
	type violationEntry struct {
		Line       int      `yaml:"line"`
		Violations []string `yaml:"violations"`
	}
	
	type violationsFile struct {
		Violations []violationEntry `yaml:"violations"`
	}
	
	var vf violationsFile
	for _, rej := range rejections {
		vf.Violations = append(vf.Violations, violationEntry{
			Line:       rej.Line,
			Violations: rej.Violations,
		})
	}
	
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create violations file: %w", err)
	}
	defer file.Close()
	
	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(vf); err != nil {
		return fmt.Errorf("failed to write violations YAML: %w", err)
	}
	
	return nil
}