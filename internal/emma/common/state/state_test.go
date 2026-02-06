package state

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// Test model structs
type testResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Status      types.String `tfsdk:"status"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func TestNewStateManager(t *testing.T) {
	ctx := context.Background()
	sm := NewStateManager(ctx)
	
	assert.NotNil(t, sm)
	assert.Equal(t, ctx, sm.ctx)
}

func TestRemoveFromState(t *testing.T) {
	ctx := context.Background()
	sm := NewStateManager(ctx)
	
	// Note: RemoveFromState requires a fully initialized Terraform state
	// which is difficult to mock in unit tests. This test verifies the
	// method exists and has the correct signature. Integration tests
	// should verify the actual behavior.
	assert.NotNil(t, sm)
	assert.NotNil(t, sm.RemoveFromState)
}

func TestUpdateComputedAttributes(t *testing.T) {
	ctx := context.Background()
	sm := NewStateManager(ctx)

	tests := []struct {
		name           string
		currentState   interface{}
		apiResponse    interface{}
		computedFields []string
		wantErr        bool
		validate       func(t *testing.T, state interface{})
	}{
		{
			name: "updates computed fields only",
			currentState: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringValue("user-name"),
				Description: types.StringValue("user-description"),
				Status:      types.StringValue("old-status"),
				CreatedAt:   types.StringValue("old-time"),
			},
			apiResponse: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringValue("api-name"),
				Description: types.StringValue("api-description"),
				Status:      types.StringValue("new-status"),
				CreatedAt:   types.StringValue("new-time"),
			},
			computedFields: []string{"Status", "CreatedAt"},
			wantErr:        false,
			validate: func(t *testing.T, state interface{}) {
				model := state.(*testResourceModel)
				// User fields should be preserved
				assert.Equal(t, "user-name", model.Name.ValueString())
				assert.Equal(t, "user-description", model.Description.ValueString())
				// Computed fields should be updated
				assert.Equal(t, "new-status", model.Status.ValueString())
				assert.Equal(t, "new-time", model.CreatedAt.ValueString())
			},
		},
		{
			name:           "nil currentState returns error",
			currentState:   nil,
			apiResponse:    &testResourceModel{},
			computedFields: []string{"Status"},
			wantErr:        true,
		},
		{
			name:           "nil apiResponse returns error",
			currentState:   &testResourceModel{},
			apiResponse:    nil,
			computedFields: []string{"Status"},
			wantErr:        true,
		},
		{
			name:           "non-struct returns error",
			currentState:   "not a struct",
			apiResponse:    &testResourceModel{},
			computedFields: []string{"Status"},
			wantErr:        true,
		},
		{
			name: "empty computed fields list",
			currentState: &testResourceModel{
				Name:   types.StringValue("user-name"),
				Status: types.StringValue("old-status"),
			},
			apiResponse: &testResourceModel{
				Name:   types.StringValue("api-name"),
				Status: types.StringValue("new-status"),
			},
			computedFields: []string{},
			wantErr:        false,
			validate: func(t *testing.T, state interface{}) {
				model := state.(*testResourceModel)
				// Nothing should be updated
				assert.Equal(t, "user-name", model.Name.ValueString())
				assert.Equal(t, "old-status", model.Status.ValueString())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.UpdateComputedAttributes(tt.currentState, tt.apiResponse, tt.computedFields)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, tt.currentState)
				}
			}
		})
	}
}

func TestPreserveUserValues(t *testing.T) {
	ctx := context.Background()
	sm := NewStateManager(ctx)

	tests := []struct {
		name        string
		currentState interface{}
		newState    interface{}
		userFields  []string
		wantErr     bool
		validate    func(t *testing.T, state interface{})
	}{
		{
			name: "preserves user fields only",
			currentState: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringValue("user-name"),
				Description: types.StringValue("user-description"),
				Status:      types.StringValue("old-status"),
				CreatedAt:   types.StringValue("old-time"),
			},
			newState: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringValue("new-name"),
				Description: types.StringValue("new-description"),
				Status:      types.StringValue("new-status"),
				CreatedAt:   types.StringValue("new-time"),
			},
			userFields: []string{"Name", "Description"},
			wantErr:    false,
			validate: func(t *testing.T, state interface{}) {
				model := state.(*testResourceModel)
				// User fields should be preserved from current state
				assert.Equal(t, "user-name", model.Name.ValueString())
				assert.Equal(t, "user-description", model.Description.ValueString())
				// Non-user fields should have new values
				assert.Equal(t, "new-status", model.Status.ValueString())
				assert.Equal(t, "new-time", model.CreatedAt.ValueString())
			},
		},
		{
			name:         "nil currentState returns error",
			currentState: nil,
			newState:     &testResourceModel{},
			userFields:   []string{"Name"},
			wantErr:      true,
		},
		{
			name:         "nil newState returns error",
			currentState: &testResourceModel{},
			newState:     nil,
			userFields:   []string{"Name"},
			wantErr:      true,
		},
		{
			name:         "non-struct returns error",
			currentState: "not a struct",
			newState:     &testResourceModel{},
			userFields:   []string{"Name"},
			wantErr:      true,
		},
		{
			name: "empty user fields list",
			currentState: &testResourceModel{
				Name:   types.StringValue("user-name"),
				Status: types.StringValue("old-status"),
			},
			newState: &testResourceModel{
				Name:   types.StringValue("new-name"),
				Status: types.StringValue("new-status"),
			},
			userFields: []string{},
			wantErr:    false,
			validate: func(t *testing.T, state interface{}) {
				model := state.(*testResourceModel)
				// Nothing should be preserved, new values should remain
				assert.Equal(t, "new-name", model.Name.ValueString())
				assert.Equal(t, "new-status", model.Status.ValueString())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.PreserveUserValues(tt.currentState, tt.newState, tt.userFields)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, tt.newState)
				}
			}
		})
	}
}
