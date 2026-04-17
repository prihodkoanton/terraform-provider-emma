package errors

import (
	"fmt"
	"time"
)

// StateTransitionError represents an error during state transition
type StateTransitionError struct {
	ResourceType  string
	ResourceID    string
	CurrentState  string
	ExpectedState string
	Operation     string
	Timeout       time.Duration
}

// Error implements the error interface
func (e *StateTransitionError) Error() string {
	return fmt.Sprintf(
		"Timeout waiting for %s %s to reach state %s for operation %s. "+
			"Current state: %s. Timeout: %v. "+
			"Check the Emma console for more details or increase the timeout.",
		e.ResourceType,
		e.ResourceID,
		e.ExpectedState,
		e.Operation,
		e.CurrentState,
		e.Timeout,
	)
}

// StateConflictError represents a resource state conflict
type StateConflictError struct {
	ResourceType string
	ResourceID   string
	CurrentState string
	Operation    string
}

// Error implements the error interface
func (e *StateConflictError) Error() string {
	return fmt.Sprintf(
		"Cannot perform %s on %s %s: resource is in %s state. "+
			"The provider will automatically retry this operation.",
		e.Operation,
		e.ResourceType,
		e.ResourceID,
		e.CurrentState,
	)
}
