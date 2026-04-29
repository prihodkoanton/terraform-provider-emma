package emma

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"strings"
)

// MinimumVolumeSize validates that volume_gb is at least 1
type MinimumVolumeSize struct {
}

func (v MinimumVolumeSize) Description(ctx context.Context) string {
	return "volume size must be at least 1 GB"
}

func (v MinimumVolumeSize) MarkdownDescription(ctx context.Context) string {
	return "volume size must be at least 1 GB"
}

func (v MinimumVolumeSize) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	if req.ConfigValue.ValueInt64() < 1 {
		resp.Diagnostics.AddError("Validation Error", req.Path.String()+" must be at least 1 GB")
	}
}

// NonEmptyVolumeType validates that volume_type is not empty
type NonEmptyVolumeType struct {
}

func (v NonEmptyVolumeType) Description(ctx context.Context) string {
	return "volume type must not be empty"
}

func (v NonEmptyVolumeType) MarkdownDescription(ctx context.Context) string {
	return "volume type must not be empty"
}

func (v NonEmptyVolumeType) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	if len(strings.TrimSpace(req.ConfigValue.ValueString())) == 0 {
		resp.Diagnostics.AddError("Validation Error", req.Path.String()+" must not be empty")
	}
}

// ValidDataCenterId validates that data_center_id is not empty
type ValidDataCenterId struct {
}

func (v ValidDataCenterId) Description(ctx context.Context) string {
	return "data center ID must not be empty"
}

func (v ValidDataCenterId) MarkdownDescription(ctx context.Context) string {
	return "data center ID must not be empty"
}

func (v ValidDataCenterId) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	if len(strings.TrimSpace(req.ConfigValue.ValueString())) == 0 {
		resp.Diagnostics.AddError("Validation Error", req.Path.String()+" must not be empty")
	}
}
