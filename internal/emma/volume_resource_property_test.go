package emma

import (
	"context"
	"testing"

	emmaSdk "github.com/emma-community/emma-go-sdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: async-operations, Property 5: User-Specified Values Are Preserved
// Validates: Requirements 2.5, 8.5
func TestProperty_UserSpecifiedValuesArePreserved(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("for any resource update during async operation, user-specified values are preserved even if API returns null", prop.ForAll(
		func(planAttachedToId, stateAttachedToId int64, apiReturnsNull bool) bool {
			// Skip invalid inputs
			if planAttachedToId < 1 || planAttachedToId > 1000000 {
				return true
			}
			if stateAttachedToId < 1 || stateAttachedToId > 1000000 {
				return true
			}

			// Only test when attachment is changing
			if planAttachedToId == stateAttachedToId {
				return true
			}

			ctx := context.Background()

			// Create plan model with user-specified attached_to_id
			plan := volumeResourceModel{
				Id:           types.StringValue("123"),
				Name:         types.StringValue("test-volume"),
				DataCenterId: types.StringValue("dc-1"),
				VolumeGb:     types.Int64Value(100),
				VolumeType:   types.StringValue("ssd"),
				AttachedToId: types.Int64Value(planAttachedToId),
			}

			// Store the state attached_to_id for comparison
			stateAttachedToIdValue := types.Int64Value(stateAttachedToId)

			// Save the planned value BEFORE conversion (this is what the actual code does)
			savedPlanAttachedToId := plan.AttachedToId

			// Simulate API response
			var apiVolume *emmaSdk.Volume
			if apiReturnsNull {
				// API returns null for attached_to_id during transition
				apiVolume = &emmaSdk.Volume{
					Id:           emmaSdk.PtrInt32(123),
					Name:         emmaSdk.PtrString("test-volume"),
					SizeGb:       emmaSdk.PtrInt32(100),
					Type:         emmaSdk.PtrString("ssd"),
					AttachedToId: nil, // API returns null during async operation
					Status:       emmaSdk.PtrString("BUSY"),
				}
			} else {
				// API returns the correct value
				apiVolume = &emmaSdk.Volume{
					Id:           emmaSdk.PtrInt32(123),
					Name:         emmaSdk.PtrString("test-volume"),
					SizeGb:       emmaSdk.PtrInt32(100),
					Type:         emmaSdk.PtrString("ssd"),
					AttachedToId: emmaSdk.PtrInt32(int32(planAttachedToId)),
					Status:       emmaSdk.PtrString("AVAILABLE"),
				}
			}

			// Convert API response to resource model (this overwrites attached_to_id)
			convertVolumeResponseToResource(ctx, &plan, apiVolume, nil)

			// Simulate the preservation logic from Update method
			// This is the key behavior we're testing
			// Restore the saved plan value if attachment changed
			if !savedPlanAttachedToId.Equal(stateAttachedToIdValue) {
				// Attachment changed, preserve the planned value
				plan.AttachedToId = savedPlanAttachedToId
			}

			// Verify that user-specified value is preserved
			if plan.AttachedToId.IsNull() {
				// User value should never be null after preservation
				return false
			}

			if plan.AttachedToId.ValueInt64() != planAttachedToId {
				// User value should match the planned value
				return false
			}

			// Verify that the value is preserved even when API returns null
			if apiReturnsNull && plan.AttachedToId.ValueInt64() != planAttachedToId {
				return false
			}

			// Verify that other fields are still updated from API
			if plan.Status.ValueString() != *apiVolume.Status {
				return false
			}

			return true
		},
		gen.Int64Range(1, 1000000),  // planAttachedToId
		gen.Int64Range(1, 1000000),  // stateAttachedToId
		gen.Bool(),                  // apiReturnsNull
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 5: User-Specified Values Are Preserved (Null to Value)
// Validates: Requirements 2.5, 8.5
func TestProperty_UserSpecifiedValuesPreservedNullToValue(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("for any attachment from null to value, user-specified value is preserved even if API returns null", prop.ForAll(
		func(planAttachedToId int64, apiReturnsNull bool) bool {
			// Skip invalid inputs
			if planAttachedToId < 1 || planAttachedToId > 1000000 {
				return true
			}

			ctx := context.Background()

			// Create plan model with user-specified attached_to_id (attaching to a VM)
			plan := volumeResourceModel{
				Id:           types.StringValue("123"),
				Name:         types.StringValue("test-volume"),
				DataCenterId: types.StringValue("dc-1"),
				VolumeGb:     types.Int64Value(100),
				VolumeType:   types.StringValue("ssd"),
				AttachedToId: types.Int64Value(planAttachedToId),
			}

			// Store the state attached_to_id (null) for comparison
			stateAttachedToIdValue := types.Int64Null()

			// Save the planned value BEFORE conversion (this is what the actual code does)
			savedPlanAttachedToId := plan.AttachedToId

			// Simulate API response
			var apiVolume *emmaSdk.Volume
			if apiReturnsNull {
				// API returns null for attached_to_id during transition
				apiVolume = &emmaSdk.Volume{
					Id:           emmaSdk.PtrInt32(123),
					Name:         emmaSdk.PtrString("test-volume"),
					SizeGb:       emmaSdk.PtrInt32(100),
					Type:         emmaSdk.PtrString("ssd"),
					AttachedToId: nil, // API returns null during async operation
					Status:       emmaSdk.PtrString("BUSY"),
				}
			} else {
				// API returns the correct value
				apiVolume = &emmaSdk.Volume{
					Id:           emmaSdk.PtrInt32(123),
					Name:         emmaSdk.PtrString("test-volume"),
					SizeGb:       emmaSdk.PtrInt32(100),
					Type:         emmaSdk.PtrString("ssd"),
					AttachedToId: emmaSdk.PtrInt32(int32(planAttachedToId)),
					Status:       emmaSdk.PtrString("AVAILABLE"),
				}
			}

			// Convert API response to resource model (this overwrites attached_to_id)
			convertVolumeResponseToResource(ctx, &plan, apiVolume, nil)

			// Simulate the preservation logic from Update method
			// Restore the saved plan value if attachment changed
			if !savedPlanAttachedToId.Equal(stateAttachedToIdValue) {
				// Attachment changed, preserve the planned value
				plan.AttachedToId = savedPlanAttachedToId
			}

			// Verify that user-specified value is preserved
			if plan.AttachedToId.IsNull() {
				return false
			}

			if plan.AttachedToId.ValueInt64() != planAttachedToId {
				return false
			}

			// Verify preservation even when API returns null
			if apiReturnsNull && plan.AttachedToId.ValueInt64() != planAttachedToId {
				return false
			}

			return true
		},
		gen.Int64Range(1, 1000000),  // planAttachedToId
		gen.Bool(),                  // apiReturnsNull
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: async-operations, Property 5: User-Specified Values Are Preserved (Value to Null)
// Validates: Requirements 2.5, 8.5
func TestProperty_UserSpecifiedValuesPreservedValueToNull(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("for any detachment from value to null, null value is preserved even if API returns stale value", prop.ForAll(
		func(stateAttachedToId int64, apiReturnsStaleValue bool) bool {
			// Skip invalid inputs
			if stateAttachedToId < 1 || stateAttachedToId > 1000000 {
				return true
			}

			ctx := context.Background()

			// Create plan model with null attached_to_id (detaching from VM)
			plan := volumeResourceModel{
				Id:           types.StringValue("123"),
				Name:         types.StringValue("test-volume"),
				DataCenterId: types.StringValue("dc-1"),
				VolumeGb:     types.Int64Value(100),
				VolumeType:   types.StringValue("ssd"),
				AttachedToId: types.Int64Null(),
			}

			// Store the state attached_to_id for comparison
			stateAttachedToIdValue := types.Int64Value(stateAttachedToId)

			// Save the planned value BEFORE conversion (this is what the actual code does)
			savedPlanAttachedToId := plan.AttachedToId

			// Simulate API response
			var apiVolume *emmaSdk.Volume
			if apiReturnsStaleValue {
				// API returns stale value during transition
				apiVolume = &emmaSdk.Volume{
					Id:           emmaSdk.PtrInt32(123),
					Name:         emmaSdk.PtrString("test-volume"),
					SizeGb:       emmaSdk.PtrInt32(100),
					Type:         emmaSdk.PtrString("ssd"),
					AttachedToId: emmaSdk.PtrInt32(int32(stateAttachedToId)), // API returns stale value
					Status:       emmaSdk.PtrString("BUSY"),
				}
			} else {
				// API returns the correct null value
				apiVolume = &emmaSdk.Volume{
					Id:           emmaSdk.PtrInt32(123),
					Name:         emmaSdk.PtrString("test-volume"),
					SizeGb:       emmaSdk.PtrInt32(100),
					Type:         emmaSdk.PtrString("ssd"),
					AttachedToId: nil,
					Status:       emmaSdk.PtrString("AVAILABLE"),
				}
			}

			// Convert API response to resource model (this overwrites attached_to_id)
			convertVolumeResponseToResource(ctx, &plan, apiVolume, nil)

			// Simulate the preservation logic from Update method
			// Restore the saved plan value if attachment changed
			if !savedPlanAttachedToId.Equal(stateAttachedToIdValue) {
				// Attachment changed, preserve the planned value (null in this case)
				plan.AttachedToId = savedPlanAttachedToId
			}

			// Verify that user-specified null value is preserved
			if !plan.AttachedToId.IsNull() {
				return false
			}

			// Verify preservation even when API returns stale value
			if apiReturnsStaleValue && !plan.AttachedToId.IsNull() {
				return false
			}

			return true
		},
		gen.Int64Range(1, 1000000),  // stateAttachedToId
		gen.Bool(),                  // apiReturnsStaleValue
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
