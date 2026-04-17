package state

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestParallelOperations_ConcurrentStateChecks tests that concurrent
// state checks on different resources work independently
// Validates: Requirements 9.1, 9.2, 9.3, 9.4, 9.5
func TestParallelOperations_ConcurrentStateChecks(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		numResources  int
		resourceTypes []string
		initialStates []string
		targetStates  [][]string
	}{
		{
			name:          "Two VMs with different states",
			numResources:  2,
			resourceTypes: []string{"vm", "vm"},
			initialStates: []string{"POWERED_ON", "POWERED_OFF"},
			targetStates:  [][]string{{"POWERED_ON"}, {"POWERED_OFF"}},
		},
		{
			name:          "VM and Volume concurrently",
			numResources:  2,
			resourceTypes: []string{"vm", "volume"},
			initialStates: []string{"POWERED_ON", "AVAILABLE"},
			targetStates:  [][]string{{"POWERED_ON"}, {"AVAILABLE"}},
		},
		{
			name:          "Multiple resources of different types",
			numResources:  3,
			resourceTypes: []string{"vm", "volume", "security_group"},
			initialStates: []string{"POWERED_ON", "AVAILABLE", "RECOMPOSED"},
			targetStates:  [][]string{{"POWERED_ON"}, {"AVAILABLE"}, {"RECOMPOSED"}},
		},
		{
			name:          "Five VMs concurrently",
			numResources:  5,
			resourceTypes: []string{"vm", "vm", "vm", "vm", "vm"},
			initialStates: []string{"POWERED_ON", "POWERED_OFF", "POWERED_ON", "POWERED_OFF", "POWERED_ON"},
			targetStates:  [][]string{{"POWERED_ON"}, {"POWERED_OFF"}, {"POWERED_ON"}, {"POWERED_OFF"}, {"POWERED_ON"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			results := make([]error, tt.numResources)

			// Create and run managers concurrently
			for i := 0; i < tt.numResources; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()

					manager := NewStateTransitionManager(StateTransitionConfig{
						ResourceType: tt.resourceTypes[index],
						ResourceID:   fmt.Sprintf("resource-%d", index),
						StatusChecker: func(ctx context.Context) (string, error) {
							return tt.initialStates[index], nil
						},
						TargetStates:  tt.targetStates[index],
						FailureStates: []string{"error", "failed"},
						Timeout:       5 * time.Second,
						PollInterval:  100 * time.Millisecond,
					})

					results[index] = manager.WaitForStableState(ctx)
				}(i)
			}

			// Wait for all operations to complete
			wg.Wait()

			// Verify all operations succeeded
			for i, err := range results {
				assert.NoError(t, err, "Resource %d should complete successfully", i)
			}
		})
	}
}

// TestParallelOperations_NoSharedStateCorruption tests that concurrent
// operations don't corrupt each other's state
// Validates: Requirements 9.1, 9.5
func TestParallelOperations_NoSharedStateCorruption(t *testing.T) {
	ctx := context.Background()

	numGoroutines := 20
	var wg sync.WaitGroup
	results := make([]string, numGoroutines)
	errors := make([]error, numGoroutines)

	// Each goroutine tracks its own state independently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Each manager has its own unique state
			expectedState := fmt.Sprintf("STATE_%d", index)

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   fmt.Sprintf("vm-%d", index),
				StatusChecker: func(ctx context.Context) (string, error) {
					// Return the unique state for this resource
					return expectedState, nil
				},
				TargetStates:  []string{expectedState},
				FailureStates: []string{"error"},
				Timeout:       5 * time.Second,
				PollInterval:  100 * time.Millisecond,
			})

			errors[index] = manager.WaitForStableState(ctx)

			// Verify the state is correct for this resource
			state, err := manager.CheckCurrentState(ctx)
			if err == nil {
				results[index] = state
			}
		}(i)
	}

	wg.Wait()

	// Verify no errors occurred
	for i, err := range errors {
		assert.NoError(t, err, "Goroutine %d should complete without error", i)
	}

	// Verify each goroutine got its own unique state (no corruption)
	for i, state := range results {
		expectedState := fmt.Sprintf("STATE_%d", i)
		assert.Equal(t, expectedState, state,
			"Goroutine %d should have its own state, not corrupted by others", i)
	}
}

