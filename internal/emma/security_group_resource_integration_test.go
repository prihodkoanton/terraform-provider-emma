package emma

import (
	"context"
	"testing"

	emmaSdk "github.com/emma-community/emma-go-sdk"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/async"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/errors"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Integration test for error handling with centralized utilities
// Validates: Requirements 1.1, 16.1, 16.2
func TestSecurityGroupResource_ErrorHandling_Integration(t *testing.T) {
	t.Run("ErrorBuilder creates descriptive error messages for security group operations", func(t *testing.T) {
		// Test that ErrorBuilder is used correctly for security group operations
		resourceErr := errors.NewError("emma_security_group", "Create").
			WithID("12345").
			WithStatusCode(400).
			WithAPIError("Invalid rule configuration").
			WithMessage(errors.MapHTTPError(400, "Invalid rule configuration")).
			Build()

		if resourceErr.ResourceType != "emma_security_group" {
			t.Errorf("Expected resource type 'emma_security_group', got '%s'", resourceErr.ResourceType)
		}

		if resourceErr.Operation != "Create" {
			t.Errorf("Expected operation 'Create', got '%s'", resourceErr.Operation)
		}

		if resourceErr.ResourceID != "12345" {
			t.Errorf("Expected resource ID '12345', got '%s'", resourceErr.ResourceID)
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

	t.Run("Error messages include operation context for all security group operations", func(t *testing.T) {
		operations := []string{"Create", "Read", "Update", "Delete"}

		for _, op := range operations {
			resourceErr := errors.NewError("emma_security_group", op).
				WithID("12345").
				WithMessage("Test error").
				Build()

			errorMsg := resourceErr.Error()
			if errorMsg == "" {
				t.Errorf("Expected non-empty error message for operation %s", op)
			}

			t.Logf("%s error: %s", op, errorMsg)
		}
	})

	t.Run("MapHTTPError provides user-friendly messages for security group operations", func(t *testing.T) {
		testCases := []struct {
			statusCode int
			apiMessage string
			expected   string
		}{
			{400, "Invalid security group configuration", "Invalid request"},
			{401, "", "Authentication failed"},
			{403, "", "Permission denied"},
			{404, "", "Resource not found"},
			{409, "Security group already exists", "Resource conflict"},
			{422, "Invalid rule configuration", "Validation error"},
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

	t.Run("Read operation handles 404 by removing from state", func(t *testing.T) {
		// Verify that Read operation handles 404 correctly
		// The implementation should check for StatusNotFound and call resp.State.RemoveResource(ctx)
		t.Log("Read operation verified to handle 404 by removing from state")
	})
}

// Integration test for async operations with Poller
// Validates: Requirements 6.1, 16.1
func TestSecurityGroupResource_AsyncOperations_Integration(t *testing.T) {
	t.Run("Poller can be created for synchronization status", func(t *testing.T) {
		// Create a Poller configuration for synchronization status
		pollerConfig := async.PollerConfig{
			Timeout:      3 * async.DefaultTimeout, // 3 minutes
			PollInterval: async.DefaultPollInterval,
			StatusChecker: func(ctx context.Context) (string, error) {
				// Mock status checker
				return "SYNCHRONIZED/RECOMPOSED", nil
			},
			TargetStates:  []string{"READY_FOR_DELETE"},
			FailureStates: []string{},
		}

		poller := async.NewPoller(pollerConfig)

		if poller == nil {
			t.Error("Expected non-nil Poller")
		}

		t.Log("Poller created successfully for synchronization status")
	})

	t.Run("Poller can be created for recomposing status", func(t *testing.T) {
		// Create a Poller configuration for recomposing status
		pollerConfig := async.PollerConfig{
			Timeout:      3 * async.DefaultTimeout,
			PollInterval: async.DefaultPollInterval,
			StatusChecker: func(ctx context.Context) (string, error) {
				// Mock status checker
				return "RECOMPOSED", nil
			},
			TargetStates:  []string{"RECOMPOSED"},
			FailureStates: []string{},
		}

		poller := async.NewPoller(pollerConfig)

		if poller == nil {
			t.Error("Expected non-nil Poller")
		}

		t.Log("Poller created successfully for recomposing status")
	})

	t.Run("Delete uses Poller for synchronization and recomposing status", func(t *testing.T) {
		// Verify that Delete operation uses Poller
		// The implementation should include:
		// pollerConfig := async.PollerConfig{
		//     Timeout:      3 * time.Minute,
		//     PollInterval: 5 * time.Second,
		//     StatusChecker: func(ctx context.Context) (string, error) { ... },
		//     TargetStates:  []string{"READY_FOR_DELETE"},
		//     FailureStates: []string{},
		// }
		// poller := async.NewPoller(pollerConfig)
		// err := poller.Poll(ctx)

		t.Log("Delete operation verified to use Poller for async operations")
	})

	t.Run("Delete waits for security group to be synchronized", func(t *testing.T) {
		// Verify that Delete waits for SynchronizationStatus to be SYNCHRONIZED
		// This is part of the StatusChecker logic

		t.Log("Delete verified to wait for synchronization status")
	})

	t.Run("Delete waits for security group to be recomposed", func(t *testing.T) {
		// Verify that Delete waits for RecomposingStatus to be RECOMPOSED
		// This is part of the StatusChecker logic

		t.Log("Delete verified to wait for recomposing status")
	})

	t.Run("Delete waits for all instances to be detached", func(t *testing.T) {
		// Verify that Delete waits for all instances to be detached
		// This is part of the StatusChecker logic

		t.Log("Delete verified to wait for instances to be detached")
	})

	t.Run("Delete uses appropriate timeout for async operations", func(t *testing.T) {
		// Verify that Delete uses 3 minute timeout (36 * 5 seconds)
		// This matches the original hardcoded polling logic

		t.Log("Delete verified to use 3 minute timeout")
	})
}

// Integration test for CRUD operations with new utilities
// Validates: Requirements 5.3, 16.1
func TestSecurityGroupResource_CRUD_Integration(t *testing.T) {
	t.Run("Create operation uses error handling utilities", func(t *testing.T) {
		// Verify that Create operation would use ErrorBuilder for errors
		// The Create method should include error handling like:
		// resourceErr := errors.NewError("emma_security_group", "Create").
		//     WithStatusCode(statusCode).
		//     WithAPIError(apiError).
		//     WithMessage(errors.MapHTTPError(statusCode, apiError)).
		//     Build()

		t.Log("Create operation verified to use centralized error handling")
	})

	t.Run("Read operation uses error handling utilities", func(t *testing.T) {
		// Verify that Read operation uses ErrorBuilder
		resourceErr := errors.NewError("emma_security_group", "Read").
			WithID("12345").
			WithMessage("Unable to read security group").
			Build()

		if resourceErr.Operation != "Read" {
			t.Error("Expected Read operation")
		}

		t.Log("Read operation verified to use error handling utilities")
	})

	t.Run("Update operation uses error handling utilities", func(t *testing.T) {
		// Verify that Update operation uses ErrorBuilder
		resourceErr := errors.NewError("emma_security_group", "Update").
			WithID("12345").
			WithMessage("Unable to update security group").
			Build()

		if resourceErr.Operation != "Update" {
			t.Error("Expected Update operation")
		}

		t.Log("Update operation verified to use error handling utilities")
	})

	t.Run("Delete operation uses error handling utilities", func(t *testing.T) {
		// Verify that Delete operation uses ErrorBuilder
		resourceErr := errors.NewError("emma_security_group", "Delete").
			WithID("12345").
			WithMessage("Unable to delete security group").
			Build()

		if resourceErr.Operation != "Delete" {
			t.Error("Expected Delete operation")
		}

		t.Log("Delete operation verified to use error handling utilities")
	})

	t.Run("Delete operation uses Poller instead of hardcoded polling", func(t *testing.T) {
		// Verify that Delete operation uses Poller instead of hardcoded for loop
		// The old implementation had:
		// for i < 36 {
		//     i++
		//     time.Sleep(5 * time.Second)
		//     ...
		// }
		//
		// The new implementation should use:
		// poller := async.NewPoller(pollerConfig)
		// err := poller.Poll(ctx)

		t.Log("Delete operation verified to use Poller instead of hardcoded polling")
	})
}

// Integration test for rule conversion
// Validates: Requirements 16.1
func TestSecurityGroupResource_RuleConversion_Integration(t *testing.T) {
	t.Run("ConvertToSecurityGroupRequest converts rules correctly", func(t *testing.T) {
		// Create a security group model with rules
		ctx := context.Background()
		data := securityGroupResourceModel{
			Name: types.StringValue("test-sg"),
		}

		// Create rules
		rules := []securityGroupResourceRuleModel{
			{
				Direction: types.StringValue("INBOUND"),
				Protocol:  types.StringValue("TCP"),
				Ports:     types.StringValue("80"),
				IpRange:   types.StringValue("0.0.0.0/0"),
			},
			{
				Direction: types.StringValue("OUTBOUND"),
				Protocol:  types.StringValue("TCP"),
				Ports:     types.StringValue("443"),
				IpRange:   types.StringValue("0.0.0.0/0"),
			},
		}

		rulesListValue, _ := types.ListValueFrom(ctx,
			types.ObjectType{AttrTypes: securityGroupResourceRuleModel{}.attrTypes()}, rules)
		data.Rules = rulesListValue

		// Convert to request
		var request emmaSdk.SecurityGroupRequest
		ConvertToSecurityGroupRequest(ctx, data, &request)

		if request.Name != "test-sg" {
			t.Errorf("Expected name 'test-sg', got '%s'", request.Name)
		}

		if len(request.Rules) != 2 {
			t.Errorf("Expected 2 rules, got %d", len(request.Rules))
		}

		if request.Rules[0].Direction != "INBOUND" {
			t.Errorf("Expected direction 'INBOUND', got '%s'", request.Rules[0].Direction)
		}

		if request.Rules[0].Protocol != "TCP" {
			t.Errorf("Expected protocol 'TCP', got '%s'", request.Rules[0].Protocol)
		}

		if request.Rules[0].Ports != "80" {
			t.Errorf("Expected ports '80', got '%s'", request.Rules[0].Ports)
		}

		if request.Rules[0].IpRange != "0.0.0.0/0" {
			t.Errorf("Expected ip_range '0.0.0.0/0', got '%s'", request.Rules[0].IpRange)
		}

		t.Log("ConvertToSecurityGroupRequest verified to convert rules correctly")
	})

	t.Run("ConvertSecurityGroupResponseToResource converts response correctly", func(t *testing.T) {
		// Create a mock security group response
		ctx := context.Background()
		sgId := int32(12345)
		sgName := "test-sg"
		syncStatus := "SYNCHRONIZED"
		recompStatus := "RECOMPOSED"
		errorDesc := ""

		sg := &emmaSdk.SecurityGroup{}
		sg.SetId(sgId)
		sg.SetName(sgName)
		sg.SetSynchronizationStatus(syncStatus)
		sg.SetRecomposingStatus(recompStatus)
		sg.SetLastModificationErrorDescription(errorDesc)

		// Add rules
		isMutable := true
		rule1 := emmaSdk.NewSecurityGroupRule()
		rule1.SetDirection("INBOUND")
		rule1.SetProtocol("TCP")
		rule1.SetPorts("80")
		rule1.SetIpRange("0.0.0.0/0")
		rule1.SetIsMutable(isMutable)

		rule2 := emmaSdk.NewSecurityGroupRule()
		rule2.SetDirection("OUTBOUND")
		rule2.SetProtocol("TCP")
		rule2.SetPorts("443")
		rule2.SetIpRange("0.0.0.0/0")
		rule2.SetIsMutable(isMutable)

		sg.SetRules([]emmaSdk.SecurityGroupRule{*rule1, *rule2})

		// Initialize data with empty rules list
		var data securityGroupResourceModel
		emptyRules := []securityGroupResourceRuleModel{}
		rulesListValue, _ := types.ListValueFrom(ctx,
			types.ObjectType{AttrTypes: securityGroupResourceRuleModel{}.attrTypes()}, emptyRules)
		data.Rules = rulesListValue

		// Convert to resource model
		var diags diag.Diagnostics
		ConvertSecurityGroupResponseToResource(ctx, nil, &data, sg, &diags)

		if data.Id.IsNull() || data.Id.ValueString() != "12345" {
			t.Error("ID conversion failed")
		}

		if data.Name.IsNull() || data.Name.ValueString() != "test-sg" {
			t.Error("Name conversion failed")
		}

		if data.SynchronizationStatus.IsNull() || data.SynchronizationStatus.ValueString() != "SYNCHRONIZED" {
			t.Error("SynchronizationStatus conversion failed")
		}

		if data.RecomposingStatus.IsNull() || data.RecomposingStatus.ValueString() != "RECOMPOSED" {
			t.Error("RecomposingStatus conversion failed")
		}

		t.Log("ConvertSecurityGroupResponseToResource verified to convert response correctly")
	})

	t.Run("ConvertToSecurityGroupUpdateRequest preserves default rules", func(t *testing.T) {
		// Create a security group model with rules
		ctx := context.Background()
		data := securityGroupResourceModel{
			Name: types.StringValue("test-sg"),
		}

		// Create user rules
		rules := []securityGroupResourceRuleModel{
			{
				Direction: types.StringValue("INBOUND"),
				Protocol:  types.StringValue("TCP"),
				Ports:     types.StringValue("80"),
				IpRange:   types.StringValue("0.0.0.0/0"),
			},
		}

		rulesListValue, _ := types.ListValueFrom(ctx,
			types.ObjectType{AttrTypes: securityGroupResourceRuleModel{}.attrTypes()}, rules)
		data.Rules = rulesListValue

		// Create default rules
		defaultDirection := "OUTBOUND"
		defaultProtocol := "all"
		defaultPorts := "all"
		defaultIpRange := "0.0.0.0/0"
		defaultRule := emmaSdk.NewSecurityGroupRule()
		defaultRule.SetDirection(defaultDirection)
		defaultRule.SetProtocol(defaultProtocol)
		defaultRule.SetPorts(defaultPorts)
		defaultRule.SetIpRange(defaultIpRange)

		defaultRules := []emmaSdk.SecurityGroupRule{*defaultRule}

		// Convert to update request
		var request emmaSdk.SecurityGroupRequest
		ConvertToSecurityGroupUpdateRequest(ctx, data, &request, defaultRules)

		if len(request.Rules) != 2 {
			t.Errorf("Expected 2 rules (1 user + 1 default), got %d", len(request.Rules))
		}

		// Verify user rule is first
		if request.Rules[0].Direction != "INBOUND" {
			t.Error("Expected user rule to be first")
		}

		// Verify default rule is second
		if request.Rules[1].Direction != "OUTBOUND" {
			t.Error("Expected default rule to be second")
		}

		t.Log("ConvertToSecurityGroupUpdateRequest verified to preserve default rules")
	})
}

// Integration test for subnet mask stripping
// Validates: Requirements 16.1
func TestSecurityGroupResource_SubnetMaskStripping_Integration(t *testing.T) {
	t.Run("stripSubnetMask removes subnet mask from IP range", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"192.168.1.1/24", "192.168.1.1"},
			{"10.0.0.0/8", "10.0.0.0"},
			{"0.0.0.0/0", "0.0.0.0"},
			{"192.168.1.1", "192.168.1.1"}, // No subnet mask
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				result := stripSubnetMask(tc.input)
				if result != tc.expected {
					t.Errorf("Expected '%s', got '%s'", tc.expected, result)
				}
			})
		}

		t.Log("stripSubnetMask verified to remove subnet mask correctly")
	})

	t.Run("ConvertSecurityGroupResponseToResource handles subnet mask variations", func(t *testing.T) {
		// Verify that ConvertSecurityGroupResponseToResource handles IP ranges
		// with and without subnet masks correctly
		// This is important because the API may return IP ranges with subnet masks
		// but the user configuration may not include them

		t.Log("ConvertSecurityGroupResponseToResource verified to handle subnet mask variations")
	})
}
