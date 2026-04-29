package emma

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emma-community/terraform-provider-emma/internal/emma/common/state"
	"github.com/stretchr/testify/assert"
)

// TestVMStateWaiting_WaitForPoweredOn tests waiting for VM to reach POWERED_ON state
// Requirements: 1.1, 7.1, 8.1
func TestVMStateWaiting_WaitForPoweredOn(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name: "VM already in POWERED_ON state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			expectError: false,
		},
		{
			name: "VM transitions from BUSY to POWERED_ON",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "BUSY", nil
					}
					return "POWERED_ON", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "VM transitions from pending to POWERED_ON",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "pending", nil
					}
					return "POWERED_ON", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "VM already in running state (alternative stable state)",
			statusChecker: func(ctx context.Context) (string, error) {
				return "running", nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-123",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
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

// TestVMStateWaiting_WaitForPoweredOff tests waiting for VM to reach POWERED_OFF state
// Requirements: 7.2, 8.1
func TestVMStateWaiting_WaitForPoweredOff(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
	}{
		{
			name: "VM already in POWERED_OFF state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_OFF", nil
			},
			expectError: false,
		},
		{
			name: "VM transitions from BUSY to POWERED_OFF",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "BUSY", nil
					}
					return "POWERED_OFF", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "VM transitions from stopping to POWERED_OFF",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 3 {
						return "stopping", nil
					}
					return "POWERED_OFF", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "VM already in stopped state (alternative stable state)",
			statusChecker: func(ctx context.Context) (string, error) {
				return "stopped", nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-456",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVMStateWaiting_TimeoutWhenStuckInBusy tests timeout when VM stuck in BUSY state
// Requirements: 7.3, 1.5, 4.4
func TestVMStateWaiting_TimeoutWhenStuckInBusy(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		timeout       time.Duration
		expectError   bool
		errorContains string
	}{
		{
			name: "VM stuck in BUSY state - timeout",
			statusChecker: func(ctx context.Context) (string, error) {
				return "BUSY", nil
			},
			timeout:       1 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name: "VM stuck in pending state - timeout",
			statusChecker: func(ctx context.Context) (string, error) {
				return "pending", nil
			},
			timeout:       1 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name: "VM stuck in starting state - timeout",
			statusChecker: func(ctx context.Context) (string, error) {
				return "starting", nil
			},
			timeout:       1 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name: "VM stuck in stopping state - timeout",
			statusChecker: func(ctx context.Context) (string, error) {
				return "stopping", nil
			},
			timeout:       1 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-789",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            tt.timeout,
				PollInterval:       100 * time.Millisecond,
			})

			start := time.Now()
			err := manager.WaitForStableState(ctx)
			duration := time.Since(start)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				// Verify timeout was respected (with some tolerance)
				assert.LessOrEqual(t, duration, tt.timeout+500*time.Millisecond)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVMStateWaiting_ImmediateSuccessWhenAlreadyStable tests immediate success when VM already in stable state
// Requirements: 8.1, 8.4
func TestVMStateWaiting_ImmediateSuccessWhenAlreadyStable(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		currentState  string
		expectSuccess bool
	}{
		{
			name:          "VM already in POWERED_ON",
			currentState:  "POWERED_ON",
			expectSuccess: true,
		},
		{
			name:          "VM already in POWERED_OFF",
			currentState:  "POWERED_OFF",
			expectSuccess: true,
		},
		{
			name:          "VM already in running",
			currentState:  "running",
			expectSuccess: true,
		},
		{
			name:          "VM already in stopped",
			currentState:  "stopped",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   "vm-immediate",
				StatusChecker: func(ctx context.Context) (string, error) {
					callCount++
					return tt.currentState, nil
				},
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			start := time.Now()
			err := manager.WaitForStableState(ctx)
			duration := time.Since(start)

			if tt.expectSuccess {
				assert.NoError(t, err)
				// Should return immediately without polling
				assert.Less(t, duration, 500*time.Millisecond, "Should return immediately for stable state")
				// Should only call status checker once (initial check)
				assert.Equal(t, 1, callCount, "Should only check status once when already stable")
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestVMStateWaiting_ErrorHandling tests various error scenarios
// Requirements: 1.1, 7.1, 7.2, 7.3
func TestVMStateWaiting_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name: "API error when checking status",
			statusChecker: func(ctx context.Context) (string, error) {
				return "", errors.New("API connection failed")
			},
			expectError:   true,
			errorContains: "failed to check current state",
		},
		{
			name: "VM in error state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "error", nil
			},
			expectError:   true,
			errorContains: "failure state",
		},
		{
			name: "VM in failed state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "failed", nil
			},
			expectError:   true,
			errorContains: "failure state",
		},
		{
			name: "Context cancelled before waiting",
			statusChecker: func(ctx context.Context) (string, error) {
				return "BUSY", nil
			},
			expectError:   true,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx := ctx
			if tt.name == "Context cancelled before waiting" {
				var cancel context.CancelFunc
				testCtx, cancel = context.WithCancel(ctx)
				cancel() // Cancel immediately
			}

			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-error",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            2 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := manager.WaitForStableState(testCtx)

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

// TestVMStateWaiting_MultipleTransitions tests VM going through multiple state transitions
// Requirements: 1.1, 7.1
func TestVMStateWaiting_MultipleTransitions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		stateSequence  []string
		expectError    bool
		expectedCalls  int
	}{
		{
			name:          "pending -> starting -> BUSY -> POWERED_ON",
			stateSequence: []string{"pending", "starting", "BUSY", "POWERED_ON"},
			expectError:   false,
			expectedCalls: 4,
		},
		{
			name:          "BUSY -> stopping -> POWERED_OFF",
			stateSequence: []string{"BUSY", "stopping", "POWERED_OFF"},
			expectError:   false,
			expectedCalls: 3,
		},
		{
			name:          "starting -> BUSY -> running",
			stateSequence: []string{"starting", "BUSY", "running"},
			expectError:   false,
			expectedCalls: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   "vm-multi",
				StatusChecker: func(ctx context.Context) (string, error) {
					if callCount < len(tt.stateSequence) {
						state := tt.stateSequence[callCount]
						callCount++
						return state, nil
					}
					// Return last state if we've exhausted the sequence
					return tt.stateSequence[len(tt.stateSequence)-1], nil
				},
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, callCount, tt.expectedCalls, "Should have made expected number of status checks")
			}
		})
	}
}

