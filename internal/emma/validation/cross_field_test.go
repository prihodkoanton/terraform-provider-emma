package emma

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
)

// createTestConfigSimple creates a simple test config for cross-field validation testing
func createTestConfigSimple(t *testing.T, values map[string]interface{}) tfsdk.Config {
	// Build tftypes.Value
	tfTypeAttrs := make(map[string]tftypes.Type)
	tfValues := make(map[string]tftypes.Value)
	
	for name, val := range values {
		switch v := val.(type) {
		case string:
			tfTypeAttrs[name] = tftypes.String
			tfValues[name] = tftypes.NewValue(tftypes.String, v)
		case int64:
			tfTypeAttrs[name] = tftypes.Number
			tfValues[name] = tftypes.NewValue(tftypes.Number, v)
		case nil:
			tfTypeAttrs[name] = tftypes.String
			tfValues[name] = tftypes.NewValue(tftypes.String, nil)
		}
	}
	
	objType := tftypes.Object{AttributeTypes: tfTypeAttrs}
	tfValue := tftypes.NewValue(objType, tfValues)
	
	// Create schema
	schemaAttrs := make(map[string]schema.Attribute)
	for name := range values {
		schemaAttrs[name] = schema.StringAttribute{
			Optional: true,
		}
	}
	
	testSchema := schema.Schema{
		Attributes: schemaAttrs,
	}
	
	return tfsdk.Config{
		Raw:    tfValue,
		Schema: testSchema,
	}
}

// TestMutuallyExclusive_ValidateString_OnlyOneFieldSet tests that validation passes when only one field is set
func TestMutuallyExclusive_ValidateString_OnlyOneFieldSet(t *testing.T) {
	v := MutuallyExclusive{
		Fields: []string{"field_a", "field_b", "field_c"},
	}
	
	var resp validator.StringResponse
	var req validator.StringRequest
	
	// Set up config with only field_a set
	config := createTestConfigSimple(t, map[string]interface{}{
		"field_a": "value_a",
		"field_b": nil,
		"field_c": nil,
	})
	
	req.ConfigValue = types.StringValue("value_a")
	req.Path = path.Root("field_a")
	req.Config = config
	
	v.ValidateString(context.Background(), req, &resp)
	
	assert.False(t, resp.Diagnostics.HasError(), "Validation should pass when only one field is set")
}

// TestMutuallyExclusive_ValidateString_MultipleFieldsSet tests that validation fails when multiple fields are set
func TestMutuallyExclusive_ValidateString_MultipleFieldsSet(t *testing.T) {
	v := MutuallyExclusive{
		Fields: []string{"field_a", "field_b", "field_c"},
	}
	
	var resp validator.StringResponse
	var req validator.StringRequest
	
	// Set up config with field_a and field_b both set
	config := createTestConfigSimple(t, map[string]interface{}{
		"field_a": "value_a",
		"field_b": "value_b",
		"field_c": nil,
	})
	
	req.ConfigValue = types.StringValue("value_a")
	req.Path = path.Root("field_a")
	req.Config = config
	
	v.ValidateString(context.Background(), req, &resp)
	
	assert.True(t, resp.Diagnostics.HasError(), "Validation should fail when multiple fields are set")
	if resp.Diagnostics.HasError() {
		actualMsg := resp.Diagnostics.Errors()[0].Summary()
		assert.Equal(t, "Mutually Exclusive Fields", actualMsg)
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "Only one of")
	}
}

// TestMutuallyExclusive_ValidateString_NullValue tests that validation passes when current field is null
func TestMutuallyExclusive_ValidateString_NullValue(t *testing.T) {
	v := MutuallyExclusive{
		Fields: []string{"field_a", "field_b"},
	}
	
	var resp validator.StringResponse
	var req validator.StringRequest
	
	config := createTestConfigSimple(t, map[string]interface{}{
		"field_a": nil,
		"field_b": "value_b",
	})
	
	req.ConfigValue = types.StringNull()
	req.Path = path.Root("field_a")
	req.Config = config
	
	v.ValidateString(context.Background(), req, &resp)
	
	assert.False(t, resp.Diagnostics.HasError(), "Validation should pass when current field is null")
}

// TestMutuallyExclusive_ValidateString_UnknownValue tests that validation passes when current field is unknown
func TestMutuallyExclusive_ValidateString_UnknownValue(t *testing.T) {
	v := MutuallyExclusive{
		Fields: []string{"field_a", "field_b"},
	}
	
	var resp validator.StringResponse
	var req validator.StringRequest
	
	config := createTestConfigSimple(t, map[string]interface{}{
		"field_a": nil,
		"field_b": "value_b",
	})
	
	req.ConfigValue = types.StringUnknown()
	req.Path = path.Root("field_a")
	req.Config = config
	
	v.ValidateString(context.Background(), req, &resp)
	
	assert.False(t, resp.Diagnostics.HasError(), "Validation should pass when current field is unknown")
}

