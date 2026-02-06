package state

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewStateTransitionManager(t *testing.T) {
	config := StateTransitionConfig{
		ResourceType:       "vm",
		ResourceID:         "test-123",
		StatusChecker:      func(ctx context.Context) (string, error) { return "POWERED_ON", nil },
		TargetStates:       []string{"POWERED_ON"},
		TransitionalStates: []string{"BUSY"},
		FailureStates:      []string{"error"},
		Timeout:            5 * time.Minute,
		PollInterval:       5 * time.Second,
	}

	manager := NewStateTransitionManager(config)

	assert.NotNil(t, manager)
	assert.Equal(t, "vm", manager.config.ResourceType)
	assert.Equal(t, "test-123", manager.config.ResourceID)
}

func TestIsTransitionalState(t *testing.T) {
	manager := NewStateTransitionManager(StateTransitionConfig{
		TransitionalStates: []string{"BUSY", "RECOMPOSING", "DRAFT"},
	})

	tests := []struct {
		name     string
		state    string
		expected bool
	}{
		{"BUSY is transitional", "BUSY", true},
		{"RECOMPOSING is transitional", "RECOMPOSING", true},
		{"DRAFT is transitional", "DRAFT", true},
		{"POWERED_ON is not transitional", "POWERED_ON", false},
		{"AVAILABLE is not transitional", "AVAILABLE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.IsTransitionalState(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsStableState(t *testing.T) {
	manager := NewStateTransitionManager(StateTransitionConfig{
		TargetStates: []string{"POWERED_ON", "POWERED_OFF", "AVAILABLE"},
	})

	tests := []struct {
		name     string
		state    string
		expected bool
	}{
		{"POWERED_ON is stable", "POWERED_ON", true},
		{"POWERED_OFF is stable", "POWERED_OFF", true},
		{"AVAILABLE is stable", "AVAILABLE", true},
		{"BUSY is not stable", "BUSY", false},
		{"RECOMPOSING is not stable", "RECOMPOSING", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.IsStableState(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsFailureState(t *testing.T) {
	manager := NewStateTransitionManager(StateTransitionConfig{
		FailureStates: []string{"error", "failed"},
	})

	tests := []struct {
		name     string
		state    string
		expected bool
	}{
		{"error is failure", "error", true},
		{"failed is failure", "failed", true},
		{"POWERED_ON is not failure", "POWERED_ON", false},
		{"BUSY is not failure", "BUSY", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.IsFailureState(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckCurrentState(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		statusChecker ResourceStateChecker
		expectedState string
		expectError   bool
	}{
		{
			name: "successfully checks state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			expectedState: "POWERED_ON",
			expectError:   false,
		},
		{
			name: "returns error on check failure",
			statusChecker: func(ctx context.Context) (string, error) {
				return "", errors.New("API error")
			},
			expectedState: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType:  "vm",
				ResourceID:    "test-123",
				StatusChecker: tt.statusChecker,
			})

			state, err := manager.CheckCurrentState(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedState, state)
			}
		})
	}
}

func TestCheckCurrentState_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "test-123",
		StatusChecker: func(ctx context.Context) (string, error) {
			return "POWERED_ON", nil
		},
	})

	state, err := manager.CheckCurrentState(ctx)

	assert.Error(t, err)
	assert.Equal(t, "", state)
	assert.Equal(t, context.Canceled, err)
}

func TestIsInTargetState(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		statusChecker  ResourceStateChecker
		targetStates   []string
		expectedResult bool
		expectedState  string
		expectError    bool
	}{
		{
			name: "resource is in target state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			targetStates:   []string{"POWERED_ON", "POWERED_OFF"},
			expectedResult: true,
			expectedState:  "POWERED_ON",
			expectError:    false,
		},
		{
			name: "resource is not in target state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "BUSY", nil
			},
			targetStates:   []string{"POWERED_ON", "POWERED_OFF"},
			expectedResult: false,
			expectedState:  "BUSY",
			expectError:    false,
		},
		{
			name: "error checking state",
			statusChecker: func(ctx context.Context) (string, error) {
				return "", errors.New("API error")
			},
			targetStates:   []string{"POWERED_ON"},
			expectedResult: false,
			expectedState:  "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType:  "vm",
				ResourceID:    "test-123",
				StatusChecker: tt.statusChecker,
				TargetStates:  tt.targetStates,
			})

			isTarget, state, err := manager.IsInTargetState(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, isTarget)
				assert.Equal(t, tt.expectedState, state)
			}
		})
	}
}

func TestWaitForStableState_AlreadyStable(t *testing.T) {
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

	// Should return immediately without polling
	err := manager.WaitForStableState(ctx)

	assert.NoError(t, err)
}