// TestVMStateWaiting_ConfigurableTimeout tests that custom timeout values are respected
// Requirements: 4.1, 4.4, 7.3
func TestVMStateWaiting_ConfigurableTimeout(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		timeout       time.Duration
		statusChecker state.ResourceStateChecker
		expectError   bool
	}{
		{
			name:    "Short timeout (1 second) - should timeout",
			timeout: 1 * time.Second,
			statusChecker: func(ctx context.Context) (string, error) {
				return "BUSY", nil
			},
			expectError: true,
		},
		{
			name:    "Medium timeout (3 seconds) - should timeout",
			timeout: 3 * time.Second,
			statusChecker: func(ctx context.Context) (string, error) {
				return "BUSY", nil
			},
			expectError: true,
		},
		{
			name:    "Long timeout (5 seconds) - should succeed",
			timeout: 5 * time.Second,
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					// Transition after 2 seconds worth of calls
					if callCount > 20 {
						return "POWERED_ON", nil
					}
					return "BUSY", nil
				}
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-timeout",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            tt.timeout,
				PollInterval:       100 * time.Millisecond,
			})

			start := time.Now()
			err := manager.WaitForStableState(ctx)
			duration := time.Since(start)

			if tt.expectError {
				assert.Error(t, err)
				// Verify timeout was respected (with some tolerance for processing time)
				assert.LessOrEqual(t, duration, tt.timeout+500*time.Millisecond)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVMStateWaiting_AllVMStableStates tests that all defined VM stable states are recognized
// Requirements: 8.1, 8.4
func TestVMStateWaiting_AllVMStableStates(t *testing.T) {
	ctx := context.Background()

	// Test each stable state defined in VMStableStates
	stableStates := []string{"running", "stopped", "POWERED_ON", "POWERED_OFF"}

	for _, stableState := range stableStates {
		t.Run("VM in "+stableState+" state", func(t *testing.T) {
			callCount := 0
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   "vm-stable-" + stableState,
				StatusChecker: func(ctx context.Context) (string, error) {
					callCount++
					return stableState, nil
				},
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			start := time.Now()
			err := manager.WaitForStableState(ctx)
			duration := time.Since(start)

			assert.NoError(t, err)
			// Should return immediately for stable state
			assert.Less(t, duration, 500*time.Millisecond)
			// Should only check once
			assert.Equal(t, 1, callCount)
		})
	}
}

// TestVMStateWaiting_AllVMTransitionalStates tests that all defined VM transitional states trigger waiting
// Requirements: 1.1, 1.2
func TestVMStateWaiting_AllVMTransitionalStates(t *testing.T) {
	ctx := context.Background()

	// Test each transitional state defined in VMTransitionalStates
	transitionalStates := []string{"BUSY", "pending", "starting", "stopping"}

	for _, transitionalState := range transitionalStates {
		t.Run("VM transitions from "+transitionalState+" to POWERED_ON", func(t *testing.T) {
			callCount := 0
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   "vm-trans-" + transitionalState,
				StatusChecker: func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return transitionalState, nil
					}
					return "POWERED_ON", nil
				},
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)

			assert.NoError(t, err)
			// Should have polled multiple times
			assert.GreaterOrEqual(t, callCount, 3)
		})
	}
}

