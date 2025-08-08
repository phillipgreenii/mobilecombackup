package importer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	
	"gopkg.in/yaml.v3"
)

// XMLRejectionWriter writes rejected entries to XML files
type XMLRejectionWriter struct {
	repoRoot string
}

// NewXMLRejectionWriter creates a new XML rejection writer
func NewXMLRejectionWriter(repoRoot string) *XMLRejectionWriter {
	// Ensure rejected directory exists
	rejectedDir := filepath.Join(repoRoot, "rejected")
	os.MkdirAll(rejectedDir, 0755)
	
	return &XMLRejectionWriter{
		repoRoot: repoRoot,
	}
}

// WriteRejections writes rejected entries to a file
func (w *XMLRejectionWriter) WriteRejections(originalFile string, rejections []RejectedEntry) (string, error) {
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
	
	// Create rejection file name
	rejectedDir := filepath.Join(w.repoRoot, "rejected")
	rejectFile := fmt.Sprintf("%s-%s-%s-rejects.xml", nameNoExt, hash[:8], timestamp)
	rejectPath := filepath.Join(rejectedDir, rejectFile)
	
	// Write XML rejection file
	if err := w.writeXMLRejections(rejectPath, rejections, nameNoExt); err != nil {
		return "", err
	}
	
	// Write violations YAML file
	violationsFile := fmt.Sprintf("%s-%s-%s-violations.yaml", nameNoExt, hash[:8], timestamp)
	violationsPath := filepath.Join(rejectedDir, violationsFile)
	if err := w.writeViolations(violationsPath, rejections); err != nil {
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
func (w *XMLRejectionWriter) writeXMLRejections(filename string, rejections []RejectedEntry, rootElement string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create rejection file: %w", err)
	}
	defer file.Close()
	
	// Write XML header
	file.WriteString(`<?xml version='1.0' encoding='UTF-8' standalone='yes' ?>` + "\n")
	
	// Determine root element name based on type
	rootName := rootElement + "es" // calls -> callses, sms -> smses
	if rootElement == "sms" {
		rootName = "smses"
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