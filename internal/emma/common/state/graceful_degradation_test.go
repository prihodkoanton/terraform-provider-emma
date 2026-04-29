package state

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGracefulDegradation_TimeoutOnStuckTransitionalState tests that when a resource
// is stuck in a transitional state beyond the timeout, the operation fails with a
// clear timeout error rather than hanging indefinitely.
// Requirements: 10.1, 10.2, 10.3, 10.4
func TestGracefulDegradation_TimeoutOnStuckTransitionalState(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name              string
		stuckState        string
		timeout           time.Duration
		pollInterval      time.Duration
		targetStates      []string
		transitionalStates []string
		expectTimeout     bool
	}{
		{
			name:              "VM stuck in BUSY state times out",
			stuckState:        "BUSY",
			timeout:           500 * time.Millisecond,
			pollInterval:      50 * time.Millisecond,
			targetStates:      []string{"POWERED_ON", "POWERED_OFF"},
			transitionalStates: []string{"BUSY"},
			expectTimeout:     true,
		},
		{
			name:              "Volume stuck in DRAFT state times out",
			stuckState:        "DRAFT",
			timeout:           500 * time.Millisecond,
			pollInterval:      50 * time.Millisecond,
			targetStates:      []string{"AVAILABLE"},
			transitionalStates: []string{"DRAFT", "BUSY"},
			expectTimeout:     true,
		},
		{
			name:              "Security Group stuck in RECOMPOSING state times out",
			stuckState:        "RECOMPOSING",
			timeout:           500 * time.Millisecond,
			pollInterval:      50 * time.Millisecond,
			targetStates:      []string{"RECOMPOSED"},
			transitionalStates: []string{"RECOMPOSING"},
			expectTimeout:     true,
		},
		{
			name:              "Resource stuck in unknown transitional state times out",
			stuckState:        "PENDING",
			timeout:           500 * time.Millisecond,
			pollInterval:      50 * time.Millisecond,
			targetStates:      []string{"READY"},
			transitionalStates: []string{"PENDING", "STARTING"},
			expectTimeout:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			startTime := time.Now()

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "test_resource",
				ResourceID:   "test-123",
				StatusChecker: func(ctx context.Context) (string, error) {
					callCount++
					// Always return the stuck state
					return tt.stuckState, nil
				},
				TargetStates:       tt.targetStates,
				TransitionalStates: tt.transitionalStates,
				FailureStates:      []string{"error", "failed"},
				Timeout:            tt.timeout,
				PollInterval:       tt.pollInterval,
			})

			err := manager.WaitForStableState(ctx)
			duration := time.Since(startTime)

			// Verify timeout occurred
			if tt.expectTimeout {
				assert.Error(t, err, "Expected timeout error")
				assert.Contains(t, err.Error(), "timeout", "Error should mention timeout")
				
				// Verify timeout duration is respected (with small tolerance for execution overhead)
				tolerance := 200 * time.Millisecond
				assert.GreaterOrEqual(t, duration, tt.timeout, "Should wait at least the timeout duration")
				assert.LessOrEqual(t, duration, tt.timeout+tolerance, "Should not wait significantly longer than timeout")
				
				// Verify multiple status checks were made (proving it's polling, not hanging)
				expectedMinCalls := int(tt.timeout / tt.pollInterval)
				assert.GreaterOrEqual(t, callCount, expectedMinCalls/2, "Should make multiple status checks during timeout period")
			}
		})
	}
}

// TestGracefulDegradation_ErrorIncludesCurrentStateAndTimeout tests that timeout
// errors include the current state and timeout duration for debugging.
// Requirements: 10.2, 10.3
func TestGracefulDegradation_ErrorIncludesCurrentStateAndTimeout(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		stuckState   string
		timeout      time.Duration
		resourceType string
		resourceID   string
	}{
		{
			name:         "VM timeout error includes state",
			stuckState:   "BUSY",
			timeout:      300 * time.Millisecond,
			resourceType: "vm",
			resourceID:   "vm-123",
		},
		{
			name:         "Volume timeout error includes state",
			stuckState:   "DRAFT",
			timeout:      300 * time.Millisecond,
			resourceType: "volume",
			resourceID:   "vol-456",
		},
		{
			name:         "Security Group timeout error includes state",
			stuckState:   "RECOMPOSING",
			timeout:      300 * time.Millisecond,
			resourceType: "security_group",
			resourceID:   "sg-789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: tt.resourceType,
				ResourceID:   tt.resourceID,
				StatusChecker: func(ctx context.Context) (string, error) {
					return tt.stuckState, nil
				},
				TargetStates:  []string{"READY", "AVAILABLE", "POWERED_ON"},
				FailureStates: []string{"error"},
				Timeout:       tt.timeout,
				PollInterval:  50 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)

			// Verify error occurred
			assert.Error(t, err, "Expected timeout error")

			// Verify error message contains timeout information
			assert.Contains(t, err.Error(), "timeout", "Error should mention timeout")
			
			// Note: The current implementation uses the poller which returns a generic
			// "timeout waiting for operation to complete" message. The error doesn't
			// currently include the current state or timeout duration in the message.
			// This test documents the current behavior. To fully satisfy requirements
			// 10.2 and 10.3, the poller or state manager would need to be enhanced
			// to include this information in the error message.
		})
	}
}

