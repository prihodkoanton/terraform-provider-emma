package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.InitialDelay)
	assert.Equal(t, 30*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.Multiplier)
	assert.NotNil(t, config.ShouldRetry)

	// Test default ShouldRetry returns true
	assert.True(t, config.ShouldRetry(errors.New("test error")))
}

func TestRetry_SuccessfulRetry(t *testing.T) {
	ctx := context.Background()
	attemptCount := 0

	config := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		ShouldRetry: func(err error) bool {
			return true
		},
	}

	operation := func() error {
		attemptCount++
		if attemptCount < 2 {
			return errors.New("temporary error")
		}
		return nil
	}

	err := Retry(ctx, config, operation)

	assert.NoError(t, err)
	assert.Equal(t, 2, attemptCount)
}

func TestRetry_MaxAttemptsRespected(t *testing.T) {
	ctx := context.Background()
	attemptCount := 0

	config := RetryConfig{
		MaxAttempts:  3,
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

	err := Retry(ctx, config, operation)

	assert.Error(t, err)
	assert.Equal(t, 3, attemptCount)
	assert.Contains(t, err.Error(), "operation failed after 3 attempts")
}

func TestRetry_ExponentialBackoffCalculation(t *testing.T) {
	config := RetryConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}

	// Test exponential growth
	delay0 := calculateDelay(0, config)
	delay1 := calculateDelay(1, config)
	delay2 := calculateDelay(2, config)

	assert.Equal(t, 1*time.Second, delay0)
	assert.Equal(t, 2*time.Second, delay1)
	assert.Equal(t, 4*time.Second, delay2)

	// Test max delay cap
	delay10 := calculateDelay(10, config)
	assert.Equal(t, 30*time.Second, delay10)
}

func TestRetry_NonRetryableErrors(t *testing.T) {
	ctx := context.Background()
	attemptCount := 0

	config := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Millisecond,
		MaxDelay:     10 * time.Millisecond,
		Multiplier:   2.0,
		ShouldRetry: func(err error) bool {
			// Don't retry on specific error
			return err.Error() != "non-retryable error"
		},
	}

	operation := func() error {
		attemptCount++
		return errors.New("non-retryable error")
	}

	err := Retry(ctx, config, operation)

	assert.Error(t, err)
	assert.Equal(t, 1, attemptCount) // Should fail immediately
	assert.Equal(t, "non-retryable error", err.Error())
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attemptCount := 0

	config := RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		ShouldRetry: func(err error) bool {
			return true
		},
	}

	operation := func() error {
		attemptCount++
		if attemptCount == 2 {
			cancel() // Cancel context after second attempt
		}
		return errors.New("error")
	}

	err := Retry(ctx, config, operation)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.LessOrEqual(t, attemptCount, 2)
}

func TestRetry_ImmediateSuccess(t *testing.T) {
	ctx := context.Background()
	attemptCount := 0

	config := DefaultRetryConfig()

	operation := func() error {
		attemptCount++
		return nil
	}

	err := Retry(ctx, config, operation)

	assert.NoError(t, err)
	assert.Equal(t, 1, attemptCount)
}
