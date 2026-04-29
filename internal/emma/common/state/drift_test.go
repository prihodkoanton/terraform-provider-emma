package state

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestNewDriftDetector(t *testing.T) {
	dd := NewDriftDetector()
	assert.NotNil(t, dd)
}

func TestDetectDrift(t *testing.T) {
	dd := NewDriftDetector()

	tests := []struct {
		name       string
		stateValue interface{}
		apiValue   interface{}
		wantDrifts []string
		wantErr    bool
	}{
		{
			name: "no drift when values match",
			stateValue: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringValue("test-name"),
				Description: types.StringValue("test-desc"),
				Status:      types.StringValue("active"),
			},
			apiValue: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringValue("test-name"),
				Description: types.StringValue("test-desc"),
				Status:      types.StringValue("active"),
			},
			wantDrifts: []string{},
			wantErr:    false,
		},
		{
			name: "detects drift in single field",
			stateValue: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringValue("test-name"),
				Description: types.StringValue("test-desc"),
				Status:      types.StringValue("active"),
			},
			apiValue: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringValue("test-name"),
				Description: types.StringValue("test-desc"),
				Status:      types.StringValue("inactive"),
			},
			wantDrifts: []string{"Status"},
			wantErr:    false,
		},
		{
			name: "detects drift in multiple fields",
			stateValue: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringValue("old-name"),
				Description: types.StringValue("old-desc"),
				Status:      types.StringValue("active"),
			},
			apiValue: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringValue("new-name"),
				Description: types.StringValue("new-desc"),
				Status:      types.StringValue("active"),
			},
			wantDrifts: []string{"Name", "Description"},
			wantErr:    false,
		},
		{
			name: "handles null values",
			stateValue: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringNull(),
				Description: types.StringValue("test-desc"),
				Status:      types.StringValue("active"),
			},
			apiValue: &testResourceModel{
				ID:          types.StringValue("123"),
				Name:        types.StringValue("new-name"),
				Description: types.StringValue("test-desc"),
				Status:      types.StringValue("active"),
			},
			wantDrifts: []string{"Name"},
			wantErr:    false,
		},
		{
			name:       "nil stateValue returns error",
			stateValue: nil,
			apiValue:   &testResourceModel{},
			wantDrifts: nil,
			wantErr:    true,
		},
		{
			name:       "nil apiValue returns error",
			stateValue: &testResourceModel{},
			apiValue:   nil,
			wantDrifts: nil,
			wantErr:    true,
		},
		{
			name:       "non-struct returns error",
			stateValue: "not a struct",
			apiValue:   &testResourceModel{},
			wantDrifts: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drifts, err := dd.DetectDrift(tt.stateValue, tt.apiValue)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.wantDrifts, drifts)
			}
		})
	}
}

func TestDetectDrift_NestedObjects(t *testing.T) {
	dd := NewDriftDetector()

	type nestedModel struct {
		ID     types.String `tfsdk:"id"`
		Name   types.String `tfsdk:"name"`
		Config types.Object `tfsdk:"config"`
	}

	// Create nested object attribute types
	configAttrTypes := map[string]attr.Type{
		"enabled": types.BoolType,
		"count":   types.Int64Type,
	}

	// Create matching nested objects
	config1, _ := types.ObjectValue(configAttrTypes, map[string]attr.Value{
		"enabled": types.BoolValue(true),
		"count":   types.Int64Value(5),
	})

	config2, _ := types.ObjectValue(configAttrTypes, map[string]attr.Value{
		"enabled": types.BoolValue(false),
		"count":   types.Int64Value(5),
	})

	tests := []struct {
		name       string
		stateValue *nestedModel
		apiValue   *nestedModel
		wantDrifts []string
	}{
		{
			name: "no drift in nested objects",
			stateValue: &nestedModel{
				ID:     types.StringValue("123"),
				Name:   types.StringValue("test"),
				Config: config1,
			},
			apiValue: &nestedModel{
				ID:     types.StringValue("123"),
				Name:   types.StringValue("test"),
				Config: config1,
			},
			wantDrifts: []string{},
		},
		{
			name: "drift in nested object",
			stateValue: &nestedModel{
				ID:     types.StringValue("123"),
				Name:   types.StringValue("test"),
				Config: config1,
			},
			apiValue: &nestedModel{
				ID:     types.StringValue("123"),
				Name:   types.StringValue("test"),
				Config: config2,
			},
			wantDrifts: []string{"Config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drifts, err := dd.DetectDrift(tt.stateValue, tt.apiValue)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.wantDrifts, drifts)
		})
	}
}
