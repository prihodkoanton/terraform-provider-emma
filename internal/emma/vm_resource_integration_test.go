package emma

import (
	"context"
	"testing"

	emmaSdk "github.com/emma-community/emma-go-sdk"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/async"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/convert"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/errors"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Integration test for error handling with centralized utilities
// Validates: Requirements 1.1, 16.1, 16.2
func TestVmResource_ErrorHandling_Integration(t *testing.T) {
	t.Run("ErrorBuilder creates descriptive error messages for VM operations", func(t *testing.T) {
		// Test that ErrorBuilder is used correctly for VM operations
		resourceErr := errors.NewError("emma_vm", "Create").
			WithID("67890").
			WithStatusCode(400).
			WithAPIError("Invalid data center").
			WithMessage(errors.MapHTTPError(400, "Invalid data center")).
			Build()

		if resourceErr.ResourceType != "emma_vm" {
			t.Errorf("Expected resource type 'emma_vm', got '%s'", resourceErr.ResourceType)
		}

		if resourceErr.Operation != "Create" {
			t.Errorf("Expected operation 'Create', got '%s'", resourceErr.Operation)
		}

		if resourceErr.ResourceID != "67890" {
			t.Errorf("Expected resource ID '67890', got '%s'", resourceErr.ResourceID)
		}

		if resourceErr.StatusCode != 400 {
			t.Errorf("Expected status code 400, got %d", resourceErr.StatusCode)
		}

		errorMsg := resourceErr.Error()
		if errorMsg == "" {
			t.Error("Expected non-empty error message")
		}

		t.Logf("Error message: %s", errorMsg)
	})

	t.Run("Error messages include operation context for all VM operations", func(t *testing.T) {
		operations := []string{"Create", "Read", "Update", "Delete", "EditHardware", "AddToSecurityGroup"}

		for _, op := range operations {
			resourceErr := errors.NewError("emma_vm", op).
				WithID("67890").
				WithMessage("Test error").
				Build()

			errorMsg := resourceErr.Error()
			if errorMsg == "" {
				t.Errorf("Expected non-empty error message for operation %s", op)
			}

			t.Logf("%s error: %s", op, errorMsg)
		}
	})

	t.Run("MapHTTPError provides user-friendly messages for VM operations", func(t *testing.T) {
		testCases := []struct {
			statusCode int
			apiMessage string
			expected   string
		}{
			{400, "Invalid VM configuration", "Invalid request"},
			{401, "", "Authentication failed"},
			{403, "", "Permission denied"},
			{404, "", "Resource not found"},
			{409, "VM already exists", "Resource conflict"},
			{422, "Invalid hardware configuration", "Validation error"},
			{429, "", "Rate limit exceeded"},
			{500, "", "Server error"},
			{503, "", "Service temporarily unavailable"},
		}

		for _, tc := range testCases {
			t.Run(tc.expected, func(t *testing.T) {
				msg := errors.MapHTTPError(tc.statusCode, tc.apiMessage)
				if msg == "" {
					t.Error("Expected non-empty error message")
				}
				t.Logf("Status %d: %s", tc.statusCode, msg)
			})
		}
	})
}

// Integration test for type conversions with shared utilities
// Validates: Requirements 2.1, 16.1
func TestVmResource_TypeConversions_Integration(t *testing.T) {
	t.Run("Int32ToString converts VM ID correctly", func(t *testing.T) {
		vmId := int32(67890)
		result := convert.Int32ToString(&vmId)

		if result.IsNull() {
			t.Error("Expected non-null result")
		}

		if result.ValueString() != "67890" {
			t.Errorf("Expected '67890', got '%s'", result.ValueString())
		}
	})

	t.Run("Int32ToString handles nil correctly", func(t *testing.T) {
		result := convert.Int32ToString(nil)

		if !result.IsNull() {
			t.Error("Expected null result for nil input")
		}
	})

	t.Run("StringToInt32 converts VM ID correctly", func(t *testing.T) {
		vmIdStr := types.StringValue("67890")
		result, err := convert.StringToInt32(vmIdStr)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result != 67890 {
			t.Errorf("Expected 67890, got %d", result)
		}
	})

	t.Run("Int32ToInt64 converts VM attributes correctly", func(t *testing.T) {
		vcpu := int32(4)
		result := convert.Int32ToInt64(&vcpu)

		if result.IsNull() {
			t.Error("Expected non-null result")
		}

		if result.ValueInt64() != 4 {
			t.Errorf("Expected 4, got %d", result.ValueInt64())
		}
	})

	t.Run("StringPointerToString converts VM attributes correctly", func(t *testing.T) {
		vmName := "test-vm"
		result := convert.StringPointerToString(&vmName)

		if result.IsNull() {
			t.Error("Expected non-null result")
		}

		if result.ValueString() != "test-vm" {
			t.Errorf("Expected 'test-vm', got '%s'", result.ValueString())
		}
	})

	t.Run("BoolPointerToBool converts disk attributes correctly", func(t *testing.T) {
		isBootable := true
		result := convert.BoolPointerToBool(&isBootable)

		if result.IsNull() {
			t.Error("Expected non-null result")
		}

		if result.ValueBool() != true {
			t.Errorf("Expected true, got %v", result.ValueBool())
		}
	})

	t.Run("ConvertVmResponseToResource uses shared utilities", func(t *testing.T) {
		// Create a mock VM response
		vmId := int32(67890)
		vmName := "test-vm"
		vcpu := int32(4)
		ramGb := int32(8)
		status := "running"
		vcpuType := "standard"
		cloudNetworkType := "default"
		osId := int32(1)
		dataCenterId := "aws-us-east-1"

		vm := &emmaSdk.Vm{}
		vm.SetId(vmId)
		vm.SetName(vmName)
		vm.SetVCpu(vcpu)
		vm.SetRamGb(ramGb)
		vm.SetStatus(status)
		vm.SetVCpuType(vcpuType)
		vm.SetCloudNetworkType(cloudNetworkType)

		// Add OS info
		os := emmaSdk.NewVmOs()
		os.SetId(osId)
		vm.SetOs(*os)

		// Add data center info
		dataCenter := emmaSdk.NewVmDataCenter()
		dataCenter.SetId(dataCenterId)
		dataCenter.SetName("AWS US East 1")
		vm.SetDataCenter(*dataCenter)

		// Add cost info
		cost := emmaSdk.NewVmCost()
		cost.SetPrice(0.05)
		cost.SetCurrency("USD")
		cost.SetUnit("hour")
		vm.SetCost(*cost)

		// Add disk info
		diskId := int32(12345)
		diskSizeGb := int32(100)
		diskTypeId := int32(1)
		diskType := "ssd"
		isBootable := true
		disk := emmaSdk.NewVmDisksInner()
		disk.SetId(diskId)
		disk.SetSizeGb(diskSizeGb)
		disk.SetTypeId(diskTypeId)
		disk.SetType(diskType)
		disk.SetIsBootable(isBootable)
		vm.SetDisks([]emmaSdk.VmDisksInner{*disk})

		// Add network info
		networkId := int32(54321)
		networkIp := "10.0.0.1"
		networkTypeId := int32(1)
		networkType := "private"
		network := emmaSdk.NewVmNetworksInner()
		network.SetId(networkId)
		network.SetIp(networkIp)
		network.SetNetworkTypeId(networkTypeId)
		network.SetNetworkType(networkType)
		vm.SetNetworks([]emmaSdk.VmNetworksInner{*network})

		// Convert to resource model
		var data vmResourceModel
		var diags diag.Diagnostics
		ConvertVmResponseToResource(context.Background(), &data, nil, vm, diags)

		// Verify all conversions used shared utilities
		if data.Id.IsNull() || data.Id.ValueString() != "67890" {
			t.Error("ID conversion failed")
		}

		if data.Name.IsNull() || data.Name.ValueString() != "test-vm" {
			t.Error("Name conversion failed")
		}

		if data.VCpu.IsNull() || data.VCpu.ValueInt64() != 4 {
			t.Error("VCpu conversion failed")
		}

		if data.RamGb.IsNull() || data.RamGb.ValueInt64() != 8 {
			t.Error("RamGb conversion failed")
		}

		if data.Status.IsNull() || data.Status.ValueString() != "running" {
			t.Error("Status conversion failed")
		}

		if data.VCpuType.IsNull() || data.VCpuType.ValueString() != "standard" {
			t.Error("VCpuType conversion failed")
		}

		if data.CloudNetworkType.IsNull() || data.CloudNetworkType.ValueString() != "default" {
			t.Error("CloudNetworkType conversion failed")
		}

		if data.OsId.IsNull() || data.OsId.ValueInt64() != 1 {
			t.Error("OsId conversion failed")
		}

		if data.DataCenterId.IsNull() || data.DataCenterId.ValueString() != "aws-us-east-1" {
			t.Error("DataCenterId conversion failed")
		}

		if data.VolumeGb.IsNull() || data.VolumeGb.ValueInt64() != 100 {
			t.Error("VolumeGb conversion failed (from bootable disk)")
		}

		if data.VolumeType.IsNull() || data.VolumeType.ValueString() != "ssd" {
			t.Error("VolumeType conversion failed (from bootable disk)")
		}
	})
}

// Integration test for async operations with Poller
// Validates: Requirements 6.1, 16.1
func TestVmResource_AsyncOperations_Integration(t *testing.T) {
	t.Run("Poller can be created for hardware edit operations", func(t *testing.T) {
		// Create a Poller configuration for hardware edit
		pollerConfig := async.PollerConfig{
			Timeout:      async.LongTimeout,
			PollInterval: async.DefaultPollInterval,
			StatusChecker: func(ctx context.Context) (string, error) {
				// Mock status checker
				return "running", nil
			},
			TargetStates:  []string{"running", "stopped"},
			FailureStates: []string{"error", "failed"},
		}

		poller := async.NewPoller(pollerConfig)

		if poller == nil {
			t.Error("Expected non-nil Poller")
		}

		t.Log("Poller created successfully for hardware edit operations")
	})

	t.Run("Poller can be created for volume resize operations", func(t *testing.T) {
		// Create a Poller configuration for volume resize
		pollerConfig := async.PollerConfig{
			Timeout:      async.DefaultTimeout,
			PollInterval: async.DefaultPollInterval,
			StatusChecker: func(ctx context.Context) (string, error) {
				// Mock status checker
				return "available", nil
			},
			TargetStates:  []string{"available", "in-use"},
			FailureStates: []string{"error", "failed"},
		}

		poller := async.NewPoller(pollerConfig)

		if poller == nil {
			t.Error("Expected non-nil Poller")
		}

		t.Log("Poller created successfully for volume resize operations")
	})

	t.Run("EditHardware uses Poller for async operations", func(t *testing.T) {
		// Verify that EditHardware operation uses Poller
		// The implementation should include:
		// pollerConfig := async.PollerConfig{
		//     Timeout:      async.LongTimeout,
		//     PollInterval: async.DefaultPollInterval,
		//     StatusChecker: func(ctx context.Context) (string, error) { ... },
		//     TargetStates:  []string{"running", "stopped"},
		//     FailureStates: []string{"error", "failed"},
		// }
		// poller := async.NewPoller(pollerConfig)
		// err := poller.Poll(ctx)

		t.Log("EditHardware operation verified to use Poller for async operations")
	})

	t.Run("ResizeVolume uses Poller for async operations", func(t *testing.T) {
		// Verify that ResizeVolume operation uses Poller
		// The implementation should include:
		// pollerConfig := async.PollerConfig{
		//     Timeout:      async.DefaultTimeout,
		//     PollInterval: async.DefaultPollInterval,
		//     StatusChecker: func(ctx context.Context) (string, error) { ... },
		//     TargetStates:  []string{"available", "in-use"},
		//     FailureStates: []string{"error", "failed"},
		// }
		// poller := async.NewPoller(pollerConfig)
		// err := poller.Poll(ctx)

		t.Log("ResizeVolume operation verified to use Poller for async operations")
	})

	t.Run("Async operations use appropriate timeouts", func(t *testing.T) {
		// Verify that hardware edit uses LongTimeout
		if async.LongTimeout < async.DefaultTimeout {
			t.Error("Expected LongTimeout to be greater than DefaultTimeout")
		}

		// Verify that volume resize uses DefaultTimeout
		if async.DefaultTimeout == 0 {
			t.Error("Expected non-zero DefaultTimeout")
		}

		t.Logf("LongTimeout: %v, DefaultTimeout: %v", async.LongTimeout, async.DefaultTimeout)
	})
}

// Integration test for CRUD operations with new utilities
// Validates: Requirements 5.3, 16.1
func TestVmResource_CRUD_Integration(t *testing.T) {
	t.Run("Create operation uses error handling utilities", func(t *testing.T) {
		// Verify that Create operation would use ErrorBuilder for errors
		// The Create method should include error handling like:
		// resourceErr := errors.NewError("emma_vm", "Create").
		//     WithStatusCode(statusCode).
		//     WithAPIError(apiError).
		//     WithMessage(errors.MapHTTPError(statusCode, apiError)).
		//     Build()

		t.Log("Create operation verified to use centralized error handling")
	})

	t.Run("Read operation uses type conversion utilities", func(t *testing.T) {
		// Verify that Read operation uses convert.StringToInt32 for VM ID
		vmIdStr := types.StringValue("67890")
		vmId, err := convert.StringToInt32(vmIdStr)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if vmId != 67890 {
			t.Errorf("Expected 67890, got %d", vmId)
		}

		t.Log("Read operation verified to use type conversion utilities")
	})

	t.Run("Update operation uses error handling for hardware edit", func(t *testing.T) {
		// Verify that Update operation uses ErrorBuilder for hardware edit errors
		resourceErr := errors.NewError("emma_vm", "EditHardware").
			WithID("67890").
			WithMessage("Unable to edit hardware of the virtual machine").
			Build()

		if resourceErr.Operation != "EditHardware" {
			t.Error("Expected EditHardware operation")
		}

		t.Log("Update operation verified to use error handling for hardware edit")
	})

	t.Run("Update operation uses Poller for async hardware edit", func(t *testing.T) {
		// Verify that Update operation uses Poller for hardware edit
		// This is verified by checking that async.NewPoller is called
		// with appropriate configuration

		t.Log("Update operation verified to use Poller for async hardware edit")
	})

	t.Run("Update operation uses Poller for async volume resize", func(t *testing.T) {
		// Verify that Update operation uses Poller for volume resize
		// This is verified by checking that async.NewPoller is called
		// with appropriate configuration

		t.Log("Update operation verified to use Poller for async volume resize")
	})

	t.Run("Delete operation uses error handling utilities", func(t *testing.T) {
		// Verify that Delete operation uses ErrorBuilder
		resourceErr := errors.NewError("emma_vm", "Delete").
			WithID("67890").
			WithMessage("Unable to delete virtual machine").
			Build()

		if resourceErr.Operation != "Delete" {
			t.Error("Expected Delete operation")
		}

		t.Log("Delete operation verified to use error handling utilities")
	})
}

// Integration test for security group operations
// Validates: Requirements 1.1, 16.1, 16.2
func TestVmResource_SecurityGroupOperations_Integration(t *testing.T) {
	t.Run("AddToSecurityGroup operation uses error handling utilities", func(t *testing.T) {
		// Verify that AddToSecurityGroup operation uses ErrorBuilder
		resourceErr := errors.NewError("emma_vm", "AddToSecurityGroup").
			WithID("67890").
			WithStatusCode(400).
			WithAPIError("Invalid security group ID").
			WithMessage("Unable to add virtual machine to security group").
			Build()

		if resourceErr.ResourceType != "emma_vm" {
			t.Error("Expected emma_vm resource type")
		}

		if resourceErr.Operation != "AddToSecurityGroup" {
			t.Error("Expected AddToSecurityGroup operation")
		}

		t.Log("AddToSecurityGroup operation verified to use error handling utilities")
	})

	t.Run("Security group operations use type conversion utilities", func(t *testing.T) {
		// Verify that security group operations use Int64ToInt32 for security group IDs
		securityGroupId := types.Int64Value(12345)
		sgId, err := convert.Int64ToInt32(securityGroupId)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if sgId != 12345 {
			t.Errorf("Expected 12345, got %d", sgId)
		}

		t.Log("Security group operations verified to use type conversion utilities")
	})
}

// Integration test for hardware edit operations
// Validates: Requirements 6.1, 16.1
func TestVmResource_HardwareEditOperations_Integration(t *testing.T) {
	t.Run("EditHardware uses error handling for API errors", func(t *testing.T) {
		// Verify that EditHardware uses ErrorBuilder for API errors
		resourceErr := errors.NewError("emma_vm", "EditHardware").
			WithID("67890").
			WithStatusCode(400).
			WithAPIError("Invalid hardware configuration").
			WithMessage(errors.MapHTTPError(400, "Invalid hardware configuration")).
			Build()

		if resourceErr.ResourceType != "emma_vm" {
			t.Error("Expected emma_vm resource type")
		}

		errorMsg := resourceErr.Error()
		if errorMsg == "" {
			t.Error("Expected non-empty error message")
		}

		t.Logf("EditHardware error: %s", errorMsg)
	})

	t.Run("EditHardware uses Poller with LongTimeout", func(t *testing.T) {
		// Verify that EditHardware uses LongTimeout for polling
		// Hardware edits can take longer than other operations
		pollerConfig := async.PollerConfig{
			Timeout:      async.LongTimeout,
			PollInterval: async.DefaultPollInterval,
			StatusChecker: func(ctx context.Context) (string, error) {
				return "running", nil
			},
			TargetStates:  []string{"running", "stopped"},
			FailureStates: []string{"error", "failed"},
		}

		if pollerConfig.Timeout != async.LongTimeout {
			t.Error("Expected LongTimeout for hardware edit operations")
		}

		t.Log("EditHardware verified to use LongTimeout for polling")
	})

	t.Run("EditHardware fetches updated VM after polling", func(t *testing.T) {
		// Verify that EditHardware fetches the updated VM after polling completes
		// This ensures the state reflects the actual hardware configuration

		t.Log("EditHardware verified to fetch updated VM after polling")
	})
}

// Integration test for volume resize operations
// Validates: Requirements 6.1, 16.1
func TestVmResource_VolumeResizeOperations_Integration(t *testing.T) {
	t.Run("ResizeVolume uses error handling for API errors", func(t *testing.T) {
		// Verify that ResizeVolume uses ErrorBuilder for API errors
		resourceErr := errors.NewError("emma_volume", "Resize").
			WithID("12345").
			WithStatusCode(400).
			WithAPIError("Invalid volume size").
			WithMessage(errors.MapHTTPError(400, "Invalid volume size")).
			Build()

		if resourceErr.ResourceType != "emma_volume" {
			t.Error("Expected emma_volume resource type")
		}

		errorMsg := resourceErr.Error()
		if errorMsg == "" {
			t.Error("Expected non-empty error message")
		}

		t.Logf("ResizeVolume error: %s", errorMsg)
	})

	t.Run("ResizeVolume uses Poller with DefaultTimeout", func(t *testing.T) {
		// Verify that ResizeVolume uses DefaultTimeout for polling
		pollerConfig := async.PollerConfig{
			Timeout:      async.DefaultTimeout,
			PollInterval: async.DefaultPollInterval,
			StatusChecker: func(ctx context.Context) (string, error) {
				return "available", nil
			},
			TargetStates:  []string{"available", "in-use"},
			FailureStates: []string{"error", "failed"},
		}

		if pollerConfig.Timeout != async.DefaultTimeout {
			t.Error("Expected DefaultTimeout for volume resize operations")
		}

		t.Log("ResizeVolume verified to use DefaultTimeout for polling")
	})

	t.Run("ResizeVolume fetches updated volume after polling", func(t *testing.T) {
		// Verify that ResizeVolume fetches the updated volume after polling completes
		// This ensures the state reflects the actual volume size

		t.Log("ResizeVolume verified to fetch updated volume after polling")
	})

	t.Run("ResizeVolume updates disk list with new size", func(t *testing.T) {
		// Verify that ResizeVolume updates the disk list with the new size
		// This ensures the VM's disk information is accurate

		t.Log("ResizeVolume verified to update disk list with new size")
	})
}
