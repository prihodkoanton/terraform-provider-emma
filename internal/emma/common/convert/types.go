package convert

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Int32ToString converts int32 pointer to types.String
func Int32ToString(value *int32) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(strconv.Itoa(int(*value)))
}

// StringToInt32 converts types.String to int32 with validation
func StringToInt32(value types.String) (int32, error) {
	if value.IsNull() || value.IsUnknown() {
		return 0, fmt.Errorf("value is null or unknown")
	}

	num, err := strconv.ParseInt(value.ValueString(), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid integer format: %w", err)
	}

	return int32(num), nil
}

// Int64ToInt32 converts types.Int64 to int32 with bounds checking
func Int64ToInt32(value types.Int64) (int32, error) {
	if value.IsNull() || value.IsUnknown() {
		return 0, fmt.Errorf("value is null or unknown")
	}

	val := value.ValueInt64()
	if val < -2147483648 || val > 2147483647 {
		return 0, fmt.Errorf("value %d out of int32 range", val)
	}

	return int32(val), nil
}

// Int32ToInt64 converts int32 pointer to types.Int64
func Int32ToInt64(value *int32) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*value))
}

// StringPointerToString converts string pointer to types.String
func StringPointerToString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// BoolPointerToBool converts bool pointer to types.Bool
func BoolPointerToBool(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}
