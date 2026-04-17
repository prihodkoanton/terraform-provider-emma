package state

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// StateManager provides utilities for common state operations
type StateManager struct {
	ctx context.Context
}

// NewStateManager creates a new StateManager instance
func NewStateManager(ctx context.Context) *StateManager {
	return &StateManager{ctx: ctx}
}

// RemoveFromState removes a resource from Terraform state
// This is typically used when a resource is not found (404 response)
func (sm *StateManager) RemoveFromState(state *resource.ReadResponse) {
	state.State.RemoveResource(sm.ctx)
}

// UpdateComputedAttributes updates only computed attributes from API response
// This preserves user-specified values while updating server-generated values
func (sm *StateManager) UpdateComputedAttributes(currentState, apiResponse interface{}, computedFields []string) error {
	if currentState == nil || apiResponse == nil {
		return fmt.Errorf("currentState and apiResponse cannot be nil")
	}

	currentVal := reflect.ValueOf(currentState)
	apiVal := reflect.ValueOf(apiResponse)

	// Handle pointers
	if currentVal.Kind() == reflect.Ptr {
		currentVal = currentVal.Elem()
	}
	if apiVal.Kind() == reflect.Ptr {
		apiVal = apiVal.Elem()
	}

	// Ensure both are structs
	if currentVal.Kind() != reflect.Struct || apiVal.Kind() != reflect.Struct {
		return fmt.Errorf("both currentState and apiResponse must be structs")
	}

	// Create a map of computed fields for quick lookup
	computedFieldsMap := make(map[string]bool)
	for _, field := range computedFields {
		computedFieldsMap[field] = true
	}

	// Update only computed fields
	for i := 0; i < currentVal.NumField(); i++ {
		fieldName := currentVal.Type().Field(i).Name
		
		// Skip if not a computed field
		if !computedFieldsMap[fieldName] {
			continue
		}

		// Get the corresponding field from API response
		apiField := apiVal.FieldByName(fieldName)
		if !apiField.IsValid() {
			continue
		}

		// Get the current field
		currentField := currentVal.Field(i)
		if !currentField.CanSet() {
			continue
		}

		// Set the value from API response
		if apiField.Type() == currentField.Type() {
			currentField.Set(apiField)
		}
	}

	return nil
}

// PreserveUserValues preserves user-specified values during state updates
// This ensures that user-provided values are not overwritten by API responses
func (sm *StateManager) PreserveUserValues(currentState, newState interface{}, userFields []string) error {
	if currentState == nil || newState == nil {
		return fmt.Errorf("currentState and newState cannot be nil")
	}

	currentVal := reflect.ValueOf(currentState)
	newVal := reflect.ValueOf(newState)

	// Handle pointers
	if currentVal.Kind() == reflect.Ptr {
		currentVal = currentVal.Elem()
	}
	if newVal.Kind() == reflect.Ptr {
		newVal = newVal.Elem()
	}

	// Ensure both are structs
	if currentVal.Kind() != reflect.Struct || newVal.Kind() != reflect.Struct {
		return fmt.Errorf("both currentState and newState must be structs")
	}

	// Create a map of user fields for quick lookup
	userFieldsMap := make(map[string]bool)
	for _, field := range userFields {
		userFieldsMap[field] = true
	}

	// Preserve user-specified fields
	for i := 0; i < newVal.NumField(); i++ {
		fieldName := newVal.Type().Field(i).Name
		
		// Skip if not a user field
		if !userFieldsMap[fieldName] {
			continue
		}

		// Get the corresponding field from current state
		currentField := currentVal.FieldByName(fieldName)
		if !currentField.IsValid() {
			continue
		}

		// Get the new field
		newField := newVal.Field(i)
		if !newField.CanSet() {
			continue
		}

		// Preserve the value from current state
		if currentField.Type() == newField.Type() {
			newField.Set(currentField)
		}
	}

	return nil
}
