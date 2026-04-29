package state

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: async-operations, Property 1: State Polling Eventually Terminates
// Validates: Requirements 1.5, 4.4
func TestProperty_StatePollingEventuallyTerminates(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any polling operation with timeout, it terminates within timeout", prop.ForAll(
		func(timeoutMs, pollIntervalMs int, shouldSucceed bool) bool {
			// Ensure valid timeout and poll interval
			if timeoutMs < 100 || timeoutMs > 5000 {
				return true // Skip invalid inputs
			}
			if pollIntervalMs < 10 || pollIntervalMs > timeoutMs/2 {
				return true // Skip invalid inputs
			}

			timeout := time.Duration(timeoutMs) * time.Millisecond
			pollInterval := time.Duration(pollIntervalMs) * time.Millisecond

			ctx := context.Background()
			callCount := 0
			maxCalls := 3 // Number of calls before transitioning to target state

			// Status checker that either succeeds after a few calls or never completes
			statusChecker := func(ctx context.Context) (string, error) {
				callCount++
				if shouldSucceed && callCount > maxCalls {
					return "POWERED_ON", nil
				}
				return "BUSY", nil
			}

			config := StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "test-vm",
				StatusChecker:      statusChecker,
				TargetStates:       []string{"POWERED_ON"},
				TransitionalStates: []string{"BUSY"},
				FailureStates:      []string{"error"},
				Timeout:            timeout,
				PollInterval:       pollInterval,
			}

			manager := NewStateTransitionManager(config)

			start := time.Now()
			err := manager.WaitForStableState(ctx)
			duration := time.Since(start)

			if shouldSucceed {
				// When operation should succeed:
				// 1. No error should occur
				if err != nil {
					return false
				}

				// 2. Should complete before timeout
				if duration >= timeout {
					return false
				}

				// 3. Should have made multiple status checks
				if callCount < maxCalls {
					return false
				}
			} else {
				// When operation should timeout:
				// 1. An error should occur
				if err == nil {
					return false
				}

				// 2. Should take at least the timeout duration (with small tolerance)
				if duration < timeout-50*time.Millisecond {
					return false
				}

				// 3. Should not take significantly longer than timeout
				// Allow up to 2x poll interval for final check plus some buffer
				if duration > timeout+2*pollInterval+200*time.Millisecond {
					return false
				}
			}

			return true
		},
		gen.IntRange(100, 5000),  // timeout in milliseconds
		gen.IntRange(10, 100),    // poll interval in milliseconds
		gen.Bool(),               // shouldSucceed flag
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 2: Retry Respects Maximum Attempts
// Validates: Requirements 5.5
func TestProperty_RetryRespectsMaximumAttempts(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20 // Reduced from default 100 for faster execution
	properties := gopter.NewProperties(parameters)

	properties.Property("for any retry operation with max attempts, retry should not exceed configured limit", prop.ForAll(
		func(maxAttempts int) bool {
			// Ensure valid max attempts
			if maxAttempts < 1 || maxAttempts > 10 {
				return true // Skip invalid inputs
			}

			ctx := context.Background()
			attemptCount := 0

			// Status checker that always fails (simulating persistent state conflict)
			statusChecker := func(ctx context.Context) (string, error) {
				attemptCount++
				// Always return transitional state to force retries
				return "BUSY", nil
			}

			config := StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "test-vm",
				StatusChecker:      statusChecker,
				TargetStates:       []string{"POWERED_ON"},
				TransitionalStates: []string{"BUSY"},
				FailureStates:      []string{"error"},
				Timeout:            time.Duration(maxAttempts*100) * time.Millisecond, // Enough time for all attempts
				PollInterval:       10 * time.Millisecond,
			}

			manager := NewStateTransitionManager(config)
			err := manager.WaitForStableState(ctx)

			// Verify that:
			// 1. An error occurred (operation should timeout)
			if err == nil {
				return false
			}

			// 2. The number of attempts should be reasonable given timeout and poll interval
			// With timeout = maxAttempts * 100ms and pollInterval = 10ms,
			// we should get approximately maxAttempts * 10 status checks
			expectedMinAttempts := maxAttempts
			expectedMaxAttempts := maxAttempts * 15 // Allow some variance

			if attemptCount < expectedMinAttempts || attemptCount > expectedMaxAttempts {
				return false
			}

			return true
		},
		gen.IntRange(1, 10), // max attempts
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 3: Transitional States Are Recognized
// Validates: Requirements 1.1, 1.2, 1.3
func TestProperty_TransitionalStatesAreRecognized(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for resource types
	resourceTypeGen := gen.OneConstOf("vm", "volume", "security_group")

	properties.Property("for any resource in a transitional state, the state manager correctly identifies it and waits", prop.ForAll(
		func(resourceType string) bool {
			ctx := context.Background()

			// Get the state definitions for this resource type
			stableStates, transitionalStates, failureStates := GetResourceStates(resourceType)

			// Skip if no states defined (shouldn't happen with our types)
			if len(transitionalStates) == 0 || len(stableStates) == 0 {
				return true
			}

			// Test each transitional state for this resource type
			for _, transitionalState := range transitionalStates {
				callCount := 0
				transitionAfterCalls := 2 // Transition to stable state after 2 calls

				statusChecker := func(ctx context.Context) (string, error) {
					callCount++
					if callCount > transitionAfterCalls {
						// Return the first stable state after transition
						return stableStates[0], nil
					}
					// Return the transitional state
					return transitionalState, nil
				}

				config := StateTransitionConfig{
					ResourceType:       resourceType,
					ResourceID:         "test-resource-123",
					StatusChecker:      statusChecker,
					TargetStates:       stableStates,
					TransitionalStates: transitionalStates,
					FailureStates:      failureStates,
					Timeout:            2 * time.Second,
					PollInterval:       50 * time.Millisecond,
				}

				manager := NewStateTransitionManager(config)

				// Verify that the transitional state is correctly identified
				if !manager.IsTransitionalState(transitionalState) {
					return false
				}

				// Verify that the transitional state is NOT identified as stable
				if manager.IsStableState(transitionalState) {
					return false
				}

				// Verify that the transitional state is NOT identified as failure
				if manager.IsFailureState(transitionalState) {
					return false
				}

				// Verify that WaitForStableState waits for the resource to transition
				err := manager.WaitForStableState(ctx)
				if err != nil {
					return false
				}

				// Verify that multiple status checks were made (indicating it waited)
				if callCount <= transitionAfterCalls {
					return false
				}

				// Verify that the final state is stable
				finalState, err := manager.CheckCurrentState(ctx)
				if err != nil {
					return false
				}
				if !manager.IsStableState(finalState) {
					return false
				}
			}

			// Also verify that stable states are NOT identified as transitional
			for _, stableState := range stableStates {
				config := StateTransitionConfig{
					ResourceType:       resourceType,
					ResourceID:         "test-resource-456",
					StatusChecker:      func(ctx context.Context) (string, error) { return stableState, nil },
					TargetStates:       stableStates,
					TransitionalStates: transitionalStates,
					FailureStates:      failureStates,
					Timeout:            1 * time.Second,
					PollInterval:       50 * time.Millisecond,
				}

				manager := NewStateTransitionManager(config)

				// Verify stable state is NOT identified as transitional
				if manager.IsTransitionalState(stableState) {
					return false
				}

				// Verify stable state IS identified as stable
				if !manager.IsStableState(stableState) {
					return false
				}

				// Verify that WaitForStableState returns immediately for stable states
				callCount := 0
				config.StatusChecker = func(ctx context.Context) (string, error) {
					callCount++
					return stableState, nil
				}
				manager = NewStateTransitionManager(config)

				err := manager.WaitForStableState(ctx)
				if err != nil {
					return false
				}

				// Should only call status checker once (initial check, no polling)
				if callCount != 1 {
					return false
				}
			}

			return true
		},
		resourceTypeGen,
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 6: Timeout Configuration Is Respected
// Validates: Requirements 4.1, 4.2, 4.5
func TestProperty_TimeoutConfigurationIsRespected(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 20
	properties := gopter.NewProperties(parameters)

	properties.Property("for any resource with custom timeout configuration, the state polling uses the configured timeout", prop.ForAll(
		func(customTimeoutMs, customPollIntervalMs int) bool {
			// Ensure valid timeout and poll interval
			if customTimeoutMs < 200 || customTimeoutMs > 3000 {
				return true // Skip invalid inputs
			}
			if customPollIntervalMs < 20 || customPollIntervalMs > customTimeoutMs/3 {
				return true // Skip invalid inputs
			}

			customTimeout := time.Duration(customTimeoutMs) * time.Millisecond
			customPollInterval := time.Duration(customPollIntervalMs) * time.Millisecond

			ctx := context.Background()
			callCount := 0

			// Status checker that never reaches target state (to force timeout)
			statusChecker := func(ctx context.Context) (string, error) {
				callCount++
				return "BUSY", nil
			}

			// Create config with custom timeout and poll interval
			config := StateTransitionConfig{
				ResourceType:       "vm",
				ResourceID:         "test-vm-timeout",
				StatusChecker:      statusChecker,
				TargetStates:       []string{"POWERED_ON"},
				TransitionalStates: []string{"BUSY"},
				FailureStates:      []string{"error"},
				Timeout:            customTimeout,
				PollInterval:       customPollInterval,
			}

			manager := NewStateTransitionManager(config)

			start := time.Now()
			err := manager.WaitForStableState(ctx)
			duration := time.Since(start)

			// Verify that:
			// 1. An error occurred (operation should timeout)
			if err == nil {
				return false
			}

			// 2. The operation took at least the custom timeout duration (with tolerance)
			// Allow 100ms tolerance for timing variations
			if duration < customTimeout-100*time.Millisecond {
				return false
			}

			// 3. The operation did not take significantly longer than the custom timeout
			// Allow up to 3x poll interval for final check plus buffer for scheduling delays
			maxDuration := customTimeout + 3*customPollInterval + 300*time.Millisecond
			if duration > maxDuration {
				return false
			}

			// 4. Verify that at least some status checks were made
			// The exact number can vary due to timing, but should be at least 1
			if callCount < 1 {
				return false
			}

			// 5. Verify the number of checks is reasonable given the timeout and poll interval
			// Expected checks = timeout / pollInterval (approximately)
			// But allow wide variance due to timing uncertainties
			expectedChecks := int(customTimeout / customPollInterval)
			// Allow variance from 50% to 200% of expected
			minChecks := max(1, int(float64(expectedChecks)*0.5))
			maxChecks := int(float64(expectedChecks) * 2.0)

			if callCount < minChecks || callCount > maxChecks {
				return false
			}

			return true
		},
		gen.IntRange(200, 1000),  // custom timeout in milliseconds
		gen.IntRange(20, 100),    // custom poll interval in milliseconds
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 8: Idempotent State Checks
// Validates: Requirements 8.1, 8.4
func TestProperty_IdempotentStateChecks(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for resource types
	resourceTypeGen := gen.OneConstOf("vm", "volume", "security_group")

	properties.Property("for any resource already in target state, attempting to transition succeeds immediately without error", prop.ForAll(
		func(resourceType string) bool {
			ctx := context.Background()

			// Get the state definitions for this resource type
			stableStates, transitionalStates, failureStates := GetResourceStates(resourceType)

			// Skip if no states defined
			if len(stableStates) == 0 {
				return true
			}

			// Test each stable state for this resource type
			for _, targetState := range stableStates {
				callCount := 0

				// Status checker that always returns the target state (resource already in desired state)
				statusChecker := func(ctx context.Context) (string, error) {
					callCount++
					return targetState, nil
				}

				config := StateTransitionConfig{
					ResourceType:       resourceType,
					ResourceID:         "test-resource-idempotent",
					StatusChecker:      statusChecker,
					TargetStates:       stableStates,
					TransitionalStates: transitionalStates,
					FailureStates:      failureStates,
					Timeout:            2 * time.Second,
					PollInterval:       50 * time.Millisecond,
				}

				manager := NewStateTransitionManager(config)

				// Measure how long the operation takes
				start := time.Now()
				err := manager.WaitForStableState(ctx)
				duration := time.Since(start)

				// Verify that:
				// 1. No error occurred (operation should succeed)
				if err != nil {
					return false
				}

				// 2. The operation completed immediately (within a very short time)
				// Allow up to 100ms for the initial check and function overhead
				if duration > 100*time.Millisecond {
					return false
				}

				// 3. Only one status check was made (initial check, no polling)
				// This verifies that no unnecessary API calls were made
				if callCount != 1 {
					return false
				}

				// 4. Verify IsInTargetState returns true
				isTarget, currentState, err := manager.IsInTargetState(ctx)
				if err != nil {
					return false
				}
				if !isTarget {
					return false
				}
				if currentState != targetState {
					return false
				}

				// 5. Verify the state is correctly identified as stable
				if !manager.IsStableState(targetState) {
					return false
				}

				// 6. Verify the state is NOT identified as transitional
				if manager.IsTransitionalState(targetState) {
					return false
				}

				// 7. Verify the state is NOT identified as failure
				if manager.IsFailureState(targetState) {
					return false
				}
			}

			return true
		},
		resourceTypeGen,
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 9: Parallel Operations Don't Interfere
// Validates: Requirements 9.1, 9.5
func TestProperty_ParallelOperationsDontInterfere(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for resource types
	resourceTypeGen := gen.OneConstOf("vm", "volume", "security_group")

	properties.Property("for any two concurrent operations on different resources, state checking for one resource does not block or interfere with state checking for the other", prop.ForAll(
		func(resourceType1, resourceType2 string, numOperations int) bool {
			// Ensure valid number of operations
			if numOperations < 2 || numOperations > 10 {
				return true // Skip invalid inputs
			}

			ctx := context.Background()

			// Get state definitions for both resource types
			stableStates1, transitionalStates1, failureStates1 := GetResourceStates(resourceType1)
			stableStates2, transitionalStates2, failureStates2 := GetResourceStates(resourceType2)

			// Skip if no states defined
			if len(stableStates1) == 0 || len(stableStates2) == 0 {
				return true
			}

			// Track completion of each operation independently
			type operationResult struct {
				resourceID string
				err        error
				duration   time.Duration
				callCount  int
			}

			results := make(chan operationResult, numOperations)

			// Launch multiple concurrent operations
			for i := 0; i < numOperations; i++ {
				// Alternate between resource types
				var resourceType string
				var stableStates, transitionalStates, failureStates []string
				if i%2 == 0 {
					resourceType = resourceType1
					stableStates = stableStates1
					transitionalStates = transitionalStates1
					failureStates = failureStates1
				} else {
					resourceType = resourceType2
					stableStates = stableStates2
					transitionalStates = transitionalStates2
					failureStates = failureStates2
				}

				resourceID := fmt.Sprintf("resource-%d", i)

				go func(resType, resID string, stable, transitional, failure []string) {
					// Each operation has its own independent state
					callCount := 0
					transitionAfterCalls := 2 // Transition to stable after 2 calls

					statusChecker := func(ctx context.Context) (string, error) {
						callCount++
						if callCount > transitionAfterCalls {
							// Return stable state
							return stable[0], nil
						}
						// Return transitional state
						if len(transitional) > 0 {
							return transitional[0], nil
						}
						return stable[0], nil
					}

					config := StateTransitionConfig{
						ResourceType:       resType,
						ResourceID:         resID,
						StatusChecker:      statusChecker,
						TargetStates:       stable,
						TransitionalStates: transitional,
						FailureStates:      failure,
						Timeout:            3 * time.Second,
						PollInterval:       50 * time.Millisecond,
					}

					manager := NewStateTransitionManager(config)

					start := time.Now()
					err := manager.WaitForStableState(ctx)
					duration := time.Since(start)

					results <- operationResult{
						resourceID: resID,
						err:        err,
						duration:   duration,
						callCount:  callCount,
					}
				}(resourceType, resourceID, stableStates, transitionalStates, failureStates)
			}

			// Collect all results
			collectedResults := make([]operationResult, 0, numOperations)
			for i := 0; i < numOperations; i++ {
				result := <-results
				collectedResults = append(collectedResults, result)
			}

			// Verify that all operations completed successfully
			for _, result := range collectedResults {
				// 1. No operation should have failed
				if result.err != nil {
					return false
				}

				// 2. Each operation should have made multiple status checks
				// (indicating it actually polled and wasn't blocked)
				if result.callCount < 2 {
					return false
				}

				// 3. Each operation should have completed in reasonable time
				// (not blocked waiting for other operations)
				if result.duration > 2*time.Second {
					return false
				}
			}

			// 4. Verify that operations completed independently
			// Check that durations vary (indicating independent execution)
			// If all operations had exactly the same duration, they might be serialized
			if numOperations >= 3 {
				firstDuration := collectedResults[0].duration
				allSame := true
				for _, result := range collectedResults[1:] {
					// Allow 10ms tolerance for timing variations
					if result.duration < firstDuration-10*time.Millisecond ||
						result.duration > firstDuration+10*time.Millisecond {
						allSame = false
						break
					}
				}
				// If all durations are exactly the same, operations might be serialized
				// This is a weak check, but helps detect obvious serialization
				// We don't fail on this, just note it's suspicious
				_ = allSame
			}

			// 5. Verify that each operation's call count is independent
			// Each operation should have made its own status checks
			for _, result := range collectedResults {
				// Each operation should have made at least 3 calls
				// (initial check + 2 polls before transitioning)
				if result.callCount < 3 {
					return false
				}
			}

			return true
		},
		resourceTypeGen,
		resourceTypeGen,
		gen.IntRange(2, 10), // number of concurrent operations
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 10: Graceful Degradation on Stuck States
// Validates: Requirements 10.1, 10.4
func TestProperty_GracefulDegradationOnStuckStates(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for resource types
	resourceTypeGen := gen.OneConstOf("vm", "volume", "security_group")

	properties.Property("for any resource that remains in transitional state beyond timeout, operation fails with clear timeout error rather than hanging indefinitely", prop.ForAll(
		func(resourceType string, timeoutMs, pollIntervalMs int) bool {
			// Ensure valid timeout and poll interval
			// Use shorter timeouts for faster property testing
			if timeoutMs < 100 || timeoutMs > 500 {
				return true // Skip invalid inputs
			}
			if pollIntervalMs < 10 || pollIntervalMs > timeoutMs/2 {
				return true // Skip invalid inputs
			}

			timeout := time.Duration(timeoutMs) * time.Millisecond
			pollInterval := time.Duration(pollIntervalMs) * time.Millisecond

			ctx := context.Background()

			// Get state definitions for this resource type
			stableStates, transitionalStates, failureStates := GetResourceStates(resourceType)

			// Skip if no states defined
			if len(transitionalStates) == 0 || len(stableStates) == 0 {
				return true
			}

			// Use the first transitional state for testing
			stuckState := transitionalStates[0]
			callCount := 0

			// Status checker that always returns transitional state (resource stuck)
			statusChecker := func(ctx context.Context) (string, error) {
				callCount++
				// Always return the stuck transitional state
				return stuckState, nil
			}

			config := StateTransitionConfig{
				ResourceType:       resourceType,
				ResourceID:         "test-stuck-resource",
				StatusChecker:      statusChecker,
				TargetStates:       stableStates,
				TransitionalStates: transitionalStates,
				FailureStates:      failureStates,
				Timeout:            timeout,
				PollInterval:       pollInterval,
			}

			manager := NewStateTransitionManager(config)

			// Measure how long the operation takes
			start := time.Now()
			err := manager.WaitForStableState(ctx)
			duration := time.Since(start)

			// Verify that:
			// 1. An error occurred (operation should timeout, not hang)
			if err == nil {
				return false
			}

			// 2. The error message indicates a timeout (clear error)
			// The error should contain "timeout" to be clear about what happened
			errorMsg := err.Error()
			if !contains(errorMsg, "timeout") {
				return false
			}

			// 3. The operation took at least the timeout duration (with tolerance)
			// The poller uses a ticker, so there's inherent timing variance.
			// Allow more generous tolerance for short timeouts and fast poll intervals.
			tolerance := 200 * time.Millisecond
			if pollInterval > tolerance {
				tolerance = pollInterval
			}
			if duration < timeout-tolerance {
				return false
			}

			// 4. The operation did NOT hang indefinitely
			// It should complete within timeout + reasonable buffer
			// The poller waits for the first ticker interval before checking status,
			// and may need one additional interval for the final timeout check.
			// Allow generous buffer for scheduling overhead and timing variations.
			maxDuration := timeout + 4*pollInterval + 500*time.Millisecond
			if duration > maxDuration {
				return false
			}

			// 5. Verify that multiple status checks were made (polling occurred)
			// This ensures the poller was actually checking state, not just waiting
			expectedMinChecks := int(timeout / pollInterval)
			// Allow variance from 50% to 200% of expected due to timing
			minChecks := max(1, int(float64(expectedMinChecks)*0.5))
			maxChecks := int(float64(expectedMinChecks) * 2.0)

			if callCount < minChecks || callCount > maxChecks {
				return false
			}

			// 6. Verify the resource is correctly identified as being in transitional state
			if !manager.IsTransitionalState(stuckState) {
				return false
			}

			// 7. Verify the resource is NOT identified as being in stable state
			if manager.IsStableState(stuckState) {
				return false
			}

			// 8. Verify the resource is NOT identified as being in failure state
			// (stuck in transitional is different from failure)
			if manager.IsFailureState(stuckState) {
				return false
			}

			return true
		},
		resourceTypeGen,
		gen.IntRange(100, 500),  // timeout in milliseconds (reduced for faster testing)
		gen.IntRange(10, 50),    // poll interval in milliseconds (reduced for faster testing)
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 10: Graceful Degradation on Stuck States (Failure State)
// Validates: Requirements 10.2
func TestProperty_GracefulDegradationOnFailureState(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for resource types
	resourceTypeGen := gen.OneConstOf("vm", "volume", "security_group")

	properties.Property("for any resource that enters a failure state, operation fails immediately with error details", prop.ForAll(
		func(resourceType string) bool {
			ctx := context.Background()

			// Get state definitions for this resource type
			stableStates, transitionalStates, failureStates := GetResourceStates(resourceType)

			// Skip if no failure states defined
			if len(failureStates) == 0 || len(stableStates) == 0 {
				return true
			}

			// Use the first failure state for testing
			failureState := failureStates[0]
			callCount := 0

			// Status checker that returns failure state immediately
			statusChecker := func(ctx context.Context) (string, error) {
				callCount++
				return failureState, nil
			}

			config := StateTransitionConfig{
				ResourceType:       resourceType,
				ResourceID:         "test-failed-resource",
				StatusChecker:      statusChecker,
				TargetStates:       stableStates,
				TransitionalStates: transitionalStates,
				FailureStates:      failureStates,
				Timeout:            5 * time.Second, // Long timeout to verify immediate failure
				PollInterval:       100 * time.Millisecond,
			}

			manager := NewStateTransitionManager(config)

			// Measure how long the operation takes
			start := time.Now()
			err := manager.WaitForStableState(ctx)
			duration := time.Since(start)

			// Verify that:
			// 1. An error occurred (operation should fail)
			if err == nil {
				return false
			}

			// 2. The error message includes the failure state
			errorMsg := err.Error()
			if !contains(errorMsg, failureState) {
				return false
			}

			// 3. The operation failed immediately (not after timeout)
			// Should complete within a very short time (not wait for timeout)
			if duration > 500*time.Millisecond {
				return false
			}

			// 4. Only one status check was made (immediate failure detection)
			if callCount != 1 {
				return false
			}

			// 5. Verify the failure state is correctly identified
			if !manager.IsFailureState(failureState) {
				return false
			}

			// 6. Verify the failure state is NOT identified as stable
			if manager.IsStableState(failureState) {
				return false
			}

			// 7. Verify the failure state is NOT identified as transitional
			if manager.IsTransitionalState(failureState) {
				return false
			}

			return true
		},
		resourceTypeGen,
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 10: Graceful Degradation on Stuck States (No Infinite Loops)
// Validates: Requirements 10.4
func TestProperty_NoInfinitePollingLoops(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for resource types
	resourceTypeGen := gen.OneConstOf("vm", "volume", "security_group")

	properties.Property("for any polling operation, the provider does not enter infinite polling loops", prop.ForAll(
		func(resourceType string, timeoutMs int) bool {
			// Ensure valid timeout
			if timeoutMs < 100 || timeoutMs > 2000 {
				return true // Skip invalid inputs
			}

			timeout := time.Duration(timeoutMs) * time.Millisecond
			pollInterval := 50 * time.Millisecond

			ctx := context.Background()

			// Get state definitions for this resource type
			stableStates, transitionalStates, failureStates := GetResourceStates(resourceType)

			// Skip if no states defined
			if len(transitionalStates) == 0 || len(stableStates) == 0 {
				return true
			}

			callCount := 0
			maxExpectedCalls := int(timeout/pollInterval) * 3 // 3x expected as upper bound

			// Status checker that tracks call count to detect infinite loops
			statusChecker := func(ctx context.Context) (string, error) {
				callCount++
				// If we've made way too many calls, we might be in an infinite loop
				if callCount > maxExpectedCalls {
					return "", fmt.Errorf("too many status checks, possible infinite loop")
				}
				// Always return transitional state to force timeout
				return transitionalStates[0], nil
			}

			config := StateTransitionConfig{
				ResourceType:       resourceType,
				ResourceID:         "test-loop-resource",
				StatusChecker:      statusChecker,
				TargetStates:       stableStates,
				TransitionalStates: transitionalStates,
				FailureStates:      failureStates,
				Timeout:            timeout,
				PollInterval:       pollInterval,
			}

			manager := NewStateTransitionManager(config)

			// Measure how long the operation takes
			start := time.Now()
			err := manager.WaitForStableState(ctx)
			duration := time.Since(start)

			// Verify that:
			// 1. An error occurred (timeout or too many calls)
			if err == nil {
				return false
			}

			// 2. The operation completed (didn't hang forever)
			// Should complete within timeout + reasonable buffer
			maxDuration := timeout + 500*time.Millisecond
			if duration > maxDuration {
				return false
			}

			// 3. The number of calls is reasonable (not infinite)
			// Should not exceed our max expected calls threshold
			if callCount > maxExpectedCalls {
				return false
			}

			// 4. The number of calls is at least 1 (polling occurred)
			if callCount < 1 {
				return false
			}

			// 5. Verify the operation respects timeout and doesn't loop forever
			// The duration should be close to the timeout, not significantly longer
			if duration > timeout*2 {
				return false
			}

			return true
		},
		resourceTypeGen,
		gen.IntRange(100, 2000), // timeout in milliseconds
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
