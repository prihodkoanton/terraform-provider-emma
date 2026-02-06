package emma

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	commonErrors "github.com/emma-community/terraform-provider-emma/internal/emma/common/errors"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/retry"
)

// Integration test for retry behavior with provider configuration
// Validates: Requirements 9.1, 9.4, 9.5
func TestProvider_RetryBehavior_Integration(t *testing.T) {
	t.Run("Retry on transient errors", func(t *testing.T) {
		// Test that transient errors (5xx, 429, 503) are retried
		ctx := context.Background()
		
		// Create a retry config with short delays for testing
		retryConfig := retry.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			ShouldRetry: func(err error) bool {
				// Simulate checking if error is retryable
				return true
			},
		}
		
		// Simulate an operation that fails twice then succeeds
		attemptCount := 0
		operation := func() error {
			attemptCount++
			if attemptCount < 3 {
				return fmt.Errorf("transient error: attempt %d", attemptCount)
			}
			return nil
		}
		
		err := retry.Retry(ctx, retryConfig, operation)
		
		if err != nil {
			t.Errorf("Expected operation to succeed after retries, got error: %v", err)
		}
		
		if attemptCount != 3 {
			t.Errorf("Expected 3 attempts, got %d", attemptCount)
		}
		
		t.Logf("Operation succeeded after %d attempts", attemptCount)
	})
	
	t.Run("No retry on permanent errors", func(t *testing.T) {
		// Test that permanent errors (4xx except 429) are not retried
		ctx := context.Background()
		
		// Create a retry config that checks for retryable status codes
		retryConfig := retry.RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			ShouldRetry: func(err error) bool {
				// Simulate checking if error is retryable based on status code
				// For 400 Bad Request, should not retry
				return false
			},
		}
		
		// Simulate an operation that always fails with a permanent error
		attemptCount := 0
		operation := func() error {
			attemptCount++
			return fmt.Errorf("permanent error: bad request")
		}
		
		err := retry.Retry(ctx, retryConfig, operation)
		
		if err == nil {
			t.Error("Expected operation to fail with permanent error")
		}
		
		if attemptCount != 1 {
			t.Errorf("Expected 1 attempt (no retries), got %d", attemptCount)
		}
		
		t.Logf("Operation failed immediately without retries: %v", err)
	})
	
	t.Run("Max attempts respected", func(t *testing.T) {
		// Test that retry logic respects max attempts configuration
		ctx := context.Background()
		
		maxAttempts := 5
		retryConfig := retry.RetryConfig{
			MaxAttempts:  maxAttempts,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     100 * time.Millisecond,
			Multiplier:   2.0,
			ShouldRetry: func(err error) bool {
				return true
			},
		}
		
		// Simulate an operation that always fails
		attemptCount := 0
		operation := func() error {
			attemptCount++
			return fmt.Errorf("persistent error: attempt %d", attemptCount)
		}
		
		err := retry.Retry(ctx, retryConfig, operation)
		
		if err == nil {
			t.Error("Expected operation to fail after max attempts")
		}
		
		if attemptCount != maxAttempts {
			t.Errorf("Expected %d attempts, got %d", maxAttempts, attemptCount)
		}
		
		t.Logf("Operation failed after %d attempts (max attempts respected)", attemptCount)
	})
}

// Integration test for IsRetryable function with various HTTP status codes
// Validates: Requirements 9.1, 9.2, 9.3, 9.4
func TestProvider_IsRetryable_Integration(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		shouldRetry    bool
		description    string
	}{
		{
			name:        "400 Bad Request - not retryable",
			statusCode:  http.StatusBadRequest,
			shouldRetry: false,
			description: "Client error - should not retry",
		},
		{
			name:        "401 Unauthorized - not retryable",
			statusCode:  http.StatusUnauthorized,
			shouldRetry: false,
			description: "Authentication error - should not retry",
		},
		{
			name:        "403 Forbidden - not retryable",
			statusCode:  http.StatusForbidden,
			shouldRetry: false,
			description: "Permission error - should not retry",
		},
		{
			name:        "404 Not Found - not retryable",
			statusCode:  http.StatusNotFound,
			shouldRetry: false,
			description: "Resource not found - should not retry",
		},
		{
			name:        "429 Too Many Requests - retryable",
			statusCode:  http.StatusTooManyRequests,
			shouldRetry: true,
			description: "Rate limit - should retry",
		},
		{
			name:        "500 Internal Server Error - retryable",
			statusCode:  http.StatusInternalServerError,
			shouldRetry: true,
			description: "Server error - should retry",
		},
		{
			name:        "502 Bad Gateway - retryable",
			statusCode:  http.StatusBadGateway,
			shouldRetry: true,
			description: "Gateway error - should retry",
		},
		{
			name:        "503 Service Unavailable - retryable",
			statusCode:  http.StatusServiceUnavailable,
			shouldRetry: true,
			description: "Service unavailable - should retry",
		},
		{
			name:        "504 Gateway Timeout - retryable",
			statusCode:  http.StatusGatewayTimeout,
			shouldRetry: true,
			description: "Gateway timeout - should retry",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isRetryable := commonErrors.IsRetryable(tc.statusCode)
			
			if isRetryable != tc.shouldRetry {
				t.Errorf("Expected IsRetryable(%d) to be %v, got %v - %s",
					tc.statusCode, tc.shouldRetry, isRetryable, tc.description)
			}
			
			t.Logf("Status %d: retryable=%v - %s", tc.statusCode, isRetryable, tc.description)
		})
	}
}

// Integration test for Client.WithRetry method
// Validates: Requirements 9.1, 9.2, 9.3, 9.5
func TestClient_WithRetry_Integration(t *testing.T) {
	t.Run("Client WithRetry wraps operations correctly", func(t *testing.T) {
		ctx := context.Background()
		
		// Create a client with custom retry config
		client := &Client{
			retryConfig: retry.RetryConfig{
				MaxAttempts:  3,
				InitialDelay: 10 * time.Millisecond,
				MaxDelay:     100 * time.Millisecond,
				Multiplier:   2.0,
				ShouldRetry: func(err error) bool {
					return true
				},
			},
		}
		
		// Test successful operation after retries
		attemptCount := 0
		err := client.WithRetry(ctx, func() error {
			attemptCount++
			if attemptCount < 2 {
				return errors.New("temporary failure")
			}
			return nil
		})
		
		if err != nil {
			t.Errorf("Expected operation to succeed, got error: %v", err)
		}
		
		if attemptCount != 2 {
			t.Errorf("Expected 2 attempts, got %d", attemptCount)
		}
		
		t.Logf("Client.WithRetry succeeded after %d attempts", attemptCount)
	})
	
	t.Run("Client IsRetryableStatusCode works correctly", func(t *testing.T) {
		client := &Client{}
		
		// Test retryable status codes
		if !client.IsRetryableStatusCode(http.StatusTooManyRequests) {
			t.Error("Expected 429 to be retryable")
		}
		
		if !client.IsRetryableStatusCode(http.StatusServiceUnavailable) {
			t.Error("Expected 503 to be retryable")
		}
		
		if !client.IsRetryableStatusCode(http.StatusInternalServerError) {
			t.Error("Expected 500 to be retryable")
		}
		
		// Test non-retryable status codes
		if client.IsRetryableStatusCode(http.StatusBadRequest) {
			t.Error("Expected 400 to not be retryable")
		}
		
		if client.IsRetryableStatusCode(http.StatusNotFound) {
			t.Error("Expected 404 to not be retryable")
		}
		
		t.Log("Client.IsRetryableStatusCode works correctly for all status codes")
	})
}
