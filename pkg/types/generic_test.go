package types

import (
	"errors"
	"testing"
)

// Test Result type
func TestResult_NewResult(t *testing.T) {
	result := NewResult(42)

	if !result.IsOk() {
		t.Error("Expected result to be Ok")
	}

	if result.IsErr() {
		t.Error("Expected result to not be error")
	}

	if result.Value != 42 {
		t.Errorf("Expected value 42, got %d", result.Value)
	}

	if result.Unwrap() != 42 {
		t.Errorf("Expected unwrap to return 42, got %d", result.Unwrap())
	}
}

func TestResult_NewResultError(t *testing.T) {
	err := errors.New("test error")
	result := NewResultError[string](err)

	if result.IsOk() {
		t.Error("Expected result to not be Ok")
	}

	if !result.IsErr() {
		t.Error("Expected result to be error")
	}

	if result.Error != err {
		t.Errorf("Expected error %v, got %v", err, result.Error)
	}
}

func TestResult_UnwrapOr(t *testing.T) {
	// Test with Ok result
	okResult := NewResult(42)
	if okResult.UnwrapOr(100) != 42 {
		t.Error("Expected UnwrapOr to return actual value for Ok result")
	}

	// Test with Error result
	errResult := NewResultError[int](errors.New("error"))
	if errResult.UnwrapOr(100) != 100 {
		t.Error("Expected UnwrapOr to return default value for Error result")
	}
}

func TestResult_UnwrapPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected Unwrap to panic on error result")
		}
	}()

	errResult := NewResultError[int](errors.New("error"))
	errResult.Unwrap()
}

// Test Optional type
func TestOptional_Some(t *testing.T) {
	opt := Some(42)

	if !opt.IsSome() {
		t.Error("Expected optional to be Some")
	}

	if opt.IsNone() {
		t.Error("Expected optional to not be None")
	}

	if opt.Unwrap() != 42 {
		t.Errorf("Expected unwrap to return 42, got %d", opt.Unwrap())
	}
}

func TestOptional_None(t *testing.T) {
	opt := None[int]()

	if opt.IsSome() {
		t.Error("Expected optional to not be Some")
	}

	if !opt.IsNone() {
		t.Error("Expected optional to be None")
	}
}

func TestOptional_UnwrapOr(t *testing.T) {
	// Test with Some
	someOpt := Some(42)
	if someOpt.UnwrapOr(100) != 42 {
		t.Error("Expected UnwrapOr to return actual value for Some")
	}

	// Test with None
	noneOpt := None[int]()
	if noneOpt.UnwrapOr(100) != 100 {
		t.Error("Expected UnwrapOr to return default value for None")
	}
}

func TestOptional_UnwrapPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected Unwrap to panic on None optional")
		}
	}()

	noneOpt := None[int]()
	noneOpt.Unwrap()
}

// Test parsing functions
func TestParseOptionalInt(t *testing.T) {
	tests := []struct {
		input    string
		expected Optional[int]
		wantSome bool
	}{
		{"42", Some(42), true},
		{"0", Some(0), true},
		{"-1", Some(-1), true},
		{"", None[int](), false},
		{"null", None[int](), false},
		{"invalid", None[int](), false},
		{"12.34", None[int](), false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseOptionalInt(tt.input)

			if tt.wantSome {
				if result.IsNone() {
					t.Errorf("Expected Some for input %q, got None", tt.input)
				}
				if result.Unwrap() != tt.expected.Unwrap() {
					t.Errorf("Expected %d, got %d", tt.expected.Unwrap(), result.Unwrap())
				}
			} else if result.IsSome() {
				t.Errorf("Expected None for input %q, got Some(%d)", tt.input, result.Unwrap())
			}
		})
	}
}

func TestParseOptionalInt64(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantSome bool
	}{
		{"9223372036854775807", 9223372036854775807, true}, // max int64
		{"0", 0, true},
		{"-9223372036854775808", -9223372036854775808, true}, // min int64
		{"", 0, false},
		{"null", 0, false},
		{"invalid", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseOptionalInt64(tt.input)

			if tt.wantSome {
				if result.IsNone() {
					t.Errorf("Expected Some for input %q, got None", tt.input)
				}
				if result.Unwrap() != tt.expected {
					t.Errorf("Expected %d, got %d", tt.expected, result.Unwrap())
				}
			} else if result.IsSome() {
				t.Errorf("Expected None for input %q, got Some", tt.input)
			}
		})
	}
}

func TestParseOptionalString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantSome bool
	}{
		{"hello", "hello", true},
		{"world", "world", true},
		{"0", "0", true}, // string "0" is valid
		{"", "", false},
		{"null", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseOptionalString(tt.input)

			if tt.wantSome {
				if result.IsNone() {
					t.Errorf("Expected Some for input %q, got None", tt.input)
				}
				if result.Unwrap() != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result.Unwrap())
				}
			} else if result.IsSome() {
				t.Errorf("Expected None for input %q, got Some", tt.input)
			}
		})
	}
}

func TestTryParseInt(t *testing.T) {
	tests := []struct {
		input     string
		expected  int
		wantError bool
	}{
		{"42", 42, false},
		{"0", 0, false},
		{"-1", -1, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"12.34", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TryParseInt(tt.input)

			if tt.wantError {
				if result.IsOk() {
					t.Errorf("Expected error for input %q, got Ok(%d)", tt.input, result.Value)
				}
			} else {
				if result.IsErr() {
					t.Errorf("Expected Ok for input %q, got error: %v", tt.input, result.Error)
				}
				if result.Value != tt.expected {
					t.Errorf("Expected %d, got %d", tt.expected, result.Value)
				}
			}
		})
	}
}

func TestTryParseInt64(t *testing.T) {
	tests := []struct {
		input     string
		expected  int64
		wantError bool
	}{
		{"9223372036854775807", 9223372036854775807, false},
		{"0", 0, false},
		{"-9223372036854775808", -9223372036854775808, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := TryParseInt64(tt.input)

			if tt.wantError {
				if result.IsOk() {
					t.Errorf("Expected error for input %q, got Ok", tt.input)
				}
			} else {
				if result.IsErr() {
					t.Errorf("Expected Ok for input %q, got error: %v", tt.input, result.Error)
				}
				if result.Value != tt.expected {
					t.Errorf("Expected %d, got %d", tt.expected, result.Value)
				}
			}
		})
	}
}

// Test generic type with different types
func TestGenericTypesWithDifferentTypes(t *testing.T) {
	// Test Result with string
	stringResult := NewResult("hello")
	if stringResult.Unwrap() != "hello" {
		t.Error("String result failed")
	}

	// Test Result with struct
	type TestStruct struct {
		Name string
		Age  int
	}

	person := TestStruct{Name: "Alice", Age: 30}
	structResult := NewResult(person)
	if structResult.Unwrap().Name != "Alice" {
		t.Error("Struct result failed")
	}

	// Test Optional with custom type
	optionalPerson := Some(person)
	if optionalPerson.Unwrap().Age != 30 {
		t.Error("Optional struct failed")
	}
}
