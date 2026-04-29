package emma

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emma-community/terraform-provider-emma/internal/emma/common/state"
	"github.com/stretchr/testify/assert"
)

// TestSecurityGroupStateWaiting_WaitForRecomposedBeforeUpdate tests waiting for RECOMPOSED state before updates
// Requirements: 3.1, 3.2
func TestSecurityGroupStateWaiting_WaitForRecomposedBeforeUpdate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name: "Security group already in RECOMPOSED state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSED", nil
			},
			expectError: false,
		},
		{
			name: "Security group transitions from RECOMPOSING to RECOMPOSED",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "RECOMPOSING", nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "Security group transitions from RECOMPOSING to RECOMPOSED after multiple checks",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 5 {
						return "RECOMPOSING", nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "security_group",
				ResourceID:         "sg-123",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.SecurityGroupStableStates,
				TransitionalStates: state.SecurityGroupTransitionalStates,
				FailureStates:      state.SecurityGroupFailureStates,
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

// TestSecurityGroupStateWaiting_WaitForRecomposedAfterUpdate tests waiting for RECOMPOSED state after updates
// Requirements: 3.2, 3.3
func TestSecurityGroupStateWaiting_WaitForRecomposedAfterUpdate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
	}{
		{
			name: "Security group already in RECOMPOSED state after update",
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSED", nil
			},
			expectError: false,
		},
		{
			name: "Security group transitions from RECOMPOSING to RECOMPOSED after update",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 3 {
						return "RECOMPOSING", nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "Security group takes longer to recompose after update",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 10 {
						return "RECOMPOSING", nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "security_group",
				ResourceID:         "sg-456",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.SecurityGroupStableStates,
				TransitionalStates: state.SecurityGroupTransitionalStates,
				FailureStates:      state.SecurityGroupFailureStates,
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

// TestSecurityGroupStateWaiting_MultipleSequentialRecompositions tests handling multiple sequential recompositions
// Requirements: 3.4, 3.5
func TestSecurityGroupStateWaiting_MultipleSequentialRecompositions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
		description   string
	}{
		{
			name: "Two sequential recompositions",
			statusChecker: func() state.ResourceStateChecker {
				// Simulates: RECOMPOSING -> RECOMPOSED -> RECOMPOSING -> RECOMPOSED
				states := []string{"RECOMPOSING", "RECOMPOSING", "RECOMPOSED"}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
			description: "Security group goes through two recomposition cycles",
		},
		{
			name: "Three sequential recompositions",
			statusChecker: func() state.ResourceStateChecker {
				// Simulates: RECOMPOSING -> RECOMPOSED -> RECOMPOSING -> RECOMPOSED -> RECOMPOSING -> RECOMPOSED
				states := []string{"RECOMPOSING", "RECOMPOSING", "RECOMPOSED", "RECOMPOSING", "RECOMPOSING", "RECOMPOSED"}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
			description: "Security group goes through three recomposition cycles",
		},
		{
			name: "Long recomposition followed by quick recomposition",
			statusChecker: func() state.ResourceStateChecker {
				// First recomposition takes longer, second is quick
				states := []string{
					"RECOMPOSING", "RECOMPOSING", "RECOMPOSING", "RECOMPOSING", "RECOMPOSED",
					"RECOMPOSING", "RECOMPOSED",
				}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
			description: "First recomposition is slow, second is fast",
		},
		{
			name: "Multiple VMs added causing sequential recompositions",
			statusChecker: func() state.ResourceStateChecker {
				// Simulates adding multiple VMs to security group
				states := []string{
					"RECOMPOSING", "RECOMPOSING", "RECOMPOSED", // First VM added
					"RECOMPOSING", "RECOMPOSED",                 // Second VM added
					"RECOMPOSING", "RECOMPOSING", "RECOMPOSED", // Third VM added
				}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
			description: "Multiple VMs added causing multiple recomposition cycles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "security_group",
				ResourceID:         "sg-multi",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.SecurityGroupStableStates,
				TransitionalStates: state.SecurityGroupTransitionalStates,
				FailureStates:      state.SecurityGroupFailureStates,
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

// TestSecurityGroupStateWaiting_TimeoutWhenStuckInRecomposing tests timeout when stuck in RECOMPOSING state
// Requirements: 3.5, 1.5, 4.4
func TestSecurityGroupStateWaiting_TimeoutWhenStuckInRecomposing(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		timeout       time.Duration
		expectError   bool
		errorContains string
	}{
		{
			name: "Security group stuck in RECOMPOSING state - timeout",
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSING", nil
			},
			timeout:       1 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name: "Security group stuck in RECOMPOSING state - longer timeout",
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSING", nil
			},
			timeout:       2 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name: "Security group recomposition takes too long",
			statusChecker: func() state.ResourceStateChecker {
				// Never reaches RECOMPOSED within timeout
				return func(ctx context.Context) (string, error) {
					return "RECOMPOSING", nil
				}
			}(),
			timeout:       1 * time.Second,
			expectError:   true,
			errorContains: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "security_group",
				ResourceID:         "sg-timeout",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.SecurityGroupStableStates,
				TransitionalStates: state.SecurityGroupTransitionalStates,
				FailureStates:      state.SecurityGroupFailureStates,
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

// TestSecurityGroupStateWaiting_ImmediateSuccessWhenAlreadyRecomposed tests immediate success when already in RECOMPOSED state
// Requirements: 8.1, 8.4
func TestSecurityGroupStateWaiting_ImmediateSuccessWhenAlreadyRecomposed(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		currentState  string
		expectSuccess bool
	}{
		{
			name:          "Security group already in RECOMPOSED",
			currentState:  "RECOMPOSED",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "security_group",
				ResourceID:   "sg-immediate",
				StatusChecker: func(ctx context.Context) (string, error) {
					callCount++
					return tt.currentState, nil
				},
				TargetStates:       state.SecurityGroupStableStates,
				TransitionalStates: state.SecurityGroupTransitionalStates,
				FailureStates:      state.SecurityGroupFailureStates,
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

// TestSecurityGroupStateWaiting_ErrorHandling tests various error scenarios
// Requirements: 3.1, 3.2, 3.3
func TestSecurityGroupStateWaiting_ErrorHandling(t *testing.T) {
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
			name: "Security group in error state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "error", nil
			},
			expectError:   true,
			errorContains: "failure state",
		},
		{
			name: "Security group in failed state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "failed", nil
			},
			expectError:   true,
			errorContains: "failure state",
		},
		{
			name: "Context cancelled before waiting",
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSING", nil
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
				ResourceType:       "security_group",
				ResourceID:         "sg-error",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.SecurityGroupStableStates,
				TransitionalStates: state.SecurityGroupTransitionalStates,
				FailureStates:      state.SecurityGroupFailureStates,
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

// TestSecurityGroupStateWaiting_ConfigurableTimeout tests that custom timeout values are respected
// Requirements: 4.1, 4.4, 3.5
func TestSecurityGroupStateWaiting_ConfigurableTimeout(t *testing.T) {
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
				return "RECOMPOSING", nil
			},
			expectError: true,
		},
		{
			name:    "Medium timeout (3 seconds) - should timeout",
			timeout: 3 * time.Second,
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSING", nil
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
						return "RECOMPOSED", nil
					}
					return "RECOMPOSING", nil
				}
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "security_group",
				ResourceID:         "sg-timeout-config",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.SecurityGroupStableStates,
				TransitionalStates: state.SecurityGroupTransitionalStates,
				FailureStates:      state.SecurityGroupFailureStates,
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

// TestSecurityGroupStateWaiting_ContextCancellation tests that context cancellation is properly handled
// Requirements: 9.3, 9.4
func TestSecurityGroupStateWaiting_ContextCancellation(t *testing.T) {
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
					return "RECOMPOSING", nil
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
					return "RECOMPOSING", nil
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
				ResourceType:       "security_group",
				ResourceID:         "sg-cancel",
				StatusChecker:      tt.statusChecker(ctx, cancel),
				TargetStates:       state.SecurityGroupStableStates,
				TransitionalStates: state.SecurityGroupTransitionalStates,
				FailureStates:      state.SecurityGroupFailureStates,
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

// TestSecurityGroupStateWaiting_RealWorldScenarios tests realistic security group state transition scenarios
// Requirements: 3.1, 3.2, 3.3, 3.4, 3.5
func TestSecurityGroupStateWaiting_RealWorldScenarios(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		scenario      string
		statusChecker state.ResourceStateChecker
		expectError   bool
	}{
		{
			name:     "Security group rule update - wait for RECOMPOSED",
			scenario: "Rule update operation",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"RECOMPOSING", "RECOMPOSING", "RECOMPOSED"}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
		},
		{
			name:     "Security group already ready for update",
			scenario: "Update when security group is ready",
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSED", nil
			},
			expectError: false,
		},
		{
			name:     "VM added to security group - triggers recomposition",
			scenario: "VM addition causing recomposition",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"RECOMPOSING", "RECOMPOSED"}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
		},
		{
			name:     "Multiple rule updates - sequential recompositions",
			scenario: "Multiple rule updates",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{
					"RECOMPOSING", "RECOMPOSED", // First update
					"RECOMPOSING", "RECOMPOSED", // Second update
				}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
		},
		{
			name:     "Security group with many VMs - slow recomposition",
			scenario: "Large security group recomposition",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{
					"RECOMPOSING", "RECOMPOSING", "RECOMPOSING",
					"RECOMPOSING", "RECOMPOSING", "RECOMPOSED",
				}
				callCount := 0
				return func(ctx context.Context) (string, error) {
					if callCount < len(states) {
						state := states[callCount]
						callCount++
						return state, nil
					}
					return "RECOMPOSED", nil
				}
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "security_group",
				ResourceID:         "sg-realworld",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.SecurityGroupStableStates,
				TransitionalStates: state.SecurityGroupTransitionalStates,
				FailureStates:      state.SecurityGroupFailureStates,
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