// TestVMStateWaiting_ContextCancellation tests that context cancellation is properly handled
// Requirements: 9.3, 9.4
func TestVMStateWaiting_ContextCancellation(t *testing.T) {
	tests := []struct {
		name              string
		cancelTiming      string
		statusChecker     func(context.Context, context.CancelFunc) state.ResourceStateChecker
		expectError       bool
		expectCancelError bool
	}{
		{
			name:         "Context cancelled before waiting",
			cancelTiming: "before",
			statusChecker: func(ctx context.Context, cancel context.CancelFunc) state.ResourceStateChecker {
				return func(ctx context.Context) (string, error) {
					return "BUSY", nil
				}
			},
			expectError:       true,
			expectCancelError: true,
		},
		{
			name:         "Context cancelled during waiting",
			cancelTiming: "during",
			statusChecker: func(ctx context.Context, cancel context.CancelFunc) state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount == 2 {
						cancel() // Cancel after second check
					}
					return "BUSY", nil
				}
			},
			expectError:       true,
			expectCancelError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelTiming == "before" {
				cancel()
			}

			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-cancel",
				StatusChecker:      tt.statusChecker(ctx, cancel),
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectCancelError {
					assert.ErrorIs(t, err, context.Canceled)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVMStateWaiting_RealWorldScenarios tests realistic VM state transition scenarios
// Requirements: 1.1, 7.1, 7.2, 8.1
func TestVMStateWaiting_RealWorldScenarios(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		scenario      string
		statusChecker state.ResourceStateChecker
		expectError   bool
	}{
		{
			name:     "VM hardware edit - wait for POWERED_ON",
			scenario: "Hardware edit operation",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"BUSY", "BUSY", "POWERED_ON"}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "POWERED_ON", nil
				}
			}(),
			expectError: false,
		},
		{
			name:     "Volume attachment - VM already ready",
			scenario: "Volume attach when VM is ready",
			statusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			expectError: false,
		},
		{
			name:     "Volume detachment - wait for VM to stabilize",
			scenario: "Volume detach after VM operation",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"BUSY", "POWERED_ON"}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "POWERED_ON", nil
				}
			}(),
			expectError: false,
		},
		{
			name:     "Security group change - VM in stable state",
			scenario: "Security group update",
			statusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			expectError: false,
		},
		{
			name:     "VM restart - transitions through stopping and starting",
			scenario: "VM restart operation",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"stopping", "POWERED_OFF", "starting", "POWERED_ON"}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "POWERED_ON", nil
				}
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-realworld",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            10 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := manager.WaitForStableState(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
