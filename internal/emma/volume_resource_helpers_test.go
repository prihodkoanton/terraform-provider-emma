package emma

import (
	"testing"

	emmaSdk "github.com/emma-community/emma-go-sdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Test convertResourceToVolumeCreateRequest helper function
func TestConvertResourceToVolumeCreateRequest(t *testing.T) {
	t.Run("converts required fields correctly", func(t *testing.T) {
		data := volumeResourceModel{
			DataCenterId: types.StringValue("dc-123"),
			VolumeGb:     types.Int64Value(100),
			VolumeType:   types.StringValue("ssd"),
		}

		request := convertResourceToVolumeCreateRequest(&data)

		if request.DataCenterId != "dc-123" {
			t.Errorf("Expected DataCenterId 'dc-123', got '%s'", request.DataCenterId)
		}
		if request.VolumeGb != 100 {
			t.Errorf("Expected VolumeGb 100, got %d", request.VolumeGb)
		}
		if request.VolumeType != "ssd" {
			t.Errorf("Expected VolumeType 'ssd', got '%s'", request.VolumeType)
		}
		if request.AttachedToId != nil {
			t.Errorf("Expected AttachedToId to be nil, got %v", *request.AttachedToId)
		}
	})

	t.Run("converts optional attached_to_id when provided", func(t *testing.T) {
		data := volumeResourceModel{
			DataCenterId: types.StringValue("dc-123"),
			VolumeGb:     types.Int64Value(100),
			VolumeType:   types.StringValue("ssd"),
			AttachedToId: types.Int64Value(456),
		}

		request := convertResourceToVolumeCreateRequest(&data)

		if request.AttachedToId == nil {
			t.Error("Expected AttachedToId to be set, got nil")
		} else if *request.AttachedToId != 456 {
			t.Errorf("Expected AttachedToId 456, got %d", *request.AttachedToId)
		}
	})

	t.Run("handles null attached_to_id", func(t *testing.T) {
		data := volumeResourceModel{
			DataCenterId: types.StringValue("dc-123"),
			VolumeGb:     types.Int64Value(100),
			VolumeType:   types.StringValue("ssd"),
			AttachedToId: types.Int64Null(),
		}

		request := convertResourceToVolumeCreateRequest(&data)

		if request.AttachedToId != nil {
			t.Errorf("Expected AttachedToId to be nil for null value, got %v", *request.AttachedToId)
		}
	})

	t.Run("handles type conversions correctly", func(t *testing.T) {
		// Test int64 to int32 conversion
		data := volumeResourceModel{
			DataCenterId: types.StringValue("dc-123"),
			VolumeGb:     types.Int64Value(2147483647), // Max int32 value
			VolumeType:   types.StringValue("ssd"),
			AttachedToId: types.Int64Value(2147483647),
		}

		request := convertResourceToVolumeCreateRequest(&data)

		if request.VolumeGb != 2147483647 {
			t.Errorf("Expected VolumeGb 2147483647, got %d", request.VolumeGb)
		}
		if request.AttachedToId == nil || *request.AttachedToId != 2147483647 {
			t.Errorf("Expected AttachedToId 2147483647, got %v", request.AttachedToId)
		}
	})
}

// Test convertResourceToVolumeEditRequest helper function
func TestConvertResourceToVolumeEditRequest(t *testing.T) {
	t.Run("converts volume size correctly", func(t *testing.T) {
		data := volumeResourceModel{
			VolumeGb: types.Int64Value(200),
		}

		request := convertResourceToVolumeEditRequest(&data)

		if request.Action != "edit" {
			t.Errorf("Expected Action 'edit', got '%s'", request.Action)
		}
		if request.VolumeGb != 200 {
			t.Errorf("Expected VolumeGb 200, got %d", request.VolumeGb)
		}
	})

	t.Run("handles large volume sizes", func(t *testing.T) {
		data := volumeResourceModel{
			VolumeGb: types.Int64Value(10000),
		}

		request := convertResourceToVolumeEditRequest(&data)

		if request.VolumeGb != 10000 {
			t.Errorf("Expected VolumeGb 10000, got %d", request.VolumeGb)
		}
	})

	t.Run("handles type conversion from int64 to int32", func(t *testing.T) {
		data := volumeResourceModel{
			VolumeGb: types.Int64Value(2147483647), // Max int32 value
		}

		request := convertResourceToVolumeEditRequest(&data)

		if request.VolumeGb != 2147483647 {
			t.Errorf("Expected VolumeGb 2147483647, got %d", request.VolumeGb)
		}
	})
}

// Test convertVolumeResponseToResource helper function
func TestConvertVolumeResponseToResource(t *testing.T) {
	t.Run("converts all fields from SDK response", func(t *testing.T) {
		// Create a mock SDK volume response
		volumeId := int32(12345)
		volumeName := "test-volume"
		sizeGb := int32(100)
		volumeType := "ssd"
		isSystem := false
		status := "available"
		projectId := int32(100)
		attachedToId := int32(456)
		createdAt := "2025-02-02T10:00:00Z"

		volume := &emmaSdk.Volume{}
		volume.SetId(volumeId)
		volume.SetName(volumeName)
		volume.SetSizeGb(sizeGb)
		volume.SetType(volumeType)
		volume.SetIsSystem(isSystem)
		volume.SetStatus(status)
		volume.SetProjectId(projectId)
		volume.SetAttachedToId(attachedToId)
		volume.SetCreatedAt(createdAt)

		// Add provider info
		provider := emmaSdk.NewVolumeProvider()
		provider.SetId(1)
		provider.SetName("AWS")
		volume.SetProvider(*provider)

		// Add location info
		location := emmaSdk.NewVolumeLocation()
		location.SetId(10)
		location.SetName("us-east-1")
		continent := "North America"
		region := "US East"
		location.SetContinent(continent)
		location.SetRegion(region)
		volume.SetLocation(*location)

		// Add data center info
		dataCenter := emmaSdk.NewVolumeDataCenter()
		dataCenter.SetId("dc-123")
		dataCenter.SetName("AWS US East 1")
		volume.SetDataCenter(*dataCenter)

		// Convert to resource model
		var data volumeResourceModel
		convertVolumeResponseToResource(nil, &data, volume, nil)

		// Verify all fields are converted correctly
		if data.Id.ValueString() != "12345" {
			t.Errorf("Expected Id '12345', got '%s'", data.Id.ValueString())
		}
		if data.Name.ValueString() != volumeName {
			t.Errorf("Expected Name '%s', got '%s'", volumeName, data.Name.ValueString())
		}
		if data.VolumeGb.ValueInt64() != 100 {
			t.Errorf("Expected VolumeGb 100, got %d", data.VolumeGb.ValueInt64())
		}
		if data.VolumeType.ValueString() != volumeType {
			t.Errorf("Expected VolumeType '%s', got '%s'", volumeType, data.VolumeType.ValueString())
		}
		if data.IsSystem.ValueBool() != isSystem {
			t.Errorf("Expected IsSystem %v, got %v", isSystem, data.IsSystem.ValueBool())
		}
		if data.Status.ValueString() != status {
			t.Errorf("Expected Status '%s', got '%s'", status, data.Status.ValueString())
		}
		if data.ProjectId.ValueInt64() != 100 {
			t.Errorf("Expected ProjectId 100, got %d", data.ProjectId.ValueInt64())
		}
		if data.AttachedToId.ValueInt64() != 456 {
			t.Errorf("Expected AttachedToId 456, got %d", data.AttachedToId.ValueInt64())
		}
		if data.CreatedAt.ValueString() != createdAt {
			t.Errorf("Expected CreatedAt '%s', got '%s'", createdAt, data.CreatedAt.ValueString())
		}
	})

	t.Run("handles null optional fields", func(t *testing.T) {
		// Create a minimal SDK volume response
		volumeId := int32(12345)
		sizeGb := int32(100)
		volumeType := "ssd"

		volume := &emmaSdk.Volume{}
		volume.SetId(volumeId)
		volume.SetSizeGb(sizeGb)
		volume.SetType(volumeType)
		// Don't set optional fields

		// Convert to resource model
		var data volumeResourceModel
		convertVolumeResponseToResource(nil, &data, volume, nil)

		// Verify null fields are handled correctly
		if !data.Name.IsNull() {
			t.Error("Expected Name to be null")
		}
		if !data.AttachedToId.IsNull() {
			t.Error("Expected AttachedToId to be null")
		}
		if !data.CreatedAt.IsNull() {
			t.Error("Expected CreatedAt to be null")
		}
	})

	t.Run("handles type conversions correctly", func(t *testing.T) {
		// Test int32 to int64 conversion
		volumeId := int32(2147483647) // Max int32 value
		sizeGb := int32(2147483647)
		volumeType := "ssd"
		projectId := int32(2147483647)
		attachedToId := int32(2147483647)

		volume := &emmaSdk.Volume{}
		volume.SetId(volumeId)
		volume.SetSizeGb(sizeGb)
		volume.SetType(volumeType)
		volume.SetProjectId(projectId)
		volume.SetAttachedToId(attachedToId)

		// Convert to resource model
		var data volumeResourceModel
		convertVolumeResponseToResource(nil, &data, volume, nil)

		// Verify conversions
		if data.Id.ValueString() != "2147483647" {
			t.Errorf("Expected Id '2147483647', got '%s'", data.Id.ValueString())
		}
		if data.VolumeGb.ValueInt64() != 2147483647 {
			t.Errorf("Expected VolumeGb 2147483647, got %d", data.VolumeGb.ValueInt64())
		}
		if data.ProjectId.ValueInt64() != 2147483647 {
			t.Errorf("Expected ProjectId 2147483647, got %d", data.ProjectId.ValueInt64())
		}
		if data.AttachedToId.ValueInt64() != 2147483647 {
			t.Errorf("Expected AttachedToId 2147483647, got %d", data.AttachedToId.ValueInt64())
		}
	})
}
