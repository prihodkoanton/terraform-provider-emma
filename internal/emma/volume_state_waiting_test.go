package emma

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emma-community/terraform-provider-emma/internal/emma/common/state"
	"github.com/stretchr/testify/assert"
)

// TestVolumeStateWaiting_WaitForAvailableBeforeAttach tests waiting for volume to reach AVAILABLE before attach
// Requirements: 1.3, 7.1
func TestVolumeStateWaiting_WaitForAvailableBeforeAttach(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name: "Volume already in AVAILABLE state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError: false,
		},
		{
			name: "Volume transitions from BUSY to AVAILABLE",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "BUSY", nil
					}
					return "AVAILABLE", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "Volume transitions from creating to AVAILABLE",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "creating", nil
					}
					return "AVAILABLE", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "Volume already in available state (alternative stable state)",
			statusChecker: func(ctx context.Context) (string, error) {
				return "available", nil
			},
			expectError: false,
		},
		{
			name: "Volume already in in-use state (stable state)",
			statusChecker: func(ctx context.Context) (string, error) {
				return "in-use", nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "volume",
				ResourceID:         "vol-123",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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

// TestVolumeStateWaiting_WaitForAvailableBeforeDetach tests waiting for volume to reach AVAILABLE before detach
// Requirements: 1.3, 7.2
func TestVolumeStateWaiting_WaitForAvailableBeforeDetach(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
	}{
		{
			name: "Volume already in AVAILABLE state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError: false,
		},
		{
			name: "Volume transitions from BUSY to AVAILABLE",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "BUSY", nil
					}
					return "AVAILABLE", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "Volume transitions from detaching to AVAILABLE",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 3 {
						return "detaching", nil
					}
					return "AVAILABLE", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "Volume already in available state (alternative stable state)",
			statusChecker: func(ctx context.Context) (string, error) {
				return "available", nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "volume",
				ResourceID:         "vol-456",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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

// TestVolumeStateWaiting_WaitForAvailableBeforeResize tests waiting for volume to reach AVAILABLE before resize
// Requirements: 1.3, 7.3
func TestVolumeStateWaiting_WaitForAvailableBeforeResize(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
	}{
		{
			name: "Volume already in AVAILABLE state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError: false,
		},
		{
			name: "Volume transitions from BUSY to AVAILABLE",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "BUSY", nil
					}
					return "AVAILABLE", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "Volume transitions from DRAFT to AVAILABLE",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "DRAFT", nil
					}
					return "AVAILABLE", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "Volume already in in-use state (can resize while attached)",
			statusChecker: func(ctx context.Context) (string, error) {
				return "in-use", nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "volume",
				ResourceID:         "vol-789",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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

// TestVolumeStateWaiting_TimeoutWhenStuckInBusy tests timeout when volume stuck in BUSY state
// Requirements: 7.4, 1.5, 4.4
func TestVolumeStateWaiting_TimeoutWhenStuckInBusy(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		timeout       time.Duration
		expectError   bool
		errorContains string
	}{
		{
			name: "Volume stuck in BUSY state - timeout",
			statusChecker: func(ctx context.Context) (string, error) {
				return "BUSY", nil
			},
			timeout:       1 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name: "Volume stuck in DRAFT state - timeout",
			statusChecker: func(ctx context.Context) (string, error) {
				return "DRAFT", nil
			},
			timeout:       1 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name: "Volume stuck in creating state - timeout",
			statusChecker: func(ctx context.Context) (string, error) {
				return "creating", nil
			},
			timeout:       1 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name: "Volume stuck in attaching state - timeout",
			statusChecker: func(ctx context.Context) (string, error) {
				return "attaching", nil
			},
			timeout:       1 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name: "Volume stuck in detaching state - timeout",
			statusChecker: func(ctx context.Context) (string, error) {
				return "detaching", nil
			},
			timeout:       1 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "volume",
				ResourceID:         "vol-timeout",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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

// TestVolumeStateWaiting_ImmediateSuccessWhenAlreadyStable tests immediate success when volume already in stable state
// Requirements: 8.1, 8.4
func TestVolumeStateWaiting_ImmediateSuccessWhenAlreadyStable(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		currentState  string
		expectSuccess bool
	}{
		{
			name:          "Volume already in AVAILABLE",
			currentState:  "AVAILABLE",
			expectSuccess: true,
		},
		{
			name:          "Volume already in available",
			currentState:  "available",
			expectSuccess: true,
		},
		{
			name:          "Volume already in in-use",
			currentState:  "in-use",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "volume",
				ResourceID:   "vol-immediate",
				StatusChecker: func(ctx context.Context) (string, error) {
					callCount++
					return tt.currentState, nil
				},
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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

// TestVolumeStateWaiting_ErrorHandling tests various error scenarios
// Requirements: 1.3, 7.1, 7.2, 7.3
func TestVolumeStateWaiting_ErrorHandling(t *testing.T) {
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
			name: "Volume in error state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "error", nil
			},
			expectError:   true,
			errorContains: "failure state",
		},
		{
			name: "Volume in failed state",
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
				ResourceType:       "volume",
				ResourceID:         "vol-error",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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

// TestVolumeStateWaiting_MultipleTransitions tests volume going through multiple state transitions
// Requirements: 1.3, 7.1
func TestVolumeStateWaiting_MultipleTransitions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		stateSequence []string
		expectError   bool
		expectedCalls int
	}{
		{
			name:          "DRAFT -> creating -> BUSY -> AVAILABLE",
			stateSequence: []string{"DRAFT", "creating", "BUSY", "AVAILABLE"},
			expectError:   false,
			expectedCalls: 4,
		},
		{
			name:          "BUSY -> attaching -> in-use",
			stateSequence: []string{"BUSY", "attaching", "in-use"},
			expectError:   false,
			expectedCalls: 3,
		},
		{
			name:          "detaching -> AVAILABLE",
			stateSequence: []string{"detaching", "AVAILABLE"},
			expectError:   false,
			expectedCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "volume",
				ResourceID:   "vol-multi",
				StatusChecker: func(ctx context.Context) (string, error) {
					if callCount < len(tt.stateSequence) {
						state := tt.stateSequence[callCount]
						callCount++
						return state, nil
					}
					// Return last state if we've exhausted the sequence
					return tt.stateSequence[len(tt.stateSequence)-1], nil
				},
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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

// TestVolumeStateWaiting_ConfigurableTimeout tests that custom timeout values are respected
// Requirements: 4.1, 4.4, 7.4
func TestVolumeStateWaiting_ConfigurableTimeout(t *testing.T) {
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
						return "AVAILABLE", nil
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
				ResourceType:       "volume",
				ResourceID:         "vol-timeout",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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

// TestVolumeStateWaiting_AllVolumeStableStates tests that all defined volume stable states are recognized
// Requirements: 8.1, 8.4
func TestVolumeStateWaiting_AllVolumeStableStates(t *testing.T) {
	ctx := context.Background()

	// Test each stable state defined in VolumeStableStates
	stableStates := []string{"available", "in-use", "AVAILABLE"}

	for _, stableState := range stableStates {
		t.Run("Volume in "+stableState+" state", func(t *testing.T) {
			callCount := 0
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "volume",
				ResourceID:   "vol-stable-" + stableState,
				StatusChecker: func(ctx context.Context) (string, error) {
					callCount++
					return stableState, nil
				},
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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

// TestVolumeStateWaiting_AllVolumeTransitionalStates tests that all defined volume transitional states trigger waiting
// Requirements: 1.3
func TestVolumeStateWaiting_AllVolumeTransitionalStates(t *testing.T) {
	ctx := context.Background()

	// Test each transitional state defined in VolumeTransitionalStates
	transitionalStates := []string{"BUSY", "DRAFT", "creating", "attaching", "detaching"}

	for _, transitionalState := range transitionalStates {
		t.Run("Volume transitions from "+transitionalState+" to AVAILABLE", func(t *testing.T) {
			callCount := 0
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "volume",
				ResourceID:   "vol-trans-" + transitionalState,
				StatusChecker: func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return transitionalState, nil
					}
					return "AVAILABLE", nil
				},
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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

// TestVolumeStateWaiting_ContextCancellation tests that context cancellation is properly handled
// Requirements: 9.3, 9.4
func TestVolumeStateWaiting_ContextCancellation(t *testing.T) {
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
				ResourceType:       "volume",
				ResourceID:         "vol-cancel",
				StatusChecker:      tt.statusChecker(ctx, cancel),
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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

// TestVolumeStateWaiting_RealWorldScenarios tests realistic volume state transition scenarios
// Requirements: 1.3, 7.1, 7.2, 7.3, 8.1
func TestVolumeStateWaiting_RealWorldScenarios(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		scenario      string
		statusChecker state.ResourceStateChecker
		expectError   bool
	}{
		{
			name:     "Volume resize - wait for AVAILABLE",
			scenario: "Volume resize operation",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"BUSY", "BUSY", "AVAILABLE"}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "AVAILABLE", nil
				}
			}(),
			expectError: false,
		},
		{
			name:     "Volume attachment - volume already ready",
			scenario: "Volume attach when volume is ready",
			statusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError: false,
		},
		{
			name:     "Volume detachment - wait for volume to stabilize",
			scenario: "Volume detach after operation",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"BUSY", "AVAILABLE"}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "AVAILABLE", nil
				}
			}(),
			expectError: false,
		},
		{
			name:     "Volume creation - transitions through creating",
			scenario: "Volume creation operation",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"creating", "DRAFT", "AVAILABLE"}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "AVAILABLE", nil
				}
			}(),
			expectError: false,
		},
		{
			name:     "Volume attachment to VM - transitions through attaching",
			scenario: "Volume attach operation",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"AVAILABLE", "attaching", "in-use"}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "in-use", nil
				}
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "volume",
				ResourceID:         "vol-realworld",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
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
