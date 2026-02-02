package emma

import (
	"context"
	"fmt"
	emmaSdk "github.com/emma-community/emma-go-sdk"
	emma "github.com/emma-community/terraform-provider-emma/internal/emma/validation"
	"github.com/emma-community/terraform-provider-emma/tools"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &volumeResource{}

func NewVolumeResource() resource.Resource {
	return &volumeResource{}
}

// volumeResource defines the resource implementation.
type volumeResource struct {
	apiClient *emmaSdk.APIClient
	token     *emmaSdk.Token
}

// volumeResourceModel describes the resource data model.
type volumeResourceModel struct {
	Id           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	DataCenterId types.String `tfsdk:"data_center_id"`
	VolumeGb     types.Int64  `tfsdk:"volume_gb"`
	VolumeType   types.String `tfsdk:"volume_type"`
	AttachedToId types.Int64  `tfsdk:"attached_to_id"`
	IsSystem     types.Bool   `tfsdk:"is_system"`
	Status       types.String `tfsdk:"status"`
	ProjectId    types.Int64  `tfsdk:"project_id"`
	Provider     types.Object `tfsdk:"provider"`
	Location     types.Object `tfsdk:"location"`
	DataCenter   types.Object `tfsdk:"data_center"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

type volumeResourceProviderModel struct {
	Id   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type volumeResourceLocationModel struct {
	Id        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Continent types.String `tfsdk:"continent"`
	Region    types.String `tfsdk:"region"`
}

type volumeResourceDataCenterModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (r *volumeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (r *volumeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This resource creates and manages storage volumes in the Emma platform.\n\n" +
			"Volumes are block storage devices that can be attached to compute instances (VMs) for persistent data storage. " +
			"To create a volume, you need to specify the data center, size in gigabytes, and volume type (e.g., ssd, hdd).\n\n" +
			"Volumes can be created independently or attached to a compute instance during creation. " +
			"You can also resize volumes (increase size only) and change attachments after creation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "ID of the volume",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "Name of the volume",
				Optional:    true,
				Computed:    true,
			},
			"data_center_id": schema.StringAttribute{
				Description:   "Data center ID where the volume will be created, volume will be recreated after changing this value",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{emma.ValidDataCenterId{}},
			},
			"volume_gb": schema.Int64Attribute{
				Description: "Volume size in gigabytes, can only be increased",
				Required:    true,
				Validators:  []validator.Int64{emma.MinimumVolumeSize{}, emma.PositiveInt64{}},
			},
			"volume_type": schema.StringAttribute{
				Description:   "Volume type (e.g., ssd, hdd), volume will be recreated after changing this value",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{emma.NonEmptyVolumeType{}},
			},
			"attached_to_id": schema.Int64Attribute{
				Description: "ID of the compute instance to attach the volume to",
				Optional:    true,
				Validators:  []validator.Int64{emma.PositiveInt64{}},
			},
			"is_system": schema.BoolAttribute{
				Description: "Indicates whether the volume contains the operating system",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status of the volume",
				Computed:    true,
			},
			"project_id": schema.Int64Attribute{
				Description: "Project ID owning the volume",
				Computed:    true,
			},
			"provider": schema.SingleNestedAttribute{
				Description: "Cloud provider information",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "Provider ID",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Provider name",
						Computed:    true,
					},
				},
			},
			"location": schema.SingleNestedAttribute{
				Description: "Geographic location information",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Description: "Location ID",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Location name",
						Computed:    true,
					},
					"continent": schema.StringAttribute{
						Description: "Continent",
						Computed:    true,
					},
					"region": schema.StringAttribute{
						Description: "Region",
						Computed:    true,
					},
				},
			},
			"data_center": schema.SingleNestedAttribute{
				Description: "Data center details",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "Data center ID",
						Computed:    true,
					},
					"name": schema.StringAttribute{
						Description: "Data center name",
						Computed:    true,
					},
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Creation timestamp",
				Computed:    true,
			},
		},
	}
}

func (r *volumeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData))
		return
	}
	r.apiClient = client.apiClient
	r.token = client.token
}

func (r *volumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data volumeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build VolumeCreate request from resource model using helper function
	volumeCreateRequest := convertResourceToVolumeCreateRequest(&data)

	// Call Emma API to create volume
	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *r.token.AccessToken)
	volume, response, err := r.apiClient.VolumesAPI.VolumeCreate(auth).VolumeCreate(*volumeCreateRequest).Execute()

	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to create volume, got error: %s",
				tools.ExtractErrorMessage(response)))
		return
	}

	// Convert API response to resource model
	convertVolumeResponseToResource(ctx, &data, volume, resp.Diagnostics)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *volumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data volumeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Extract volume ID from state
	volumeId := tools.StringToInt32(data.Id.ValueString())

	// Call Emma API to get volume
	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *r.token.AccessToken)
	volume, response, err := r.apiClient.VolumesAPI.GetVolume(auth, volumeId).Execute()

	if err != nil {
		// Handle 404 errors by removing from state
		if response != nil && response.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read volume, got error: %s",
				tools.ExtractErrorMessage(response)))
		return
	}

	// Update all computed attributes
	convertVolumeResponseToResource(ctx, &data, volume, resp.Diagnostics)

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *volumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan volumeResourceModel
	var state volumeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract volume ID from state
	volumeId := tools.StringToInt32(state.Id.ValueString())
	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *r.token.AccessToken)

	// Handle volume resize if volume_gb changed
	if !plan.VolumeGb.Equal(state.VolumeGb) {
		// Validate that size is increasing
		if plan.VolumeGb.ValueInt64() < state.VolumeGb.ValueInt64() {
			resp.Diagnostics.AddError("Validation Error",
				fmt.Sprintf("Volume size can only be increased. Current size: %d GB, requested size: %d GB",
					state.VolumeGb.ValueInt64(), plan.VolumeGb.ValueInt64()))
			return
		}

		// Create VolumeEdit request for resize using helper function
		volumeEdit := convertResourceToVolumeEditRequest(&plan)
		_, response, err := r.apiClient.VolumesAPI.VolumeActions(auth, volumeId).VolumeEdit(*volumeEdit).Execute()
		if err != nil {
			resp.Diagnostics.AddError("Client Error",
				fmt.Sprintf("Unable to resize volume, got error: %s",
					tools.ExtractErrorMessage(response)))
			return
		}
	}

	// Handle attachment changes
	planAttachedToId := plan.AttachedToId
	stateAttachedToId := state.AttachedToId

	// Check if attachment changed
	if !planAttachedToId.Equal(stateAttachedToId) {
		// Detach from old instance if currently attached
		if !stateAttachedToId.IsNull() && !stateAttachedToId.IsUnknown() {
			oldVmId := int32(stateAttachedToId.ValueInt64())
			volumeDetach := emmaSdk.NewVolumeDetach("detach", volumeId)
			
			// Create VmActionsRequest with VolumeDetach
			vmActionsReq := emmaSdk.VolumeDetachAsVmActionsRequest(volumeDetach)
			
			_, response, err := r.apiClient.VirtualMachinesAPI.VmActions(auth, oldVmId).VmActionsRequest(vmActionsReq).Execute()
			if err != nil {
				resp.Diagnostics.AddError("Client Error",
					fmt.Sprintf("Unable to detach volume from VM %d, got error: %s",
						oldVmId, tools.ExtractErrorMessage(response)))
				return
			}
		}

		// Attach to new instance if specified
		if !planAttachedToId.IsNull() && !planAttachedToId.IsUnknown() {
			newVmId := int32(planAttachedToId.ValueInt64())
			volumeAttach := emmaSdk.NewVolumeAttach("attach", volumeId)
			
			// Create VmActionsRequest with VolumeAttach
			vmActionsReq := emmaSdk.VolumeAttachAsVmActionsRequest(volumeAttach)
			
			_, response, err := r.apiClient.VirtualMachinesAPI.VmActions(auth, newVmId).VmActionsRequest(vmActionsReq).Execute()
			if err != nil {
				resp.Diagnostics.AddError("Client Error",
					fmt.Sprintf("Unable to attach volume to VM %d, got error: %s",
						newVmId, tools.ExtractErrorMessage(response)))
				return
			}
		}
	}

	// Refresh volume state after updates
	volume, response, err := r.apiClient.VolumesAPI.GetVolume(auth, volumeId).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read volume after update, got error: %s",
				tools.ExtractErrorMessage(response)))
		return
	}

	// Convert API response to resource model
	convertVolumeResponseToResource(ctx, &plan, volume, resp.Diagnostics)

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data volumeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Extract volume ID from state
	volumeId := tools.StringToInt32(data.Id.ValueString())
	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *r.token.AccessToken)

	// Check if volume is a system volume
	if !data.IsSystem.IsNull() && !data.IsSystem.IsUnknown() && data.IsSystem.ValueBool() {
		resp.Diagnostics.AddError("Validation Error",
			fmt.Sprintf("Cannot delete system volume %d. System volumes contain the operating system and cannot be deleted.",
				volumeId))
		return
	}

	// Check if volume is attached and detach if necessary
	if !data.AttachedToId.IsNull() && !data.AttachedToId.IsUnknown() {
		vmId := int32(data.AttachedToId.ValueInt64())
		volumeDetach := emmaSdk.NewVolumeDetach("detach", volumeId)

		// Create VmActionsRequest with VolumeDetach
		vmActionsReq := emmaSdk.VolumeDetachAsVmActionsRequest(volumeDetach)

		_, response, err := r.apiClient.VirtualMachinesAPI.VmActions(auth, vmId).VmActionsRequest(vmActionsReq).Execute()
		if err != nil {
			resp.Diagnostics.AddError("Client Error",
				fmt.Sprintf("Unable to detach volume %d from VM %d before deletion, got error: %s",
					volumeId, vmId, tools.ExtractErrorMessage(response)))
			return
		}
	}

	// Call Emma API to delete volume
	_, response, err := r.apiClient.VolumesAPI.VolumeDelete(auth, volumeId).Execute()

	if err != nil {
		// Handle 404 errors as successful deletion (idempotent)
		if response != nil && response.StatusCode == 404 {
			// Volume already deleted, treat as success
			return
		}

		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to delete volume %d, got error: %s",
				volumeId, tools.ExtractErrorMessage(response)))
		return
	}

	// Resource is automatically removed from state after successful Delete
}

func (r *volumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Use ImportStatePassthroughID to set the ID from the import string
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// Call Read operation to populate the rest of the state
	r.Read(ctx, resource.ReadRequest{State: resp.State, Private: resp.Private},
		&resource.ReadResponse{State: resp.State, Private: resp.Private, Diagnostics: resp.Diagnostics})
}

// Helper function to get attribute types for provider nested object
func (o volumeResourceProviderModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.Int64Type,
		"name": types.StringType,
	}
}

// Helper function to get attribute types for location nested object
func (o volumeResourceLocationModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":        types.Int64Type,
		"name":      types.StringType,
		"continent": types.StringType,
		"region":    types.StringType,
	}
}

// Helper function to get attribute types for data center nested object
func (o volumeResourceDataCenterModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
	}
}

// convertVolumeResponseToResource converts Emma API Volume response to Terraform resource model
func convertVolumeResponseToResource(ctx context.Context, data *volumeResourceModel, volume *emmaSdk.Volume, diags diag.Diagnostics) {
	// Set ID
	if volume.Id != nil {
		data.Id = types.StringValue(fmt.Sprintf("%d", *volume.Id))
	}

	// Set basic attributes
	if volume.Name != nil {
		data.Name = types.StringValue(*volume.Name)
	} else {
		data.Name = types.StringNull()
	}

	if volume.SizeGb != nil {
		data.VolumeGb = types.Int64Value(int64(*volume.SizeGb))
	}
	if volume.Type != nil {
		data.VolumeType = types.StringValue(*volume.Type)
	}
	if volume.IsSystem != nil {
		data.IsSystem = types.BoolValue(*volume.IsSystem)
	}
	if volume.Status != nil {
		data.Status = types.StringValue(*volume.Status)
	}
	if volume.ProjectId != nil {
		data.ProjectId = types.Int64Value(int64(*volume.ProjectId))
	}

	// Set attached_to_id
	if volume.AttachedToId != nil {
		data.AttachedToId = types.Int64Value(int64(*volume.AttachedToId))
	} else {
		data.AttachedToId = types.Int64Null()
	}

	// Set created_at
	if volume.CreatedAt != nil {
		data.CreatedAt = types.StringValue(*volume.CreatedAt)
	} else {
		data.CreatedAt = types.StringNull()
	}

	// Convert provider nested object
	if volume.Provider != nil {
		providerModel := volumeResourceProviderModel{
			Id:   types.Int64Value(int64(*volume.Provider.Id)),
			Name: types.StringValue(*volume.Provider.Name),
		}
		providerObj, providerDiag := types.ObjectValueFrom(ctx, providerModel.attrTypes(), providerModel)
		data.Provider = providerObj
		diags.Append(providerDiag...)
	} else {
		data.Provider = types.ObjectNull(volumeResourceProviderModel{}.attrTypes())
	}

	// Convert location nested object
	if volume.Location != nil {
		locationModel := volumeResourceLocationModel{
			Id:   types.Int64Value(int64(*volume.Location.Id)),
			Name: types.StringValue(*volume.Location.Name),
		}
		if volume.Location.Continent != nil {
			locationModel.Continent = types.StringValue(*volume.Location.Continent)
		} else {
			locationModel.Continent = types.StringNull()
		}
		if volume.Location.Region != nil {
			locationModel.Region = types.StringValue(*volume.Location.Region)
		} else {
			locationModel.Region = types.StringNull()
		}
		locationObj, locationDiag := types.ObjectValueFrom(ctx, locationModel.attrTypes(), locationModel)
		data.Location = locationObj
		diags.Append(locationDiag...)
	} else {
		data.Location = types.ObjectNull(volumeResourceLocationModel{}.attrTypes())
	}

	// Convert data center nested object
	if volume.DataCenter != nil {
		dataCenterModel := volumeResourceDataCenterModel{
			Id:   types.StringValue(*volume.DataCenter.Id),
			Name: types.StringValue(*volume.DataCenter.Name),
		}
		dataCenterObj, dataCenterDiag := types.ObjectValueFrom(ctx, dataCenterModel.attrTypes(), dataCenterModel)
		data.DataCenter = dataCenterObj
		diags.Append(dataCenterDiag...)
	} else {
		data.DataCenter = types.ObjectNull(volumeResourceDataCenterModel{}.attrTypes())
	}

	// Set data_center_id from the data center object (for consistency with API response)
	if volume.DataCenter != nil && volume.DataCenter.Id != nil {
		data.DataCenterId = types.StringValue(*volume.DataCenter.Id)
	}
}

// convertResourceToVolumeCreateRequest converts Terraform resource model to SDK VolumeCreate request
func convertResourceToVolumeCreateRequest(data *volumeResourceModel) *emmaSdk.VolumeCreate {
	// Create VolumeCreate request with required fields
	volumeCreateRequest := emmaSdk.NewVolumeCreate(
		data.DataCenterId.ValueString(),
		int32(data.VolumeGb.ValueInt64()),
		data.VolumeType.ValueString(),
	)

	// Add optional attached_to_id if provided
	if !data.AttachedToId.IsNull() && !data.AttachedToId.IsUnknown() {
		volumeCreateRequest.SetAttachedToId(int32(data.AttachedToId.ValueInt64()))
	}

	return volumeCreateRequest
}

// convertResourceToVolumeEditRequest converts Terraform resource model to SDK VolumeEdit request
func convertResourceToVolumeEditRequest(data *volumeResourceModel) *emmaSdk.VolumeEdit {
	// Create VolumeEdit request with action and new size
	volumeEdit := emmaSdk.NewVolumeEdit(
		"edit",
		int32(data.VolumeGb.ValueInt64()),
	)

	return volumeEdit
}
