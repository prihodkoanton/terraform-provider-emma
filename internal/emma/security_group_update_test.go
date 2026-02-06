package emma

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/emma-community/terraform-provider-emma/internal/emma/common/retry"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/state"
	"github.com/stretchr/testify/assert"
)

// TestSecurityGroupUpdate_WaitsForStableState tests that update waits for security group to reach stable state
// Requirements: 3.1, 3.2, 3.3
func TestSecurityGroupUpdate_WaitsForStableState(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name: "Security group already in RECOMPOSED - immediate success",
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
			name: "Security group takes multiple checks to reach RECOMPOSED",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"RECOMPOSING", "RECOMPOSING", "RECOMPOSING", "RECOMPOSED"}
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
			name: "Security group already stable - no waiting needed",
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSED", nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "security_group",
				ResourceID:         "sg-update-123",
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

// TestSecurityGroupUpdate_RetriesOnStateConflict tests that update retries when state conflict occurs
// Requirements: 5.1, 5.2
func TestSecurityGroupUpdate_RetriesOnStateConflict(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		apiError    string
		expectRetry bool
	}{
		{
			name:        "409 Conflict status code triggers retry",
			statusCode:  http.StatusConflict,
			apiError:    "Resource conflict",
			expectRetry: true,
		},
		{
			name:        "State-related error message triggers retry",
			statusCode:  http.StatusBadRequest,
			apiError:    "Security group is in recomposing state",
			expectRetry: true,
		},
		{
			name:        "Recomposing state error triggers retry",
			statusCode:  http.StatusBadRequest,
			apiError:    "Cannot update: security group is recomposing",
			expectRetry: true,
		},
		{
			name:        "Resource conflict message triggers retry",
			statusCode:  http.StatusBadRequest,
			apiError:    "Resource conflict detected",
			expectRetry: true,
		},
		{
			name:        "Busy state triggers retry",
			statusCode:  http.StatusBadRequest,
			apiError:    "Security group is busy",
			expectRetry: true,
		},
		{
			name:        "Non-state error does not trigger retry",
			statusCode:  http.StatusBadRequest,
			apiError:    "Invalid rule configuration",
			expectRetry: false,
		},
		{
			name:        "404 Not Found does not trigger retry",
			statusCode:  http.StatusNotFound,
			apiError:    "Security group not found",
			expectRetry: false,
		},
		{
			name:        "401 Unauthorized does not trigger retry",
			statusCode:  http.StatusUnauthorized,
			apiError:    "Authentication failed",
			expectRetry: false,
		},
		{
			name:        "403 Forbidden does not trigger retry",
			statusCode:  http.StatusForbidden,
			apiError:    "Access denied",
			expectRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testErr := errors.New("test error")
			result := retry.IsStateConflictError(testErr, tt.statusCode, tt.apiError)
			assert.Equal(t, tt.expectRetry, result)
		})
	}
}

// TestSecurityGroupUpdate_RetryLogic tests the retry logic for update operations
// Requirements: 5.1, 5.2, 5.5
func TestSecurityGroupUpdate_RetryLogic(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		attemptResults []error
		expectSuccess  bool
		expectAttempts int
	}{
		{
			name: "Success on first attempt",
			attemptResults: []error{
				nil,
			},
			expectSuccess:  true,
			expectAttempts: 1,
		},
		{
			name: "Success after one retry",
			attemptResults: []error{
				errors.New("state conflict"),
				nil,
			},
			expectSuccess:  true,
			expectAttempts: 2,
		},
		{
			name: "Success after multiple retries",
			attemptResults: []error{
				errors.New("state conflict"),
				errors.New("state conflict"),
				errors.New("state conflict"),
				nil,
			},
			expectSuccess:  true,
			expectAttempts: 4,
		},
		{
			name: "Failure after max attempts",
			attemptResults: []error{
				errors.New("state conflict"),
				errors.New("state conflict"),
				errors.New("state conflict"),
				errors.New("state conflict"),
				errors.New("state conflict"),
			},
			expectSuccess:  false,
			expectAttempts: 5,
		},
		{
			name: "Success on last attempt",
			attemptResults: []error{
				errors.New("state conflict"),
				errors.New("state conflict"),
				errors.New("state conflict"),
				errors.New("state conflict"),
				nil,
			},
			expectSuccess:  true,
			expectAttempts: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attemptCount := 0

			config := retry.RetryConfig{
				MaxAttempts:  5,
				InitialDelay: 10 * time.Millisecond,
				MaxDelay:     100 * time.Millisecond,
				Multiplier:   2.0,
				ShouldRetry: func(err error) bool {
					return err != nil && err.Error() == "state conflict"
				},
			}

			err := retry.Retry(ctx, config, func() error {
				if attemptCount < len(tt.attemptResults) {
					result := tt.attemptResults[attemptCount]
					attemptCount++
					return result
				}
				return nil
			})

			if tt.expectSuccess {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			assert.Equal(t, tt.expectAttempts, attemptCount)
		})
	}
}

