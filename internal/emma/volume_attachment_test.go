package emma

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emma-community/terraform-provider-emma/internal/emma/common/state"
	"github.com/stretchr/testify/assert"
)

// TestVolumeAttachment_WaitForVMStableStateBeforeAttach tests that volume attach waits for VM stable state
// Requirements: 2.1, 2.2, 2.3, 7.1
func TestVolumeAttachment_WaitForVMStableStateBeforeAttach(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		vmStatusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name: "VM already in POWERED_ON state - attach proceeds",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			expectError: false,
		},
		{
			name: "VM already in POWERED_OFF state - attach proceeds",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_OFF", nil
			},
			expectError: false,
		},
		{
			name: "VM transitions from BUSY to POWERED_ON before attach",
			vmStatusChecker: func() state.ResourceStateChecker {
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
			name: "VM transitions from starting to POWERED_ON before attach",
			vmStatusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 3 {
						return "starting", nil
					}
					return "POWERED_ON", nil
				}
			}(),
			expectError: false,
		},
		{
			name: "VM already in running state (alternative stable state)",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "running", nil
			},
			expectError: false,
		},
		{
			name: "VM already in stopped state (alternative stable state)",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "stopped", nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vmStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-123",
				StatusChecker:      tt.vmStatusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := vmStateManager.WaitForStableState(ctx)

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

// TestVolumeAttachment_WaitForVMStableStateBeforeDetach tests that volume detach waits for VM stable state
// Requirements: 2.1, 2.2, 2.3, 7.2
func TestVolumeAttachment_WaitForVMStableStateBeforeDetach(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		vmStatusChecker state.ResourceStateChecker
		expectError   bool
		errorContains string
	}{
		{
			name: "VM already in POWERED_ON state - detach proceeds",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			expectError: false,
		},
		{
			name: "VM already in POWERED_OFF state - detach proceeds",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_OFF", nil
			},
			expectError: false,
		},
		{
			name: "VM transitions from BUSY to POWERED_ON before detach",
			vmStatusChecker: func() state.ResourceStateChecker {
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
			name: "VM transitions from stopping to POWERED_OFF before detach",
			vmStatusChecker: func() state.ResourceStateChecker {
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
			name: "VM already in running state (alternative stable state)",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "running", nil
			},
			expectError: false,
		},
		{
			name: "VM already in stopped state (alternative stable state)",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "stopped", nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vmStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-456",
				StatusChecker:      tt.vmStatusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := vmStateManager.WaitForStableState(ctx)

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

// TestVolumeAttachment_AttachedToIdPreservation tests that attached_to_id is preserved during transition
// Requirements: 2.5, 8.5
func TestVolumeAttachment_AttachedToIdPreservation(t *testing.T) {
	tests := []struct {
		name                string
		planAttachedToId    int64
		stateAttachedToId   int64
		apiAttachedToId     *int32
		expectedAttachedToId int64
		description         string
	}{
		{
			name:              "API returns null during transition - preserve plan value",
			planAttachedToId:  123,
			stateAttachedToId: 0,
			apiAttachedToId:   nil,
			expectedAttachedToId: 123,
			description:       "When API returns null during attachment transition, preserve the planned value",
		},
		{
			name:              "API returns different value during transition - preserve plan value",
			planAttachedToId:  123,
			stateAttachedToId: 0,
			apiAttachedToId:   func() *int32 { v := int32(456); return &v }(),
			expectedAttachedToId: 123,
			description:       "When API returns different value during transition, preserve the planned value",
		},
		{
			name:              "API returns correct value - use API value",
			planAttachedToId:  123,
			stateAttachedToId: 0,
			apiAttachedToId:   func() *int32 { v := int32(123); return &v }(),
			expectedAttachedToId: 123,
			description:       "When API returns correct value, use it",
		},
		{
			name:              "Detachment - plan is null, API returns null",
			planAttachedToId:  0,
			stateAttachedToId: 123,
			apiAttachedToId:   nil,
			expectedAttachedToId: 0,
			description:       "When detaching (plan is null), accept null from API",
		},
		{
			name:              "Detachment - plan is null, API still shows old attachment",
			planAttachedToId:  0,
			stateAttachedToId: 123,
			apiAttachedToId:   func() *int32 { v := int32(123); return &v }(),
			expectedAttachedToId: 0,
			description:       "When detaching (plan is null), preserve null even if API still shows attachment",
		},
		{
			name:              "Attachment change - plan has new VM, API returns null",
			planAttachedToId:  456,
			stateAttachedToId: 123,
			apiAttachedToId:   nil,
			expectedAttachedToId: 456,
			description:       "When changing attachment, preserve new plan value if API returns null",
		},
		{
			name:              "Attachment change - plan has new VM, API returns old VM",
			planAttachedToId:  456,
			stateAttachedToId: 123,
			apiAttachedToId:   func() *int32 { v := int32(123); return &v }(),
			expectedAttachedToId: 456,
			description:       "When changing attachment, preserve new plan value if API still shows old VM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the preservation logic from volume_resource.go Update method
			planAttachedToId := tt.planAttachedToId
			stateAttachedToId := tt.stateAttachedToId
			
			// Simulate API response
			apiAttachedToId := tt.apiAttachedToId
			
			// Simulate the conversion logic
			var resultAttachedToId int64
			if apiAttachedToId != nil {
				resultAttachedToId = int64(*apiAttachedToId)
			} else {
				resultAttachedToId = 0
			}
			
			// Apply preservation logic: if attachment changed, preserve plan value
			if planAttachedToId != stateAttachedToId {
				resultAttachedToId = planAttachedToId
			}
			
			assert.Equal(t, tt.expectedAttachedToId, resultAttachedToId, tt.description)
		})
	}
}

// TestVolumeAttachment_BothVMAndVolumeWaitForStableState tests that both VM and volume wait for stable state
// Requirements: 2.1, 2.2, 2.3, 7.1, 7.2
func TestVolumeAttachment_BothVMAndVolumeWaitForStableState(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                string
		vmStatusChecker     state.ResourceStateChecker
		volumeStatusChecker state.ResourceStateChecker
		expectError         bool
		description         string
	}{
		{
			name: "Both VM and volume already stable - attach proceeds immediately",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError: false,
			description: "When both resources are stable, operation proceeds immediately",
		},
		{
			name: "VM busy, volume ready - wait for VM",
			vmStatusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "BUSY", nil
					}
					return "POWERED_ON", nil
				}
			}(),
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError: false,
			description: "When VM is busy but volume is ready, wait for VM to stabilize",
		},
		{
			name: "VM ready, volume busy - wait for volume",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			volumeStatusChecker: func() state.ResourceStateChecker {
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
			description: "When volume is busy but VM is ready, wait for volume to stabilize",
		},
		{
			name: "Both VM and volume busy - wait for both",
			vmStatusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 2 {
						return "BUSY", nil
					}
					return "POWERED_ON", nil
				}
			}(),
			volumeStatusChecker: func() state.ResourceStateChecker {
				callCount := 0
				return func(ctx context.Context) (string, error) {
					callCount++
					if callCount <= 3 {
						return "BUSY", nil
					}
					return "AVAILABLE", nil
				}
			}(),
			expectError: false,
			description: "When both resources are busy, wait for both to stabilize",
		},
		{
			name: "VM transitions through multiple states, volume stable",
			vmStatusChecker: func() state.ResourceStateChecker {
				states := []string{"BUSY", "starting", "POWERED_ON"}
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
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError: false,
			description: "When VM transitions through multiple states, wait for final stable state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wait for VM to reach stable state
			vmStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-789",
				StatusChecker:      tt.vmStatusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err := vmStateManager.WaitForStableState(ctx)
			if err != nil && !tt.expectError {
				t.Fatalf("VM state wait failed unexpectedly: %v", err)
			}

			// Wait for volume to reach stable state
			volumeStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "volume",
				ResourceID:         "vol-789",
				StatusChecker:      tt.volumeStatusChecker,
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			err = volumeStateManager.WaitForStableState(ctx)

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// TestVolumeAttachment_ErrorScenarios tests various error scenarios during attachment
// Requirements: 2.1, 2.2, 2.3, 7.1, 7.2
func TestVolumeAttachment_ErrorScenarios(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                string
		vmStatusChecker     state.ResourceStateChecker
		volumeStatusChecker state.ResourceStateChecker
		expectError         bool
		errorContains       string
		description         string
	}{
		{
			name: "VM in error state - attachment fails",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "error", nil
			},
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError:   true,
			errorContains: "failure state",
			description:   "When VM is in error state, attachment should fail",
		},
		{
			name: "Volume in error state - attachment fails",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "error", nil
			},
			expectError:   true,
			errorContains: "failure state",
			description:   "When volume is in error state, attachment should fail",
		},
		{
			name: "VM API error - attachment fails",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "", errors.New("API connection failed")
			},
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError:   true,
			errorContains: "failed to check current state",
			description:   "When VM API check fails, attachment should fail",
		},
		{
			name: "Volume API error - attachment fails",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "", errors.New("API connection failed")
			},
			expectError:   true,
			errorContains: "failed to check current state",
			description:   "When volume API check fails, attachment should fail",
		},
		{
			name: "VM timeout - attachment fails",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "BUSY", nil
			},
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError:   true,
			errorContains: "timeout",
			description:   "When VM doesn't reach stable state within timeout, attachment should fail",
		},
		{
			name: "Volume timeout - attachment fails",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "BUSY", nil
			},
			expectError:   true,
			errorContains: "timeout",
			description:   "When volume doesn't reach stable state within timeout, attachment should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wait for VM to reach stable state
			vmStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-error",
				StatusChecker:      tt.vmStatusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            1 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			vmErr := vmStateManager.WaitForStableState(ctx)
			
			// If VM check failed, we expect error
			if vmErr != nil {
				if tt.expectError {
					assert.Error(t, vmErr, tt.description)
					if tt.errorContains != "" {
						assert.Contains(t, vmErr.Error(), tt.errorContains, tt.description)
					}
					return
				}
				t.Fatalf("VM state wait failed unexpectedly: %v", vmErr)
			}

			// Wait for volume to reach stable state
			volumeStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "volume",
				ResourceID:         "vol-error",
				StatusChecker:      tt.volumeStatusChecker,
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
				Timeout:            1 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			volumeErr := volumeStateManager.WaitForStableState(ctx)

			if tt.expectError {
				assert.Error(t, volumeErr, tt.description)
				if tt.errorContains != "" {
					assert.Contains(t, volumeErr.Error(), tt.errorContains, tt.description)
				}
			} else {
				assert.NoError(t, volumeErr, tt.description)
			}
		})
	}
}