// TestParallelOperations_ContextCancellationPropagates tests that
// context cancellation propagates correctly to all parallel operations
// Validates: Requirements 9.3, 9.4
func TestParallelOperations_ContextCancellationPropagates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	numGoroutines := 10
	var wg sync.WaitGroup
	errors := make([]error, numGoroutines)

	// Start multiple long-running operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   fmt.Sprintf("vm-%d", index),
				StatusChecker: func(ctx context.Context) (string, error) {
					// Simulate a slow operation that checks for cancellation
					select {
					case <-ctx.Done():
						return "", ctx.Err()
					case <-time.After(50 * time.Millisecond):
						return "BUSY", nil
					}
				},
				TargetStates:       []string{"POWERED_ON"},
				TransitionalStates: []string{"BUSY"},
				FailureStates:      []string{"error"},
				Timeout:            30 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)
			errors[index] = err
		}(i)
	}

	// Cancel after a short delay
	time.Sleep(200 * time.Millisecond)
	cancel()

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify that all operations were cancelled
	for i, err := range errors {
		assert.Error(t, err, "Goroutine %d should return an error after cancellation", i)
		assert.ErrorIs(t, err, context.Canceled,
			"Goroutine %d should return context.Canceled error", i)
	}
}

// TestParallelOperations_IndependentTimeouts tests that each parallel
// operation respects its own timeout independently
// Validates: Requirements 9.1, 9.2
func TestParallelOperations_IndependentTimeouts(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		timeout1 time.Duration
		timeout2 time.Duration
	}{
		{
			name:     "Different timeouts",
			timeout1: 1 * time.Second,
			timeout2: 2 * time.Second,
		},
		{
			name:     "Same timeouts",
			timeout1: 1 * time.Second,
			timeout2: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			results := make([]error, 2)
			startTimes := make([]time.Time, 2)
			endTimes := make([]time.Time, 2)

			// Manager 1 with timeout1
			wg.Add(1)
			go func() {
				defer wg.Done()
				startTimes[0] = time.Now()

				manager := NewStateTransitionManager(StateTransitionConfig{
					ResourceType: "vm",
					ResourceID:   "vm-1",
					StatusChecker: func(ctx context.Context) (string, error) {
						return "BUSY", nil // Never reaches target state
					},
					TargetStates:       []string{"POWERED_ON"},
					TransitionalStates: []string{"BUSY"},
					FailureStates:      []string{"error"},
					Timeout:            tt.timeout1,
					PollInterval:       100 * time.Millisecond,
				})

				results[0] = manager.WaitForStableState(ctx)
				endTimes[0] = time.Now()
			}()

			// Manager 2 with timeout2
			wg.Add(1)
			go func() {
				defer wg.Done()
				startTimes[1] = time.Now()

				manager := NewStateTransitionManager(StateTransitionConfig{
					ResourceType: "volume",
					ResourceID:   "vol-1",
					StatusChecker: func(ctx context.Context) (string, error) {
						return "BUSY", nil // Never reaches target state
					},
					TargetStates:       []string{"AVAILABLE"},
					TransitionalStates: []string{"BUSY"},
					FailureStates:      []string{"error"},
					Timeout:            tt.timeout2,
					PollInterval:       100 * time.Millisecond,
				})

				results[1] = manager.WaitForStableState(ctx)
				endTimes[1] = time.Now()
			}()

			wg.Wait()

			// Both should timeout
			assert.Error(t, results[0], "Manager 1 should timeout")
			assert.Error(t, results[1], "Manager 2 should timeout")

			// Verify each respected its own timeout (with tolerance)
			duration1 := endTimes[0].Sub(startTimes[0])
			duration2 := endTimes[1].Sub(startTimes[1])

			tolerance := 500 * time.Millisecond
			assert.InDelta(t, tt.timeout1.Seconds(), duration1.Seconds(), tolerance.Seconds(),
				"Manager 1 should respect its own timeout")
			assert.InDelta(t, tt.timeout2.Seconds(), duration2.Seconds(), tolerance.Seconds(),
				"Manager 2 should respect its own timeout")
		})
	}
}

