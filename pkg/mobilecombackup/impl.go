package mobilecombackup

import (
	"fmt"
)

// TODO: This is legacy code that needs to be refactored to use the new import architecture
// Temporarily implementing minimal functionality to allow build

type processorState struct {
	outputDir string
}

func (s *processorState) Process(fileRoot string) (Result, error) {
	// Placeholder implementation
	return Result{}, fmt.Errorf("legacy Process not implemented - use new import functionality")
}

func Init(rootPath string) (Processor, error) {
	// TODO: This implementation needs to be updated to match the new architecture
	// The old coalescer pattern doesn't match the new CallsReader/SMSReader pattern
	return nil, fmt.Errorf("Init not implemented - architecture mismatch")
}
