package errors

import (
	"strings"
	"testing"
	"time"
)

// TestStateTransitionError_Error tests the StateTransitionError formatting
func TestStateTransitionError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *StateTransitionError
		expected []string // Expected substrings in error message
	}{
		{
			name: "VM timeout with all fields",
			err: &StateTransitionError{
				ResourceType:  "VM",
				ResourceID:    "vm-12345",
				CurrentState:  "BUSY",
				ExpectedState: "POWERED_ON",
				Operation:     "hardware edit",
				Timeout:       10 * time.Minute,
			},
			expected: []string{
				"Timeout waiting for",
				"VM",
				"vm-12345",
				"POWERED_ON",
				"hardware edit",
				"Current state: BUSY",
				"Timeout: 10m0s",
				"Check the Emma console",
			},
		},
		{
			name: "Volume timeout",
			err: &StateTransitionError{
				ResourceType:  "Volume",
				ResourceID:    "vol-67890",
				CurrentState:  "DRAFT",
				ExpectedState: "AVAILABLE",
				Operation:     "attach",
				Timeout:       5 * time.Minute,
			},
			expected: []string{
				"Timeout waiting for",
				"Volume",
				"vol-67890",
				"AVAILABLE",
				"attach",
				"Current state: DRAFT",
				"Timeout: 5m0s",
			},
		},
		{
			name: "Security Group timeout",
			err: &StateTransitionError{
				ResourceType:  "SecurityGroup",
				ResourceID:    "sg-abc123",
				CurrentState:  "RECOMPOSING",
				ExpectedState: "RECOMPOSED",
				Operation:     "update",
				Timeout:       3 * time.Minute,
			},
			expected: []string{
				"Timeout waiting for",
				"SecurityGroup",
				"sg-abc123",
				"RECOMPOSED",
				"update",
				"Current state: RECOMPOSING",
				"Timeout: 3m0s",
			},
		},
		{
			name: "Short timeout",
			err: &StateTransitionError{
				ResourceType:  "VM",
				ResourceID:    "vm-test",
				CurrentState:  "BUSY",
				ExpectedState: "POWERED_OFF",
				Operation:     "shutdown",
				Timeout:       30 * time.Second,
			},
			expected: []string{
				"Timeout: 30s",
				"Current state: BUSY",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()

			// Verify all expected substrings are present
			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Error message missing expected substring %q.\nGot: %q", expected, result)
				}
			}
		})
	}
}

// TestStateTransitionError_IncludesCurrentAndExpectedStates tests that error messages
// include both current and expected states as required by Requirements 6.1, 6.2
func TestStateTransitionError_IncludesCurrentAndExpectedStates(t *testing.T) {
	tests := []struct {
		name          string
		currentState  string
		expectedState string
	}{
		{
			name:          "BUSY to POWERED_ON",
			currentState:  "BUSY",
			expectedState: "POWERED_ON",
		},
		{
			name:          "DRAFT to AVAILABLE",
			currentState:  "DRAFT",
			expectedState: "AVAILABLE",
		},
		{
			name:          "RECOMPOSING to RECOMPOSED",
			currentState:  "RECOMPOSING",
			expectedState: "RECOMPOSED",
		},
		{
			name:          "pending to running",
			currentState:  "pending",
			expectedState: "running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &StateTransitionError{
				ResourceType:  "TestResource",
				ResourceID:    "test-123",
				CurrentState:  tt.currentState,
				ExpectedState: tt.expectedState,
				Operation:     "test operation",
				Timeout:       1 * time.Minute,
			}

			errorMsg := err.Error()

			// Verify current state is in error message
			if !strings.Contains(errorMsg, tt.currentState) {
				t.Errorf("Error message does not contain current state %q.\nGot: %q", tt.currentState, errorMsg)
			}

			// Verify expected state is in error message
			if !strings.Contains(errorMsg, tt.expectedState) {
				t.Errorf("Error message does not contain expected state %q.\nGot: %q", tt.expectedState, errorMsg)
			}

			// Verify "Current state:" label is present
			if !strings.Contains(errorMsg, "Current state:") {
				t.Errorf("Error message does not contain 'Current state:' label.\nGot: %q", errorMsg)
			}
		})
	}
}

// TestStateConflictError_Error tests the StateConflictError formatting
func TestStateConflictError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *StateConflictError
		expected []string // Expected substrings in error message
	}{
		{
			name: "VM state conflict",
			err: &StateConflictError{
				ResourceType: "VM",
				ResourceID:   "vm-12345",
				CurrentState: "BUSY",
				Operation:    "hardware edit",
			},
			expected: []string{
				"Cannot perform",
				"hardware edit",
				"VM",
				"vm-12345",
				"BUSY state",
				"automatically retry",
			},
		},
		{
			name: "Volume state conflict",
			err: &StateConflictError{
				ResourceType: "Volume",
				ResourceID:   "vol-67890",
				CurrentState: "DRAFT",
				Operation:    "attach",
			},
			expected: []string{
				"Cannot perform",
				"attach",
				"Volume",
				"vol-67890",
				"DRAFT state",
				"automatically retry",
			},
		},
		{
			name: "Security Group state conflict",
			err: &StateConflictError{
				ResourceType: "SecurityGroup",
				ResourceID:   "sg-abc123",
				CurrentState: "RECOMPOSING",
				Operation:    "update",
			},
			expected: []string{
				"Cannot perform",
				"update",
				"SecurityGroup",
				"sg-abc123",
				"RECOMPOSING state",
				"automatically retry",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()

			// Verify all expected substrings are present
			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Error message missing expected substring %q.\nGot: %q", expected, result)
				}
			}
		})
	}
}

