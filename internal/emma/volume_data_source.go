package emma

import (
	"context"
	"fmt"
	emmaSdk "github.com/emma-community/emma-go-sdk"
	"github.com/emma-community/terraform-provider-emma/tools"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &volumeDataSource{}

func NewVolumeDataSource() datasource.DataSource {
	return &volumeDataSource{}
}

// volumeDataSource defines the data source implementation.
type volumeDataSource struct {
	apiClient *emmaSdk.APIClient
	token     *emmaSdk.Token
}

// volumeDataSourceModel describes the data source data model.
type volumeDataSourceModel struct {
	Id             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	DataCenterId   types.String `tfsdk:"data_center_id"`
	VolumeGb       types.Int64  `tfsdk:"volume_gb"`
	VolumeType     types.String `tfsdk:"volume_type"`
	AttachedToId   types.Int64  `tfsdk:"attached_to_id"`
	IsSystem       types.Bool   `tfsdk:"is_system"`
	Status         types.String `tfsdk:"status"`
	ProjectId      types.Int64  `tfsdk:"project_id"`
	CloudProvider  types.Object `tfsdk:"cloud_provider"`
	Location       types.Object `tfsdk:"location"`
	DataCenter     types.Object `tfsdk:"data_center"`
	CreatedAt      types.String `tfsdk:"created_at"`
	CreatedByName  types.String `tfsdk:"created_by_name"`
	CreatedById    types.Int64  `tfsdk:"created_by_id"`
	ModifiedAt     types.String `tfsdk:"modified_at"`
	ModifiedByName types.String `tfsdk:"modified_by_name"`
	ModifiedById   types.Int64  `tfsdk:"modified_by_id"`
	Cost           types.Object `tfsdk:"cost"`
}

type volumeDataSourceProviderModel struct {
	Id   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type volumeDataSourceLocationModel struct {
	Id        types.Int64   `tfsdk:"id"`
	Name      types.String  `tfsdk:"name"`
	Continent types.String  `tfsdk:"continent"`
	Region    types.String  `tfsdk:"region"`
	Latitude  types.Float64 `tfsdk:"latitude"`
	Longitude types.Float64 `tfsdk:"longitude"`
}

type volumeDataSourceDataCenterModel struct {
	Id           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ProviderId   types.Int64  `tfsdk:"provider_id"`
	ProviderName types.String `tfsdk:"provider_name"`
	LocationId   types.Int64  `tfsdk:"location_id"`
	LocationName types.String `tfsdk:"location_name"`
}

type volumeDataSourceCostModel struct {
	Unit     types.String  `tfsdk:"unit"`
	Currency types.String  `tfsdk:"currency"`
	Price    types.Float64 `tfsdk:"price"`
}

func (d *volumeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (d *volumeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source retrieves information about an existing storage volume in the Emma platform.\n\n" +
			"Use this data source to query volume details by ID, including size, type, status, and attachment information.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the volume",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the volume",
				Computed:    true,
			},
			"data_center_id": schema.StringAttribute{
				Description: "Data center ID where the volume is located",
				Computed:    true,
			},
			"volume_gb": schema.Int64Attribute{
				Description: "Volume size in gigabytes",
				Computed:    true,
			},
			"volume_type": schema.StringAttribute{
				Description: "Volume type (e.g., ssd, hdd)",
				Computed:    true,
			},
			"attached_to_id": schema.Int64Attribute{
				Description: "ID of the compute instance the volume is attached to",
				Computed:    true,
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
					"latitude": schema.Float64Attribute{
						Description: "Approximate latitude of the geographical location",
						Computed:    true,
					},
					"longitude": schema.Float64Attribute{
						Description: "Approximate longitude of the geographical location",
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
					"provider_id": schema.Int64Attribute{
						Description: "ID of the cloud provider that owns the data center",
						Computed:    true,
					},
					"provider_name": schema.StringAttribute{
						Description: "Name of the cloud provider that owns the data center",
						Computed:    true,
					},
					"location_id": schema.Int64Attribute{
						Description: "ID of the data center location",
						Computed:    true,
					},
					"location_name": schema.StringAttribute{
						Description: "Name of the data center location",
						Computed:    true,
					},
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Creation timestamp",
				Computed:    true,
			},
			"created_by_name": schema.StringAttribute{
				Description: "Name of the user who created the volume",
				Computed:    true,
			},
			"created_by_id": schema.Int64Attribute{
				Description: "ID of the user who created the volume",
				Computed:    true,
			},
			"modified_at": schema.StringAttribute{
				Description: "Date and time when the volume was last modified",
				Computed:    true,
			},
			"modified_by_name": schema.StringAttribute{
				Description: "Name of the user who last modified the volume",
				Computed:    true,
			},
			"modified_by_id": schema.Int64Attribute{
				Description: "ID of the user who last modified the volume",
				Computed:    true,
			},
			"cost": schema.SingleNestedAttribute{
				Description: "Cost information for the volume",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"unit": schema.StringAttribute{
						Description: "Cost period unit",
						Computed:    true,
					},
					"currency": schema.StringAttribute{
						Description: "Currency of cost",
						Computed:    true,
					},
					"price": schema.Float64Attribute{
						Description: "Cost of the volume for the period",
						Computed:    true,
					},
				},
			},
		},
	}
}

