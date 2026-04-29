package emma

import (
	"context"
	"fmt"
	emmaSdk "github.com/emma-community/emma-go-sdk"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/async"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/errors"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/state"
	"github.com/emma-community/terraform-provider-emma/tools"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"net/http"
)

var _ resource.Resource = &subnetworkResource{}

func NewSubnetworkResource() resource.Resource {
	return &subnetworkResource{}
}

// subnetworkResource defines the resource implementation.
type subnetworkResource struct {
	apiClient *emmaSdk.APIClient
	token     *emmaSdk.Token
}

// subnetworkResourceModel describes the resource data model.
type subnetworkResourceModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	DataCenterId     types.String `tfsdk:"data_center_id"`
	SubnetworkPrefix types.String `tfsdk:"subnetwork_prefix"`
	SubnetworkSize   types.Int64  `tfsdk:"subnetwork_size"`
	Status           types.String `tfsdk:"status"`
}

func (r *subnetworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnetwork"
}

func (r *subnetworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This resource creates and manages subnetworks in the Emma platform.\n\n" +
			"Subnetworks allow you to segment your cloud network into smaller, isolated subnets within a data center. " +
			"Each subnetwork has a defined prefix and size that determines the IP address range available for resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "ID of the subnetwork",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "Name of the subnetwork",
				Optional:    true,
				Computed:    true,
			},
			"data_center_id": schema.StringAttribute{
				Description:   "Data center ID where the subnetwork will be created",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"subnetwork_prefix": schema.StringAttribute{
				Description:   "IP prefix for the subnetwork (e.g. 10.0.1.0)",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"subnetwork_size": schema.Int64Attribute{
				Description:   "Size of the subnetwork (CIDR mask, e.g. 24 for /24)",
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				Description: "Current status of the subnetwork",
				Computed:    true,
			},
		},
	}
}

func (r *subnetworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData))
		return
	}
	r.apiClient = client.apiClient
	r.token = client.token
}