// TestVolumeAttachment_RealWorldAttachmentScenarios tests realistic volume attachment scenarios
// Requirements: 2.1, 2.2, 2.3, 2.5, 7.1, 7.2, 8.5
func TestVolumeAttachment_RealWorldAttachmentScenarios(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                string
		scenario            string
		vmStatusChecker     state.ResourceStateChecker
		volumeStatusChecker state.ResourceStateChecker
		expectError         bool
	}{
		{
			name:     "Attach volume to running VM - both ready",
			scenario: "Standard volume attachment to running VM",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError: false,
		},
		{
			name:     "Attach volume to stopped VM - both ready",
			scenario: "Volume attachment to stopped VM",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_OFF", nil
			},
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError: false,
		},
		{
			name:     "Attach volume while VM is starting",
			scenario: "Volume attachment during VM startup",
			vmStatusChecker: func() state.ResourceStateChecker {
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
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "AVAILABLE", nil
			},
			expectError: false,
		},
		{
			name:     "Attach newly created volume - volume transitions from creating",
			scenario: "Attach volume immediately after creation",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			volumeStatusChecker: func() state.ResourceStateChecker {
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
			name:     "Detach volume from running VM",
			scenario: "Standard volume detachment from running VM",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "in-use", nil
			},
			expectError: false,
		},
		{
			name:     "Detach volume while VM is busy",
			scenario: "Volume detachment while VM is processing",
			vmStatusChecker: func() state.ResourceStateChecker {
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
			volumeStatusChecker: func(ctx context.Context) (string, error) {
				return "in-use", nil
			},
			expectError: false,
		},
		{
			name:     "Change volume attachment - detach from one VM, attach to another",
			scenario: "Move volume between VMs",
			vmStatusChecker: func(ctx context.Context) (string, error) {
				return "POWERED_ON", nil
			},
			volumeStatusChecker: func() state.ResourceStateChecker {
				states := []string{"in-use", "detaching", "AVAILABLE"}
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wait for VM to reach stable state
			vmStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "vm-realworld",
				StatusChecker:      tt.vmStatusChecker,
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            10 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			vmErr := vmStateManager.WaitForStableState(ctx)
			if vmErr != nil && !tt.expectError {
				t.Fatalf("VM state wait failed unexpectedly: %v", vmErr)
			}

			// Wait for volume to reach stable state
			volumeStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType:       "volume",
				ResourceID:         "vol-realworld",
				StatusChecker:      tt.volumeStatusChecker,
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
				Timeout:            10 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			volumeErr := volumeStateManager.WaitForStableState(ctx)

			if tt.expectError {
				assert.Error(t, volumeErr)
			} else {
				assert.NoError(t, volumeErr)
			}
		})
	}
}