// TestParallelOperations_MixedSuccessAndFailure tests parallel operations
// where some succeed and some fail independently
// Validates: Requirements 9.1, 9.2, 9.5
func TestParallelOperations_MixedSuccessAndFailure(t *testing.T) {
	ctx := context.Background()

	numGoroutines := 6
	var wg sync.WaitGroup
	results := make([]error, numGoroutines)

	// Define expected outcomes
	expectedOutcomes := []struct {
		shouldSucceed bool
		state         string
		targetStates  []string
	}{
		{true, "POWERED_ON", []string{"POWERED_ON"}},
		{false, "error", []string{"POWERED_ON"}},
		{true, "AVAILABLE", []string{"AVAILABLE"}},
		{false, "failed", []string{"AVAILABLE"}},
		{true, "RECOMPOSED", []string{"RECOMPOSED"}},
		{false, "BUSY", []string{"POWERED_ON"}}, // Will timeout
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			outcome := expectedOutcomes[index]

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "test",
				ResourceID:   fmt.Sprintf("resource-%d", index),
				StatusChecker: func(ctx context.Context) (string, error) {
					return outcome.state, nil
				},
				TargetStates:       outcome.targetStates,
				TransitionalStates: []string{"BUSY"},
				FailureStates:      []string{"error", "failed"},
				Timeout:            500 * time.Millisecond,
				PollInterval:       100 * time.Millisecond,
			})

			results[index] = manager.WaitForStableState(ctx)
		}(i)
	}

	wg.Wait()

	// Verify each operation had the expected outcome
	for i, result := range results {
		if expectedOutcomes[i].shouldSucceed {
			assert.NoError(t, result, "Operation %d should succeed", i)
		} else {
			assert.Error(t, result, "Operation %d should fail", i)
		}
	}
}

// TestParallelOperations_StateTransitions tests parallel operations
// where resources transition through states at different rates
// Validates: Requirements 9.1, 9.2, 9.5
func TestParallelOperations_StateTransitions(t *testing.T) {
	ctx := context.Background()

	numGoroutines := 4
	var wg sync.WaitGroup
	results := make([]error, numGoroutines)

	// Each resource transitions at a different rate
	transitionCounts := []int{1, 2, 3, 4}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			var callCount int32
			maxCalls := int32(transitionCounts[index])

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   fmt.Sprintf("vm-%d", index),
				StatusChecker: func(ctx context.Context) (string, error) {
					count := atomic.AddInt32(&callCount, 1)
					if count >= maxCalls {
						return "POWERED_ON", nil
					}
					return "BUSY", nil
				},
				TargetStates:       []string{"POWERED_ON"},
				TransitionalStates: []string{"BUSY"},
				FailureStates:      []string{"error"},
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			results[index] = manager.WaitForStableState(ctx)
		}(i)
	}

	wg.Wait()

	// All should succeed despite different transition rates
	for i, err := range results {
		assert.NoError(t, err, "Operation %d should succeed", i)
	}
}

// TestParallelOperations_HighConcurrency tests behavior under high
// concurrency with many parallel operations
// Validates: Requirements 9.1, 9.5
func TestParallelOperations_HighConcurrency(t *testing.T) {
	ctx := context.Background()

	numGoroutines := 50
	var wg sync.WaitGroup
	results := make([]error, numGoroutines)
	var successCount int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   fmt.Sprintf("vm-%d", index),
				StatusChecker: func(ctx context.Context) (string, error) {
					// Simulate some variability in response time
					time.Sleep(time.Duration(index%10) * time.Millisecond)
					return "POWERED_ON", nil
				},
				TargetStates:  []string{"POWERED_ON"},
				FailureStates: []string{"error"},
				Timeout:       5 * time.Second,
				PollInterval:  100 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)
			results[index] = err

			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// All operations should succeed
	assert.Equal(t, int32(numGoroutines), atomic.LoadInt32(&successCount),
		"All %d operations should succeed", numGoroutines)

	for i, err := range results {
		assert.NoError(t, err, "Operation %d should succeed", i)
	}
}

// TestParallelOperations_ContextCancellationPartial tests that
// cancelling one context doesn't affect other parallel operations
// Validates: Requirements 9.3, 9.4
func TestParallelOperations_ContextCancellationPartial(t *testing.T) {
	// Create separate contexts for each operation
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2 := context.Background()

	var wg sync.WaitGroup
	results := make([]error, 2)

	// Operation 1 - will be cancelled
	wg.Add(1)
	go func() {
		defer wg.Done()

		manager := NewStateTransitionManager(StateTransitionConfig{
			ResourceType: "vm",
			ResourceID:   "vm-1",
			StatusChecker: func(ctx context.Context) (string, error) {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(50 * time.Millisecond):
					return "BUSY", nil
				}
			},
			TargetStates:       []string{"POWERED_ON"},
			TransitionalStates: []string{"BUSY"},
			FailureStates:      []string{"error"},
			Timeout:            10 * time.Second,
			PollInterval:       100 * time.Millisecond,
		})

		results[0] = manager.WaitForStableState(ctx1)
	}()

	// Operation 2 - will complete successfully
	wg.Add(1)
	go func() {
		defer wg.Done()

		manager := NewStateTransitionManager(StateTransitionConfig{
			ResourceType: "volume",
			ResourceID:   "vol-1",
			StatusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			TargetStates:  []string{"AVAILABLE"},
			FailureStates: []string{"error"},
			Timeout:       5 * time.Second,
			PollInterval:  100 * time.Millisecond,
		})

		results[1] = manager.WaitForStableState(ctx2)
	}()

	// Cancel only the first operation
	time.Sleep(200 * time.Millisecond)
	cancel1()

	wg.Wait()

	// First operation should be cancelled
	assert.Error(t, results[0])
	assert.ErrorIs(t, results[0], context.Canceled)

	// Second operation should succeed
	assert.NoError(t, results[1])
}