// TestMutuallyExclusive_ValidateString_EmptyStringIgnored tests that empty strings are not counted as set
func TestMutuallyExclusive_ValidateString_EmptyStringIgnored(t *testing.T) {
	v := MutuallyExclusive{
		Fields: []string{"field_a", "field_b"},
	}
	
	var resp validator.StringResponse
	var req validator.StringRequest
	
	config := createTestConfigSimple(t, map[string]interface{}{
		"field_a": "value_a",
		"field_b": "", // Empty string should be ignored
	})
	
	req.ConfigValue = types.StringValue("value_a")
	req.Path = path.Root("field_a")
	req.Config = config
	
	v.ValidateString(context.Background(), req, &resp)
	
	assert.False(t, resp.Diagnostics.HasError(), "Validation should pass when other field is empty string")
}

// TestRequiresOneOf_ValidateString_AtLeastOneFieldSet tests that validation passes when at least one field is set
func TestRequiresOneOf_ValidateString_AtLeastOneFieldSet(t *testing.T) {
	v := RequiresOneOf{
		Fields: []string{"field_a", "field_b", "field_c"},
	}
	
	var resp validator.StringResponse
	var req validator.StringRequest
	
	config := createTestConfigSimple(t, map[string]interface{}{
		"field_a": "value_a",
		"field_b": nil,
		"field_c": nil,
	})
	
	req.ConfigValue = types.StringValue("value_a")
	req.Path = path.Root("field_a")
	req.Config = config
	
	v.ValidateString(context.Background(), req, &resp)
	
	assert.False(t, resp.Diagnostics.HasError(), "Validation should pass when at least one field is set")
}

// TestRequiresOneOf_ValidateString_NoFieldsSet tests that validation fails when no fields are set
func TestRequiresOneOf_ValidateString_NoFieldsSet(t *testing.T) {
	v := RequiresOneOf{
		Fields: []string{"field_a", "field_b", "field_c"},
	}
	
	var resp validator.StringResponse
	var req validator.StringRequest
	
	config := createTestConfigSimple(t, map[string]interface{}{
		"field_a": nil,
		"field_b": nil,
		"field_c": nil,
	})
	
	req.ConfigValue = types.StringNull()
	req.Path = path.Root("field_a")
	req.Config = config
	
	v.ValidateString(context.Background(), req, &resp)
	
	assert.True(t, resp.Diagnostics.HasError(), "Validation should fail when no fields are set")
	if resp.Diagnostics.HasError() {
		actualMsg := resp.Diagnostics.Errors()[0].Summary()
		assert.Equal(t, "Required Field Missing", actualMsg)
		assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "At least one of")
	}
}

// TestRequiresOneOf_ValidateString_MultipleFieldsSet tests that validation passes when multiple fields are set
func TestRequiresOneOf_ValidateString_MultipleFieldsSet(t *testing.T) {
	v := RequiresOneOf{
		Fields: []string{"field_a", "field_b"},
	}
	
	var resp validator.StringResponse
	var req validator.StringRequest
	
	config := createTestConfigSimple(t, map[string]interface{}{
		"field_a": "value_a",
		"field_b": "value_b",
	})
	
	req.ConfigValue = types.StringValue("value_a")
	req.Path = path.Root("field_a")
	req.Config = config
	
	v.ValidateString(context.Background(), req, &resp)
	
	assert.False(t, resp.Diagnostics.HasError(), "Validation should pass when multiple fields are set")
}

// TestRequiresOneOf_ValidateString_EmptyStringNotCounted tests that empty strings are not counted as set
func TestRequiresOneOf_ValidateString_EmptyStringNotCounted(t *testing.T) {
	v := RequiresOneOf{
		Fields: []string{"field_a", "field_b"},
	}
	
	var resp validator.StringResponse
	var req validator.StringRequest
	
	config := createTestConfigSimple(t, map[string]interface{}{
		"field_a": "", // Empty string should not count
		"field_b": nil,
	})
	
	req.ConfigValue = types.StringValue("")
	req.Path = path.Root("field_a")
	req.Config = config
	
	v.ValidateString(context.Background(), req, &resp)
	
	assert.True(t, resp.Diagnostics.HasError(), "Validation should fail when only empty strings are present")
}

// TestMutuallyExclusive_Description tests the description method
func TestMutuallyExclusive_Description(t *testing.T) {
	v := MutuallyExclusive{
		Fields: []string{"field_a", "field_b"},
	}
	
	desc := v.Description(context.Background())
	assert.Contains(t, desc, "Only one of")
	assert.Contains(t, desc, "field_a")
	assert.Contains(t, desc, "field_b")
}

// TestRequiresOneOf_Description tests the description method
func TestRequiresOneOf_Description(t *testing.T) {
	v := RequiresOneOf{
		Fields: []string{"field_a", "field_b"},
	}
	
	desc := v.Description(context.Background())
	assert.Contains(t, desc, "At least one of")
	assert.Contains(t, desc, "field_a")
	assert.Contains(t, desc, "field_b")
}


// Property-Based Tests