func TestWaitForStableState_FailureState(t *testing.T) {
	ctx := context.Background()

	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "test-123",
		StatusChecker: func(ctx context.Context) (string, error) {
			return "error", nil
		},
		TargetStates:  []string{"POWERED_ON"},
		FailureStates: []string{"error", "failed"},
		Timeout:       5 * time.Second,
		PollInterval:  1 * time.Second,
	})

	err := manager.WaitForStableState(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failure state")
}

func TestWaitForStableState_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "test-123",
		StatusChecker: func(ctx context.Context) (string, error) {
			return "BUSY", nil
		},
		TargetStates:  []string{"POWERED_ON"},
		FailureStates: []string{"error"},
		Timeout:       5 * time.Second,
		PollInterval:  1 * time.Second,
	})

	err := manager.WaitForStableState(ctx)

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestWaitForStableState_TransitionToStable(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "test-123",
		StatusChecker: func(ctx context.Context) (string, error) {
			callCount++
			if callCount <= 2 {
				return "BUSY", nil
			}
			return "POWERED_ON", nil
		},
		TargetStates:       []string{"POWERED_ON"},
		TransitionalStates: []string{"BUSY"},
		FailureStates:      []string{"error"},
		Timeout:            5 * time.Second,
		PollInterval:       100 * time.Millisecond,
	})

	err := manager.WaitForStableState(ctx)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, callCount, 3) // Initial check + at least 2 polls
}

func TestParallelOperations_NoInterference(t *testing.T) {
	ctx := context.Background()

	// Create two independent managers for different resources
	manager1 := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "vm-1",
		StatusChecker: func(ctx context.Context) (string, error) {
			return "POWERED_ON", nil
		},
		TargetStates:  []string{"POWERED_ON"},
		FailureStates: []string{"error"},
		Timeout:       5 * time.Second,
		PollInterval:  1 * time.Second,
	})

	manager2 := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "volume",
		ResourceID:   "vol-1",
		StatusChecker: func(ctx context.Context) (string, error) {
			return "AVAILABLE", nil
		},
		TargetStates:  []string{"AVAILABLE"},
		FailureStates: []string{"error"},
		Timeout:       5 * time.Second,
		PollInterval:  1 * time.Second,
	})

	// Run both operations in parallel
	done1 := make(chan error, 1)
	done2 := make(chan error, 1)

	go func() {
		done1 <- manager1.WaitForStableState(ctx)
	}()

	go func() {
		done2 <- manager2.WaitForStableState(ctx)
	}()

	// Both should complete successfully without interference
	err1 := <-done1
	err2 := <-done2

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

// Additional comprehensive tests for state manager configuration

func TestStateTransitionManager_Configuration(t *testing.T) {
	tests := []struct {
		name   string
		config StateTransitionConfig
	}{
		{
			name: "VM configuration",
			config: StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-123",
				StatusChecker:      func(ctx context.Context) (string, error) { return "POWERED_ON", nil },
				TargetStates:       VMStableStates,
				TransitionalStates: VMTransitionalStates,
				FailureStates:      VMFailureStates,
				Timeout:            10 * time.Minute,
				PollInterval:       5 * time.Second,
			},
		},
		{
			name: "Volume configuration",
			config: StateTransitionConfig{
				ResourceType:       "volume",
				ResourceID:         "vol-456",
				StatusChecker:      func(ctx context.Context) (string, error) { return "AVAILABLE", nil },
				TargetStates:       VolumeStableStates,
				TransitionalStates: VolumeTransitionalStates,
				FailureStates:      VolumeFailureStates,
				Timeout:            10 * time.Minute,
				PollInterval:       5 * time.Second,
			},
		},
		{
			name: "Security Group configuration",
			config: StateTransitionConfig{
				ResourceType:       "security_group",
				ResourceID:         "sg-789",
				StatusChecker:      func(ctx context.Context) (string, error) { return "RECOMPOSED", nil },
				TargetStates:       SecurityGroupStableStates,
				TransitionalStates: SecurityGroupTransitionalStates,
				FailureStates:      SecurityGroupFailureStates,
				Timeout:            10 * time.Minute,
				PollInterval:       5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStateTransitionManager(tt.config)

			assert.NotNil(t, manager)
			assert.Equal(t, tt.config.ResourceType, manager.config.ResourceType)
			assert.Equal(t, tt.config.ResourceID, manager.config.ResourceID)
			assert.Equal(t, tt.config.Timeout, manager.config.Timeout)
			assert.Equal(t, tt.config.PollInterval, manager.config.PollInterval)
			assert.Equal(t, tt.config.TargetStates, manager.config.TargetStates)
			assert.Equal(t, tt.config.TransitionalStates, manager.config.TransitionalStates)
			assert.Equal(t, tt.config.FailureStates, manager.config.FailureStates)
		})
	}
}