// TestStateConflictError_IncludesCurrentState tests that error messages
// include the current state as required by Requirements 6.1, 6.3
func TestStateConflictError_IncludesCurrentState(t *testing.T) {
	tests := []struct {
		name         string
		currentState string
	}{
		{
			name:         "BUSY state",
			currentState: "BUSY",
		},
		{
			name:         "DRAFT state",
			currentState: "DRAFT",
		},
		{
			name:         "RECOMPOSING state",
			currentState: "RECOMPOSING",
		},
		{
			name:         "pending state",
			currentState: "pending",
		},
		{
			name:         "starting state",
			currentState: "starting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &StateConflictError{
				ResourceType: "TestResource",
				ResourceID:   "test-123",
				CurrentState: tt.currentState,
				Operation:    "test operation",
			}

			errorMsg := err.Error()

			// Verify current state is in error message
			if !strings.Contains(errorMsg, tt.currentState) {
				t.Errorf("Error message does not contain current state %q.\nGot: %q", tt.currentState, errorMsg)
			}

			// Verify "state" keyword is present
			if !strings.Contains(errorMsg, "state") {
				t.Errorf("Error message does not contain 'state' keyword.\nGot: %q", errorMsg)
			}
		})
	}
}

// TestStateConflictError_SuggestsRetry tests that error messages suggest
// automatic retry as required by Requirement 6.3
func TestStateConflictError_SuggestsRetry(t *testing.T) {
	err := &StateConflictError{
		ResourceType: "VM",
		ResourceID:   "vm-test",
		CurrentState: "BUSY",
		Operation:    "update",
	}

	errorMsg := err.Error()

	// Verify retry suggestion is present
	retryKeywords := []string{"retry", "automatically"}
	foundRetryKeyword := false
	for _, keyword := range retryKeywords {
		if strings.Contains(strings.ToLower(errorMsg), keyword) {
			foundRetryKeyword = true
			break
		}
	}

	if !foundRetryKeyword {
		t.Errorf("Error message does not suggest automatic retry.\nGot: %q", errorMsg)
	}
}

// TestStateTransitionError_TimeoutFormatting tests various timeout durations
func TestStateTransitionError_TimeoutFormatting(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout string
	}{
		{
			name:            "seconds",
			timeout:         30 * time.Second,
			expectedTimeout: "30s",
		},
		{
			name:            "minutes",
			timeout:         5 * time.Minute,
			expectedTimeout: "5m0s",
		},
		{
			name:            "minutes and seconds",
			timeout:         2*time.Minute + 30*time.Second,
			expectedTimeout: "2m30s",
		},
		{
			name:            "hours",
			timeout:         1 * time.Hour,
			expectedTimeout: "1h0m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &StateTransitionError{
				ResourceType:  "VM",
				ResourceID:    "vm-test",
				CurrentState:  "BUSY",
				ExpectedState: "POWERED_ON",
				Operation:     "test",
				Timeout:       tt.timeout,
			}

			errorMsg := err.Error()

			// Verify timeout is formatted correctly
			if !strings.Contains(errorMsg, tt.expectedTimeout) {
				t.Errorf("Error message does not contain expected timeout %q.\nGot: %q", tt.expectedTimeout, errorMsg)
			}
		})
	}
}

// TestStateTransitionError_ImplementsError tests that StateTransitionError implements error interface
func TestStateTransitionError_ImplementsError(t *testing.T) {
	var _ error = &StateTransitionError{}
	var _ error = (*StateTransitionError)(nil)
}

// TestStateConflictError_ImplementsError tests that StateConflictError implements error interface
func TestStateConflictError_ImplementsError(t *testing.T) {
	var _ error = &StateConflictError{}
	var _ error = (*StateConflictError)(nil)
}

// TestErrorMessages_Actionable tests that error messages provide actionable guidance
// as required by Requirement 6.3
func TestErrorMessages_Actionable(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		actionableTerms []string
	}{
		{
			name: "StateTransitionError provides guidance",
			err: &StateTransitionError{
				ResourceType:  "VM",
				ResourceID:    "vm-test",
				CurrentState:  "BUSY",
				ExpectedState: "POWERED_ON",
				Operation:     "update",
				Timeout:       5 * time.Minute,
			},
			actionableTerms: []string{"Check", "console", "increase", "timeout"},
		},
		{
			name: "StateConflictError provides guidance",
			err: &StateConflictError{
				ResourceType: "VM",
				ResourceID:   "vm-test",
				CurrentState: "BUSY",
				Operation:    "update",
			},
			actionableTerms: []string{"automatically", "retry"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorMsg := tt.err.Error()

			// Check for at least one actionable term
			foundActionable := false
			for _, term := range tt.actionableTerms {
				if strings.Contains(strings.ToLower(errorMsg), strings.ToLower(term)) {
					foundActionable = true
					break
				}
			}

			if !foundActionable {
				t.Errorf("Error message does not contain actionable guidance. Expected one of %v.\nGot: %q",
					tt.actionableTerms, errorMsg)
			}
		})
	}
}
