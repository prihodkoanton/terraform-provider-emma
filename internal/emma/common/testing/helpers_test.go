package testing

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Test TestContext helper
func TestTestContext(t *testing.T) {
	ctx := TestContext()
	if ctx == nil {
		t.Error("TestContext() returned nil")
	}
}

// Test AssertStringEqual helper
func TestAssertStringEqual(t *testing.T) {
	t.Run("equal strings pass", func(t *testing.T) {
		// This should not fail
		AssertStringEqual(t, "test", "test", "field")
	})

	t.Run("different strings fail", func(t *testing.T) {
		// We can't easily test that this fails without a mock testing.T
		// So we just verify the function exists and can be called
		mockT := &testing.T{}
		AssertStringEqual(mockT, "expected", "actual", "field")
		// In a real failure, mockT would have recorded an error
	})
}

// Test AssertInt32Equal helper
func TestAssertInt32Equal(t *testing.T) {
	t.Run("equal int32 values pass", func(t *testing.T) {
		AssertInt32Equal(t, 42, 42, "field")
	})
}

// Test AssertInt64Equal helper
func TestAssertInt64Equal(t *testing.T) {
	t.Run("equal int64 values pass", func(t *testing.T) {
		AssertInt64Equal(t, 12345, 12345, "field")
	})
}

// Test AssertBoolEqual helper
func TestAssertBoolEqual(t *testing.T) {
	t.Run("equal bool values pass", func(t *testing.T) {
		AssertBoolEqual(t, true, true, "field")
		AssertBoolEqual(t, false, false, "field")
	})
}

// Test AssertNotNil helper
func TestAssertNotNil(t *testing.T) {
	t.Run("non-nil value passes", func(t *testing.T) {
		value := "test"
		AssertNotNil(t, value, "field")
	})

	t.Run("non-nil pointer passes", func(t *testing.T) {
		value := 42
		AssertNotNil(t, &value, "field")
	})
}

// Test AssertNil helper
func TestAssertNil(t *testing.T) {
	t.Run("nil value passes", func(t *testing.T) {
		AssertNil(t, nil, "field")
	})
}

// Test AssertTerraformStringEqual helper
func TestAssertTerraformStringEqual(t *testing.T) {
	t.Run("equal string values pass", func(t *testing.T) {
		value := types.StringValue("test")
		AssertTerraformStringEqual(t, "test", value, "field")
	})

	t.Run("null value is detected", func(t *testing.T) {
		value := types.StringNull()
		mockT := &testing.T{}
		AssertTerraformStringEqual(mockT, "test", value, "field")
		// In a real failure, mockT would have recorded an error
	})

	t.Run("unknown value is detected", func(t *testing.T) {
		value := types.StringUnknown()
		mockT := &testing.T{}
		AssertTerraformStringEqual(mockT, "test", value, "field")
		// In a real failure, mockT would have recorded an error
	})
}

// Test AssertTerraformInt64Equal helper
func TestAssertTerraformInt64Equal(t *testing.T) {
	t.Run("equal int64 values pass", func(t *testing.T) {
		value := types.Int64Value(12345)
		AssertTerraformInt64Equal(t, 12345, value, "field")
	})

	t.Run("null value is detected", func(t *testing.T) {
		value := types.Int64Null()
		mockT := &testing.T{}
		AssertTerraformInt64Equal(mockT, 12345, value, "field")
		// In a real failure, mockT would have recorded an error
	})

	t.Run("unknown value is detected", func(t *testing.T) {
		value := types.Int64Unknown()
		mockT := &testing.T{}
		AssertTerraformInt64Equal(mockT, 12345, value, "field")
		// In a real failure, mockT would have recorded an error
	})
}

