package convert

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// Test helper structs
type testNestedModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.Int64  `tfsdk:"value"`
}

func (m testNestedModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":  types.StringType,
		"value": types.Int64Type,
	}
}

// TestNewObjectConverter tests the constructor
func TestNewObjectConverter(t *testing.T) {
	ctx := context.Background()
	converter := NewObjectConverter(ctx)
	
	assert.NotNil(t, converter)
	assert.NotNil(t, converter.ctx)
}

// TestToObject tests object conversion
func TestToObject(t *testing.T) {
	ctx := context.Background()
	converter := NewObjectConverter(ctx)

	t.Run("converts valid struct to object", func(t *testing.T) {
		model := testNestedModel{
			Name:  types.StringValue("test-name"),
			Value: types.Int64Value(42),
		}

		result, diags := converter.ToObject(model.attrTypes(), model)
		
		assert.False(t, diags.HasError())
		assert.False(t, result.IsNull())
		assert.False(t, result.IsUnknown())
		
		// Verify the object contains the expected attributes
		attrs := result.Attributes()
		assert.Len(t, attrs, 2)
		
		nameAttr, ok := attrs["name"]
		assert.True(t, ok)
		assert.Equal(t, "test-name", nameAttr.(types.String).ValueString())
		
		valueAttr, ok := attrs["value"]
		assert.True(t, ok)
		assert.Equal(t, int64(42), valueAttr.(types.Int64).ValueInt64())
	})

	t.Run("converts struct with null values", func(t *testing.T) {
		model := testNestedModel{
			Name:  types.StringNull(),
			Value: types.Int64Null(),
		}

		result, diags := converter.ToObject(model.attrTypes(), model)
		
		assert.False(t, diags.HasError())
		assert.False(t, result.IsNull())
		
		// Verify the object contains null attributes
		attrs := result.Attributes()
		nameAttr, ok := attrs["name"]
		assert.True(t, ok)
		assert.True(t, nameAttr.(types.String).IsNull())
		
		valueAttr, ok := attrs["value"]
		assert.True(t, ok)
		assert.True(t, valueAttr.(types.Int64).IsNull())
	})

	t.Run("converts struct with unknown values", func(t *testing.T) {
		model := testNestedModel{
			Name:  types.StringUnknown(),
			Value: types.Int64Unknown(),
		}

		result, diags := converter.ToObject(model.attrTypes(), model)
		
		assert.False(t, diags.HasError())
		assert.False(t, result.IsNull())
		
		// Verify the object contains unknown attributes
		attrs := result.Attributes()
		nameAttr, ok := attrs["name"]
		assert.True(t, ok)
		assert.True(t, nameAttr.(types.String).IsUnknown())
		
		valueAttr, ok := attrs["value"]
		assert.True(t, ok)
		assert.True(t, valueAttr.(types.Int64).IsUnknown())
	})

	t.Run("handles empty attribute types", func(t *testing.T) {
		emptyModel := struct{}{}
		emptyAttrTypes := map[string]attr.Type{}

		result, diags := converter.ToObject(emptyAttrTypes, emptyModel)
		
		assert.False(t, diags.HasError())
		assert.False(t, result.IsNull())
		assert.Len(t, result.Attributes(), 0)
	})
}

// TestToList tests list conversion
func TestToList(t *testing.T) {
	ctx := context.Background()
	converter := NewObjectConverter(ctx)

	t.Run("converts valid slice to list", func(t *testing.T) {
		models := []testNestedModel{
			{
				Name:  types.StringValue("item1"),
				Value: types.Int64Value(1),
			},
			{
				Name:  types.StringValue("item2"),
				Value: types.Int64Value(2),
			},
			{
				Name:  types.StringValue("item3"),
				Value: types.Int64Value(3),
			},
		}

		elementType := types.ObjectType{AttrTypes: testNestedModel{}.attrTypes()}
		result, diags := converter.ToList(elementType, models)
		
		assert.False(t, diags.HasError())
		assert.False(t, result.IsNull())
		assert.False(t, result.IsUnknown())
		assert.Equal(t, 3, len(result.Elements()))
	})

	t.Run("converts empty slice to empty list", func(t *testing.T) {
		models := []testNestedModel{}

		elementType := types.ObjectType{AttrTypes: testNestedModel{}.attrTypes()}
		result, diags := converter.ToList(elementType, models)
		
		assert.False(t, diags.HasError())
		assert.False(t, result.IsNull())
		assert.Equal(t, 0, len(result.Elements()))
	})

	t.Run("converts slice with null values", func(t *testing.T) {
		models := []testNestedModel{
			{
				Name:  types.StringNull(),
				Value: types.Int64Null(),
			},
		}

		elementType := types.ObjectType{AttrTypes: testNestedModel{}.attrTypes()}
		result, diags := converter.ToList(elementType, models)
		
		assert.False(t, diags.HasError())
		assert.False(t, result.IsNull())
		assert.Equal(t, 1, len(result.Elements()))
	})

	t.Run("converts slice of strings", func(t *testing.T) {
		stringModels := []types.String{
			types.StringValue("one"),
			types.StringValue("two"),
			types.StringValue("three"),
		}

		result, diags := converter.ToList(types.StringType, stringModels)
		
		assert.False(t, diags.HasError())
		assert.False(t, result.IsNull())
		assert.Equal(t, 3, len(result.Elements()))
	})

	t.Run("converts slice of integers", func(t *testing.T) {
		intModels := []types.Int64{
			types.Int64Value(10),
			types.Int64Value(20),
			types.Int64Value(30),
		}

		result, diags := converter.ToList(types.Int64Type, intModels)
		
		assert.False(t, diags.HasError())
		assert.False(t, result.IsNull())
		assert.Equal(t, 3, len(result.Elements()))
	})
}

// TestObjectConverterWithDifferentContexts tests that the converter works with different contexts
func TestObjectConverterWithDifferentContexts(t *testing.T) {
	t.Run("works with background context", func(t *testing.T) {
		ctx := context.Background()
		converter := NewObjectConverter(ctx)
		
		model := testNestedModel{
			Name:  types.StringValue("test"),
			Value: types.Int64Value(100),
		}

		result, diags := converter.ToObject(model.attrTypes(), model)
		assert.False(t, diags.HasError())
		assert.False(t, result.IsNull())
	})

	t.Run("works with context with values", func(t *testing.T) {
		type contextKey string
		ctx := context.WithValue(context.Background(), contextKey("key"), "value")
		converter := NewObjectConverter(ctx)
		
		model := testNestedModel{
			Name:  types.StringValue("test"),
			Value: types.Int64Value(100),
		}

		result, diags := converter.ToObject(model.attrTypes(), model)
		assert.False(t, diags.HasError())
		assert.False(t, result.IsNull())
	})
}