func (d *volumeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.apiClient = client.apiClient
	d.token = client.token
}

func (d *volumeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data volumeDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Extract volume ID from configuration
	volumeId := tools.StringToInt32(data.Id.ValueString())

	// Call Emma API to get volume
	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *d.token.AccessToken)
	volume, response, err := d.apiClient.VolumesAPI.GetVolume(auth, volumeId).Execute()

	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read volume, got error: %s",
				tools.ExtractErrorMessage(response)))
		return
	}

	// Convert API response to data source model
	convertVolumeResponseToDataSource(ctx, &data, volume, resp.Diagnostics)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Helper function to get attribute types for provider nested object
func (o volumeDataSourceProviderModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.Int64Type,
		"name": types.StringType,
	}
}

// Helper function to get attribute types for location nested object
func (o volumeDataSourceLocationModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":        types.Int64Type,
		"name":      types.StringType,
		"continent": types.StringType,
		"region":    types.StringType,
		"latitude":  types.Float64Type,
		"longitude": types.Float64Type,
	}
}

// Helper function to get attribute types for data center nested object
func (o volumeDataSourceDataCenterModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":            types.StringType,
		"name":          types.StringType,
		"provider_id":   types.Int64Type,
		"provider_name": types.StringType,
		"location_id":   types.Int64Type,
		"location_name": types.StringType,
	}
}

// Helper function to get attribute types for cost nested object
func (o volumeDataSourceCostModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"unit":     types.StringType,
		"currency": types.StringType,
		"price":    types.Float64Type,
	}
}

