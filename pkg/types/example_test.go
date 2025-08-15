package types

import (
	"fmt"
	"log"
)

// Example showing how Optional can simplify XML attribute parsing
func ExampleOptional() {
	// Simulating XML attribute values that might be "null" or empty
	attributes := []string{"42", "null", "", "123", "invalid"}

	for _, attr := range attributes {
		timestamp := ParseOptionalInt64(attr)

		if timestamp.IsSome() {
			fmt.Printf("Valid timestamp: %d\n", timestamp.Unwrap())
		} else {
			fmt.Printf("Invalid/missing timestamp for value: %q\n", attr)
		}
	}

	// Output:
	// Valid timestamp: 42
	// Invalid/missing timestamp for value: "null"
	// Invalid/missing timestamp for value: ""
	// Valid timestamp: 123
	// Invalid/missing timestamp for value: "invalid"
}

// Example showing how Result can improve error handling
func ExampleResult() {
	// Simulating parsing operations that can fail
	inputs := []string{"42", "invalid", "100"}

	for _, input := range inputs {
		result := TryParseInt(input)

		if result.IsOk() {
			fmt.Printf("Successfully parsed: %d\n", result.Unwrap())
		} else {
			fmt.Printf("Failed to parse %q: %v\n", input, result.Error)
		}
	}

	// Output:
	// Successfully parsed: 42
	// Failed to parse "invalid": strconv.Atoi: parsing "invalid": invalid syntax
	// Successfully parsed: 100
}

// Example showing how these types work together in a realistic parsing scenario
func ExampleOptional_sMSParsing() {
	// Simulate parsing SMS attributes from XML
	smsData := map[string]string{
		"address": "555-1234",
		"date":    "1640995200000", // Unix timestamp in milliseconds
		"type":    "2",             // SMS type
		"subject": "null",          // Subject might be null
		"body":    "Hello world",
	}

	// Parse with Optional types for better null handling
	address := ParseOptionalString(smsData["address"])
	date := ParseOptionalInt64(smsData["date"])
	msgType := ParseOptionalInt(smsData["type"])
	subject := ParseOptionalString(smsData["subject"])
	body := ParseOptionalString(smsData["body"])

	fmt.Printf("Address: %s\n", address.UnwrapOr("unknown"))
	fmt.Printf("Date: %d\n", date.UnwrapOr(0))
	fmt.Printf("Type: %d\n", msgType.UnwrapOr(0))
	fmt.Printf("Subject: %s\n", subject.UnwrapOr("(no subject)"))
	fmt.Printf("Body: %s\n", body.UnwrapOr("(empty)"))

	// Output:
	// Address: 555-1234
	// Date: 1640995200000
	// Type: 2
	// Subject: (no subject)
	// Body: Hello world
}

// Example demonstrating how Result can be used for validation chains
func ExampleResult_chaining() {
	validatePositive := func(n int) Result[int] {
		if n <= 0 {
			return NewResultError[int](fmt.Errorf("number must be positive, got %d", n))
		}
		return NewResult(n)
	}

	testValues := []int{42, -5, 0, 100}

	for _, value := range testValues {
		// Chain operations using Result
		parseResult := TryParseInt(fmt.Sprintf("%d", value))
		if parseResult.IsErr() {
			log.Printf("Parse failed: %v", parseResult.Error)
			continue
		}

		validationResult := validatePositive(parseResult.Unwrap())
		if validationResult.IsErr() {
			log.Printf("Validation failed: %v", validationResult.Error)
			continue
		}

		fmt.Printf("Valid positive number: %d\n", validationResult.Unwrap())
	}

	// Output:
	// Valid positive number: 42
	// Valid positive number: 100
}

// Example showing type safety with generics
func ExampleResult_typeSafety() {
	// Different types of Results
	intResult := NewResult(42)
	stringResult := NewResult("hello")

	// Type inference works correctly
	fmt.Printf("Integer: %d\n", intResult.Unwrap())
	fmt.Printf("String: %s\n", stringResult.Unwrap())

	// Optional with different types
	optionalAge := Some(25)
	optionalName := Some("Alice")

	fmt.Printf("Age: %d\n", optionalAge.UnwrapOr(0))
	fmt.Printf("Name: %s\n", optionalName.UnwrapOr("Unknown"))

	// Output:
	// Integer: 42
	// String: hello
	// Age: 25
	// Name: Alice
}
