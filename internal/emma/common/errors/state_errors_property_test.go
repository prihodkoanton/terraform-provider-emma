package errors

import (
	"strings"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: async-operations, Property 7: Error Messages Include Current State
// Validates: Requirements 6.1, 6.2, 6.3
func TestProperty_ErrorMessagesIncludeCurrentState(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// Generator for resource types
	resourceTypeGen := gen.OneConstOf("VM", "Volume", "SecurityGroup", "SpotInstance", "SSHKey")

	// Generator for resource IDs
	resourceIDGen := gen.Identifier()

	// Generator for state names (common states across resources)
	stateGen := gen.OneConstOf(
		"BUSY", "POWERED_ON", "POWERED_OFF", "AVAILABLE", "DRAFT",
		"RECOMPOSING", "RECOMPOSED", "pending", "running", "stopped",
		"creating", "attaching", "detaching", "error", "failed",
	)

	// Generator for operation names
	operationGen := gen.OneConstOf(
		"create", "update", "delete", "attach", "detach",
		"hardware edit", "resize", "recompose", "start", "stop",
	)

	// Generator for timeout durations (in milliseconds)
	timeoutGen := gen.IntRange(1000, 600000) // 1 second to 10 minutes

	properties.Property("for any StateTransitionError, error message includes current state and expected state", prop.ForAll(
		func(resourceType, resourceID, currentState, expectedState, operation string, timeoutMs int) bool {
			// Skip if current and expected states are the same (not a valid error scenario)
			if currentState == expectedState {
				return true
			}

			timeout := time.Duration(timeoutMs) * time.Millisecond

			err := &StateTransitionError{
				ResourceType:  resourceType,
				ResourceID:    resourceID,
				CurrentState:  currentState,
				ExpectedState: expectedState,
				Operation:     operation,
				Timeout:       timeout,
			}

			errorMsg := err.Error()

			// Property 1: Error message MUST include the current state
			if !strings.Contains(errorMsg, currentState) {
				return false
			}

			// Property 2: Error message MUST include the expected state
			if !strings.Contains(errorMsg, expectedState) {
				return false
			}

			// Property 3: Error message MUST include "Current state:" label for clarity
			if !strings.Contains(errorMsg, "Current state:") {
				return false
			}

			// Property 4: Error message MUST include the resource type
			if !strings.Contains(errorMsg, resourceType) {
				return false
			}

			// Property 5: Error message MUST include the resource ID
			if !strings.Contains(errorMsg, resourceID) {
				return false
			}

			// Property 6: Error message MUST include the operation
			if !strings.Contains(errorMsg, operation) {
				return false
			}

			// Property 7: Error message MUST include timeout information
			if !strings.Contains(errorMsg, "Timeout:") {
				return false
			}

			// Property 8: Error message MUST provide actionable guidance (Requirement 6.3)
			// Check for guidance keywords
			guidanceKeywords := []string{"Check", "console", "increase", "timeout"}
			hasGuidance := false
			for _, keyword := range guidanceKeywords {
				if strings.Contains(errorMsg, keyword) {
					hasGuidance = true
					break
				}
			}
			if !hasGuidance {
				return false
			}

			return true
		},
		resourceTypeGen,
		resourceIDGen,
		stateGen,
		stateGen, // expected state
		operationGen,
		timeoutGen,
	))

	properties.Property("for any StateConflictError, error message includes current state", prop.ForAll(
		func(resourceType, resourceID, currentState, operation string) bool {
			err := &StateConflictError{
				ResourceType: resourceType,
				ResourceID:   resourceID,
				CurrentState: currentState,
				Operation:    operation,
			}

			errorMsg := err.Error()

			// Property 1: Error message MUST include the current state
			if !strings.Contains(errorMsg, currentState) {
				return false
			}

			// Property 2: Error message MUST include "state" keyword for context
			if !strings.Contains(strings.ToLower(errorMsg), "state") {
				return false
			}

			// Property 3: Error message MUST include the resource type
			if !strings.Contains(errorMsg, resourceType) {
				return false
			}

			// Property 4: Error message MUST include the resource ID
			if !strings.Contains(errorMsg, resourceID) {
				return false
			}

			// Property 5: Error message MUST include the operation
			if !strings.Contains(errorMsg, operation) {
				return false
			}

			// Property 6: Error message MUST suggest automatic retry (Requirement 6.3)
			retryKeywords := []string{"retry", "automatically"}
			hasRetryGuidance := false
			for _, keyword := range retryKeywords {
				if strings.Contains(strings.ToLower(errorMsg), keyword) {
					hasRetryGuidance = true
					break
				}
			}
			if !hasRetryGuidance {
				return false
			}

			return true
		},
		resourceTypeGen,
		resourceIDGen,
		stateGen,
		operationGen,
	))

	properties.Property("for any error with state information, the error message is non-empty and well-formed", prop.ForAll(
		func(resourceType, resourceID, currentState, expectedState, operation string, timeoutMs int) bool {
			// Skip if current and expected states are the same
			if currentState == expectedState {
				return true
			}

			// Skip if resourceID is "nil" (gopter's Identifier generator can produce this)
			if resourceID == "nil" {
				return true
			}

			timeout := time.Duration(timeoutMs) * time.Millisecond

			err := &StateTransitionError{
				ResourceType:  resourceType,
				ResourceID:    resourceID,
				CurrentState:  currentState,
				ExpectedState: expectedState,
				Operation:     operation,
				Timeout:       timeout,
			}

			errorMsg := err.Error()

			// Property 1: Error message MUST be non-empty
			if len(errorMsg) == 0 {
				return false
			}

			// Property 2: Error message MUST be reasonably long (contains useful information)
			// A well-formed error should be at least 50 characters
			if len(errorMsg) < 50 {
				return false
			}

			// Property 3: Error message MUST not contain placeholder text
			placeholders := []string{"%s", "%v", "%d", "<nil>"}
			for _, placeholder := range placeholders {
				if strings.Contains(errorMsg, placeholder) {
					return false
				}
			}

			// Property 4: Error message MUST be properly formatted (no double spaces)
			if strings.Contains(errorMsg, "  ") {
				return false
			}

			return true
		},
		resourceTypeGen,
		resourceIDGen,
		stateGen,
		stateGen, // expected state
		operationGen,
		timeoutGen,
	))

	properties.Property("for any StateConflictError, the error message is non-empty and well-formed", prop.ForAll(
		func(resourceType, resourceID, currentState, operation string) bool {
			// Skip if resourceID is "nil" (gopter's Identifier generator can produce this)
			if resourceID == "nil" {
				return true
			}

			err := &StateConflictError{
				ResourceType: resourceType,
				ResourceID:   resourceID,
				CurrentState: currentState,
				Operation:    operation,
			}

			errorMsg := err.Error()

			// Property 1: Error message MUST be non-empty
			if len(errorMsg) == 0 {
				return false
			}

			// Property 2: Error message MUST be reasonably long (contains useful information)
			if len(errorMsg) < 30 {
				return false
			}

			// Property 3: Error message MUST not contain placeholder text
			placeholders := []string{"%s", "%v", "%d", "<nil>"}
			for _, placeholder := range placeholders {
				if strings.Contains(errorMsg, placeholder) {
					return false
				}
			}

			// Property 4: Error message MUST be properly formatted (no double spaces)
			if strings.Contains(errorMsg, "  ") {
				return false
			}

			return true
		},
		resourceTypeGen,
		resourceIDGen,
		stateGen,
		operationGen,
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 7: Error Messages Include Current State (Edge Cases)
// Validates: Requirements 6.1, 6.2, 6.3
func TestProperty_ErrorMessagesHandleEdgeCases(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50
	properties := gopter.NewProperties(parameters)

	// Generator for potentially problematic strings
	edgeCaseStringGen := gen.OneConstOf(
		"", // empty string
		" ", // single space
		"  ", // multiple spaces
		"state-with-dashes",
		"state_with_underscores",
		"StateWithCamelCase",
		"state with spaces",
		"123numeric",
		"special!@#$%",
	)

	properties.Property("error messages handle edge case strings gracefully", prop.ForAll(
		func(resourceType, resourceID, currentState, expectedState, operation string) bool {
			// Skip completely empty inputs (not realistic)
			if resourceType == "" || resourceID == "" || operation == "" {
				return true
			}

			// Skip if current and expected states are the same
			if currentState == expectedState {
				return true
			}

			err := &StateTransitionError{
				ResourceType:  resourceType,
				ResourceID:    resourceID,
				CurrentState:  currentState,
				ExpectedState: expectedState,
				Operation:     operation,
				Timeout:       5 * time.Minute,
			}

			errorMsg := err.Error()

			// Property 1: Error message MUST be non-empty even with edge case inputs
			if len(errorMsg) == 0 {
				return false
			}

			// Property 2: Error message MUST include non-empty states
			if currentState != "" && !strings.Contains(errorMsg, currentState) {
				return false
			}
			if expectedState != "" && !strings.Contains(errorMsg, expectedState) {
				return false
			}

			// Property 3: Error message MUST not panic or produce malformed output
			// (if we got here without panic, this passes)

			return true
		},
		edgeCaseStringGen,
		edgeCaseStringGen,
		edgeCaseStringGen,
		edgeCaseStringGen,
		edgeCaseStringGen,
	))

	properties.Property("StateConflictError handles edge case strings gracefully", prop.ForAll(
		func(resourceType, resourceID, currentState, operation string) bool {
			// Skip completely empty inputs (not realistic)
			if resourceType == "" || resourceID == "" || operation == "" {
				return true
			}

			err := &StateConflictError{
				ResourceType: resourceType,
				ResourceID:   resourceID,
				CurrentState: currentState,
				Operation:    operation,
			}

			errorMsg := err.Error()

			// Property 1: Error message MUST be non-empty even with edge case inputs
			if len(errorMsg) == 0 {
				return false
			}

			// Property 2: Error message MUST include non-empty current state
			if currentState != "" && !strings.Contains(errorMsg, currentState) {
				return false
			}

			// Property 3: Error message MUST not panic or produce malformed output
			// (if we got here without panic, this passes)

			return true
		},
		edgeCaseStringGen,
		edgeCaseStringGen,
		edgeCaseStringGen,
		edgeCaseStringGen,
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
