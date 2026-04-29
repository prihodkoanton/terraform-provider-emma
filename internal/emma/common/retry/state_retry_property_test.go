package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: async-operations, Property 4: State Conflict Errors Trigger Retry
// Validates: Requirements 5.1, 5.2, 5.3
func TestProperty_StateConflictErrorsTriggersRetry(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any state conflict error, retry should be triggered with exponential backoff", prop.ForAll(
		func(statusCode int, errorKeywordIndex int, maxAttempts int) bool {
			// Ensure valid inputs
			if maxAttempts < 2 || maxAttempts > 10 {
				return true // Skip invalid inputs
			}

			// Map error keyword index to actual keyword
			keywords := []string{"state", "busy", "recomposing", "inappropriate compute instance state", "resource conflict"}
			keyword := keywords[errorKeywordIndex%len(keywords)]

			// Create error message with the keyword
			apiError := fmt.Sprintf("Operation failed: resource %s issue", keyword)

			// Check if this should be detected as a state conflict
			isStateConflict := IsStateConflictError(errors.New("test error"), statusCode, apiError)

			// For 409 status or messages with state keywords, should be detected as conflict
			expectedConflict := statusCode == http.StatusConflict || containsStateKeyword(apiError)

			if isStateConflict != expectedConflict {
				return false
			}

			// If it's a state conflict, test that retry is triggered
			if isStateConflict {
				attemptCount := 0

				config := StateConflictRetryConfig()
				config.MaxAttempts = maxAttempts
				config.InitialDelay = 1 * time.Millisecond // Speed up test
				config.MaxDelay = 10 * time.Millisecond
				config.ShouldRetry = func(err error) bool {
					return IsStateConflictError(err, statusCode, apiError)
				}

				operation := func() error {
					attemptCount++
					return errors.New(apiError)
				}

				ctx := context.Background()
				startTime := time.Now()
				err := Retry(ctx, config, operation)
				duration := time.Since(startTime)

				// Verify that:
				// 1. An error occurred (operation always fails)
				if err == nil {
					return false
				}

				// 2. Multiple attempts were made (retry was triggered)
				if attemptCount != maxAttempts {
					return false
				}

				// 3. The operation took some time (indicating delays between retries)
				// With exponential backoff: 1ms, 2ms, 4ms, 8ms... (capped at 10ms)
				// For maxAttempts=5: ~1+2+4+8 = 15ms minimum
				minExpectedDuration := time.Duration(0)
				for i := 0; i < maxAttempts-1; i++ {
					delay := calculateDelayHelper(i, config)
					minExpectedDuration += delay
				}

				// Allow some tolerance for test execution overhead
				if duration < minExpectedDuration/2 {
					return false
				}

				// 4. Error message indicates multiple attempts
				if attemptCount > 1 && err.Error() == apiError {
					// Should be wrapped with attempt count
					return false
				}
			}

			return true
		},
		gen.OneConstOf(
			http.StatusConflict,
			http.StatusBadRequest,
			http.StatusInternalServerError,
			http.StatusOK,
		),
		gen.IntRange(0, 4),  // Index for error keyword
		gen.IntRange(2, 10), // max attempts
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 4: State Conflict Errors Trigger Retry (409 Status)
// Validates: Requirements 5.1
func TestProperty_409StatusTriggersRetry(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any 409 status code, retry should be triggered regardless of error message", prop.ForAll(
		func(errorMessage string, maxAttempts int) bool {
			// Ensure valid inputs
			if maxAttempts < 2 || maxAttempts > 10 {
				return true // Skip invalid inputs
			}
			if errorMessage == "" {
				errorMessage = "conflict error"
			}

			// Verify 409 is detected as state conflict
			isConflict := IsStateConflictError(errors.New("test"), http.StatusConflict, errorMessage)
			if !isConflict {
				return false
			}

			// Test that retry is triggered
			attemptCount := 0

			config := StateConflictRetryConfig()
			config.MaxAttempts = maxAttempts
			config.InitialDelay = 1 * time.Millisecond
			config.MaxDelay = 10 * time.Millisecond
			config.ShouldRetry = func(err error) bool {
				return IsStateConflictError(err, http.StatusConflict, errorMessage)
			}

			operation := func() error {
				attemptCount++
				return errors.New(errorMessage)
			}

			ctx := context.Background()
			err := Retry(ctx, config, operation)

			// Verify retry was triggered
			return err != nil && attemptCount == maxAttempts
		},
		gen.AnyString(),
		gen.IntRange(2, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 4: State Conflict Errors Trigger Retry (State Keywords)
// Validates: Requirements 5.2
func TestProperty_StateKeywordsTriggersRetry(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any error message with state keywords, retry should be triggered", prop.ForAll(
		func(keywordIndex int, prefix string, suffix string, maxAttempts int) bool {
			// Ensure valid inputs
			if maxAttempts < 2 || maxAttempts > 10 {
				return true // Skip invalid inputs
			}

			// Select keyword
			keywords := []string{"state", "busy", "recomposing", "inappropriate compute instance state", "resource conflict"}
			keyword := keywords[keywordIndex%len(keywords)]

			// Create error message with keyword embedded
			errorMessage := fmt.Sprintf("%s %s %s", prefix, keyword, suffix)

			// Verify keyword is detected as state conflict
			isConflict := IsStateConflictError(errors.New("test"), http.StatusBadRequest, errorMessage)
			if !isConflict {
				return false
			}

			// Test that retry is triggered
			attemptCount := 0

			config := StateConflictRetryConfig()
			config.MaxAttempts = maxAttempts
			config.InitialDelay = 1 * time.Millisecond
			config.MaxDelay = 10 * time.Millisecond
			config.ShouldRetry = func(err error) bool {
				return IsStateConflictError(err, http.StatusBadRequest, errorMessage)
			}

			operation := func() error {
				attemptCount++
				return errors.New(errorMessage)
			}

			ctx := context.Background()
			err := Retry(ctx, config, operation)

			// Verify retry was triggered
			return err != nil && attemptCount == maxAttempts
		},
		gen.IntRange(0, 4),
		gen.AnyString(),
		gen.AnyString(),
		gen.IntRange(2, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 4: State Conflict Errors Trigger Retry (Non-State Errors)
// Validates: Requirements 5.3
func TestProperty_NonStateErrorsDoNotTriggerRetry(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any non-state error, retry should not be triggered", prop.ForAll(
		func(statusCode int, errorMessage string) bool {
			// Ensure we're testing non-conflict status codes
			if statusCode == http.StatusConflict {
				return true // Skip 409
			}

			// Ensure error message doesn't contain state keywords
			if containsStateKeyword(errorMessage) {
				return true // Skip messages with state keywords
			}

			// Verify it's not detected as state conflict
			isConflict := IsStateConflictError(errors.New("test"), statusCode, errorMessage)
			if isConflict {
				return false
			}

			// Test that retry is NOT triggered (only 1 attempt)
			attemptCount := 0

			config := StateConflictRetryConfig()
			config.MaxAttempts = 5
			config.InitialDelay = 1 * time.Millisecond
			config.MaxDelay = 10 * time.Millisecond
			config.ShouldRetry = func(err error) bool {
				return IsStateConflictError(err, statusCode, errorMessage)
			}

			operation := func() error {
				attemptCount++
				return errors.New(errorMessage)
			}

			ctx := context.Background()
			err := Retry(ctx, config, operation)

			// Verify only one attempt was made (no retry)
			return err != nil && attemptCount == 1
		},
		gen.OneConstOf(
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusInternalServerError,
		),
		gen.RegexMatch("[a-zA-Z0-9 ]+"), // Simple alphanumeric strings without state keywords
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper function to check if error message contains state keywords
func containsStateKeyword(message string) bool {
	keywords := []string{"state", "busy", "recomposing", "inappropriate compute instance state", "resource conflict"}
	lowerMessage := ""
	for _, c := range message {
		if c >= 'A' && c <= 'Z' {
			lowerMessage += string(c + 32)
		} else {
			lowerMessage += string(c)
		}
	}

	for _, keyword := range keywords {
		if contains(lowerMessage, keyword) {
			return true
		}
	}
	return false
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function to calculate delay (mirrors retry.go's calculateDelay)
func calculateDelayHelper(attempt int, config RetryConfig) time.Duration {
	delay := float64(config.InitialDelay)
	for i := 0; i < attempt; i++ {
		delay *= config.Multiplier
	}
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	return time.Duration(delay)
}
