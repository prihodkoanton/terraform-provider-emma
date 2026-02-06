package errors

import (
	"errors"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestResourceError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ResourceError
		expected string
	}{
		{
			name: "error with resource ID",
			err: &ResourceError{
				ResourceType: "emma_volume",
				ResourceID:   "12345",
				Operation:    "Create",
				Message:      "volume creation failed",
			},
			expected: "Create emma_volume (ID: 12345) failed: volume creation failed",
		},
		{
			name: "error without resource ID",
			err: &ResourceError{
				ResourceType: "emma_vm",
				Operation:    "Update",
				Message:      "update operation failed",
			},
			expected: "Update emma_vm failed: update operation failed",
		},
		{
			name: "error with empty resource ID",
			err: &ResourceError{
				ResourceType: "emma_ssh_key",
				ResourceID:   "",
				Operation:    "Delete",
				Message:      "deletion failed",
			},
			expected: "Delete emma_ssh_key failed: deletion failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestErrorBuilder_FluentAPI(t *testing.T) {
	// Test fluent API chaining
	err := NewError("emma_volume", "Create").
		WithID("12345").
		WithStatusCode(400).
		WithAPIError("Invalid volume size").
		WithMessage("Volume size must be between 1 and 10000 GB").
		WithCause(errors.New("validation error")).
		Build()

	if err.ResourceType != "emma_volume" {
		t.Errorf("ResourceType = %q, want %q", err.ResourceType, "emma_volume")
	}
	if err.ResourceID != "12345" {
		t.Errorf("ResourceID = %q, want %q", err.ResourceID, "12345")
	}
	if err.Operation != "Create" {
		t.Errorf("Operation = %q, want %q", err.Operation, "Create")
	}
	if err.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want %d", err.StatusCode, 400)
	}
	if err.APIError != "Invalid volume size" {
		t.Errorf("APIError = %q, want %q", err.APIError, "Invalid volume size")
	}
	if err.Message != "Volume size must be between 1 and 10000 GB" {
		t.Errorf("Message = %q, want %q", err.Message, "Volume size must be between 1 and 10000 GB")
	}
	if err.Cause == nil || err.Cause.Error() != "validation error" {
		t.Errorf("Cause = %v, want validation error", err.Cause)
	}
}

func TestErrorBuilder_PartialBuild(t *testing.T) {
	// Test building error with only required fields
	err := NewError("emma_vm", "Read").
		WithMessage("resource not found").
		Build()

	if err.ResourceType != "emma_vm" {
		t.Errorf("ResourceType = %q, want %q", err.ResourceType, "emma_vm")
	}
	if err.Operation != "Read" {
		t.Errorf("Operation = %q, want %q", err.Operation, "Read")
	}
	if err.Message != "resource not found" {
		t.Errorf("Message = %q, want %q", err.Message, "resource not found")
	}
	if err.ResourceID != "" {
		t.Errorf("ResourceID = %q, want empty string", err.ResourceID)
	}
	if err.StatusCode != 0 {
		t.Errorf("StatusCode = %d, want 0", err.StatusCode)
	}
}

func TestErrorBuilder_ErrorContextInclusion(t *testing.T) {
	tests := []struct {
		name            string
		builder         *ErrorBuilder
		expectedInError []string
	}{
		{
			name: "includes resource type and operation",
			builder: NewError("emma_volume", "Create").
				WithMessage("creation failed"),
			expectedInError: []string{"Create", "emma_volume", "creation failed"},
		},
		{
			name: "includes resource ID when provided",
			builder: NewError("emma_vm", "Update").
				WithID("67890").
				WithMessage("update failed"),
			expectedInError: []string{"Update", "emma_vm", "ID: 67890", "update failed"},
		},
		{
			name: "includes all context fields",
			builder: NewError("emma_security_group", "Delete").
				WithID("sg-123").
				WithStatusCode(500).
				WithAPIError("Internal server error").
				WithMessage("deletion failed"),
			expectedInError: []string{"Delete", "emma_security_group", "ID: sg-123", "deletion failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.builder.Build()
			errorMsg := err.Error()

			for _, expected := range tt.expectedInError {
				if !strings.Contains(errorMsg, expected) {
					t.Errorf("Error message %q does not contain expected string %q", errorMsg, expected)
				}
			}
		})
	}
}

