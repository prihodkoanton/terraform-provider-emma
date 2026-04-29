package state

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestIdempotentOperation_AlreadyInTargetState tests that operations succeed
// immediately when the resource is already in the target state
// Validates: Requirements 8.1, 8.4
func TestIdempotentOperation_AlreadyInTargetState(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		resourceType string
		currentState string
		targetStates []string
	}{
		{
			name:         "VM already POWERED_ON",
			resourceType: "vm",
			currentState: "POWERED_ON",
			targetStates: []string{"POWERED_ON", "POWERED_OFF"},
		},
		{
			name:         "VM already POWERED_OFF",
			resourceType: "vm",
			currentState: "POWERED_OFF",
			targetStates: []string{"POWERED_ON", "POWERED_OFF"},
		},
		{
			name:         "Volume already AVAILABLE",
			resourceType: "volume",
			currentState: "AVAILABLE",
			targetStates: []string{"AVAILABLE"},
		},
		{
			name:         "Security Group already RECOMPOSED",
			resourceType: "security_group",
			currentState: "RECOMPOSED",
			targetStates: []string{"RECOMPOSED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: tt.resourceType,
				ResourceID:   "test-123",
				StatusChecker: func(ctx context.Context) (string, error) {
					return tt.currentState, nil
				},
				TargetStates:  tt.targetStates,
				FailureStates: []string{"error", "failed"},
				Timeout:       5 * time.Second,
				PollInterval:  1 * time.Second,
			})

			// Operation should succeed immediately
			err := manager.WaitForStableState(ctx)

			assert.NoError(t, err, "Operation should succeed when already in target state")
		})
	}
}

// TestIdempotentOperation_NoUnnecessaryAPICalls tests that when a resource
// is already in the target state, no unnecessary polling occurs
// Validates: Requirements 8.1, 8.4
func TestIdempotentOperation_NoUnnecessaryAPICalls(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		resourceType string
		currentState string
		targetStates []string
	}{
		{
			name:         "VM already in stable state",
			resourceType: "vm",
			currentState: "POWERED_ON",
			targetStates: []string{"POWERED_ON", "POWERED_OFF"},
		},
		{
			name:         "Volume already in stable state",
			resourceType: "volume",
			currentState: "AVAILABLE",
			targetStates: []string{"AVAILABLE"},
		},
		{
			name:         "Security Group already in stable state",
			resourceType: "security_group",
			currentState: "RECOMPOSED",
			targetStates: []string{"RECOMPOSED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var apiCallCount int32

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: tt.resourceType,
				ResourceID:   "test-123",
				StatusChecker: func(ctx context.Context) (string, error) {
					atomic.AddInt32(&apiCallCount, 1)
					return tt.currentState, nil
				},
				TargetStates:  tt.targetStates,
				FailureStates: []string{"error", "failed"},
				Timeout:       5 * time.Second,
				PollInterval:  100 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)

			assert.NoError(t, err)
			// Should only call the API once to check current state, no polling
			assert.Equal(t, int32(1), atomic.LoadInt32(&apiCallCount),
				"Should only make one API call when already in target state")
		})
	}
}

// TestIdempotentOperation_MultipleCallsSameResult tests that calling
// WaitForStableState multiple times on the same resource produces the same result
// Validates: Requirements 8.1, 8.4
func TestIdempotentOperation_MultipleCallsSameResult(t *testing.T) {
	ctx := context.Background()

	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "test-123",
		StatusChecker: func(ctx context.Context) (string, error) {
			return "POWERED_ON", nil
		},
		TargetStates:  []string{"POWERED_ON", "POWERED_OFF"},
		FailureStates: []string{"error"},
		Timeout:       5 * time.Second,
		PollInterval:  1 * time.Second,
	})

	// Call multiple times
	for i := 0; i < 5; i++ {
		err := manager.WaitForStableState(ctx)
		assert.NoError(t, err, "Call %d should succeed", i+1)
	}
}