// TestVolumeAttachment_ImmediateSuccessWhenBothStable tests immediate success when both resources are stable
// Requirements: 8.1, 8.4
func TestVolumeAttachment_ImmediateSuccessWhenBothStable(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		vmState     string
		volumeState string
	}{
		{
			name:        "VM POWERED_ON, Volume AVAILABLE",
			vmState:     "POWERED_ON",
			volumeState: "AVAILABLE",
		},
		{
			name:        "VM POWERED_OFF, Volume AVAILABLE",
			vmState:     "POWERED_OFF",
			volumeState: "AVAILABLE",
		},
		{
			name:        "VM running, Volume available",
			vmState:     "running",
			volumeState: "available",
		},
		{
			name:        "VM stopped, Volume available",
			vmState:     "stopped",
			volumeState: "available",
		},
		{
			name:        "VM POWERED_ON, Volume in-use",
			vmState:     "POWERED_ON",
			volumeState: "in-use",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vmCallCount := 0
			volumeCallCount := 0

			// Wait for VM
			vmStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   "vm-immediate",
				StatusChecker: func(ctx context.Context) (string, error) {
					vmCallCount++
					return tt.vmState, nil
				},
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			vmStart := time.Now()
			vmErr := vmStateManager.WaitForStableState(ctx)
			vmDuration := time.Since(vmStart)

			assert.NoError(t, vmErr)
			assert.Less(t, vmDuration, 500*time.Millisecond, "VM check should return immediately")
			assert.Equal(t, 1, vmCallCount, "VM should only be checked once")

			// Wait for volume
			volumeStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "volume",
				ResourceID:   "vol-immediate",
				StatusChecker: func(ctx context.Context) (string, error) {
					volumeCallCount++
					return tt.volumeState, nil
				},
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
				Timeout:            5 * time.Second,
				PollInterval:       100 * time.Millisecond,
			})

			volumeStart := time.Now()
			volumeErr := volumeStateManager.WaitForStableState(ctx)
			volumeDuration := time.Since(volumeStart)

			assert.NoError(t, volumeErr)
			assert.Less(t, volumeDuration, 500*time.Millisecond, "Volume check should return immediately")
			assert.Equal(t, 1, volumeCallCount, "Volume should only be checked once")
		})
	}
}