// Property-Based Tests

// Feature: provider-improvements, Property 1: Error Messages Include Context
// For any resource operation that fails, the error message should include
// the resource type, operation name, and resource ID (when available).
// Validates: Requirements 1.1, 1.4

func TestProperty_ErrorMessagesIncludeContext(t *testing.T) {
	// Import gopter for property-based testing
	properties := gopter.NewProperties(nil)

	// Property: All errors include resource type and operation
	properties.Property("error messages include resource type and operation", prop.ForAll(
		func(resourceType, operation, message string) bool {
			err := NewError(resourceType, operation).
				WithMessage(message).
				Build()

			errorMsg := err.Error()

			// Verify resource type is in error message
			if !strings.Contains(errorMsg, resourceType) {
				t.Logf("Error message missing resource type: %q not in %q", resourceType, errorMsg)
				return false
			}

			// Verify operation is in error message
			if !strings.Contains(errorMsg, operation) {
				t.Logf("Error message missing operation: %q not in %q", operation, errorMsg)
				return false
			}

			// Verify message is in error message
			if !strings.Contains(errorMsg, message) {
				t.Logf("Error message missing message: %q not in %q", message, errorMsg)
				return false
			}

			return true
		},
		gen.Identifier(),        // resourceType
		gen.Identifier(),        // operation
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }), // message
	))

	// Property: Errors with resource ID include the ID in the message
	properties.Property("error messages with ID include resource ID", prop.ForAll(
		func(resourceType, operation, resourceID, message string) bool {
			err := NewError(resourceType, operation).
				WithID(resourceID).
				WithMessage(message).
				Build()

			errorMsg := err.Error()

			// Only check for ID in message if resourceID is non-empty
			if resourceID != "" {
				// Verify resource ID is in error message
				if !strings.Contains(errorMsg, resourceID) {
					t.Logf("Error message missing resource ID: %q not in %q", resourceID, errorMsg)
					return false
				}

				// Verify "ID:" label is present
				if !strings.Contains(errorMsg, "ID:") {
					t.Logf("Error message missing 'ID:' label in %q", errorMsg)
					return false
				}
			} else {
				// When ID is empty, it should not appear in the error message
				if strings.Contains(errorMsg, "ID:") {
					t.Logf("Error message should not contain 'ID:' when resource ID is empty: %q", errorMsg)
					return false
				}
			}

			return true
		},
		gen.Identifier(),        // resourceType
		gen.Identifier(),        // operation
		gen.NumString(),         // resourceID (numeric string, can be empty)
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }), // message
	))

	// Property: Errors with status code preserve the status code
	properties.Property("error messages preserve status code", prop.ForAll(
		func(resourceType, operation, message string, statusCode int) bool {
			err := NewError(resourceType, operation).
				WithStatusCode(statusCode).
				WithMessage(message).
				Build()

			// Verify status code is preserved
			if err.StatusCode != statusCode {
				t.Logf("Status code not preserved: got %d, want %d", err.StatusCode, statusCode)
				return false
			}

			return true
		},
		gen.Identifier(),        // resourceType
		gen.Identifier(),        // operation
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }), // message
		gen.IntRange(100, 599),  // statusCode (valid HTTP status codes)
	))

	// Property: Errors with API error preserve the API error message
	properties.Property("error messages preserve API error", prop.ForAll(
		func(resourceType, operation, message, apiError string) bool {
			err := NewError(resourceType, operation).
				WithAPIError(apiError).
				WithMessage(message).
				Build()

			// Verify API error is preserved
			if err.APIError != apiError {
				t.Logf("API error not preserved: got %q, want %q", err.APIError, apiError)
				return false
			}

			return true
		},
		gen.Identifier(),        // resourceType
		gen.Identifier(),        // operation
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }), // message
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }), // apiError
	))

	// Run all properties with 100 iterations each
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