// TestStateRefresh_DetectsExternalChanges tests that state refresh
// correctly detects when a resource state has changed externally
// Validates: Requirements 8.2, 8.3
func TestStateRefresh_DetectsExternalChanges(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		initialState  string
		changedState  string
		targetStates  []string
		expectSuccess bool
	}{
		{
			name:          "VM changed from BUSY to POWERED_ON",
			initialState:  "BUSY",
			changedState:  "POWERED_ON",
			targetStates:  []string{"POWERED_ON", "POWERED_OFF"},
			expectSuccess: true,
		},
		{
			name:          "Volume changed from DRAFT to AVAILABLE",
			initialState:  "DRAFT",
			changedState:  "AVAILABLE",
			targetStates:  []string{"AVAILABLE"},
			expectSuccess: true,
		},
		{
			name:          "Security Group changed from RECOMPOSING to RECOMPOSED",
			initialState:  "RECOMPOSING",
			changedState:  "RECOMPOSED",
			targetStates:  []string{"RECOMPOSED"},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			currentState := tt.initialState

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "test",
				ResourceID:   "test-123",
				StatusChecker: func(ctx context.Context) (string, error) {
					callCount++
					// Simulate external state change after first check
					if callCount > 1 {
						currentState = tt.changedState
					}
					return currentState, nil
				},
				TargetStates:       tt.targetStates,
				TransitionalStates: []string{"BUSY", "DRAFT", "RECOMPOSING"},
				FailureStates:      []string{"error", "failed"},
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)

			if tt.expectSuccess {
				assert.NoError(t, err, "Should detect external state change and succeed")
				assert.GreaterOrEqual(t, callCount, 2, "Should have polled at least twice to detect change")
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestStateRefresh_DetectsExternalFailure tests that state refresh
// detects when a resource enters a failure state externally
// Validates: Requirements 8.2, 8.3
func TestStateRefresh_DetectsExternalFailure(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	currentState := "BUSY"

	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "test-123",
		StatusChecker: func(ctx context.Context) (string, error) {
			callCount++
			// Simulate external failure after first check
			if callCount > 1 {
				currentState = "error"
			}
			return currentState, nil
		},
		TargetStates:       []string{"POWERED_ON"},
		TransitionalStates: []string{"BUSY"},
		FailureStates:      []string{"error", "failed"},
		Timeout:            5 * time.Second,
		PollInterval:       100 * time.Millisecond,
	})

	err := manager.WaitForStableState(ctx)

	assert.Error(t, err, "Should detect external failure")
	// The error could be from WaitForStableState (contains "failure state") or from poller (contains "operation failed")
	hasFailureMessage := err.Error() == "resource vm test-123 is in failure state: error" ||
		err.Error() == "operation failed with status: error"
	assert.True(t, hasFailureMessage, "Error should indicate failure: %s", err.Error())
	assert.GreaterOrEqual(t, callCount, 2, "Should have polled at least twice to detect failure")
}

// TestIsInTargetState_RefreshesFromAPI tests that IsInTargetState
// always refreshes state from the API rather than using cached values
// Validates: Requirements 8.2
func TestIsInTargetState_RefreshesFromAPI(t *testing.T) {
	ctx := context.Background()

	var apiCallCount int32
	currentState := "BUSY"

	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "test-123",
		StatusChecker: func(ctx context.Context) (string, error) {
			atomic.AddInt32(&apiCallCount, 1)
			return currentState, nil
		},
		TargetStates:  []string{"POWERED_ON", "POWERED_OFF"},
		FailureStates: []string{"error"},
		Timeout:       5 * time.Second,
		PollInterval:  1 * time.Second,
	})

	// First check - not in target state
	isTarget, state, err := manager.IsInTargetState(ctx)
	assert.NoError(t, err)
	assert.False(t, isTarget)
	assert.Equal(t, "BUSY", state)
	firstCallCount := atomic.LoadInt32(&apiCallCount)

	// Simulate external state change
	currentState = "POWERED_ON"

	// Second check - should detect the change
	isTarget, state, err = manager.IsInTargetState(ctx)
	assert.NoError(t, err)
	assert.True(t, isTarget)
	assert.Equal(t, "POWERED_ON", state)
	secondCallCount := atomic.LoadInt32(&apiCallCount)

	// Verify that API was called again (not using cached value)
	assert.Greater(t, secondCallCount, firstCallCount,
		"Should make new API call to refresh state, not use cached value")
}