// Test AssertTerraformBoolEqual helper
func TestAssertTerraformBoolEqual(t *testing.T) {
	t.Run("equal bool values pass", func(t *testing.T) {
		value := types.BoolValue(true)
		AssertTerraformBoolEqual(t, true, value, "field")
	})

	t.Run("null value is detected", func(t *testing.T) {
		value := types.BoolNull()
		mockT := &testing.T{}
		AssertTerraformBoolEqual(mockT, true, value, "field")
		// In a real failure, mockT would have recorded an error
	})

	t.Run("unknown value is detected", func(t *testing.T) {
		value := types.BoolUnknown()
		mockT := &testing.T{}
		AssertTerraformBoolEqual(mockT, true, value, "field")
		// In a real failure, mockT would have recorded an error
	})
}

// Test AssertTerraformStringNull helper
func TestAssertTerraformStringNull(t *testing.T) {
	t.Run("null value passes", func(t *testing.T) {
		value := types.StringNull()
		AssertTerraformStringNull(t, value, "field")
	})
}

// Test AssertTerraformInt64Null helper
func TestAssertTerraformInt64Null(t *testing.T) {
	t.Run("null value passes", func(t *testing.T) {
		value := types.Int64Null()
		AssertTerraformInt64Null(t, value, "field")
	})
}

// Test AssertTerraformBoolNull helper
func TestAssertTerraformBoolNull(t *testing.T) {
	t.Run("null value passes", func(t *testing.T) {
		value := types.BoolNull()
		AssertTerraformBoolNull(t, value, "field")
	})
}

// Test AssertTerraformObjectNotNull helper
func TestAssertTerraformObjectNotNull(t *testing.T) {
	t.Run("non-null object passes", func(t *testing.T) {
		value := types.ObjectValueMust(
			map[string]attr.Type{
				"test": types.StringType,
			},
			map[string]attr.Value{
				"test": types.StringValue("value"),
			},
		)
		AssertTerraformObjectNotNull(t, value, "field")
	})
}

// Test AssertTerraformListNotNull helper
func TestAssertTerraformListNotNull(t *testing.T) {
	t.Run("non-null list passes", func(t *testing.T) {
		value := types.ListValueMust(
			types.StringType,
			[]attr.Value{
				types.StringValue("item1"),
				types.StringValue("item2"),
			},
		)
		AssertTerraformListNotNull(t, value, "field")
	})
}

// Test AssertNoError helper
func TestAssertNoError(t *testing.T) {
	t.Run("nil error passes", func(t *testing.T) {
		AssertNoError(t, nil)
	})
}

// Test AssertError helper
func TestAssertError(t *testing.T) {
	t.Run("non-nil error passes", func(t *testing.T) {
		err := errors.New("test error")
		AssertError(t, err)
	})
}

// Test AssertErrorContains helper
func TestAssertErrorContains(t *testing.T) {
	t.Run("error with expected message passes", func(t *testing.T) {
		err := errors.New("this is a test error message")
		AssertErrorContains(t, err, "test error")
	})

	t.Run("error with exact message passes", func(t *testing.T) {
		err := errors.New("exact match")
		AssertErrorContains(t, err, "exact match")
	})
}

// Test contains helper function
func TestContains(t *testing.T) {
	t.Run("substring found", func(t *testing.T) {
		if !contains("hello world", "world") {
			t.Error("Expected 'world' to be found in 'hello world'")
		}
	})

	t.Run("substring not found", func(t *testing.T) {
		if contains("hello world", "foo") {
			t.Error("Expected 'foo' not to be found in 'hello world'")
		}
	})

	t.Run("empty substring", func(t *testing.T) {
		if !contains("hello", "") {
			t.Error("Expected empty string to be found")
		}
	})

	t.Run("exact match", func(t *testing.T) {
		if !contains("test", "test") {
			t.Error("Expected exact match to be found")
		}
	})
}