func TestStateDetection_AllResourceTypes(t *testing.T) {
	tests := []struct {
		name               string
		resourceType       string
		stableStates       []string
		transitionalStates []string
		failureStates      []string
	}{
		{
			name:               "VM states",
			resourceType:       "vm",
			stableStates:       VMStableStates,
			transitionalStates: VMTransitionalStates,
			failureStates:      VMFailureStates,
		},
		{
			name:               "Volume states",
			resourceType:       "volume",
			stableStates:       VolumeStableStates,
			transitionalStates: VolumeTransitionalStates,
			failureStates:      VolumeFailureStates,
		},
		{
			name:               "Security Group states",
			resourceType:       "security_group",
			stableStates:       SecurityGroupStableStates,
			transitionalStates: SecurityGroupTransitionalStates,
			failureStates:      SecurityGroupFailureStates,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType:       tt.resourceType,
				TargetStates:       tt.stableStates,
				TransitionalStates: tt.transitionalStates,
				FailureStates:      tt.failureStates,
			})

			// Test stable state detection
			for _, state := range tt.stableStates {
				assert.True(t, manager.IsStableState(state), "Expected %s to be stable for %s", state, tt.resourceType)
				assert.False(t, manager.IsTransitionalState(state), "Expected %s not to be transitional for %s", state, tt.resourceType)
				assert.False(t, manager.IsFailureState(state), "Expected %s not to be failure for %s", state, tt.resourceType)
			}

			// Test transitional state detection
			for _, state := range tt.transitionalStates {
				assert.True(t, manager.IsTransitionalState(state), "Expected %s to be transitional for %s", state, tt.resourceType)
				assert.False(t, manager.IsStableState(state), "Expected %s not to be stable for %s", state, tt.resourceType)
				assert.False(t, manager.IsFailureState(state), "Expected %s not to be failure for %s", state, tt.resourceType)
			}

			// Test failure state detection
			for _, state := range tt.failureStates {
				assert.True(t, manager.IsFailureState(state), "Expected %s to be failure for %s", state, tt.resourceType)
				assert.False(t, manager.IsStableState(state), "Expected %s not to be stable for %s", state, tt.resourceType)
				assert.False(t, manager.IsTransitionalState(state), "Expected %s not to be transitional for %s", state, tt.resourceType)
			}
		})
	}
}

func TestStateTransitionManager_EmptyStates(t *testing.T) {
	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType:       "test",
		TargetStates:       []string{},
		TransitionalStates: []string{},
		FailureStates:      []string{},
	})

	// All checks should return false for empty state lists
	assert.False(t, manager.IsStableState("any_state"))
	assert.False(t, manager.IsTransitionalState("any_state"))
	assert.False(t, manager.IsFailureState("any_state"))
}

func TestWaitForStableState_StatusCheckerError(t *testing.T) {
	ctx := context.Background()

	manager := NewStateTransitionManager(StateTransitionConfig{
		ResourceType: "vm",
		ResourceID:   "test-123",
		StatusChecker: func(ctx context.Context) (string, error) {
			return "", errors.New("API connection failed")
		},
		TargetStates:  []string{"POWERED_ON"},
		FailureStates: []string{"error"},
		Timeout:       5 * time.Second,
		PollInterval:  1 * time.Second,
	})

	err := manager.WaitForStableState(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check current state")
}

func TestWaitForStableState_MultipleTargetStates(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		currentState  string
		targetStates  []string
		shouldSucceed bool
	}{
		{
			name:          "matches first target state",
			currentState:  "POWERED_ON",
			targetStates:  []string{"POWERED_ON", "POWERED_OFF"},
			shouldSucceed: true,
		},
		{
			name:          "matches second target state",
			currentState:  "POWERED_OFF",
			targetStates:  []string{"POWERED_ON", "POWERED_OFF"},
			shouldSucceed: true,
		},
		{
			name:          "matches third target state",
			currentState:  "AVAILABLE",
			targetStates:  []string{"POWERED_ON", "POWERED_OFF", "AVAILABLE"},
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewStateTransitionManager(StateTransitionConfig{
				ResourceType: "test",
				ResourceID:   "test-123",
				StatusChecker: func(ctx context.Context) (string, error) {
					return tt.currentState, nil
				},
				TargetStates:  tt.targetStates,
				FailureStates: []string{"error"},
				Timeout:       5 * time.Second,
				PollInterval:  1 * time.Second,
			})

			err := manager.WaitForStableState(ctx)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