// TestIdempotentOperation_DifferentTargetStates tests idempotent behavior
// when a resource is in one target state but operation expects another
// Validates: Requirements 8.1, 8.4
func TestIdempotentOperation_DifferentTargetStates(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		currentState  string
		targetStates  []string
		expectSuccess bool
	}{
		{
			name:          "VM is POWERED_ON, expecting POWERED_ON or POWERED_OFF",
			currentState:  "POWERED_ON",
			targetStates:  []string{"POWERED_ON", "POWERED_OFF"},
			expectSuccess: true,
		},
		{
			name:          "VM is POWERED_OFF, expecting POWERED_ON or POWERED_OFF",
			currentState:  "POWERED_OFF",
			targetStates:  []string{"POWERED_ON", "POWERED_OFF"},
			expectSuccess: true,
		},
		{
			name:          "VM is POWERED_ON, expecting only POWERED_OFF",
			currentState:  "POWERED_ON",
			targetStates:  []string{"POWERED_OFF"},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   "test-123",
				StatusChecker: func(ctx context.Context) (string, error) {
					return tt.currentState, nil
				},
				TargetStates:       tt.targetStates,
				TransitionalStates: []string{"BUSY"},
				FailureStates:      []string{"error"},
				Timeout:            1 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)

			if tt.expectSuccess {
				assert.NoError(t, err, "Should succeed when in any target state")
			} else {
				assert.Error(t, err, "Should timeout when not in target state")
			}
		})
	}
}

// TestCheckCurrentState_AlwaysRefreshes tests that CheckCurrentState
// always makes an API call and doesn't cache results
// Validates: Requirements 8.2
func TestCheckCurrentState_AlwaysRefreshes(t *testing.T) {
	ctx := context.Background()

	var apiCallCount int32
	states := []string{"BUSY", "BUSY", "POWERED_ON", "POWERED_ON"}
	stateIndex := 0

	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "test-123",
		StatusChecker: func(ctx context.Context) (string, error) {
			atomic.AddInt32(&apiCallCount, 1)
			state := states[stateIndex]
			if stateIndex < len(states)-1 {
				stateIndex++
			}
			return state, nil
		},
		TargetStates:  []string{"POWERED_ON"},
		FailureStates: []string{"error"},
	})

	// Make multiple calls to CheckCurrentState
	for i := 0; i < 4; i++ {
		state, err := manager.CheckCurrentState(ctx)
		assert.NoError(t, err)
		assert.Equal(t, states[i], state, "Should return current state from API")
	}

	// Verify API was called for each check
	assert.Equal(t, int32(4), atomic.LoadInt32(&apiCallCount),
		"Should make API call for each CheckCurrentState call")
}

// TestIdempotentOperation_ConcurrentChecks tests that concurrent
// idempotent checks on the same resource work correctly
// Validates: Requirements 8.1, 8.4
func TestIdempotentOperation_ConcurrentChecks(t *testing.T) {
	ctx := context.Background()

	var apiCallCount int32

	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "test-123",
		StatusChecker: func(ctx context.Context) (string, error) {
			atomic.AddInt32(&apiCallCount, 1)
			return "POWERED_ON", nil
		},
		TargetStates:  []string{"POWERED_ON", "POWERED_OFF"},
		FailureStates: []string{"error"},
		Timeout:       5 * time.Second,
		PollInterval:  1 * time.Second,
	})

	// Run multiple concurrent checks
	numGoroutines := 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			done <- manager.WaitForStableState(ctx)
		}()
	}

	// All should succeed
	for i := 0; i < numGoroutines; i++ {
		err := <-done
		assert.NoError(t, err, "Concurrent check %d should succeed", i+1)
	}

	// Each goroutine should have made exactly one API call
	assert.Equal(t, int32(numGoroutines), atomic.LoadInt32(&apiCallCount),
		"Each concurrent check should make exactly one API call")
}

