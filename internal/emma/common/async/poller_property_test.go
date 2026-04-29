package async

import (
	"context"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: provider-improvements, Property 10: Async Operations Timeout
// Validates: Requirements 6.2, 6.6
func TestProperty_AsyncOperationsTimeout(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any async operation, if operation does not complete within timeout, it should return timeout error", prop.ForAll(
		func(timeoutMs, pollIntervalMs int) bool {
			// Ensure valid timeout and poll interval
			if timeoutMs < 50 || timeoutMs > 1000 {
				return true // Skip invalid inputs
			}
			if pollIntervalMs < 10 || pollIntervalMs > timeoutMs {
				return true // Skip invalid inputs
			}

			timeout := time.Duration(timeoutMs) * time.Millisecond
			pollInterval := time.Duration(pollIntervalMs) * time.Millisecond

			// Status checker that never completes
			statusChecker := func(ctx context.Context) (string, error) {
				return "in_progress", nil
			}

			config := PollerConfig{
				Timeout:       timeout,
				PollInterval:  pollInterval,
				StatusChecker: statusChecker,
				TargetStates:  []string{"completed"},
				FailureStates: []string{"failed"},
			}

			poller := NewPoller(config)
			ctx := context.Background()

			start := time.Now()
			err := poller.Poll(ctx)
			duration := time.Since(start)

			// Verify that:
			// 1. An error occurred (timeout)
			if err == nil {
				return false
			}

			// 2. The error is a timeout error
			if err.Error() != "timeout waiting for operation to complete" {
				return false
			}

			// 3. The operation took at least the timeout duration
			// Allow some tolerance for timing precision
			if duration < timeout-50*time.Millisecond {
				return false
			}

			// 4. The operation didn't take significantly longer than timeout
			// Allow up to 2x poll interval for final check
			if duration > timeout+2*pollInterval+100*time.Millisecond {
				return false
			}

			return true
		},
		gen.IntRange(50, 1000),   // timeout in milliseconds
		gen.IntRange(10, 100),    // poll interval in milliseconds
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
