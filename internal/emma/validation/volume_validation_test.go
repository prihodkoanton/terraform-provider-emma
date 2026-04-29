package emma

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

// MinimumVolumeSize tests

func TestMinimumVolumeSize_ValidateInt64_ZeroValue(t *testing.T) {
	v := MinimumVolumeSize{}
	var resp validator.Int64Response
	var req validator.Int64Request

	req.ConfigValue = types.Int64Value(0)
	req.Path = path.Root("volume_gb")

	v.ValidateInt64(context.Background(), req, &resp)

	assert.Equal(t, 1, resp.Diagnostics.ErrorsCount())
	if resp.Diagnostics.HasError() {
		actualMsg := resp.Diagnostics.Errors()[0].Detail()
		assert.Equal(t, "volume_gb must be at least 1 GB", actualMsg)
	} else {
		assert.Fail(t, "MinimumVolumeSize is not validating zero values")
	}
}

func TestMinimumVolumeSize_ValidateInt64_NegativeValue(t *testing.T) {
	v := MinimumVolumeSize{}
	var resp validator.Int64Response
	var req validator.Int64Request

	req.ConfigValue = types.Int64Value(-10)
	req.Path = path.Root("volume_gb")

	v.ValidateInt64(context.Background(), req, &resp)

	assert.Equal(t, 1, resp.Diagnostics.ErrorsCount())
	if resp.Diagnostics.HasError() {
		actualMsg := resp.Diagnostics.Errors()[0].Detail()
		assert.Equal(t, "volume_gb must be at least 1 GB", actualMsg)
	} else {
		assert.Fail(t, "MinimumVolumeSize is not validating negative values")
	}
}

func TestMinimumVolumeSize_ValidateInt64_ValidValue(t *testing.T) {
	v := MinimumVolumeSize{}
	var resp validator.Int64Response
	var req validator.Int64Request

	req.ConfigValue = types.Int64Value(1)
	req.Path = path.Root("volume_gb")

	v.ValidateInt64(context.Background(), req, &resp)

	assert.False(t, resp.Diagnostics.HasError(), "MinimumVolumeSize should accept value of 1")
}

func TestMinimumVolumeSize_ValidateInt64_LargeValue(t *testing.T) {
	v := MinimumVolumeSize{}
	var resp validator.Int64Response
	var req validator.Int64Request

	req.ConfigValue = types.Int64Value(1000)
	req.Path = path.Root("volume_gb")

	v.ValidateInt64(context.Background(), req, &resp)

	assert.False(t, resp.Diagnostics.HasError(), "MinimumVolumeSize should accept large values")
}

func TestMinimumVolumeSize_ValidateInt64_NullValue(t *testing.T) {
	v := MinimumVolumeSize{}
	var resp validator.Int64Response
	var req validator.Int64Request

	req.ConfigValue = types.Int64Null()
	req.Path = path.Root("volume_gb")

	v.ValidateInt64(context.Background(), req, &resp)

	assert.False(t, resp.Diagnostics.HasError(), "MinimumVolumeSize should skip null values")
}

func TestMinimumVolumeSize_ValidateInt64_UnknownValue(t *testing.T) {
	v := MinimumVolumeSize{}
	var resp validator.Int64Response
	var req validator.Int64Request

	req.ConfigValue = types.Int64Unknown()
	req.Path = path.Root("volume_gb")

	v.ValidateInt64(context.Background(), req, &resp)

	assert.False(t, resp.Diagnostics.HasError(), "MinimumVolumeSize should skip unknown values")
}

// NonEmptyVolumeType tests

func TestNonEmptyVolumeType_ValidateString_EmptyValue(t *testing.T) {
	v := NonEmptyVolumeType{}
	var resp validator.StringResponse
	var req validator.StringRequest

	req.ConfigValue = types.StringValue("")
	req.Path = path.Root("volume_type")

	v.ValidateString(context.Background(), req, &resp)

	assert.Equal(t, 1, resp.Diagnostics.ErrorsCount())
	if resp.Diagnostics.HasError() {
		actualMsg := resp.Diagnostics.Errors()[0].Detail()
		assert.Equal(t, "volume_type must not be empty", actualMsg)
	} else {
		assert.Fail(t, "NonEmptyVolumeType is not validating empty values")
	}
}