// TestParallelOperations_NoRaceConditions tests that there are no
// race conditions when accessing manager state concurrently
// Validates: Requirements 9.1, 9.5
func TestParallelOperations_NoRaceConditions(t *testing.T) {
	ctx := context.Background()

	// Create a single manager
	var callCount int32
	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "vm-1",
		StatusChecker: func(ctx context.Context) (string, error) {
			atomic.AddInt32(&callCount, 1)
			return "POWERED_ON", nil
		},
		TargetStates:  []string{"POWERED_ON"},
		FailureStates: []string{"error"},
		Timeout:       5 * time.Second,
		PollInterval:  100 * time.Millisecond,
	})

	// Call methods concurrently on the same manager
	numGoroutines := 20
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Call various methods concurrently
			_ = manager.IsStableState("POWERED_ON")
			_ = manager.IsTransitionalState("BUSY")
			_ = manager.IsFailureState("error")
			_, _ = manager.CheckCurrentState(ctx)
		}()

	}

	wg.Wait()

	// If there were race conditions, the test would fail with -race flag
	// The test passing means no race conditions were detected
	assert.Greater(t, atomic.LoadInt32(&callCount), int32(0),
		"Status checker should have been called")
}

// TestParallelOperations_DifferentResourceTypes tests parallel operations
// on different resource types to ensure no interference
// Validates: Requirements 9.1, 9.2, 9.5
func TestParallelOperations_DifferentResourceTypes(t *testing.T) {
	ctx := context.Background()

	resourceConfigs := []struct {
		resourceType       string
		resourceID         string
		currentState       string
		targetStates       []string
		transitionalStates []string
		failureStates      []string
	}{
		{
			resourceType:       "vm",
			resourceID:         "vm-1",
			currentState:       "POWERED_ON",
			targetStates:       VMStableStates,
			transitionalStates: VMTransitionalStates,
			failureStates:      VMFailureStates,
		},
		{
			resourceType:       "volume",
			resourceID:         "vol-1",
			currentState:       "AVAILABLE",
			targetStates:       VolumeStableStates,
			transitionalStates: VolumeTransitionalStates,
			failureStates:      VolumeFailureStates,
		},
		{
			resourceType:       "security_group",
			resourceID:         "sg-1",
			currentState:       "RECOMPOSED",
			targetStates:       SecurityGroupStableStates,
			transitionalStates: SecurityGroupTransitionalStates,
			failureStates:      SecurityGroupFailureStates,
		},
	}

	var wg sync.WaitGroup
	results := make([]error, len(resourceConfigs))

	for i, config := range resourceConfigs {
		wg.Add(1)
		go func(index int, cfg struct {
			resourceType       string
			resourceID         string
			currentState       string
			targetStates       []string
			transitionalStates []string
			failureStates      []string
		}) {
			defer wg.Done()

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType:       cfg.resourceType,
				ResourceID:         cfg.resourceID,
				StatusChecker: func(ctx context.Context) (string, error) {
					return cfg.currentState, nil
				},
				TargetStates:       cfg.targetStates,
				TransitionalStates: cfg.transitionalStates,
				FailureStates:      cfg.failureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			results[index] = manager.WaitForStableState(ctx)
		}(i, config)
	}

	wg.Wait()

	// All operations should succeed
	for i, err := range results {
		assert.NoError(t, err, "Resource type %s should complete successfully",
			resourceConfigs[i].resourceType)
	}
}

