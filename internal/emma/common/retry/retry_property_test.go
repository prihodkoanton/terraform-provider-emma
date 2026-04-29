package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: provider-improvements, Property 7: Retry Logic Respects Max Attempts
// Validates: Requirements 9.1, 9.5
func TestProperty_RetryLogicRespectsMaxAttempts(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any retry configuration, retry mechanism should not exceed max attempts", prop.ForAll(
		func(maxAttempts int) bool {
			// Ensure valid max attempts
			if maxAttempts < 1 || maxAttempts > 10 {
				return true // Skip invalid inputs
			}

			attemptCount := 0

			config := RetryConfig{
				MaxAttempts:  maxAttempts,
				InitialDelay: 1 * time.Millisecond,
				MaxDelay:     10 * time.Millisecond,
				Multiplier:   2.0,
				ShouldRetry: func(err error) bool {
					return true
				},
			}

			operation := func() error {
				attemptCount++
				return errors.New("persistent error")
			}

			ctx := context.Background()
			err := Retry(ctx, config, operation)

			// Verify that:
			// 1. An error occurred (operation always fails)
			if err == nil {
				return false
			}

			// 2. The number of attempts equals max attempts
			if attemptCount != maxAttempts {
				return false
			}

			// 3. The error message indicates the correct number of attempts
			expectedMsg := "operation failed after"
			if len(err.Error()) < len(expectedMsg) || err.Error()[:len(expectedMsg)] != expectedMsg {
				return false
			}

			return true
		},
		gen.IntRange(1, 10), // max attempts
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: provider-improvements, Property 8: Exponential Backoff Increases Delay
// Validates: Requirements 9.1
func TestProperty_ExponentialBackoffIncreasesDelay(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any retry sequence, each subsequent delay should be larger than previous up to max", prop.ForAll(
		func(initialDelayMs, maxDelayMs int, multiplier float64) bool {
			// Ensure valid parameters
			if initialDelayMs < 1 || initialDelayMs > 1000 {
				return true // Skip invalid inputs
			}
			if maxDelayMs < initialDelayMs || maxDelayMs > 10000 {
				return true // Skip invalid inputs
			}
			if multiplier < 1.1 || multiplier > 5.0 {
				return true // Skip invalid inputs
			}

			config := RetryConfig{
				InitialDelay: time.Duration(initialDelayMs) * time.Millisecond,
				MaxDelay:     time.Duration(maxDelayMs) * time.Millisecond,
				Multiplier:   multiplier,
			}

			// Test delays for first 5 attempts
			var delays []time.Duration
			for attempt := 0; attempt < 5; attempt++ {
				delay := calculateDelay(attempt, config)
				delays = append(delays, delay)
			}

			// Verify that:
			// 1. First delay equals initial delay
			if delays[0] != config.InitialDelay {
				return false
			}

			// 2. Each delay is either larger than previous or at max
			for i := 1; i < len(delays); i++ {
				if delays[i] < delays[i-1] {
					return false
				}
				// Once we hit max, all subsequent delays should be max
				if delays[i-1] == config.MaxDelay && delays[i] != config.MaxDelay {
					return false
				}
			}

			// 3. No delay exceeds max delay
			for _, delay := range delays {
				if delay > config.MaxDelay {
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 1000),      // initial delay in milliseconds
		gen.IntRange(100, 10000),   // max delay in milliseconds
		gen.Float64Range(1.1, 5.0), // multiplier
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: provider-improvements, Property 9: Non-Retryable Errors Fail Immediately
// Validates: Requirements 9.4
func TestProperty_NonRetryableErrorsFailImmediately(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any non-retryable error, operation should fail immediately without retrying", prop.ForAll(
		func(maxAttempts int) bool {
			// Ensure valid max attempts
			if maxAttempts < 2 || maxAttempts > 10 {
				return true // Skip invalid inputs
			}

			attemptCount := 0

			config := RetryConfig{
				MaxAttempts:  maxAttempts,
				InitialDelay: 1 * time.Millisecond,
				MaxDelay:     10 * time.Millisecond,
				Multiplier:   2.0,
				ShouldRetry: func(err error) bool {
					// Simulate non-retryable error (e.g., 4xx except 429)
					return err.Error() != "non-retryable error"
				},
			}

			operation := func() error {
				attemptCount++
				return errors.New("non-retryable error")
			}

			ctx := context.Background()
			err := Retry(ctx, config, operation)

			// Verify that:
			// 1. An error occurred
			if err == nil {
				return false
			}

			// 2. Only one attempt was made
			if attemptCount != 1 {
				return false
			}

			// 3. The error is the original non-retryable error
			if err.Error() != "non-retryable error" {
				return false
			}

			return true
		},
		gen.IntRange(2, 10), // max attempts (must be > 1 to test immediate failure)
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
