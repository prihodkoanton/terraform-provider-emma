package emma

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/emma-community/terraform-provider-emma/internal/emma/common/async"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/retry"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/state"
	"github.com/stretchr/testify/assert"
)

// TestVMHardwareEdit_WaitsForStableState tests that hardware edit waits for VM to reach stable state
// Requirements: 2.1, 2.2, 2.3
func TestVMHardwareEdit_WaitsForStableState(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name: "VM already in POWERED_ON - immediate success",
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
			name: "VM transitions from pending to starting to POWERED_ON",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"pending", "starting", "BUSY", "POWERED_ON"}
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
			name: "VM already in POWERED_OFF - also stable",
			statusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_OFF", nil
			},
			expectError: false,
		},
		{
			name: "VM transitions from stopping to POWERED_OFF",
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "stopping", nil
					}
					return "POWERED_OFF", nil
				}
			}(),
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

// TestVMHardwareEdit_RetriesOnStateConflict tests that hardware edit retries when state conflict occurs
// Requirements: 5.1, 5.2
func TestVMHardwareEdit_RetriesOnStateConflict(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		apiError      string
		expectRetry   bool
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
			apiError:    "Resource is in busy state",
			expectRetry: true,
		},
		{
			name:        "Inappropriate compute instance state triggers retry",
			statusCode:  http.StatusBadRequest,
			apiError:    "Inappropriate compute instance state for operation",
			expectRetry: true,
		},
		{
			name:        "Resource conflict message triggers retry",
			statusCode:  http.StatusBadRequest,
			apiError:    "Resource conflict detected",
			expectRetry: true,
		},
		{
			name:        "Recomposing state triggers retry",
			statusCode:  http.StatusBadRequest,
			apiError:    "Resource is recomposing",
			expectRetry: true,
		},
		{
			name:        "Non-state error does not trigger retry",
			statusCode:  http.StatusBadRequest,
			apiError:    "Invalid parameter value",
			expectRetry: false,
		},
		{
			name:        "404 Not Found does not trigger retry",
			statusCode:  http.StatusNotFound,
			apiError:    "Resource not found",
			expectRetry: false,
		},
		{
			name:        "401 Unauthorized does not trigger retry",
			statusCode:  http.StatusUnauthorized,
			apiError:    "Authentication failed",
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

// TestVMHardwareEdit_RetryLogic tests the retry logic for hardware edit operations
// Requirements: 5.1, 5.2, 5.5
func TestVMHardwareEdit_RetryLogic(t *testing.T) {
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

// TestVMHardwareEdit_TimeoutHandling tests timeout scenarios during hardware edit
// Requirements: 2.1, 2.2, 2.3
func TestVMHardwareEdit_TimeoutHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		timeout       time.Duration
		statusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name:    "Timeout when VM stuck in BUSY state",
			timeout: 1 * time.Second,
			statusChecker: func(ctx context.Context) (string, error) {
				return "BUSY", nil
			},
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name:    "Timeout when VM stuck in pending state",
			timeout: 1 * time.Second,
			statusChecker: func(ctx context.Context) (string, error) {
				return "pending", nil
			},
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name:    "Timeout when VM stuck in starting state",
			timeout: 1 * time.Second,
			statusChecker: func(ctx context.Context) (string, error) {
				return "starting", nil
			},
			expectError:   true,
			errorContains: "timeout",
		},
		{
			name:    "Success when VM reaches stable state before timeout",
			timeout: 3 * time.Second,
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					// Transition after a few calls
					if callCount > 5 {
						return "POWERED_ON", nil
					}
					return "BUSY", nil
				}
			}(),
			expectError: false,
		},
		{
			name:    "Timeout respected with tolerance",
			timeout: 2 * time.Second,
			statusChecker: func(ctx context.Context) (string, error) {
				return "BUSY", nil
			},
			expectError:   true,
			errorContains: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-timeout-test",
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
				// Verify timeout was respected (with some tolerance for processing)
				assert.LessOrEqual(t, duration, tt.timeout+500*time.Millisecond)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVMHardwareEdit_ErrorHandling tests various error scenarios during hardware edit
// Requirements: 2.1, 2.2, 2.3
func TestVMHardwareEdit_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name: "API error when checking VM status",
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
			name: "Nil status returned",
			statusChecker: func(ctx context.Context) (string, error) {
				return "", fmt.Errorf("VM status is nil")
			},
			expectError:   true,
			errorContains: "failed to check current state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-error-test",
				StatusChecker:      tt.statusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
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

// TestVMHardwareEdit_StateConflictRetryConfig tests the retry configuration for hardware edit
// Requirements: 5.1, 5.2, 5.5
func TestVMHardwareEdit_StateConflictRetryConfig(t *testing.T) {
	config := retry.StateConflictRetryConfig()

	// Verify configuration is appropriate for hardware edit operations
	assert.Equal(t, 5, config.MaxAttempts, "Should allow 5 retry attempts")
	assert.Equal(t, 2*time.Second, config.InitialDelay, "Initial delay should be 2 seconds")
	assert.Equal(t, 30*time.Second, config.MaxDelay, "Max delay should be 30 seconds")
	assert.Equal(t, 2.0, config.Multiplier, "Multiplier should be 2.0 for exponential backoff")
	assert.NotNil(t, config.ShouldRetry, "ShouldRetry function should be configured")
}

// TestVMHardwareEdit_RealWorldScenarios tests realistic hardware edit scenarios
// Requirements: 2.1, 2.2, 2.3, 5.1, 5.2
func TestVMHardwareEdit_RealWorldScenarios(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		scenario      string
		statusChecker state.ResourceStateChecker
		expectError   bool
	}{
		{
			name:     "Hardware edit - VM ready immediately",
			scenario: "VM is already in stable state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			expectError: false,
		},
		{
			name:     "Hardware edit - VM busy from previous operation",
			scenario: "VM transitions from BUSY to POWERED_ON",
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
			name:     "Hardware edit - VM starting up",
			scenario: "VM transitions from starting to POWERED_ON",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"starting", "BUSY", "POWERED_ON"}
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
			name:     "Hardware edit - VM powered off",
			scenario: "VM is in POWERED_OFF state (also stable)",
			statusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_OFF", nil
			},
			expectError: false,
		},
		{
			name:     "Hardware edit - Multiple state transitions",
			scenario: "VM goes through multiple transitional states",
			statusChecker: func() state.ResourceStateChecker {
				states := []string{"pending", "starting", "BUSY", "BUSY", "POWERED_ON"}
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

// TestVMHardwareEdit_ConfigurableTimeout tests that custom timeout values are respected
// Requirements: 2.1, 2.2, 2.3
func TestVMHardwareEdit_ConfigurableTimeout(t *testing.T) {
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
					// Transition after 2 seconds worth of calls (20 calls at 100ms interval)
					if callCount > 20 {
						return "POWERED_ON", nil
					}
					return "BUSY", nil
				}
			}(),
			expectError: false,
		},
		{
			name:    "Default timeout should be sufficient for normal operations",
			timeout: async.DefaultTimeout,
			statusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount > 5 {
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
				ResourceID:         "vm-timeout-config",
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
				// Verify timeout was respected (with some tolerance)
				assert.LessOrEqual(t, duration, tt.timeout+500*time.Millisecond)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
