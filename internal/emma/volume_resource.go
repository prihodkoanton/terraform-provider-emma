package emma

import (
	"context"
	"fmt"
	emmaSdk "github.com/emma-community/emma-go-sdk"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/async"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/convert"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/errors"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/retry"
	"github.com/emma-community/terraform-provider-emma/internal/emma/common/state"
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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"net/http"
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
	ProjectId      types.Int64  `tfsdk:"project_id"`
	CloudProvider  types.Object `tfsdk:"cloud_provider"`
	Location       types.Object `tfsdk:"location"`
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
				Description:   "Name of the volume",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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
			"cloud_provider": schema.SingleNestedAttribute{
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
		statusCode := 0
		apiError := ""
		if response != nil {
			statusCode = response.StatusCode
			apiError = tools.ExtractErrorMessage(response)
		}
		
		resourceErr := errors.NewError("emma_volume", "Create").
			WithStatusCode(statusCode).
			WithAPIError(apiError).
			WithMessage(errors.MapHTTPError(statusCode, apiError)).
			Build()
		
		resp.Diagnostics.AddError("Client Error", resourceErr.Error())
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
	volumeId, err := convert.StringToInt32(data.Id)
	if err != nil {
		resourceErr := errors.NewError("emma_volume", "Read").
			WithID(data.Id.ValueString()).
			WithMessage(fmt.Sprintf("Invalid volume ID: %v", err)).
			Build()
		
		resp.Diagnostics.AddError("Validation Error", resourceErr.Error())
		return
	}

	// Call Emma API to get volume
	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *r.token.AccessToken)
	volume, response, err := r.apiClient.VolumesAPI.GetVolume(auth, volumeId).Execute()

	if err != nil {
		// Handle 404 errors by removing from state using StateManager
		if response != nil && response.StatusCode == 404 {
			stateManager := state.NewStateManager(ctx)
			stateManager.RemoveFromState(resp)
			return
		}

		statusCode := 0
		apiError := ""
		if response != nil {
			statusCode = response.StatusCode
			apiError = tools.ExtractErrorMessage(response)
		}
		
		resourceErr := errors.NewError("emma_volume", "Read").
			WithID(data.Id.ValueString()).
			WithStatusCode(statusCode).
			WithAPIError(apiError).
			WithMessage(errors.MapHTTPError(statusCode, apiError)).
			Build()
		
		resp.Diagnostics.AddError("Client Error", resourceErr.Error())
		return
	}

	// Update all computed attributes
	convertVolumeResponseToResource(ctx, &data, volume, resp.Diagnostics)

	// Save updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *volumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan volumeResourceModel
	var stateData volumeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract volume ID from state
	volumeId, err := convert.StringToInt32(stateData.Id)
	if err != nil {
		resourceErr := errors.NewError("emma_volume", "Update").
			WithID(stateData.Id.ValueString()).
			WithMessage(fmt.Sprintf("Invalid volume ID: %v", err)).
			Build()
		
		resp.Diagnostics.AddError("Validation Error", resourceErr.Error())
		return
	}
	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *r.token.AccessToken)

	// Preserve the planned attached_to_id for later comparison
	planAttachedToId := plan.AttachedToId
	stateAttachedToId := stateData.AttachedToId

	// Handle volume resize if volume_gb changed
	if !plan.VolumeGb.Equal(stateData.VolumeGb) {
		// Validate that size is increasing
		if plan.VolumeGb.ValueInt64() < stateData.VolumeGb.ValueInt64() {
			resourceErr := errors.NewError("emma_volume", "Update").
				WithID(stateData.Id.ValueString()).
				WithMessage(fmt.Sprintf("Volume size can only be increased. Current size: %d GB, requested size: %d GB",
					stateData.VolumeGb.ValueInt64(), plan.VolumeGb.ValueInt64())).
				Build()
			
			resp.Diagnostics.AddError("Validation Error", resourceErr.Error())
			return
		}

		tflog.Debug(ctx, "Waiting for volume to reach stable state before resize", map[string]interface{}{
			"volume_id": stateData.Id.ValueString(),
		})

		// Wait for volume to reach stable state before resize
		stateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
			ResourceType: "volume",
			ResourceID:   stateData.Id.ValueString(),
			StatusChecker: func(ctx context.Context) (string, error) {
				vol, _, err := r.apiClient.VolumesAPI.GetVolume(auth, volumeId).Execute()
				if err != nil {
					return "", err
				}
				if vol.Status == nil {
					return "", fmt.Errorf("volume status is nil")
				}
				return *vol.Status, nil
			},
			TargetStates:       state.VolumeStableStates,
			TransitionalStates: state.VolumeTransitionalStates,
			FailureStates:      state.VolumeFailureStates,
			Timeout:            async.DefaultTimeout,
			PollInterval:       async.DefaultPollInterval,
		})

		if err := stateManager.WaitForStableState(auth); err != nil {
			resp.Diagnostics.AddError("State Transition Error",
				fmt.Sprintf("Volume did not reach stable state before resize: %s", err.Error()))
			return
		}

		tflog.Info(ctx, "Volume reached stable state, proceeding with resize", map[string]interface{}{
			"volume_id": stateData.Id.ValueString(),
		})

		// Perform resize with retry on state conflicts
		retryConfig := retry.StateConflictRetryConfig()
		var lastResponse *http.Response
		var lastAPIError string

		err := retry.Retry(auth, retryConfig, func() error {
			volumeEdit := convertResourceToVolumeEditRequest(&plan)
			_, response, err := r.apiClient.VolumesAPI.VolumeActions(auth, volumeId).VolumeEdit(*volumeEdit).Execute()

			lastResponse = response
			if err != nil {
				lastAPIError = tools.ExtractErrorMessage(response)
				statusCode := 0
				if response != nil {
					statusCode = response.StatusCode
				}

				// Check if this is a state conflict error that should be retried
				if retry.IsStateConflictError(err, statusCode, lastAPIError) {
					tflog.Warn(ctx, "Volume resize failed due to state conflict, will retry", map[string]interface{}{
						"volume_id":   stateData.Id.ValueString(),
						"status_code": statusCode,
						"error":       lastAPIError,
					})
					return err
				}

				// Non-retryable error
				return fmt.Errorf("non-retryable error: %w", err)
			}

			return nil
		})

		if err != nil {
			statusCode := 0
			apiError := ""
			if lastResponse != nil {
				statusCode = lastResponse.StatusCode
				apiError = lastAPIError
			}
			
			resourceErr := errors.NewError("emma_volume", "Update").
				WithID(stateData.Id.ValueString()).
				WithStatusCode(statusCode).
				WithAPIError(apiError).
				WithMessage(errors.MapHTTPError(statusCode, apiError)).
				Build()
			
			resp.Diagnostics.AddError("Client Error", resourceErr.Error())
			return
		}

		tflog.Info(ctx, "Volume resize completed successfully", map[string]interface{}{
			"volume_id": stateData.Id.ValueString(),
		})
	}

	// Handle attachment changes
	// Check if attachment changed
	if !planAttachedToId.Equal(stateAttachedToId) {
		// Detach from old instance if currently attached
		if !stateAttachedToId.IsNull() && !stateAttachedToId.IsUnknown() {
			oldVmId, err := convert.Int64ToInt32(stateAttachedToId)
			if err != nil {
				resourceErr := errors.NewError("emma_volume", "Update").
					WithID(stateData.Id.ValueString()).
					WithMessage(fmt.Sprintf("Invalid VM ID for detachment: %v", err)).
					Build()
				
				resp.Diagnostics.AddError("Validation Error", resourceErr.Error())
				return
			}

			tflog.Debug(ctx, "Waiting for VM to reach stable state before volume detach", map[string]interface{}{
				"vm_id":     oldVmId,
				"volume_id": stateData.Id.ValueString(),
			})

			// Wait for VM to reach stable state before detach
			vmStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   fmt.Sprintf("%d", oldVmId),
				StatusChecker: func(ctx context.Context) (string, error) {
					vm, _, err := r.apiClient.VirtualMachinesAPI.GetVm(auth, oldVmId).Execute()
					if err != nil {
						return "", err
					}
					if vm.Status == nil {
						return "", fmt.Errorf("VM status is nil")
					}
					return *vm.Status, nil
				},
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            async.DefaultTimeout,
				PollInterval:       async.DefaultPollInterval,
			})

			if err := vmStateManager.WaitForStableState(auth); err != nil {
				resp.Diagnostics.AddError("State Transition Error",
					fmt.Sprintf("VM did not reach stable state before volume detach: %s", err.Error()))
				return
			}

			tflog.Debug(ctx, "Waiting for volume to reach stable state before detach", map[string]interface{}{
				"volume_id": stateData.Id.ValueString(),
			})

			// Wait for volume to reach stable state before detach
			volumeStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "volume",
				ResourceID:   stateData.Id.ValueString(),
				StatusChecker: func(ctx context.Context) (string, error) {
					vol, _, err := r.apiClient.VolumesAPI.GetVolume(auth, volumeId).Execute()
					if err != nil {
						return "", err
					}
					if vol.Status == nil {
						return "", fmt.Errorf("volume status is nil")
					}
					return *vol.Status, nil
				},
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
				Timeout:            async.DefaultTimeout,
				PollInterval:       async.DefaultPollInterval,
			})

			if err := volumeStateManager.WaitForStableState(auth); err != nil {
				resp.Diagnostics.AddError("State Transition Error",
					fmt.Sprintf("Volume did not reach stable state before detach: %s", err.Error()))
				return
			}

			tflog.Info(ctx, "VM and volume reached stable state, proceeding with detach", map[string]interface{}{
				"vm_id":     oldVmId,
				"volume_id": stateData.Id.ValueString(),
			})

			// Perform detach with retry on state conflicts
			retryConfig := retry.StateConflictRetryConfig()
			var lastResponse *http.Response
			var lastAPIError string

			err = retry.Retry(auth, retryConfig, func() error {
				volumeDetach := emmaSdk.NewVolumeDetach("detach", volumeId)
				vmActionsReq := emmaSdk.VolumeDetachAsVmActionsRequest(volumeDetach)
				
				_, response, err := r.apiClient.VirtualMachinesAPI.VmActions(auth, oldVmId).VmActionsRequest(vmActionsReq).Execute()

				lastResponse = response
				if err != nil {
					lastAPIError = tools.ExtractErrorMessage(response)
					statusCode := 0
					if response != nil {
						statusCode = response.StatusCode
					}

					// Check if this is a state conflict error that should be retried
					if retry.IsStateConflictError(err, statusCode, lastAPIError) {
						tflog.Warn(ctx, "Volume detach failed due to state conflict, will retry", map[string]interface{}{
							"vm_id":       oldVmId,
							"volume_id":   stateData.Id.ValueString(),
							"status_code": statusCode,
							"error":       lastAPIError,
						})
						return err
					}

					// Non-retryable error
					return fmt.Errorf("non-retryable error: %w", err)
				}

				return nil
			})

			if err != nil {
				statusCode := 0
				apiError := ""
				if lastResponse != nil {
					statusCode = lastResponse.StatusCode
					apiError = lastAPIError
				}
				
				resourceErr := errors.NewError("emma_volume", "Update").
					WithID(stateData.Id.ValueString()).
					WithStatusCode(statusCode).
					WithAPIError(apiError).
					WithMessage(fmt.Sprintf("Unable to detach volume from VM %d: %s", oldVmId, errors.MapHTTPError(statusCode, apiError))).
					Build()
				
				resp.Diagnostics.AddError("Client Error", resourceErr.Error())
				return
			}

			tflog.Info(ctx, "Volume detached successfully", map[string]interface{}{
				"vm_id":     oldVmId,
				"volume_id": stateData.Id.ValueString(),
			})
		}

		// Attach to new instance if specified
		if !planAttachedToId.IsNull() && !planAttachedToId.IsUnknown() {
			newVmId, err := convert.Int64ToInt32(planAttachedToId)
			if err != nil {
				resourceErr := errors.NewError("emma_volume", "Update").
					WithID(stateData.Id.ValueString()).
					WithMessage(fmt.Sprintf("Invalid VM ID for attachment: %v", err)).
					Build()
				
				resp.Diagnostics.AddError("Validation Error", resourceErr.Error())
				return
			}

			tflog.Debug(ctx, "Waiting for VM to reach stable state before volume attach", map[string]interface{}{
				"vm_id":     newVmId,
				"volume_id": stateData.Id.ValueString(),
			})

			// Wait for VM to reach stable state before attach
			vmStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "vm",
				ResourceID:   fmt.Sprintf("%d", newVmId),
				StatusChecker: func(ctx context.Context) (string, error) {
					vm, _, err := r.apiClient.VirtualMachinesAPI.GetVm(auth, newVmId).Execute()
					if err != nil {
						return "", err
					}
					if vm.Status == nil {
						return "", fmt.Errorf("VM status is nil")
					}
					return *vm.Status, nil
				},
				TargetStates:       state.VMStableStates,
				TransitionalStates: state.VMTransitionalStates,
				FailureStates:      state.VMFailureStates,
				Timeout:            async.DefaultTimeout,
				PollInterval:       async.DefaultPollInterval,
			})

			if err := vmStateManager.WaitForStableState(auth); err != nil {
				resp.Diagnostics.AddError("State Transition Error",
					fmt.Sprintf("VM did not reach stable state before volume attach: %s", err.Error()))
				return
			}

			tflog.Debug(ctx, "Waiting for volume to reach stable state before attach", map[string]interface{}{
				"volume_id": stateData.Id.ValueString(),
			})

			// Wait for volume to reach stable state before attach
			volumeStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
				ResourceType: "volume",
				ResourceID:   stateData.Id.ValueString(),
				StatusChecker: func(ctx context.Context) (string, error) {
					vol, _, err := r.apiClient.VolumesAPI.GetVolume(auth, volumeId).Execute()
					if err != nil {
						return "", err
					}
					if vol.Status == nil {
						return "", fmt.Errorf("volume status is nil")
					}
					return *vol.Status, nil
				},
				TargetStates:       state.VolumeStableStates,
				TransitionalStates: state.VolumeTransitionalStates,
				FailureStates:      state.VolumeFailureStates,
				Timeout:            async.DefaultTimeout,
				PollInterval:       async.DefaultPollInterval,
			})

			if err := volumeStateManager.WaitForStableState(auth); err != nil {
				resp.Diagnostics.AddError("State Transition Error",
					fmt.Sprintf("Volume did not reach stable state before attach: %s", err.Error()))
				return
			}

			tflog.Info(ctx, "VM and volume reached stable state, proceeding with attach", map[string]interface{}{
				"vm_id":     newVmId,
				"volume_id": stateData.Id.ValueString(),
			})

			// Perform attach with retry on state conflicts
			retryConfig := retry.StateConflictRetryConfig()
			var lastResponse *http.Response
			var lastAPIError string

			err = retry.Retry(auth, retryConfig, func() error {
				volumeAttach := emmaSdk.NewVolumeAttach("attach", volumeId)
				vmActionsReq := emmaSdk.VolumeAttachAsVmActionsRequest(volumeAttach)
				
				_, response, err := r.apiClient.VirtualMachinesAPI.VmActions(auth, newVmId).VmActionsRequest(vmActionsReq).Execute()

				lastResponse = response
				if err != nil {
					lastAPIError = tools.ExtractErrorMessage(response)
					statusCode := 0
					if response != nil {
						statusCode = response.StatusCode
					}

					// Check if this is a state conflict error that should be retried
					if retry.IsStateConflictError(err, statusCode, lastAPIError) {
						tflog.Warn(ctx, "Volume attach failed due to state conflict, will retry", map[string]interface{}{
							"vm_id":       newVmId,
							"volume_id":   stateData.Id.ValueString(),
							"status_code": statusCode,
							"error":       lastAPIError,
						})
						return err
					}

					// Non-retryable error
					return fmt.Errorf("non-retryable error: %w", err)
				}

				return nil
			})

			if err != nil {
				statusCode := 0
				apiError := ""
				if lastResponse != nil {
					statusCode = lastResponse.StatusCode
					apiError = lastAPIError
				}
				
				resourceErr := errors.NewError("emma_volume", "Update").
					WithID(stateData.Id.ValueString()).
					WithStatusCode(statusCode).
					WithAPIError(apiError).
					WithMessage(fmt.Sprintf("Unable to attach volume to VM %d: %s", newVmId, errors.MapHTTPError(statusCode, apiError))).
					Build()
				
				resp.Diagnostics.AddError("Client Error", resourceErr.Error())
				return
			}

			tflog.Info(ctx, "Volume attached successfully", map[string]interface{}{
				"vm_id":     newVmId,
				"volume_id": stateData.Id.ValueString(),
			})
		}
	}

	// Refresh volume state after updates
	volume, response, err := r.apiClient.VolumesAPI.GetVolume(auth, volumeId).Execute()
	if err != nil {
		statusCode := 0
		apiError := ""
		if response != nil {
			statusCode = response.StatusCode
			apiError = tools.ExtractErrorMessage(response)
		}
		
		resourceErr := errors.NewError("emma_volume", "Update").
			WithID(stateData.Id.ValueString()).
			WithStatusCode(statusCode).
			WithAPIError(apiError).
			WithMessage(errors.MapHTTPError(statusCode, apiError)).
			Build()
		
		resp.Diagnostics.AddError("Client Error", resourceErr.Error())
		return
	}

	// Convert API response to resource model
	convertVolumeResponseToResource(ctx, &plan, volume, resp.Diagnostics)

	// Preserve the planned attached_to_id if attachment just changed
	// The API may return null temporarily during the attachment transition
	if !planAttachedToId.Equal(stateAttachedToId) {
		plan.AttachedToId = planAttachedToId
	}

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
	volumeId, err := convert.StringToInt32(data.Id)
	if err != nil {
		resourceErr := errors.NewError("emma_volume", "Delete").
			WithID(data.Id.ValueString()).
			WithMessage(fmt.Sprintf("Invalid volume ID: %v", err)).
			Build()
		
		resp.Diagnostics.AddError("Validation Error", resourceErr.Error())
		return
	}
	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *r.token.AccessToken)

	// Check if volume is a system volume
	if !data.IsSystem.IsNull() && !data.IsSystem.IsUnknown() && data.IsSystem.ValueBool() {
		resourceErr := errors.NewError("emma_volume", "Delete").
			WithID(data.Id.ValueString()).
			WithMessage("Cannot delete system volume. System volumes contain the operating system and cannot be deleted.").
			Build()
		
		resp.Diagnostics.AddError("Validation Error", resourceErr.Error())
		return
	}

	// Check if volume is attached
	// Note: During destroy, we don't need to explicitly detach volumes from VMs
	// The Emma API automatically detaches volumes when a VM is deleted
	// We only need to detach if we're changing attachment, not during destroy
	if !data.AttachedToId.IsNull() && !data.AttachedToId.IsUnknown() {
		vmId, err := convert.Int64ToInt32(data.AttachedToId)
		if err != nil {
			resourceErr := errors.NewError("emma_volume", "Delete").
				WithID(data.Id.ValueString()).
				WithMessage(fmt.Sprintf("Invalid VM ID for detachment: %v", err)).
				Build()
			
			resp.Diagnostics.AddError("Validation Error", resourceErr.Error())
			return
		}

		tflog.Info(ctx, "Volume is attached to VM, checking if VM still exists", map[string]interface{}{
			"vm_id":     vmId,
			"volume_id": data.Id.ValueString(),
		})
		
		// Check if VM still exists
		_, response, err := r.apiClient.VirtualMachinesAPI.GetVm(auth, vmId).Execute()
		if err != nil && response != nil && response.StatusCode == 404 {
			tflog.Info(ctx, "VM no longer exists, volume is already detached, proceeding to delete", map[string]interface{}{
				"vm_id":     vmId,
				"volume_id": data.Id.ValueString(),
			})
			// VM is gone, volume is already detached, skip to deletion
			goto deleteVolume
		}
		
		tflog.Info(ctx, "VM still exists, but skipping explicit detach during destroy - API will handle it", map[string]interface{}{
			"vm_id":     vmId,
			"volume_id": data.Id.ValueString(),
		})
		
		// During destroy, we skip the detach step entirely
		// The volume deletion API will handle detachment if needed
		// This avoids race conditions with VM deletion
		goto deleteVolume

		tflog.Debug(ctx, "Waiting for VM to reach stable state before volume detach", map[string]interface{}{
			"vm_id":     vmId,
			"volume_id": data.Id.ValueString(),
		})

		// Wait for VM to reach stable state before detach
		vmStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
			ResourceType: "vm",
			ResourceID:   fmt.Sprintf("%d", vmId),
			StatusChecker: func(ctx context.Context) (string, error) {
				vm, response, err := r.apiClient.VirtualMachinesAPI.GetVm(auth, vmId).Execute()
				if err != nil {
					// If VM is already deleted (404), volume is implicitly detached
					if response != nil && response.StatusCode == 404 {
						tflog.Info(ctx, "VM no longer exists during state check, volume is implicitly detached", map[string]interface{}{
							"vm_id":     vmId,
							"volume_id": data.Id.ValueString(),
						})
						// Return a special marker to indicate VM is gone
						return "DELETED", nil
					}
					return "", err
				}
				if vm.Status == nil {
					return "", fmt.Errorf("VM status is nil")
				}
				return *vm.Status, nil
			},
			TargetStates:       append(state.VMStableStates, "DELETED"), // Add DELETED as a valid target state
			TransitionalStates: state.VMTransitionalStates,
			FailureStates:      state.VMFailureStates,
			Timeout:            async.DefaultTimeout,
			PollInterval:       async.DefaultPollInterval,
		})

		if err := vmStateManager.WaitForStableState(auth); err != nil {
			resp.Diagnostics.AddError("State Transition Error",
				fmt.Sprintf("VM did not reach stable state before volume detach: %s", err.Error()))
			return
		}
		
		// Check if VM was deleted during wait
		vm, response, err := r.apiClient.VirtualMachinesAPI.GetVm(auth, vmId).Execute()
		if err != nil && response != nil && response.StatusCode == 404 {
			tflog.Info(ctx, "VM was deleted, skipping volume detach", map[string]interface{}{
				"vm_id":     vmId,
				"volume_id": data.Id.ValueString(),
			})
			// Skip detach and go straight to volume deletion
			goto deleteVolume
		}
		if err != nil {
			resourceErr := errors.NewError("emma_volume", "Delete").
				WithID(data.Id.ValueString()).
				WithMessage(fmt.Sprintf("Failed to check VM status: %v", err)).
				Build()
			
			resp.Diagnostics.AddError("Client Error", resourceErr.Error())
			return
		}
		if vm.Status != nil && *vm.Status == "DELETED" {
			tflog.Info(ctx, "VM is deleted, skipping volume detach", map[string]interface{}{
				"vm_id":     vmId,
				"volume_id": data.Id.ValueString(),
			})
			// Skip detach and go straight to volume deletion
			goto deleteVolume
		}

		tflog.Debug(ctx, "Waiting for volume to reach stable state before detach", map[string]interface{}{
			"volume_id": data.Id.ValueString(),
		})

		// Wait for volume to reach stable state before detach
		// During destroy, we're more lenient - if volume is already detaching or times out, we proceed anyway
		volumeStateManager := state.NewStateTransitionManager(state.StateTransitionConfig{
			ResourceType: "volume",
			ResourceID:   data.Id.ValueString(),
			StatusChecker: func(ctx context.Context) (string, error) {
				vol, response, err := r.apiClient.VolumesAPI.GetVolume(auth, volumeId).Execute()
				if err != nil {
					// If volume is already deleted (404), that's fine
					if response != nil && response.StatusCode == 404 {
						tflog.Info(ctx, "Volume no longer exists, skipping detach", map[string]interface{}{
							"volume_id": data.Id.ValueString(),
						})
						return "DELETED", nil
					}
					return "", err
				}
				if vol.Status == nil {
					return "", fmt.Errorf("volume status is nil")
				}
				return *vol.Status, nil
			},
			TargetStates:       append(state.VolumeStableStates, "DELETED"), // Add DELETED as valid target
			TransitionalStates: state.VolumeTransitionalStates,
			FailureStates:      state.VolumeFailureStates,
			Timeout:            async.DefaultTimeout,
			PollInterval:       async.DefaultPollInterval,
		})

		if err := volumeStateManager.WaitForStableState(auth); err != nil {
			// During destroy, if we timeout waiting for stable state, log warning but continue
			tflog.Warn(ctx, "Volume did not reach stable state before detach, will attempt detach anyway", map[string]interface{}{
				"volume_id": data.Id.ValueString(),
				"error":     err.Error(),
			})
		}
		
		// Check if volume was deleted during wait
		vol, response, err := r.apiClient.VolumesAPI.GetVolume(auth, volumeId).Execute()
		if err != nil && response != nil && response.StatusCode == 404 {
			tflog.Info(ctx, "Volume was deleted, skipping detach", map[string]interface{}{
				"volume_id": data.Id.ValueString(),
			})
			// Volume is gone, we're done
			return
		}
		if vol != nil && vol.Status != nil && *vol.Status == "DELETED" {
			tflog.Info(ctx, "Volume is deleted, skipping detach", map[string]interface{}{
				"volume_id": data.Id.ValueString(),
			})
			return
		}

		tflog.Info(ctx, "VM and volume reached stable state, proceeding with detach", map[string]interface{}{
			"vm_id":     vmId,
			"volume_id": data.Id.ValueString(),
		})

		// Perform detach with retry on state conflicts
		retryConfig := retry.StateConflictRetryConfig()
		var lastResponse *http.Response
		var lastAPIError string

		err = retry.Retry(auth, retryConfig, func() error {
			volumeDetach := emmaSdk.NewVolumeDetach("detach", volumeId)
			vmActionsReq := emmaSdk.VolumeDetachAsVmActionsRequest(volumeDetach)

			_, response, err := r.apiClient.VirtualMachinesAPI.VmActions(auth, vmId).VmActionsRequest(vmActionsReq).Execute()

			lastResponse = response
			if err != nil {
				lastAPIError = tools.ExtractErrorMessage(response)
				// If VM is already deleted (404), that's fine - volume is already detached
				if response != nil && response.StatusCode == 404 {
					tflog.Info(ctx, "VM no longer exists, volume is implicitly detached", map[string]interface{}{
						"vm_id":     vmId,
						"volume_id": data.Id.ValueString(),
					})
					return nil
				}

				statusCode := 0
				if response != nil {
					statusCode = response.StatusCode
				}

				// Check if this is a state conflict error that should be retried
				if retry.IsStateConflictError(err, statusCode, lastAPIError) {
					tflog.Warn(ctx, "Volume detach failed due to state conflict, will retry", map[string]interface{}{
						"vm_id":       vmId,
						"volume_id":   data.Id.ValueString(),
						"status_code": statusCode,
						"error":       lastAPIError,
					})
					return err
				}

				// Non-retryable error
				return fmt.Errorf("non-retryable error: %w", err)
			}

			return nil
		})

		if err != nil {
			statusCode := 0
			apiError := ""
			if lastResponse != nil {
				statusCode = lastResponse.StatusCode
				apiError = lastAPIError
			}
			
			resourceErr := errors.NewError("emma_volume", "Delete").
				WithID(data.Id.ValueString()).
				WithStatusCode(statusCode).
				WithAPIError(apiError).
				WithMessage(fmt.Sprintf("Unable to detach volume from VM %d before deletion: %s", vmId, errors.MapHTTPError(statusCode, apiError))).
				Build()
			
			resp.Diagnostics.AddError("Client Error", resourceErr.Error())
			return
		}

		tflog.Info(ctx, "Volume detached successfully", map[string]interface{}{
			"vm_id":     vmId,
			"volume_id": data.Id.ValueString(),
		})
	}