// TestParallelOperations_ErrorPropagation tests that errors in one
// parallel operation don't affect other operations
// Validates: Requirements 9.1, 9.2
func TestParallelOperations_ErrorPropagation(t *testing.T) {
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make([]error, 3)

	// Operation 1 - will succeed
	wg.Add(1)
	go func() {
		defer wg.Done()

		manager := NewStateTransitionManager(StateTransitionConfig{
			ResourceType: "vm",
			ResourceID:   "vm-1",
			StatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			TargetStates:  []string{"POWERED_ON"},
			FailureStates: []string{"error"},
			Timeout:       5 * time.Second,
			PollInterval:  100 * time.Millisecond,
		})

		results[0] = manager.WaitForStableState(ctx)
	}()

	// Operation 2 - will fail with API error
	wg.Add(1)
	go func() {
		defer wg.Done()

		manager := NewStateTransitionManager(StateTransitionConfig{
			ResourceType: "volume",
			ResourceID:   "vol-1",
			StatusChecker: func(ctx context.Context) (string, error) {
				return "", errors.New("API connection failed")
			},
			TargetStates:  []string{"AVAILABLE"},
			FailureStates: []string{"error"},
			Timeout:       5 * time.Second,
			PollInterval:  100 * time.Millisecond,
		})

		results[1] = manager.WaitForStableState(ctx)
	}()

	// Operation 3 - will succeed
	wg.Add(1)
	go func() {
		defer wg.Done()

		manager := NewStateTransitionManager(StateTransitionConfig{
			ResourceType: "security_group",
			ResourceID:   "sg-1",
			StatusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSED", nil
			},
			TargetStates:  []string{"RECOMPOSED"},
			FailureStates: []string{"error"},
			Timeout:       5 * time.Second,
			PollInterval:  100 * time.Millisecond,
		})

		results[2] = manager.WaitForStableState(ctx)
	}()

	wg.Wait()

	// First operation should succeed
	assert.NoError(t, results[0], "First operation should succeed")

	// Second operation should fail
	assert.Error(t, results[1], "Second operation should fail")
	assert.Contains(t, results[1].Error(), "failed to check current state")

	// Third operation should succeed (not affected by second operation's error)
	assert.NoError(t, results[2], "Third operation should succeed despite second operation's failure")
}

// TestParallelOperations_ContextDeadline tests that context deadlines
// are respected independently for each parallel operation
// Validates: Requirements 9.3, 9.4
func TestParallelOperations_ContextDeadline(t *testing.T) {
	var wg sync.WaitGroup
	results := make([]error, 2)

	// Operation 1 - short deadline
	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		manager := NewStateTransitionManager(StateTransitionConfig{
			ResourceType: "vm",
			ResourceID:   "vm-1",
			StatusChecker: func(ctx context.Context) (string, error) {
				return "BUSY", nil // Never reaches target
			},
			TargetStates:       []string{"POWERED_ON"},
			TransitionalStates: []string{"BUSY"},
			FailureStates:      []string{"error"},
			Timeout:            10 * time.Second, // Longer than context deadline
			PollInterval:       100 * time.Millisecond,
		})

		results[0] = manager.WaitForStableState(ctx)
	}()

	// Operation 2 - longer deadline, should complete
	wg.Add(1)
	go func() {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		manager := NewStateTransitionManager(StateTransitionConfig{
			ResourceType: "volume",
			ResourceID:   "vol-1",
			StatusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			TargetStates:  []string{"AVAILABLE"},
			FailureStates: []string{"error"},
			Timeout:       10 * time.Second,
			PollInterval:  100 * time.Millisecond,
		})

		results[1] = manager.WaitForStableState(ctx)
	}()

	wg.Wait()

	// First operation should timeout due to context deadline
	assert.Error(t, results[0])
	assert.ErrorIs(t, results[0], context.DeadlineExceeded)

	// Second operation should succeed
	assert.NoError(t, results[1])
}

// TestParallelOperations_IsInTargetState tests concurrent calls to
// IsInTargetState on different managers
// Validates: Requirements 9.1, 9.5
func TestParallelOperations_IsInTargetState(t *testing.T) {
	ctx := context.Background()

	numGoroutines := 10
	var wg sync.WaitGroup
	results := make([]bool, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Half in target state, half not
			state := "BUSY"
			if index%2 == 0 {
				state = "POWERED_ON"
			}

			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   fmt.Sprintf("vm-%d", index),
				StatusChecker: func(ctx context.Context) (string, error) {
					return state, nil
				},
				TargetStates:  []string{"POWERED_ON"},
				FailureStates: []string{"error"},
				Timeout:       5 * time.Second,
				PollInterval:  100 * time.Millisecond,
			})

			isTarget, _, err := manager.IsInTargetState(ctx)
			results[index] = isTarget
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify results
	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, errors[i], "Operation %d should not error", i)

		if i%2 == 0 {
			assert.True(t, results[i], "Operation %d should be in target state", i)
		} else {
			assert.False(t, results[i], "Operation %d should not be in target state", i)
		}
	}
}