// TestIdempotentOperation_ErrorHandling tests that errors during
// state checks are handled correctly in idempotent operations
// Validates: Requirements 8.2, 8.3
func TestIdempotentOperation_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name: "API error on first check",
			statusChecker: func(ctx context.Context) (string, error) {
				return "", errors.New("API connection failed")
			},
			expectError:   true,
			errorContains: "failed to check current state",
		},
		{
			name: "Successful check returns stable state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType:  "vm",
				ResourceID:    "test-123",
				StatusChecker: tt.statusChecker,
				TargetStates:  []string{"POWERED_ON"},
				FailureStates: []string{"error"},
				Timeout:       5 * time.Second,
				PollInterval:  1 * time.Second,
			})

			err := manager.WaitForStableState(ctx)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIdempotentOperation_StateTransitionDuringCheck tests behavior
// when state transitions occur during the idempotent check
// Validates: Requirements 8.2, 8.3
func TestIdempotentOperation_StateTransitionDuringCheck(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		stateSequence  []string
		targetStates   []string
		expectSuccess  bool
		minAPICalls    int
	}{
		{
			name:          "Transitions from BUSY to POWERED_ON",
			stateSequence: []string{"BUSY", "BUSY", "POWERED_ON"},
			targetStates:  []string{"POWERED_ON"},
			expectSuccess: true,
			minAPICalls:   3,
		},
		{
			name:          "Already in target state",
			stateSequence: []string{"POWERED_ON"},
			targetStates:  []string{"POWERED_ON"},
			expectSuccess: true,
			minAPICalls:   1,
		},
		{
			name:          "Transitions through multiple states",
			stateSequence: []string{"BUSY", "starting", "POWERED_ON"},
			targetStates:  []string{"POWERED_ON"},
			expectSuccess: true,
			minAPICalls:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var apiCallCount int32

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   "test-123",
				StatusChecker: func(ctx context.Context) (string, error) {
					callNum := atomic.AddInt32(&apiCallCount, 1)
					idx := int(callNum) - 1
					if idx >= len(tt.stateSequence) {
						idx = len(tt.stateSequence) - 1
					}
					return tt.stateSequence[idx], nil
				},
				TargetStates:       tt.targetStates,
				TransitionalStates: []string{"BUSY", "starting"},
				FailureStates:      []string{"error"},
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, int(atomic.LoadInt32(&apiCallCount)), tt.minAPICalls,
					"Should make at least %d API calls", tt.minAPICalls)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestIdempotentOperation_ContextCancellation tests that context
// cancellation is respected during idempotent operations
// Validates: Requirements 8.2, 8.3
func TestIdempotentOperation_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var apiCallCount int32

	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "test-123",
		StatusChecker: func(ctx context.Context) (string, error) {
			atomic.AddInt32(&apiCallCount, 1)
			// Cancel after first call
			if atomic.LoadInt32(&apiCallCount) == 1 {
				cancel()
			}
			return "BUSY", nil
		},
		TargetStates:       []string{"POWERED_ON"},
		TransitionalStates: []string{"BUSY"},
		FailureStates:      []string{"error"},
		Timeout:            5 * time.Second,
		PollInterval:       100 * time.Millisecond,
	})

	err := manager.WaitForStableState(ctx)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	// Should have made at least one call before cancellation
	assert.GreaterOrEqual(t, atomic.LoadInt32(&apiCallCount), int32(1))
}
