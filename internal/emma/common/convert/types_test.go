package convert

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
)

// Test Int32ToString
func TestInt32ToString(t *testing.T) {
	t.Run("converts valid int32 pointer", func(t *testing.T) {
		value := int32(42)
		result := Int32ToString(&value)
		assert.False(t, result.IsNull())
		assert.Equal(t, "42", result.ValueString())
	})

	t.Run("converts negative int32", func(t *testing.T) {
		value := int32(-100)
		result := Int32ToString(&value)
		assert.False(t, result.IsNull())
		assert.Equal(t, "-100", result.ValueString())
	})

	t.Run("converts zero", func(t *testing.T) {
		value := int32(0)
		result := Int32ToString(&value)
		assert.False(t, result.IsNull())
		assert.Equal(t, "0", result.ValueString())
	})

	t.Run("returns null for nil pointer", func(t *testing.T) {
		result := Int32ToString(nil)
		assert.True(t, result.IsNull())
	})
}

// Test StringToInt32
func TestStringToInt32(t *testing.T) {
	t.Run("converts valid string", func(t *testing.T) {
		value := types.StringValue("42")
		result, err := StringToInt32(value)
		assert.NoError(t, err)
		assert.Equal(t, int32(42), result)
	})

	t.Run("converts negative string", func(t *testing.T) {
		value := types.StringValue("-100")
		result, err := StringToInt32(value)
		assert.NoError(t, err)
		assert.Equal(t, int32(-100), result)
	})

	t.Run("converts zero string", func(t *testing.T) {
		value := types.StringValue("0")
		result, err := StringToInt32(value)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), result)
	})

	t.Run("returns error for null value", func(t *testing.T) {
		value := types.StringNull()
		_, err := StringToInt32(value)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "null or unknown")
	})

	t.Run("returns error for unknown value", func(t *testing.T) {
		value := types.StringUnknown()
		_, err := StringToInt32(value)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "null or unknown")
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		value := types.StringValue("not-a-number")
		_, err := StringToInt32(value)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid integer format")
	})

	t.Run("returns error for float string", func(t *testing.T) {
		value := types.StringValue("42.5")
		_, err := StringToInt32(value)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid integer format")
	})
}

// Test Int64ToInt32
func TestInt64ToInt32(t *testing.T) {
	t.Run("converts valid int64", func(t *testing.T) {
		value := types.Int64Value(42)
		result, err := Int64ToInt32(value)
		assert.NoError(t, err)
		assert.Equal(t, int32(42), result)
	})

	t.Run("converts negative int64", func(t *testing.T) {
		value := types.Int64Value(-100)
		result, err := Int64ToInt32(value)
		assert.NoError(t, err)
		assert.Equal(t, int32(-100), result)
	})

	t.Run("converts zero", func(t *testing.T) {
		value := types.Int64Value(0)
		result, err := Int64ToInt32(value)
		assert.NoError(t, err)
		assert.Equal(t, int32(0), result)
	})

	t.Run("converts max int32 value", func(t *testing.T) {
		value := types.Int64Value(2147483647)
		result, err := Int64ToInt32(value)
		assert.NoError(t, err)
		assert.Equal(t, int32(2147483647), result)
	})

	t.Run("converts min int32 value", func(t *testing.T) {
		value := types.Int64Value(-2147483648)
		result, err := Int64ToInt32(value)
		assert.NoError(t, err)
		assert.Equal(t, int32(-2147483648), result)
	})

	t.Run("returns error for value above int32 max", func(t *testing.T) {
		value := types.Int64Value(2147483648)
		_, err := Int64ToInt32(value)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "out of int32 range")
	})

	t.Run("returns error for value below int32 min", func(t *testing.T) {
		value := types.Int64Value(-2147483649)
		_, err := Int64ToInt32(value)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "out of int32 range")
	})

	t.Run("returns error for null value", func(t *testing.T) {
		value := types.Int64Null()
		_, err := Int64ToInt32(value)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "null or unknown")
	})

	t.Run("returns error for unknown value", func(t *testing.T) {
		value := types.Int64Unknown()
		_, err := Int64ToInt32(value)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "null or unknown")
	})
}