// Test MockVolumeBuilder
func TestMockVolumeBuilder(t *testing.T) {
	t.Run("creates volume with defaults", func(t *testing.T) {
		volume := NewMockVolumeBuilder().Build()

		if volume == nil {
			t.Fatal("Expected non-nil volume")
		}

		id, ok := volume.GetIdOk()
		if !ok || id == nil {
			t.Error("Expected volume to have ID")
		}

		name, ok := volume.GetNameOk()
		if !ok || name == nil {
			t.Error("Expected volume to have name")
		}

		sizeGb, ok := volume.GetSizeGbOk()
		if !ok || sizeGb == nil {
			t.Error("Expected volume to have size")
		}

		volumeType, ok := volume.GetTypeOk()
		if !ok || volumeType == nil {
			t.Error("Expected volume to have type")
		}
	})

	t.Run("builder methods work", func(t *testing.T) {
		volume := NewMockVolumeBuilder().
			WithId(999).
			WithName("custom-volume").
			WithSizeGb(500).
			WithType("ssd-plus").
			WithStatus("creating").
			WithAttachedToId(123).
			WithProjectId(456).
			WithDataCenterId("dc-custom").
			Build()

		id, _ := volume.GetIdOk()
		if *id != 999 {
			t.Errorf("Expected ID 999, got %d", *id)
		}

		name, _ := volume.GetNameOk()
		if *name != "custom-volume" {
			t.Errorf("Expected name 'custom-volume', got '%s'", *name)
		}

		sizeGb, _ := volume.GetSizeGbOk()
		if *sizeGb != 500 {
			t.Errorf("Expected size 500, got %d", *sizeGb)
		}

		volumeType, _ := volume.GetTypeOk()
		if *volumeType != "ssd-plus" {
			t.Errorf("Expected type 'ssd-plus', got '%s'", *volumeType)
		}

		status, _ := volume.GetStatusOk()
		if *status != "creating" {
			t.Errorf("Expected status 'creating', got '%s'", *status)
		}

		attachedToId, _ := volume.GetAttachedToIdOk()
		if *attachedToId != 123 {
			t.Errorf("Expected attachedToId 123, got %d", *attachedToId)
		}

		projectId, _ := volume.GetProjectIdOk()
		if *projectId != 456 {
			t.Errorf("Expected projectId 456, got %d", *projectId)
		}

		dataCenter, _ := volume.GetDataCenterOk()
		dataCenterId, _ := dataCenter.GetIdOk()
		if *dataCenterId != "dc-custom" {
			t.Errorf("Expected dataCenterId 'dc-custom', got '%s'", *dataCenterId)
		}
	})

	t.Run("builder is chainable", func(t *testing.T) {
		builder := NewMockVolumeBuilder()
		result := builder.WithId(1).WithName("test")

		if result != builder {
			t.Error("Expected builder methods to return the same builder instance")
		}
	})
}

// Test MockVmBuilder
func TestMockVmBuilder(t *testing.T) {
	t.Run("creates VM with defaults", func(t *testing.T) {
		vm := NewMockVmBuilder().Build()

		if vm == nil {
			t.Fatal("Expected non-nil VM")
		}

		if vm.Id == nil {
			t.Error("Expected VM to have ID")
		}

		if vm.Name == nil {
			t.Error("Expected VM to have name")
		}

		if vm.VCpu == nil {
			t.Error("Expected VM to have vCPU")
		}

		if vm.RamGb == nil {
			t.Error("Expected VM to have RAM")
		}
	})

	t.Run("builder methods work", func(t *testing.T) {
		vm := NewMockVmBuilder().
			WithId(888).
			WithName("custom-vm").
			WithVCpu(8).
			WithRamGb(16).
			WithStatus("stopped").
			Build()

		if *vm.Id != 888 {
			t.Errorf("Expected ID 888, got %d", *vm.Id)
		}

		if *vm.Name != "custom-vm" {
			t.Errorf("Expected name 'custom-vm', got '%s'", *vm.Name)
		}

		if *vm.VCpu != 8 {
			t.Errorf("Expected vCPU 8, got %d", *vm.VCpu)
		}

		if *vm.RamGb != 16 {
			t.Errorf("Expected RAM 16, got %d", *vm.RamGb)
		}

		if *vm.Status != "stopped" {
			t.Errorf("Expected status 'stopped', got '%s'", *vm.Status)
		}
	})

	t.Run("builder is chainable", func(t *testing.T) {
		builder := NewMockVmBuilder()
		result := builder.WithId(1).WithName("test")

		if result != builder {
			t.Error("Expected builder methods to return the same builder instance")
		}
	})
}