// TestSecurityGroupUpdate_HandlesRecompositionCorrectly tests that update handles recomposition correctly
// Requirements: 3.1, 3.2, 3.3
func TestSecurityGroupUpdate_HandlesRecompositionCorrectly(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		scenario      string
		statusChecker state.ResourceStateChecker
		expectError   bool
	}{
		{
			name:     "Update triggers recomposition - wait for completion",
			scenario: "Security group recomposes after update",
			statusChecker: func() state.ResourceStateChecker {
				// Simulates: RECOMPOSED (before update) -> RECOMPOSING (after update) -> RECOMPOSED (complete)
				states := []string{"RECOMPOSED", "RECOMPOSING", "RECOMPOSING", "RECOMPOSED"}
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
			name:     "Update with quick recomposition",
			scenario: "Security group recomposes quickly",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"RECOMPOSED", "RECOMPOSING", "RECOMPOSED"}
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
			name:     "Update with slow recomposition",
			scenario: "Security group takes longer to recompose",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{
					"RECOMPOSED",
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
		{
			name:     "Multiple sequential recompositions during update",
			scenario: "Security group recomposes multiple times",
			statusChecker: func() state.ResourceStateChecker {
				// Simulates multiple recomposition cycles
				states := []string{
					"RECOMPOSED",
					"RECOMPOSING", "RECOMPOSED", // First recomposition
					"RECOMPOSING", "RECOMPOSED", // Second recomposition
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
			name:     "Update when already recomposing",
			scenario: "Security group is already recomposing when update starts",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "security_group",
				ResourceID:         "sg-recompose",
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

// TestSecurityGroupUpdate_TimeoutHandling tests timeout scenarios during update
// Requirements: 3.1, 3.2, 3.3
func TestSecurityGroupUpdate_TimeoutHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		timeout       time.Duration
		statusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name:    "Timeout when security group stuck in RECOMPOSING state",
			timeout: 1 * time.Second,
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSING", nil
			},
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name:    "Timeout with longer duration",
			timeout: 2 * time.Second,
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSING", nil
			},
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name:    "Success when security group reaches RECOMPOSED before timeout",
			timeout: 3 * time.Second,
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					// Transition after a few calls
					if callCount > 5 {
						return "RECOMPOSED", nil
					}
					return "RECOMPOSING", nil
				}
			}(),
			expectError: false,
		},
		{
			name:    "Timeout respected with tolerance",
			timeout: 1500 * time.Millisecond,
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSING", nil
			},
			expectError:   true,
			errorContains: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "security_group",
				ResourceID:         "sg-timeout-test",
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
				// Verify timeout was respected (with some tolerance for processing)
				assert.LessOrEqual(t, duration, tt.timeout+500*time.Millisecond)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSecurityGroupUpdate_ErrorHandling tests various error scenarios during update
// Requirements: 3.1, 3.2, 3.3
func TestSecurityGroupUpdate_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name: "API error when checking security group status",
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
			name: "Nil status returned",
			statusChecker: func(ctx context.Context) (string, error) {
				return "", errors.New("security group status is nil")
			},
			expectError:   true,
			errorContains: "failed to check current state",
		},
		{
			name: "Unexpected error during status check",
			statusChecker: func(ctx context.Context) (string, error) {
				return "", errors.New("unexpected error occurred")
			},
			expectError:   true,
			errorContains: "failed to check current state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "security_group",
				ResourceID:         "sg-error-test",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.SecurityGroupStableStates,
				TransitionalStates: state.SecurityGroupTransitionalStates,
				FailureStates:      state.SecurityGroupFailureStates,
				Timeout:            2 * time.Second,
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

// TestSecurityGroupUpdate_StateConflictRetryConfig tests the retry configuration for update operations
// Requirements: 5.1, 5.2, 5.5
func TestSecurityGroupUpdate_StateConflictRetryConfig(t *testing.T) {
	config := retry.StateConflictRetryConfig()

	// Verify configuration is appropriate for security group update operations
	assert.Equal(t, 5, config.MaxAttempts, "Should allow 5 retry attempts")
	assert.Equal(t, 2*time.Second, config.InitialDelay, "Initial delay should be 2 seconds")
	assert.Equal(t, 30*time.Second, config.MaxDelay, "Max delay should be 30 seconds")
	assert.Equal(t, 2.0, config.Multiplier, "Multiplier should be 2.0 for exponential backoff")
	assert.NotNil(t, config.ShouldRetry, "ShouldRetry function should be configured")
}

// TestSecurityGroupUpdate_RealWorldScenarios tests realistic security group update scenarios
// Requirements: 3.1, 3.2, 3.3, 5.1, 5.2
func TestSecurityGroupUpdate_RealWorldScenarios(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		scenario      string
		statusChecker state.ResourceStateChecker
		expectError   bool
	}{
		{
			name:     "Rule update - security group ready immediately",
			scenario: "Security group is already in RECOMPOSED state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "RECOMPOSED", nil
			},
			expectError: false,
		},
		{
			name:     "Rule update - security group recomposing from previous operation",
			scenario: "Security group transitions from RECOMPOSING to RECOMPOSED",
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
			scenario: "Multiple rule updates causing sequential recompositions",
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
			scenario: "Large security group takes longer to recompose",
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
		{
			name:     "Update during active recomposition",
			scenario: "Update attempted while security group is actively recomposing",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{
					"RECOMPOSING", "RECOMPOSING", "RECOMPOSING", "RECOMPOSED",
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

// TestSecurityGroupUpdate_ImmediateSuccessWhenRecomposed tests immediate success when already in RECOMPOSED state
// Requirements: 8.1, 8.4
func TestSecurityGroupUpdate_ImmediateSuccessWhenRecomposed(t *testing.T) {
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

// TestSecurityGroupUpdate_ContextCancellation tests that context cancellation is properly handled
// Requirements: 9.3, 9.4
func TestSecurityGroupUpdate_ContextCancellation(t *testing.T) {
	tests := []struct {
		name              string
		cancelTiming      string
		statusChecker     func(context.Context, context.CancelFunc) state.ResourceStateChecker
		expectError       bool
		expectCancelError bool
	}{
		{
			name:         "Context cancelled before update",
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