// convertVolumeResponseToDataSource converts Emma API Volume response to Terraform data source model
func convertVolumeResponseToDataSource(ctx context.Context, data *volumeDataSourceModel, volume *emmaSdk.Volume, diags diag.Diagnostics) {
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
		providerModel := volumeDataSourceProviderModel{
			Id:   types.Int64Value(int64(*volume.Provider.Id)),
			Name: types.StringValue(*volume.Provider.Name),
		}
		providerObj, providerDiag := types.ObjectValueFrom(ctx, providerModel.attrTypes(), providerModel)
		data.CloudProvider = providerObj
		diags.Append(providerDiag...)
	} else {
		data.CloudProvider = types.ObjectNull(volumeDataSourceProviderModel{}.attrTypes())
	}

	// Convert location nested object
	if volume.Location != nil {
		locationModel := volumeDataSourceLocationModel{
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
		if volume.Location.Latitude != nil {
			locationModel.Latitude = types.Float64Value(*volume.Location.Latitude)
		} else {
			locationModel.Latitude = types.Float64Null()
		}
		if volume.Location.Longitude != nil {
			locationModel.Longitude = types.Float64Value(*volume.Location.Longitude)
		} else {
			locationModel.Longitude = types.Float64Null()
		}
		locationObj, locationDiag := types.ObjectValueFrom(ctx, locationModel.attrTypes(), locationModel)
		data.Location = locationObj
		diags.Append(locationDiag...)
	} else {
		data.Location = types.ObjectNull(volumeDataSourceLocationModel{}.attrTypes())
	}

	// Convert data center nested object
	if volume.DataCenter != nil {
		dataCenterModel := volumeDataSourceDataCenterModel{
			Id:   types.StringValue(*volume.DataCenter.Id),
			Name: types.StringValue(*volume.DataCenter.Name),
		}
		if volume.DataCenter.ProviderId != nil {
			dataCenterModel.ProviderId = types.Int64Value(int64(*volume.DataCenter.ProviderId))
		} else {
			dataCenterModel.ProviderId = types.Int64Null()
		}
		if volume.DataCenter.ProviderName != nil {
			dataCenterModel.ProviderName = types.StringValue(*volume.DataCenter.ProviderName)
		} else {
			dataCenterModel.ProviderName = types.StringNull()
		}
		if volume.DataCenter.LocationId != nil {
			dataCenterModel.LocationId = types.Int64Value(int64(*volume.DataCenter.LocationId))
		} else {
			dataCenterModel.LocationId = types.Int64Null()
		}
		if volume.DataCenter.LocationName != nil {
			dataCenterModel.LocationName = types.StringValue(*volume.DataCenter.LocationName)
		} else {
			dataCenterModel.LocationName = types.StringNull()
		}
		dataCenterObj, dataCenterDiag := types.ObjectValueFrom(ctx, dataCenterModel.attrTypes(), dataCenterModel)
		data.DataCenter = dataCenterObj
		diags.Append(dataCenterDiag...)
	} else {
		data.DataCenter = types.ObjectNull(volumeDataSourceDataCenterModel{}.attrTypes())
	}

	// Set data_center_id from the data center object
	if volume.DataCenter != nil && volume.DataCenter.Id != nil {
		data.DataCenterId = types.StringValue(*volume.DataCenter.Id)
	} else {
		data.DataCenterId = types.StringNull()
	}

	// Set created_by / modified_by fields
	if volume.CreatedByName != nil {
		data.CreatedByName = types.StringValue(*volume.CreatedByName)
	} else {
		data.CreatedByName = types.StringNull()
	}
	if volume.CreatedById != nil {
		data.CreatedById = types.Int64Value(int64(*volume.CreatedById))
	} else {
		data.CreatedById = types.Int64Null()
	}
	if volume.ModifiedAt != nil {
		data.ModifiedAt = types.StringValue(*volume.ModifiedAt)
	} else {
		data.ModifiedAt = types.StringNull()
	}
	if volume.ModifiedByName != nil {
		data.ModifiedByName = types.StringValue(*volume.ModifiedByName)
	} else {
		data.ModifiedByName = types.StringNull()
	}
	if volume.ModifiedById != nil {
		data.ModifiedById = types.Int64Value(int64(*volume.ModifiedById))
	} else {
		data.ModifiedById = types.Int64Null()
	}

	// Convert cost nested object
	if volume.Cost != nil {
		costModel := volumeDataSourceCostModel{}
		if volume.Cost.Unit != nil {
			costModel.Unit = types.StringValue(*volume.Cost.Unit)
		} else {
			costModel.Unit = types.StringNull()
		}
		if volume.Cost.Currency != nil {
			costModel.Currency = types.StringValue(*volume.Cost.Currency)
		} else {
			costModel.Currency = types.StringNull()
		}
		if volume.Cost.Price != nil {
			costModel.Price = types.Float64Value(float64(*volume.Cost.Price))
		} else {
			costModel.Price = types.Float64Null()
		}
		costObj, costDiag := types.ObjectValueFrom(ctx, costModel.attrTypes(), costModel)
		data.Cost = costObj
		diags.Append(costDiag...)
	} else {
		data.Cost = types.ObjectNull(volumeDataSourceCostModel{}.attrTypes())
	}
}