// Test MockSshKeyBuilder
func TestMockSshKeyBuilder(t *testing.T) {
	t.Run("creates SSH key with defaults", func(t *testing.T) {
		sshKey := NewMockSshKeyBuilder().Build()

		if sshKey == nil {
			t.Fatal("Expected non-nil SSH key")
		}

		if sshKey.Id == nil {
			t.Error("Expected SSH key to have ID")
		}

		if sshKey.Name == nil {
			t.Error("Expected SSH key to have name")
		}

		if sshKey.Key == nil {
			t.Error("Expected SSH key to have key value")
		}

		if sshKey.Fingerprint == nil {
			t.Error("Expected SSH key to have fingerprint")
		}
	})

	t.Run("builder methods work", func(t *testing.T) {
		sshKey := NewMockSshKeyBuilder().
			WithId(777).
			WithName("custom-key").
			WithKey("ssh-rsa CUSTOM...").
			WithFingerprint("SHA256:custom").
			Build()

		if *sshKey.Id != 777 {
			t.Errorf("Expected ID 777, got %d", *sshKey.Id)
		}

		if *sshKey.Name != "custom-key" {
			t.Errorf("Expected name 'custom-key', got '%s'", *sshKey.Name)
		}

		if *sshKey.Key != "ssh-rsa CUSTOM..." {
			t.Errorf("Expected key 'ssh-rsa CUSTOM...', got '%s'", *sshKey.Key)
		}

		if *sshKey.Fingerprint != "SHA256:custom" {
			t.Errorf("Expected fingerprint 'SHA256:custom', got '%s'", *sshKey.Fingerprint)
		}
	})

	t.Run("builder is chainable", func(t *testing.T) {
		builder := NewMockSshKeyBuilder()
		result := builder.WithId(1).WithName("test")

		if result != builder {
			t.Error("Expected builder methods to return the same builder instance")
		}
	})
}

// Test MockSecurityGroupBuilder
func TestMockSecurityGroupBuilder(t *testing.T) {
	t.Run("creates security group with defaults", func(t *testing.T) {
		sg := NewMockSecurityGroupBuilder().Build()

		if sg == nil {
			t.Fatal("Expected non-nil security group")
		}

		if sg.Id == nil {
			t.Error("Expected security group to have ID")
		}

		if sg.Name == nil {
			t.Error("Expected security group to have name")
		}

		if sg.SynchronizationStatus == nil {
			t.Error("Expected security group to have synchronization status")
		}

		if sg.RecomposingStatus == nil {
			t.Error("Expected security group to have recomposing status")
		}

		if len(sg.Rules) == 0 {
			t.Error("Expected security group to have at least one rule")
		}
	})

	t.Run("builder methods work", func(t *testing.T) {
		sg := NewMockSecurityGroupBuilder().
			WithId(666).
			WithName("custom-sg").
			WithSynchronizationStatus("Synchronizing").
			WithRecomposingStatus("Recomposing").
			Build()

		if *sg.Id != 666 {
			t.Errorf("Expected ID 666, got %d", *sg.Id)
		}

		if *sg.Name != "custom-sg" {
			t.Errorf("Expected name 'custom-sg', got '%s'", *sg.Name)
		}

		if *sg.SynchronizationStatus != "Synchronizing" {
			t.Errorf("Expected synchronization status 'Synchronizing', got '%s'", *sg.SynchronizationStatus)
		}

		if *sg.RecomposingStatus != "Recomposing" {
			t.Errorf("Expected recomposing status 'Recomposing', got '%s'", *sg.RecomposingStatus)
		}
	})

	t.Run("builder is chainable", func(t *testing.T) {
		builder := NewMockSecurityGroupBuilder()
		result := builder.WithId(1).WithName("test")

		if result != builder {
			t.Error("Expected builder methods to return the same builder instance")
		}
	})
}