func TestNonEmptyVolumeType_ValidateString_BlankValue(t *testing.T) {
	v := NonEmptyVolumeType{}
	var resp validator.StringResponse
	var req validator.StringRequest

	req.ConfigValue = types.StringValue("   ")
	req.Path = path.Root("volume_type")

	v.ValidateString(context.Background(), req, &resp)

	assert.Equal(t, 1, resp.Diagnostics.ErrorsCount())
	if resp.Diagnostics.HasError() {
		actualMsg := resp.Diagnostics.Errors()[0].Detail()
		assert.Equal(t, "volume_type must not be empty", actualMsg)
	} else {
		assert.Fail(t, "NonEmptyVolumeType is not validating blank values")
	}
}

func TestNonEmptyVolumeType_ValidateString_ValidValue(t *testing.T) {
	v := NonEmptyVolumeType{}
	var resp validator.StringResponse
	var req validator.StringRequest

	req.ConfigValue = types.StringValue("ssd")
	req.Path = path.Root("volume_type")

	v.ValidateString(context.Background(), req, &resp)

	assert.False(t, resp.Diagnostics.HasError(), "NonEmptyVolumeType should accept valid volume type")
}

func TestNonEmptyVolumeType_ValidateString_NullValue(t *testing.T) {
	v := NonEmptyVolumeType{}
	var resp validator.StringResponse
	var req validator.StringRequest

	req.ConfigValue = types.StringNull()
	req.Path = path.Root("volume_type")

	v.ValidateString(context.Background(), req, &resp)

	assert.False(t, resp.Diagnostics.HasError(), "NonEmptyVolumeType should skip null values")
}

func TestNonEmptyVolumeType_ValidateString_UnknownValue(t *testing.T) {
	v := NonEmptyVolumeType{}
	var resp validator.StringResponse
	var req validator.StringRequest

	req.ConfigValue = types.StringUnknown()
	req.Path = path.Root("volume_type")

	v.ValidateString(context.Background(), req, &resp)

	assert.False(t, resp.Diagnostics.HasError(), "NonEmptyVolumeType should skip unknown values")
}

// ValidDataCenterId tests

func TestValidDataCenterId_ValidateString_EmptyValue(t *testing.T) {
	v := ValidDataCenterId{}
	var resp validator.StringResponse
	var req validator.StringRequest

	req.ConfigValue = types.StringValue("")
	req.Path = path.Root("data_center_id")

	v.ValidateString(context.Background(), req, &resp)

	assert.Equal(t, 1, resp.Diagnostics.ErrorsCount())
	if resp.Diagnostics.HasError() {
		actualMsg := resp.Diagnostics.Errors()[0].Detail()
		assert.Equal(t, "data_center_id must not be empty", actualMsg)
	} else {
		assert.Fail(t, "ValidDataCenterId is not validating empty values")
	}
}

func TestValidDataCenterId_ValidateString_BlankValue(t *testing.T) {
	v := ValidDataCenterId{}
	var resp validator.StringResponse
	var req validator.StringRequest

	req.ConfigValue = types.StringValue("   ")
	req.Path = path.Root("data_center_id")

	v.ValidateString(context.Background(), req, &resp)

	assert.Equal(t, 1, resp.Diagnostics.ErrorsCount())
	if resp.Diagnostics.HasError() {
		actualMsg := resp.Diagnostics.Errors()[0].Detail()
		assert.Equal(t, "data_center_id must not be empty", actualMsg)
	} else {
		assert.Fail(t, "ValidDataCenterId is not validating blank values")
	}
}

func TestValidDataCenterId_ValidateString_ValidValue(t *testing.T) {
	v := ValidDataCenterId{}
	var resp validator.StringResponse
	var req validator.StringRequest

	req.ConfigValue = types.StringValue("dc-123")
	req.Path = path.Root("data_center_id")

	v.ValidateString(context.Background(), req, &resp)

	assert.False(t, resp.Diagnostics.HasError(), "ValidDataCenterId should accept valid data center ID")
}

func TestValidDataCenterId_ValidateString_NullValue(t *testing.T) {
	v := ValidDataCenterId{}
	var resp validator.StringResponse
	var req validator.StringRequest

	req.ConfigValue = types.StringNull()
	req.Path = path.Root("data_center_id")

	v.ValidateString(context.Background(), req, &resp)

	assert.False(t, resp.Diagnostics.HasError(), "ValidDataCenterId should skip null values")
}

func TestValidDataCenterId_ValidateString_UnknownValue(t *testing.T) {
	v := ValidDataCenterId{}
	var resp validator.StringResponse
	var req validator.StringRequest

	req.ConfigValue = types.StringUnknown()
	req.Path = path.Root("data_center_id")

	v.ValidateString(context.Background(), req, &resp)

	assert.False(t, resp.Diagnostics.HasError(), "ValidDataCenterId should skip unknown values")
}