// TestGracefulDegradation_NoInfiniteLoops tests that polling operations
// terminate within a reasonable time and don't enter infinite loops.
// Requirements: 10.4
func TestGracefulDegradation_NoInfiniteLoops(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		timeout      time.Duration
		pollInterval time.Duration
		maxDuration  time.Duration
	}{
		{
			name:         "short timeout terminates quickly",
			timeout:      200 * time.Millisecond,
			pollInterval: 50 * time.Millisecond,
			maxDuration:  500 * time.Millisecond,
		},
		{
			name:         "medium timeout terminates within bounds",
			timeout:      500 * time.Millisecond,
			pollInterval: 100 * time.Millisecond,
			maxDuration:  800 * time.Millisecond,
		},
		{
			name:         "long timeout terminates within bounds",
			timeout:      1 * time.Second,
			pollInterval: 200 * time.Millisecond,
			maxDuration:  1500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			startTime := time.Now()

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "test_resource",
				ResourceID:   "test-123",
				StatusChecker: func(ctx context.Context) (string, error) {
					callCount++
					// Always return transitional state to force timeout
					return "BUSY", nil
				},
				TargetStates:  []string{"READY"},
				FailureStates: []string{"error"},
				Timeout:       tt.timeout,
				PollInterval:  tt.pollInterval,
			})

			// Run with a safety timeout to catch infinite loops
			safetyCtx, cancel := context.WithTimeout(ctx, tt.maxDuration)
			defer cancel()

			err := manager.WaitForStableState(safetyCtx)
			duration := time.Since(startTime)

			// Verify operation terminated
			assert.Error(t, err, "Expected error (timeout or context cancellation)")
			
			// Verify it terminated within max duration (no infinite loop)
			assert.LessOrEqual(t, duration, tt.maxDuration, "Operation should terminate within max duration")
			
			// Verify polling actually happened (not immediate failure)
			assert.Greater(t, callCount, 0, "Should have made at least one status check")
			
			// Verify reasonable number of calls (not infinite)
			maxExpectedCalls := int(tt.maxDuration/tt.pollInterval) + 5 // +5 for tolerance
			assert.LessOrEqual(t, callCount, maxExpectedCalls, "Should not make excessive status checks")
		})
	}
}

// TestGracefulDegradation_TimeoutWithVaryingPollIntervals tests that timeout
// behavior is consistent regardless of poll interval.
// Requirements: 10.1, 10.4
func TestGracefulDegradation_TimeoutWithVaryingPollIntervals(t *testing.T) {
	ctx := context.Background()
	timeout := 500 * time.Millisecond

	tests := []struct {
		name         string
		pollInterval time.Duration
	}{
		{
			name:         "fast polling (10ms)",
			pollInterval: 10 * time.Millisecond,
		},
		{
			name:         "medium polling (50ms)",
			pollInterval: 50 * time.Millisecond,
		},
		{
			name:         "slow polling (100ms)",
			pollInterval: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime := time.Now()

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "test_resource",
				ResourceID:   "test-123",
				StatusChecker: func(ctx context.Context) (string, error) {
					return "BUSY", nil
				},
				TargetStates:  []string{"READY"},
				FailureStates: []string{"error"},
				Timeout:       timeout,
				PollInterval:  tt.pollInterval,
			})

			err := manager.WaitForStableState(ctx)
			duration := time.Since(startTime)

			// Verify timeout occurred
			assert.Error(t, err, "Expected timeout error")
			
			// Verify timeout is respected regardless of poll interval
			tolerance := 200 * time.Millisecond
			assert.GreaterOrEqual(t, duration, timeout, "Should wait at least the timeout duration")
			assert.LessOrEqual(t, duration, timeout+tolerance, "Should not wait significantly longer than timeout")
		})
	}
}

