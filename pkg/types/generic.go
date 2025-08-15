// Package types provides generic types and utilities for improved type safety
// and reduced code duplication across the mobilecombackup codebase.
package types

import (
	"strconv"
)

const (
	// XMLNullValue represents the null value in XML attributes
	XMLNullValue = "null"
)

// Result represents a generic operation result that can either contain a value or an error
type Result[T any] struct {
	Value T
	Error error
}

// NewResult creates a Result with a successful value
func NewResult[T any](value T) Result[T] {
	return Result[T]{Value: value}
}

// NewResultError creates a Result with an error
func NewResultError[T any](err error) Result[T] {
	var zero T
	return Result[T]{Value: zero, Error: err}
}

// IsOk returns true if the result contains a value (no error)
func (r Result[T]) IsOk() bool {
	return r.Error == nil
}

// IsErr returns true if the result contains an error
func (r Result[T]) IsErr() bool {
	return r.Error != nil
}

// Unwrap returns the value, panicking if there's an error
// Use this only when you're certain the result is Ok
func (r Result[T]) Unwrap() T {
	if r.Error != nil {
		panic("called Unwrap on Result with error: " + r.Error.Error())
	}
	return r.Value
}

// UnwrapOr returns the value if Ok, otherwise returns the provided default
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.Error != nil {
		return defaultValue
	}
	return r.Value
}

// Optional represents a value that may or may not be present
// This is useful for parsing XML attributes that might be "null" or empty
type Optional[T any] struct {
	value    T
	hasValue bool
}

// None creates an empty Optional
func None[T any]() Optional[T] {
	return Optional[T]{hasValue: false}
}

// Some creates an Optional with a value
func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, hasValue: true}
}

// IsSome returns true if the Optional contains a value
func (o Optional[T]) IsSome() bool {
	return o.hasValue
}

// IsNone returns true if the Optional is empty
func (o Optional[T]) IsNone() bool {
	return !o.hasValue
}

// Unwrap returns the value, panicking if empty
func (o Optional[T]) Unwrap() T {
	if !o.hasValue {
		panic("called Unwrap on empty Optional")
	}
	return o.value
}

// UnwrapOr returns the value if present, otherwise returns the provided default
func (o Optional[T]) UnwrapOr(defaultValue T) T {
	if !o.hasValue {
		return defaultValue
	}
	return o.value
}

// ParseOptionalInt parses a string to an optional integer
// Returns None if the string is empty, "null", or invalid
func ParseOptionalInt(s string) Optional[int] {
	if s == "" || s == XMLNullValue {
		return None[int]()
	}

	value, err := strconv.Atoi(s)
	if err != nil {
		return None[int]()
	}

	return Some(value)
}

// ParseOptionalInt64 parses a string to an optional int64
// Returns None if the string is empty, "null", or invalid
func ParseOptionalInt64(s string) Optional[int64] {
	if s == "" || s == XMLNullValue {
		return None[int64]()
	}

	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return None[int64]()
	}

	return Some(value)
}

// ParseOptionalString parses a string to an optional string
// Returns None if the string is empty or "null"
func ParseOptionalString(s string) Optional[string] {
	if s == "" || s == XMLNullValue {
		return None[string]()
	}

	return Some(s)
}

// TryParseInt attempts to parse a string as an integer, returning a Result
func TryParseInt(s string) Result[int] {
	value, err := strconv.Atoi(s)
	if err != nil {
		return NewResultError[int](err)
	}
	return NewResult(value)
}

// TryParseInt64 attempts to parse a string as an int64, returning a Result
func TryParseInt64(s string) Result[int64] {
	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return NewResultError[int64](err)
	}
	return NewResult(value)
}
