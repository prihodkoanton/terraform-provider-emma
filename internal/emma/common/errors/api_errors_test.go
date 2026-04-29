package errors

import (
	"net/http"
	"strings"
	"testing"
)

func TestMapHTTPError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		apiMessage     string
		expectedSubstr []string
	}{
		{
			name:           "400 Bad Request",
			statusCode:     http.StatusBadRequest,
			apiMessage:     "missing required field",
			expectedSubstr: []string{"Invalid request", "missing required field"},
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			apiMessage:     "invalid token",
			expectedSubstr: []string{"Authentication failed", "credentials"},
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			apiMessage:     "access denied",
			expectedSubstr: []string{"Permission denied", "access"},
		},
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			apiMessage:     "resource does not exist",
			expectedSubstr: []string{"Resource not found", "deleted"},
		},
		{
			name:           "409 Conflict",
			statusCode:     http.StatusConflict,
			apiMessage:     "resource already exists",
			expectedSubstr: []string{"Resource conflict", "resource already exists"},
		},
		{
			name:           "422 Unprocessable Entity",
			statusCode:     http.StatusUnprocessableEntity,
			apiMessage:     "invalid volume size",
			expectedSubstr: []string{"Validation error", "invalid volume size"},
		},
		{
			name:           "429 Too Many Requests",
			statusCode:     http.StatusTooManyRequests,
			apiMessage:     "rate limit exceeded",
			expectedSubstr: []string{"Rate limit exceeded", "try again later"},
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			apiMessage:     "database connection failed",
			expectedSubstr: []string{"Server error", "try again", "contact support"},
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			apiMessage:     "maintenance in progress",
			expectedSubstr: []string{"Service temporarily unavailable", "try again later"},
		},
		{
			name:           "Unknown status code",
			statusCode:     418,
			apiMessage:     "I'm a teapot",
			expectedSubstr: []string{"API error", "status 418", "I'm a teapot"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapHTTPError(tt.statusCode, tt.apiMessage)

			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(result, substr) {
					t.Errorf("MapHTTPError(%d, %q) = %q, expected to contain %q",
						tt.statusCode, tt.apiMessage, result, substr)
				}
			}
		})
	}
}

func TestMapHTTPError_MessageFormatting(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		apiMessage string
		wantPrefix string
	}{
		{
			name:       "400 includes API message",
			statusCode: http.StatusBadRequest,
			apiMessage: "custom error",
			wantPrefix: "Invalid request: custom error",
		},
		{
			name:       "409 includes API message",
			statusCode: http.StatusConflict,
			apiMessage: "duplicate name",
			wantPrefix: "Resource conflict: duplicate name",
		},
		{
			name:       "422 includes API message",
			statusCode: http.StatusUnprocessableEntity,
			apiMessage: "field validation failed",
			wantPrefix: "Validation error: field validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapHTTPError(tt.statusCode, tt.apiMessage)
			if result != tt.wantPrefix {
				t.Errorf("MapHTTPError(%d, %q) = %q, want %q",
					tt.statusCode, tt.apiMessage, result, tt.wantPrefix)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		// Retryable errors
		{
			name:       "429 Too Many Requests is retryable",
			statusCode: http.StatusTooManyRequests,
			want:       true,
		},
		{
			name:       "500 Internal Server Error is retryable",
			statusCode: http.StatusInternalServerError,
			want:       true,
		},
		{
			name:       "502 Bad Gateway is retryable",
			statusCode: http.StatusBadGateway,
			want:       true,
		},
		{
			name:       "503 Service Unavailable is retryable",
			statusCode: http.StatusServiceUnavailable,
			want:       true,
		},
		{
			name:       "504 Gateway Timeout is retryable",
			statusCode: http.StatusGatewayTimeout,
			want:       true,
		},
		// Non-retryable errors
		{
			name:       "400 Bad Request is not retryable",
			statusCode: http.StatusBadRequest,
			want:       false,
		},
		{
			name:       "401 Unauthorized is not retryable",
			statusCode: http.StatusUnauthorized,
			want:       false,
		},
		{
			name:       "403 Forbidden is not retryable",
			statusCode: http.StatusForbidden,
			want:       false,
		},
		{
			name:       "404 Not Found is not retryable",
			statusCode: http.StatusNotFound,
			want:       false,
		},
		{
			name:       "409 Conflict is not retryable",
			statusCode: http.StatusConflict,
			want:       false,
		},
		{
			name:       "422 Unprocessable Entity is not retryable",
			statusCode: http.StatusUnprocessableEntity,
			want:       false,
		},
		{
			name:       "200 OK is not retryable",
			statusCode: http.StatusOK,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.statusCode)
			if result != tt.want {
				t.Errorf("IsRetryable(%d) = %v, want %v", tt.statusCode, result, tt.want)
			}
		})
	}
}

func TestIsRetryable_RetryDecisionLogic(t *testing.T) {
	// Test the retry decision logic comprehensively
	retryableCodes := []int{
		http.StatusTooManyRequests,      // 429
		http.StatusInternalServerError,  // 500
		http.StatusBadGateway,           // 502
		http.StatusServiceUnavailable,   // 503
		http.StatusGatewayTimeout,       // 504
	}

	nonRetryableCodes := []int{
		http.StatusBadRequest,          // 400
		http.StatusUnauthorized,        // 401
		http.StatusForbidden,           // 403
		http.StatusNotFound,            // 404
		http.StatusConflict,            // 409
		http.StatusUnprocessableEntity, // 422
	}

	for _, code := range retryableCodes {
		if !IsRetryable(code) {
			t.Errorf("IsRetryable(%d) = false, want true (should be retryable)", code)
		}
	}

	for _, code := range nonRetryableCodes {
		if IsRetryable(code) {
			t.Errorf("IsRetryable(%d) = true, want false (should not be retryable)", code)
		}
	}
}
