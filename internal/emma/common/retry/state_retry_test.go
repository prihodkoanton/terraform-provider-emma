package retry

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestIsStateConflictError_409StatusCode tests detection of 409 Conflict status code
// Requirements: 5.1
func TestIsStateConflictError_409StatusCode(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		apiError   string
		expected   bool
	}{
		{
			name:       "409 status code with any message",
			err:        errors.New("conflict error"),
			statusCode: http.StatusConflict,
			apiError:   "some conflict message",
			expected:   true,
		},
		{
			name:       "409 status code with empty message",
			err:        errors.New("conflict error"),
			statusCode: http.StatusConflict,
			apiError:   "",
			expected:   true,
		},
		{
			name:       "409 status code with nil error",
			err:        nil,
			statusCode: http.StatusConflict,
			apiError:   "conflict",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStateConflictError(tt.err, tt.statusCode, tt.apiError)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsStateConflictError_StateRelatedMessages tests detection of state-related error messages
// Requirements: 5.2
func TestIsStateConflictError_StateRelatedMessages(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		apiError   string
		expected   bool
	}{
		{
			name:       "message contains 'state'",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "Resource is in invalid state",
			expected:   true,
		},
		{
			name:       "message contains 'busy'",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "Resource is busy",
			expected:   true,
		},
		{
			name:       "message contains 'recomposing'",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "Security group is recomposing",
			expected:   true,
		},
		{
			name:       "message contains 'inappropriate compute instance state'",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "Inappropriate compute instance state for operation",
			expected:   true,
		},
		{
			name:       "message contains 'resource conflict'",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "Resource conflict detected",
			expected:   true,
		},
		{
			name:       "case insensitive - uppercase STATE",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "Resource is in invalid STATE",
			expected:   true,
		},
		{
			name:       "case insensitive - uppercase BUSY",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "Resource is BUSY",
			expected:   true,
		},
		{
			name:       "case insensitive - mixed case ReComposing",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "Security group is ReComposing",
			expected:   true,
		},
		{
			name:       "message with state keyword in middle",
			err:        errors.New("error"),
			statusCode: http.StatusInternalServerError,
			apiError:   "Cannot perform operation: resource state is transitioning",
			expected:   true,
		},
		{
			name:       "message with busy keyword in middle",
			err:        errors.New("error"),
			statusCode: http.StatusInternalServerError,
			apiError:   "VM is currently busy processing another request",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStateConflictError(tt.err, tt.statusCode, tt.apiError)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsStateConflictError_NonRetryableErrors tests detection of non-retryable errors
// Requirements: 5.3
func TestIsStateConflictError_NonRetryableErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		apiError   string
		expected   bool
	}{
		{
			name:       "400 Bad Request without state keywords",
			err:        errors.New("bad request"),
			statusCode: http.StatusBadRequest,
			apiError:   "Invalid parameter value",
			expected:   false,
		},
		{
			name:       "401 Unauthorized",
			err:        errors.New("unauthorized"),
			statusCode: http.StatusUnauthorized,
			apiError:   "Authentication failed",
			expected:   false,
		},
		{
			name:       "403 Forbidden",
			err:        errors.New("forbidden"),
			statusCode: http.StatusForbidden,
			apiError:   "Access denied",
			expected:   false,
		},
		{
			name:       "404 Not Found",
			err:        errors.New("not found"),
			statusCode: http.StatusNotFound,
			apiError:   "Resource not found",
			expected:   false,
		},
		{
			name:       "422 Unprocessable Entity",
			err:        errors.New("validation error"),
			statusCode: http.StatusUnprocessableEntity,
			apiError:   "Validation failed",
			expected:   false,
		},
		{
			name:       "500 Internal Server Error without state keywords",
			err:        errors.New("server error"),
			statusCode: http.StatusInternalServerError,
			apiError:   "Internal server error",
			expected:   false,
		},
		{
			name:       "503 Service Unavailable",
			err:        errors.New("service unavailable"),
			statusCode: http.StatusServiceUnavailable,
			apiError:   "Service temporarily unavailable",
			expected:   false,
		},
		{
			name:       "200 OK (success status)",
			err:        nil,
			statusCode: http.StatusOK,
			apiError:   "",
			expected:   false,
		},
		{
			name:       "empty error message",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "",
			expected:   false,
		},
		{
			name:       "message without state keywords",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "Invalid configuration provided",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStateConflictError(tt.err, tt.statusCode, tt.apiError)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsStateConflictError_EdgeCases tests edge cases and boundary conditions
func TestIsStateConflictError_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		apiError   string
		expected   bool
	}{
		{
			name:       "409 with state keyword (both conditions true)",
			err:        errors.New("conflict"),
			statusCode: http.StatusConflict,
			apiError:   "Resource is in busy state",
			expected:   true,
		},
		{
			name:       "zero status code with state keyword",
			err:        errors.New("error"),
			statusCode: 0,
			apiError:   "Resource state is invalid",
			expected:   true,
		},
		{
			name:       "negative status code",
			err:        errors.New("error"),
			statusCode: -1,
			apiError:   "some error",
			expected:   false,
		},
		{
			name:       "very long error message with state keyword",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "This is a very long error message that contains many words and eventually mentions that the resource state is invalid somewhere in the middle of this lengthy description",
			expected:   true,
		},
		{
			name:       "state keyword as part of another word",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "The statement is invalid",
			expected:   true, // Should still match because we use Contains
		},
		{
			name:       "multiple state keywords",
			err:        errors.New("error"),
			statusCode: http.StatusBadRequest,
			apiError:   "Resource is busy and in recomposing state",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStateConflictError(tt.err, tt.statusCode, tt.apiError)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestStateConflictRetryConfig tests the configuration for state conflict retries
// Requirements: 5.4, 5.5
func TestStateConflictRetryConfig(t *testing.T) {
	config := StateConflictRetryConfig()

	// Verify configuration values
	assert.Equal(t, 5, config.MaxAttempts, "MaxAttempts should be 5")
	assert.Equal(t, 2*time.Second, config.InitialDelay, "InitialDelay should be 2 seconds")
	assert.Equal(t, 30*time.Second, config.MaxDelay, "MaxDelay should be 30 seconds")
	assert.Equal(t, 2.0, config.Multiplier, "Multiplier should be 2.0")
	assert.NotNil(t, config.ShouldRetry, "ShouldRetry function should not be nil")

	// Test that ShouldRetry returns true (default behavior for state conflicts)
	testErr := errors.New("test error")
	assert.True(t, config.ShouldRetry(testErr), "ShouldRetry should return true by default")
}

// TestStateConflictRetryConfig_ExponentialBackoff tests the exponential backoff behavior
func TestStateConflictRetryConfig_ExponentialBackoff(t *testing.T) {
	config := StateConflictRetryConfig()

	// Verify exponential backoff parameters are suitable for state conflicts
	// Initial: 2s, then 4s, 8s, 16s, 30s (capped at MaxDelay)
	assert.Equal(t, 2*time.Second, config.InitialDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.Multiplier)

	// Verify max attempts allows sufficient retries for transient state issues
	assert.Equal(t, 5, config.MaxAttempts)
}