// Test Int32ToInt64
func TestInt32ToInt64(t *testing.T) {
	t.Run("converts valid int32 pointer", func(t *testing.T) {
		value := int32(42)
		result := Int32ToInt64(&value)
		assert.False(t, result.IsNull())
		assert.Equal(t, int64(42), result.ValueInt64())
	})

	t.Run("converts negative int32", func(t *testing.T) {
		value := int32(-100)
		result := Int32ToInt64(&value)
		assert.False(t, result.IsNull())
		assert.Equal(t, int64(-100), result.ValueInt64())
	})

	t.Run("converts zero", func(t *testing.T) {
		value := int32(0)
		result := Int32ToInt64(&value)
		assert.False(t, result.IsNull())
		assert.Equal(t, int64(0), result.ValueInt64())
	})

	t.Run("returns null for nil pointer", func(t *testing.T) {
		result := Int32ToInt64(nil)
		assert.True(t, result.IsNull())
	})
}

// Test StringPointerToString
func TestStringPointerToString(t *testing.T) {
	t.Run("converts valid string pointer", func(t *testing.T) {
		value := "test-string"
		result := StringPointerToString(&value)
		assert.False(t, result.IsNull())
		assert.Equal(t, "test-string", result.ValueString())
	})

	t.Run("converts empty string", func(t *testing.T) {
		value := ""
		result := StringPointerToString(&value)
		assert.False(t, result.IsNull())
		assert.Equal(t, "", result.ValueString())
	})

	t.Run("returns null for nil pointer", func(t *testing.T) {
		result := StringPointerToString(nil)
		assert.True(t, result.IsNull())
	})
}

// Test BoolPointerToBool
func TestBoolPointerToBool(t *testing.T) {
	t.Run("converts true pointer", func(t *testing.T) {
		value := true
		result := BoolPointerToBool(&value)
		assert.False(t, result.IsNull())
		assert.True(t, result.ValueBool())
	})

	t.Run("converts false pointer", func(t *testing.T) {
		value := false
		result := BoolPointerToBool(&value)
		assert.False(t, result.IsNull())
		assert.False(t, result.ValueBool())
	})

	t.Run("returns null for nil pointer", func(t *testing.T) {
		result := BoolPointerToBool(nil)
		assert.True(t, result.IsNull())
	})
}

// Property-Based Tests