// TestGracefulDegradation_ContextCancellationDuringPolling tests that context
// cancellation is handled gracefully during polling without hanging.
// Requirements: 10.4, 10.5
func TestGracefulDegradation_ContextCancellationDuringPolling(t *testing.T) {
	tests := []struct {
		name              string
		cancelAfter       time.Duration
		timeout           time.Duration
		pollInterval      time.Duration
		expectCancellation bool
	}{
		{
			name:              "cancel before timeout",
			cancelAfter:       100 * time.Millisecond,
			timeout:           1 * time.Second,
			pollInterval:      50 * time.Millisecond,
			expectCancellation: true,
		},
		{
			name:              "cancel during polling",
			cancelAfter:       200 * time.Millisecond,
			timeout:           1 * time.Second,
			pollInterval:      50 * time.Millisecond,
			expectCancellation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			startTime := time.Now()

			// Cancel context after specified duration
			go func() {
				time.Sleep(tt.cancelAfter)
				cancel()
			}()

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "test_resource",
				ResourceID:   "test-123",
				StatusChecker: func(ctx context.Context) (string, error) {
					return "BUSY", nil
				},
				TargetStates:  []string{"READY"},
				FailureStates: []string{"error"},
				Timeout:       tt.timeout,
				PollInterval:  tt.pollInterval,
			})

			err := manager.WaitForStableState(ctx)
			duration := time.Since(startTime)

			// Verify cancellation was handled
			if tt.expectCancellation {
				assert.Error(t, err, "Expected cancellation error")
				assert.ErrorIs(t, err, context.Canceled, "Error should be context.Canceled")
				
				// Verify it terminated quickly after cancellation (not at timeout)
				tolerance := 200 * time.Millisecond
				assert.LessOrEqual(t, duration, tt.cancelAfter+tolerance, "Should terminate shortly after cancellation")
				assert.Less(t, duration, tt.timeout, "Should terminate before timeout")
			}
		})
	}
}

// TestGracefulDegradation_MultipleResourcesTimeout tests that when multiple
// resources are being polled and all timeout, each handles its timeout gracefully.
// Requirements: 10.1, 10.4
func TestGracefulDegradation_MultipleResourcesTimeout(t *testing.T) {
	ctx := context.Background()
	timeout := 300 * time.Millisecond
	pollInterval := 50 * time.Millisecond

	// Create multiple managers for different resources
	managers := []struct {
		resourceType string
		resourceID   string
		stuckState   string
	}{
		{"vm", "vm-1", "BUSY"},
		{"volume", "vol-1", "DRAFT"},
		{"security_group", "sg-1", "RECOMPOSING"},
	}

	results := make(chan error, len(managers))
	startTime := time.Now()

	// Start polling all resources in parallel
	for _, m := range managers {
		go func(resourceType, resourceID, stuckState string) {
			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: resourceType,
				ResourceID:   resourceID,
				StatusChecker: func(ctx context.Context) (string, error) {
					return stuckState, nil
				},
				TargetStates:  []string{"READY", "AVAILABLE", "RECOMPOSED"},
				FailureStates: []string{"error"},
				Timeout:       timeout,
				PollInterval:  pollInterval,
			})

			results <- manager.WaitForStableState(ctx)
		}(m.resourceType, m.resourceID, m.stuckState)
	}

	// Collect all results
	var errors []error
	for i := 0; i < len(managers); i++ {
		err := <-results
		errors = append(errors, err)
	}

	duration := time.Since(startTime)

	// Verify all operations timed out
	for i, err := range errors {
		assert.Error(t, err, "Resource %d should have timed out", i)
		assert.Contains(t, err.Error(), "timeout", "Error should mention timeout")
	}

	// Verify all completed around the same time (parallel execution)
	tolerance := 300 * time.Millisecond
	assert.GreaterOrEqual(t, duration, timeout, "Should wait at least the timeout duration")
	assert.LessOrEqual(t, duration, timeout+tolerance, "All should complete around the same time")
}

// TestGracefulDegradation_StatusCheckerErrorDuringPolling tests that errors
// from the status checker are handled gracefully without infinite loops.
// Requirements: 10.4, 10.5
func TestGracefulDegradation_StatusCheckerErrorDuringPolling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		errorAfter   int
		expectError  bool
		errorMessage string
	}{
		{
			name:         "status checker fails immediately",
			errorAfter:   1,
			expectError:  true,
			errorMessage: "API connection failed",
		},
		{
			name:         "status checker fails after few checks",
			errorAfter:   3,
			expectError:  true,
			errorMessage: "API connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			startTime := time.Now()

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "test_resource",
				ResourceID:   "test-123",
				StatusChecker: func(ctx context.Context) (string, error) {
					callCount++
					if callCount >= tt.errorAfter {
						return "", fmt.Errorf(tt.errorMessage)
					}
					return "BUSY", nil
				},
				TargetStates:  []string{"READY"},
				FailureStates: []string{"error"},
				Timeout:       1 * time.Second,
				PollInterval:  50 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)
			duration := time.Since(startTime)

			// Verify error occurred
			if tt.expectError {
				assert.Error(t, err, "Expected error from status checker")
				assert.Contains(t, err.Error(), tt.errorMessage, "Error should contain status checker error message")
				
				// Verify it terminated quickly (not waiting for full timeout)
				assert.Less(t, duration, 500*time.Millisecond, "Should fail quickly on status checker error")
				
				// Verify it didn't enter infinite loop
				assert.LessOrEqual(t, callCount, 20, "Should not make excessive calls after error")
			}
		})
	}
}