func (r *subnetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data subnetworkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Create subnetwork")

	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *r.token.AccessToken)

	subnetworkCreate := emmaSdk.NewSubnetworkCreate(data.DataCenterId.ValueString(), int32(data.SubnetworkSize.ValueInt64()))
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		subnetworkCreate.SetName(data.Name.ValueString())
	}
	if !data.SubnetworkPrefix.IsNull() && !data.SubnetworkPrefix.IsUnknown() {
		subnetworkCreate.SetSubnetworkPrefix(data.SubnetworkPrefix.ValueString())
	}

	subnetwork, response, err := r.apiClient.SubnetworksAPI.SubnetworkCreate(auth).SubnetworkCreate(*subnetworkCreate).Execute()
	if err != nil {
		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}
		resourceErr := errors.NewError("emma_subnetwork", "Create").
			WithStatusCode(statusCode).
			WithAPIError(tools.ExtractErrorMessage(response)).
			WithMessage(errors.MapHTTPError(statusCode, tools.ExtractErrorMessage(response))).
			WithCause(err).
			Build()
		resp.Diagnostics.AddError("Client Error", resourceErr.Error())
		return
	}

	ConvertSubnetworkResponseToResource(&data, subnetwork)

	subnetworkId := data.Id.ValueString()

	stateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
		ResourceType: "subnetwork",
		ResourceID:   subnetworkId,
		StatusChecker: func(ctx context.Context) (string, error) {
			sn, _, err := r.apiClient.SubnetworksAPI.GetSubnetwork(auth, subnetworkId).Execute()
			if err != nil {
				return "", err
			}
			if sn.Status == nil {
				return "", fmt.Errorf("subnetwork status is nil")
			}
			return *sn.Status, nil
		},
		TargetStates:       state.SubnetworkStableStates,
		TransitionalStates: state.SubnetworkTransitionalStates,
		FailureStates:      state.SubnetworkFailureStates,
		Timeout:            async.DefaultTimeout,
		PollInterval:       async.DefaultPollInterval,
	})

	if err := stateManager.WaitForStableState(auth); err != nil {
		resp.Diagnostics.AddError("State Transition Error",
			fmt.Sprintf("Subnetwork did not reach active state after create: %s", err.Error()))
		return
	}

	subnetworkRefreshed, _, err := r.apiClient.SubnetworksAPI.GetSubnetwork(auth, subnetworkId).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read subnetwork after create: %s", err.Error()))
		return
	}
	ConvertSubnetworkResponseToResource(&data, subnetworkRefreshed)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *subnetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data subnetworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Read subnetwork")

	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *r.token.AccessToken)
	subnetwork, response, err := r.apiClient.SubnetworksAPI.GetSubnetwork(auth, data.Id.ValueString()).Execute()
	if err != nil {
		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}
		if statusCode == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resourceErr := errors.NewError("emma_subnetwork", "Read").
			WithID(data.Id.ValueString()).
			WithStatusCode(statusCode).
			WithAPIError(tools.ExtractErrorMessage(response)).
			WithMessage(errors.MapHTTPError(statusCode, tools.ExtractErrorMessage(response))).
			WithCause(err).
			Build()
		resp.Diagnostics.AddError("Client Error", resourceErr.Error())
		return
	}

	ConvertSubnetworkResponseToResource(&data, subnetwork)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *subnetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData subnetworkResourceModel
	var stateData subnetworkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Update subnetwork")

	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *r.token.AccessToken)

	subnetworkEdit := emmaSdk.SubnetworkEdit{}
	if !planData.Name.IsNull() && !planData.Name.IsUnknown() {
		name := planData.Name.ValueString()
		subnetworkEdit.Name = &name
	}

	subnetwork, response, err := r.apiClient.SubnetworksAPI.SubnetworkUpdate(auth, stateData.Id.ValueString()).SubnetworkEdit(subnetworkEdit).Execute()
	if err != nil {
		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}
		resourceErr := errors.NewError("emma_subnetwork", "Update").
			WithID(stateData.Id.ValueString()).
			WithStatusCode(statusCode).
			WithAPIError(tools.ExtractErrorMessage(response)).
			WithMessage(errors.MapHTTPError(statusCode, tools.ExtractErrorMessage(response))).
			WithCause(err).
			Build()
		resp.Diagnostics.AddError("Client Error", resourceErr.Error())
		return
	}

	ConvertSubnetworkResponseToResource(&stateData, subnetwork)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
}

func (r *subnetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data subnetworkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Delete subnetwork")

	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *r.token.AccessToken)
	_, response, err := r.apiClient.SubnetworksAPI.SubnetworkDelete(auth, data.Id.ValueString()).Execute()
	if err != nil {
		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}
		resourceErr := errors.NewError("emma_subnetwork", "Delete").
			WithID(data.Id.ValueString()).
			WithStatusCode(statusCode).
			WithAPIError(tools.ExtractErrorMessage(response)).
			WithMessage(errors.MapHTTPError(statusCode, tools.ExtractErrorMessage(response))).
			WithCause(err).
			Build()
		resp.Diagnostics.AddError("Client Error", resourceErr.Error())
		return
	}
}

func (r *subnetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "Import subnetwork")
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	r.Read(ctx, resource.ReadRequest{State: resp.State, Private: resp.Private},
		&resource.ReadResponse{State: resp.State, Private: resp.Private, Diagnostics: resp.Diagnostics})
}

func ConvertSubnetworkResponseToResource(data *subnetworkResourceModel, subnetwork *emmaSdk.Subnetwork) {
	if subnetwork.Id != nil {
		data.Id = types.StringValue(*subnetwork.Id)
	}
	if subnetwork.Name != nil {
		data.Name = types.StringValue(*subnetwork.Name)
	}
	if subnetwork.DataCenterId != nil {
		data.DataCenterId = types.StringValue(*subnetwork.DataCenterId)
	}
	if subnetwork.SubnetworkPrefix != nil {
		data.SubnetworkPrefix = types.StringValue(*subnetwork.SubnetworkPrefix)
	}
	if subnetwork.SubnetworkSize != nil {
		data.SubnetworkSize = types.Int64Value(int64(*subnetwork.SubnetworkSize))
	}
	if subnetwork.Status != nil {
		data.Status = types.StringValue(*subnetwork.Status)
	}
}