// TestProperty2_TypeConversionsPreserveValues tests that round-trip conversions preserve values
// Feature: provider-improvements, Property 2: Type Conversions Preserve Values
// Validates: Requirements 2.1, 2.2, 2.4
func TestProperty2_TypeConversionsPreserveValues(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Int32 -> String -> Int32 round-trip preserves value
	properties.Property("int32 to string to int32 round-trip preserves value", prop.ForAll(
		func(value int32) bool {
			// Convert int32 -> types.String
			strValue := Int32ToString(&value)
			
			// Convert types.String -> int32
			result, err := StringToInt32(strValue)
			
			if err != nil {
				t.Logf("Unexpected error during conversion: %v", err)
				return false
			}
			
			if result != value {
				t.Logf("Round-trip failed: original=%d, result=%d", value, result)
				return false
			}
			
			return true
		},
		gen.Int32(),
	))

	// Property: Int32 -> Int64 -> Int32 round-trip preserves value
	properties.Property("int32 to int64 to int32 round-trip preserves value", prop.ForAll(
		func(value int32) bool {
			// Convert int32 -> types.Int64
			int64Value := Int32ToInt64(&value)
			
			// Convert types.Int64 -> int32
			result, err := Int64ToInt32(int64Value)
			
			if err != nil {
				t.Logf("Unexpected error during conversion: %v", err)
				return false
			}
			
			if result != value {
				t.Logf("Round-trip failed: original=%d, result=%d", value, result)
				return false
			}
			
			return true
		},
		gen.Int32(),
	))

	// Property: String pointer -> types.String -> String pointer round-trip preserves value
	properties.Property("string pointer to types.String preserves value", prop.ForAll(
		func(value string) bool {
			// Convert string -> types.String
			tfValue := StringPointerToString(&value)
			
			// Verify the value is preserved
			if tfValue.IsNull() {
				t.Logf("Unexpected null value for non-nil pointer")
				return false
			}
			
			if tfValue.ValueString() != value {
				t.Logf("Value not preserved: original=%q, result=%q", value, tfValue.ValueString())
				return false
			}
			
			return true
		},
		gen.AlphaString(),
	))

	// Property: Bool pointer -> types.Bool preserves value
	properties.Property("bool pointer to types.Bool preserves value", prop.ForAll(
		func(value bool) bool {
			// Convert bool -> types.Bool
			tfValue := BoolPointerToBool(&value)
			
			// Verify the value is preserved
			if tfValue.IsNull() {
				t.Logf("Unexpected null value for non-nil pointer")
				return false
			}
			
			if tfValue.ValueBool() != value {
				t.Logf("Value not preserved: original=%v, result=%v", value, tfValue.ValueBool())
				return false
			}
			
			return true
		},
		gen.Bool(),
	))

	// Property: Valid int64 values within int32 range convert successfully
	properties.Property("int64 values within int32 range convert successfully", prop.ForAll(
		func(value int32) bool {
			// Use int32 value as int64 (guaranteed to be in range)
			int64Value := types.Int64Value(int64(value))
			
			// Convert to int32
			result, err := Int64ToInt32(int64Value)
			
			if err != nil {
				t.Logf("Unexpected error for value in range: %v", err)
				return false
			}
			
			if result != value {
				t.Logf("Value not preserved: original=%d, result=%d", value, result)
				return false
			}
			
			return true
		},
		gen.Int32(),
	))

	// Run all properties with 100 iterations each
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty3_NullHandlingConsistency tests that null and unknown values are handled consistently
// Feature: provider-improvements, Property 3: Null Handling Consistency
// Validates: Requirements 2.3
func TestProperty3_NullHandlingConsistency(t *testing.T) {
	properties := gopter.NewProperties(nil)

	// Property: Null pointer conversions return null Terraform values without panicking
	properties.Property("null pointers convert to null terraform values without panic", prop.ForAll(
		func(shouldBeNil bool) bool {
			// Test Int32ToString with nil
			if shouldBeNil {
				result := Int32ToString(nil)
				if !result.IsNull() {
					t.Logf("Int32ToString(nil) should return null value")
					return false
				}
			}
			
			// Test Int32ToInt64 with nil
			if shouldBeNil {
				result := Int32ToInt64(nil)
				if !result.IsNull() {
					t.Logf("Int32ToInt64(nil) should return null value")
					return false
				}
			}
			
			// Test StringPointerToString with nil
			if shouldBeNil {
				result := StringPointerToString(nil)
				if !result.IsNull() {
					t.Logf("StringPointerToString(nil) should return null value")
					return false
				}
			}
			
			// Test BoolPointerToBool with nil
			if shouldBeNil {
				result := BoolPointerToBool(nil)
				if !result.IsNull() {
					t.Logf("BoolPointerToBool(nil) should return null value")
					return false
				}
			}
			
			return true
		},
		gen.Bool(),
	))

	// Property: Null Terraform values return errors without panicking
	properties.Property("null terraform values return errors without panic", prop.ForAll(
		func(useNull bool) bool {
			// Test StringToInt32 with null
			if useNull {
				_, err := StringToInt32(types.StringNull())
				if err == nil {
					t.Logf("StringToInt32 should return error for null value")
					return false
				}
			}
			
			// Test Int64ToInt32 with null
			if useNull {
				_, err := Int64ToInt32(types.Int64Null())
				if err == nil {
					t.Logf("Int64ToInt32 should return error for null value")
					return false
				}
			}
			
			return true
		},
		gen.Bool(),
	))

	// Property: Unknown Terraform values return errors without panicking
	properties.Property("unknown terraform values return errors without panic", prop.ForAll(
		func(useUnknown bool) bool {
			// Test StringToInt32 with unknown
			if useUnknown {
				_, err := StringToInt32(types.StringUnknown())
				if err == nil {
					t.Logf("StringToInt32 should return error for unknown value")
					return false
				}
			}
			
			// Test Int64ToInt32 with unknown
			if useUnknown {
				_, err := Int64ToInt32(types.Int64Unknown())
				if err == nil {
					t.Logf("Int64ToInt32 should return error for unknown value")
					return false
				}
			}
			
			return true
		},
		gen.Bool(),
	))

	// Property: All conversion functions handle edge cases without panicking
	properties.Property("conversion functions never panic on edge cases", prop.ForAll(
		func(testCase int) bool {
			// This property tests that no conversion function panics
			// We use a defer/recover to catch any panics
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Panic detected: %v", r)
				}
			}()
			
			switch testCase % 10 {
			case 0:
				Int32ToString(nil)
			case 1:
				_, _ = StringToInt32(types.StringNull())
			case 2:
				_, _ = StringToInt32(types.StringUnknown())
			case 3:
				_, _ = Int64ToInt32(types.Int64Null())
			case 4:
				_, _ = Int64ToInt32(types.Int64Unknown())
			case 5:
				Int32ToInt64(nil)
			case 6:
				StringPointerToString(nil)
			case 7:
				BoolPointerToBool(nil)
			case 8:
				// Test with empty string
				_, _ = StringToInt32(types.StringValue(""))
			case 9:
				// Test with invalid string
				_, _ = StringToInt32(types.StringValue("not-a-number"))
			}
			
			// If we reach here without panic, test passes
			return true
		},
		gen.IntRange(0, 100),
	))

	// Run all properties with 100 iterations each
	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
