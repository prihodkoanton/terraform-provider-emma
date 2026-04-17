package errors

import (
	"fmt"
)

// ResourceError represents a provider error with context
type ResourceError struct {
	ResourceType string
	ResourceID   string
	Operation    string
	StatusCode   int
	APIError     string
	Message      string
	Cause        error
}

// Error implements the error interface
func (e *ResourceError) Error() string {
	if e.ResourceID != "" {
		return fmt.Sprintf("%s %s (ID: %s) failed: %s", e.Operation, e.ResourceType, e.ResourceID, e.Message)
	}
	return fmt.Sprintf("%s %s failed: %s", e.Operation, e.ResourceType, e.Message)
}

// ErrorBuilder provides fluent API for building errors
type ErrorBuilder struct {
	err *ResourceError
}

// NewError creates a new ErrorBuilder with resource type and operation
func NewError(resourceType, operation string) *ErrorBuilder {
	return &ErrorBuilder{
		err: &ResourceError{
			ResourceType: resourceType,
			Operation:    operation,
		},
	}
}

// WithID sets the resource ID
func (b *ErrorBuilder) WithID(id string) *ErrorBuilder {
	b.err.ResourceID = id
	return b
}

// WithStatusCode sets the HTTP status code
func (b *ErrorBuilder) WithStatusCode(code int) *ErrorBuilder {
	b.err.StatusCode = code
	return b
}

// WithAPIError sets the API error message
func (b *ErrorBuilder) WithAPIError(apiErr string) *ErrorBuilder {
	b.err.APIError = apiErr
	return b
}

// WithMessage sets the user-friendly message
func (b *ErrorBuilder) WithMessage(msg string) *ErrorBuilder {
	b.err.Message = msg
	return b
}

// WithCause sets the underlying error
func (b *ErrorBuilder) WithCause(err error) *ErrorBuilder {
	b.err.Cause = err
	return b
}

// Build returns the constructed ResourceError
func (b *ErrorBuilder) Build() *ResourceError {
	return b.err
}