deleteVolume:
	// Call Emma API to delete volume
	_, response, err := r.apiClient.VolumesAPI.VolumeDelete(auth, volumeId).Execute()

	if err != nil {
		// Handle 404 errors as successful deletion (idempotent)
		if response != nil && response.StatusCode == 404 {
			// Volume already deleted, treat as success
			return
		}

		statusCode := 0
		apiError := ""
		if response != nil {
			statusCode = response.StatusCode
			apiError = tools.ExtractErrorMessage(response)
		}
		
		resourceErr := errors.NewError("emma_volume", "Delete").
			WithID(data.Id.ValueString()).
			WithStatusCode(statusCode).
			WithAPIError(apiError).
			WithMessage(errors.MapHTTPError(statusCode, apiError)).
			Build()
		
		resp.Diagnostics.AddError("Client Error", resourceErr.Error())
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
	// Set ID using shared utility
	data.Id = convert.Int32ToString(volume.Id)

	// Set basic attributes using shared utilities
	data.Name = convert.StringPointerToString(volume.Name)
	data.VolumeGb = convert.Int32ToInt64(volume.SizeGb)
	data.VolumeType = convert.StringPointerToString(volume.Type)
	data.IsSystem = convert.BoolPointerToBool(volume.IsSystem)
	data.Status = convert.StringPointerToString(volume.Status)
	data.ProjectId = convert.Int32ToInt64(volume.ProjectId)
	data.AttachedToId = convert.Int32ToInt64(volume.AttachedToId)
	data.CreatedAt = convert.StringPointerToString(volume.CreatedAt)

	// Convert provider nested object
	if volume.Provider != nil {
		providerModel := volumeResourceProviderModel{
			Id:   convert.Int32ToInt64(volume.Provider.Id),
			Name: convert.StringPointerToString(volume.Provider.Name),
		}
		providerObj, providerDiag := types.ObjectValueFrom(ctx, providerModel.attrTypes(), providerModel)
		data.CloudProvider = providerObj
		diags.Append(providerDiag...)
	} else {
		data.CloudProvider = types.ObjectNull(volumeResourceProviderModel{}.attrTypes())
	}

	// Convert location nested object
	if volume.Location != nil {
		locationModel := volumeResourceLocationModel{
			Id:        convert.Int32ToInt64(volume.Location.Id),
			Name:      convert.StringPointerToString(volume.Location.Name),
			Continent: convert.StringPointerToString(volume.Location.Continent),
			Region:    convert.StringPointerToString(volume.Location.Region),
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
			Id:   convert.StringPointerToString(volume.DataCenter.Id),
			Name: convert.StringPointerToString(volume.DataCenter.Name),
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
		attachedToId, err := convert.Int64ToInt32(data.AttachedToId)
		if err == nil {
			volumeCreateRequest.SetAttachedToId(attachedToId)
		}
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
