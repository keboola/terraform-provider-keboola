package test

import (
	"fmt"
)

// Error represents attribute mismatch errors in test validations.
type Error struct {
	AttributeName string
	ExpectedValue string
	ActualValue   string
}

// NewAttributeMismatchError creates a new Error for attribute mismatches.
func NewAttributeMismatchError(attributeName, expected, actual string) *Error {
	return &Error{
		AttributeName: attributeName,
		ExpectedValue: expected,
		ActualValue:   actual,
	}
}

// Error implements the error interface for Error.
func (e *Error) Error() string {
	return fmt.Sprintf(
		"Stored configuration doesn't match state, attribute: %s \n expected: %s \n actual: %s \n",
		e.AttributeName,
		e.ExpectedValue,
		e.ActualValue,
	)
}

// ResourceNotFoundError represents a resource not found error.
type ResourceNotFoundError struct {
	ResourceName string
}

// NewResourceNotFoundError creates a new ResourceNotFoundError.
func NewResourceNotFoundError(resourceName string) *ResourceNotFoundError {
	return &ResourceNotFoundError{
		ResourceName: resourceName,
	}
}

// Error implements the error interface for ResourceNotFoundError.
func (e *ResourceNotFoundError) Error() string {
	return "Not found: " + e.ResourceName
}

// ConfigParseError represents an error parsing configuration.
type ConfigParseError struct {
	OriginalError error
}

// NewConfigParseError creates a new ConfigParseError.
func NewConfigParseError(err error) *ConfigParseError {
	return &ConfigParseError{
		OriginalError: err,
	}
}

// Error implements the error interface for ConfigParseError.
func (e *ConfigParseError) Error() string {
	return fmt.Sprintf("Could not parse configuration: %v", e.OriginalError)
}

// PathNotFoundError represents an error when a path is not found in config.
type PathNotFoundError struct {
	Path string
}

// NewPathNotFoundError creates a new PathNotFoundError.
func NewPathNotFoundError(path string) *PathNotFoundError {
	return &PathNotFoundError{
		Path: path,
	}
}

// Error implements the error interface for PathNotFoundError.
func (e *PathNotFoundError) Error() string {
	return fmt.Sprintf("Get path %s not found", e.Path)
}

// PathValueMismatchError represents an error when a path value doesn't match.
type PathValueMismatchError struct {
	Path     string
	Expected string
	Actual   interface{}
}

// NewPathValueMismatchError creates a new PathValueMismatchError.
func NewPathValueMismatchError(path, expected string, actual interface{}) *PathValueMismatchError {
	return &PathValueMismatchError{
		Path:     path,
		Expected: expected,
		Actual:   actual,
	}
}

// Error implements the error interface for PathValueMismatchError.
func (e *PathValueMismatchError) Error() string {
	return fmt.Sprintf("Get path %s value didn't match: expected: %s found: %v",
		e.Path, e.Expected, e.Actual)
}