// TestProperty4_ValidationErrorsAreDescriptive tests that validation errors are clear and specific
// Feature: provider-improvements, Property 4: Validation Errors Are Descriptive
// Validates: Requirements 3.2, 3.5
func TestProperty4_ValidationErrorsAreDescriptive(t *testing.T) {
	// Import gopter for property-based testing
	properties := gopter.NewProperties(nil)
	
	// Property 1: MutuallyExclusive errors include field names
	properties.Property("mutually exclusive errors include all conflicting field names", prop.ForAll(
		func(fieldCount int) bool {
			// Generate field names
			fields := make([]string, fieldCount)
			for i := 0; i < fieldCount; i++ {
				fields[i] = fmt.Sprintf("field_%d", i)
			}
			
			v := MutuallyExclusive{Fields: fields}
			
			// Create config with multiple fields set
			configValues := make(map[string]interface{})
			for i := 0; i < fieldCount; i++ {
				configValues[fields[i]] = fmt.Sprintf("value_%d", i)
			}
			
			config := createTestConfigSimple(t, configValues)
			
			var resp validator.StringResponse
			var req validator.StringRequest
			req.ConfigValue = types.StringValue("value_0")
			req.Path = path.Root(fields[0])
			req.Config = config
			
			v.ValidateString(context.Background(), req, &resp)
			
			// Should have error
			if !resp.Diagnostics.HasError() {
				return false
			}
			
			// Error message should contain "Only one of"
			errDetail := resp.Diagnostics.Errors()[0].Detail()
			if !strings.Contains(errDetail, "Only one of") {
				return false
			}
			
			// Error message should mention the fields
			for _, field := range fields {
				if !strings.Contains(errDetail, field) {
					return false
				}
			}
			
			return true
		},
		gen.IntRange(2, 5), // Test with 2-5 fields
	))
	
	// Property 2: RequiresOneOf errors include field names
	properties.Property("requires one of errors include all required field names", prop.ForAll(
		func(fieldCount int) bool {
			// Generate field names
			fields := make([]string, fieldCount)
			for i := 0; i < fieldCount; i++ {
				fields[i] = fmt.Sprintf("field_%d", i)
			}
			
			v := RequiresOneOf{Fields: fields}
			
			// Create config with no fields set
			configValues := make(map[string]interface{})
			for i := 0; i < fieldCount; i++ {
				configValues[fields[i]] = nil
			}
			
			config := createTestConfigSimple(t, configValues)
			
			var resp validator.StringResponse
			var req validator.StringRequest
			req.ConfigValue = types.StringNull()
			req.Path = path.Root(fields[0])
			req.Config = config
			
			v.ValidateString(context.Background(), req, &resp)
			
			// Should have error
			if !resp.Diagnostics.HasError() {
				return false
			}
			
			// Error message should contain "At least one of"
			errDetail := resp.Diagnostics.Errors()[0].Detail()
			if !strings.Contains(errDetail, "At least one of") {
				return false
			}
			
			// Error message should mention the fields
			for _, field := range fields {
				if !strings.Contains(errDetail, field) {
					return false
				}
			}
			
			return true
		},
		gen.IntRange(2, 5), // Test with 2-5 fields
	))
	
	// Property 3: Error summaries are clear and consistent
	properties.Property("error summaries are clear and consistent", prop.ForAll(
		func(fieldCount int) bool {
			fields := make([]string, fieldCount)
			for i := 0; i < fieldCount; i++ {
				fields[i] = fmt.Sprintf("field_%d", i)
			}
			
			// Test MutuallyExclusive
			v1 := MutuallyExclusive{Fields: fields}
			configValues := make(map[string]interface{})
			for i := 0; i < fieldCount; i++ {
				configValues[fields[i]] = fmt.Sprintf("value_%d", i)
			}
			config := createTestConfigSimple(t, configValues)
			
			var resp1 validator.StringResponse
			var req1 validator.StringRequest
			req1.ConfigValue = types.StringValue("value_0")
			req1.Path = path.Root(fields[0])
			req1.Config = config
			
			v1.ValidateString(context.Background(), req1, &resp1)
			
			if resp1.Diagnostics.HasError() {
				summary := resp1.Diagnostics.Errors()[0].Summary()
				if summary != "Mutually Exclusive Fields" {
					return false
				}
			}
			
			// Test RequiresOneOf
			v2 := RequiresOneOf{Fields: fields}
			configValues2 := make(map[string]interface{})
			for i := 0; i < fieldCount; i++ {
				configValues2[fields[i]] = nil
			}
			config2 := createTestConfigSimple(t, configValues2)
			
			var resp2 validator.StringResponse
			var req2 validator.StringRequest
			req2.ConfigValue = types.StringNull()
			req2.Path = path.Root(fields[0])
			req2.Config = config2
			
			v2.ValidateString(context.Background(), req2, &resp2)
			
			if resp2.Diagnostics.HasError() {
				summary := resp2.Diagnostics.Errors()[0].Summary()
				if summary != "Required Field Missing" {
					return false
				}
			}
			
			return true
		},
		gen.IntRange(2, 5),
	))
	
	// Run all properties with 100 iterations each
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
