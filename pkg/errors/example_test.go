package errors_test

import (
	"fmt"
	"os"

	"github.com/phillipgreen/mobilecombackup/pkg/errors"
)

func ExampleNewValidationError() {
	// Simulate a validation failure
	err := fmt.Errorf("value must be positive, got -5")

	// Wrap with validation context
	validationErr := errors.NewValidationError("validate user input", err)

	fmt.Println(validationErr.Error())
	// Output will contain file and line information, e.g.:
	// example_test.go:12: validate user input failed: value must be positive, got -5
}

func ExampleNewProcessingError() {
	// Simulate a processing failure
	inputFile := "backup.xml"
	stage := "parsing XML"
	err := fmt.Errorf("unexpected end of XML input")

	processingErr := errors.NewProcessingError(stage, inputFile, err)

	fmt.Println(processingErr.Error())
	// Output: processing failed at stage 'parsing XML' for file 'backup.xml': unexpected end of XML input
}

func ExampleNewImportError() {
	// Simulate an import failure after processing some items
	phase := "validation"
	entity := "contacts"
	count := 150
	err := fmt.Errorf("duplicate contact name found")

	importErr := errors.NewImportError(phase, entity, count, err)

	fmt.Println(importErr.Error())
	// Output: import failed during 'validation' phase for contacts after processing 150 items: duplicate contact name found
}

func ExampleNewFileError() {
	// Simulate a file operation failure
	filePath := "/path/to/config.yaml"
	operation := "read"
	err := &os.PathError{Op: "open", Path: filePath, Err: fmt.Errorf("no such file or directory")}

	fileErr := errors.NewFileError(filePath, operation, err)

	fmt.Println(fileErr.Error())
	// Output: file read failed for '/path/to/config.yaml': open /path/to/config.yaml: no such file or directory
}

func ExampleWrapWithValidation() {
	// Function that might return an error
	validateData := func() error {
		return fmt.Errorf("invalid format")
	}

	// Wrap the error with validation context
	err := errors.WrapWithValidation("validate input data", validateData())
	if err != nil {
		fmt.Println(err.Error())
		// Output will contain file and line, e.g.:
		// example_test.go:48: validate input data failed: invalid format
	}
}

func ExampleGetErrorCode() {
	// Create a structured error
	err := errors.NewProcessingError("parsing", "input.xml", fmt.Errorf("malformed XML"))

	// Extract the error code
	if code, ok := errors.GetErrorCode(err); ok {
		fmt.Printf("Error code: %s\n", code)
		// Output: Error code: PROCESSING_ERROR
	}
}

func ExampleIsErrorCode() {
	// Create a structured error
	err := errors.NewValidationError("validate schema", fmt.Errorf("missing required field"))

	// Check if it's a validation error
	if errors.IsErrorCode(err, errors.ErrCodeValidation) {
		fmt.Println("This is a validation error")
	}

	// Check if it's a different type of error
	if !errors.IsErrorCode(err, errors.ErrCodeFileNotFound) {
		fmt.Println("This is not a file not found error")
	}
	// Output:
	// This is a validation error
	// This is not a file not found error
}

func Example() {
	// Demonstrate basic usage of structured errors
	err := errors.NewValidationError("validate input", fmt.Errorf("invalid value"))
	fmt.Println("Error:", err.Error())

	// Check error type
	if errors.IsErrorCode(err, errors.ErrCodeValidation) {
		fmt.Println("Type: validation error")
	}

	// Output:
	// Error: example_test.go:105: validate input failed: invalid value
	// Type: validation error
}
