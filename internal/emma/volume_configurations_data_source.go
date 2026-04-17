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

var _ datasource.DataSource = &volumeConfigurationsDataSource{}

func NewVolumeConfigurationsDataSource() datasource.DataSource {
	return &volumeConfigurationsDataSource{}
}

// volumeConfigurationsDataSource defines the data source implementation.
type volumeConfigurationsDataSource struct {
	apiClient *emmaSdk.APIClient
	token     *emmaSdk.Token
}

// volumeConfigurationsDataSourceModel describes the data source data model.
type volumeConfigurationsDataSourceModel struct {
	DataCenterId   types.String `tfsdk:"data_center_id"`
	Configurations types.List   `tfsdk:"configurations"`
}

// volumeConfigurationModel describes individual configuration items in the list.
type volumeConfigurationModel struct {
	ProviderId     types.Int64   `tfsdk:"provider_id"`
	ProviderName   types.String  `tfsdk:"provider_name"`
	LocationId     types.Int64   `tfsdk:"location_id"`
	LocationName   types.String  `tfsdk:"location_name"`
	DataCenterId   types.String  `tfsdk:"data_center_id"`
	DataCenterName types.String  `tfsdk:"data_center_name"`
	VolumeGb       types.Int64   `tfsdk:"volume_gb"`
	VolumeType     types.String  `tfsdk:"volume_type"`
	PricePerMonth  types.Float64 `tfsdk:"price_per_month"`
}

func (d *volumeConfigurationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_configurations"
}

func (d *volumeConfigurationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "This data source retrieves available volume configurations for a specific data center in the Emma platform.\n\n" +
			"Use this data source to discover valid volume types, size ranges, and pricing information when planning volume deployments.",
		Attributes: map[string]schema.Attribute{
			"data_center_id": schema.StringAttribute{
				Description: "ID of the data center to query volume configurations for",
				Required:    true,
			},
			"configurations": schema.ListNestedAttribute{
				Description: "List of available volume configurations",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"provider_id": schema.Int64Attribute{
							Description: "ID of the cloud provider",
							Computed:    true,
						},
						"provider_name": schema.StringAttribute{
							Description: "Name of the cloud provider",
							Computed:    true,
						},
						"location_id": schema.Int64Attribute{
							Description: "Location ID",
							Computed:    true,
						},
						"location_name": schema.StringAttribute{
							Description: "Location name (city or state)",
							Computed:    true,
						},
						"data_center_id": schema.StringAttribute{
							Description: "ID of the data center",
							Computed:    true,
						},
						"data_center_name": schema.StringAttribute{
							Description: "Name of the data center",
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
						"price_per_month": schema.Float64Attribute{
							Description: "Price per month for this configuration",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *volumeConfigurationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *volumeConfigurationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data volumeConfigurationsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Extract data_center_id from configuration
	dataCenterId := data.DataCenterId.ValueString()

	// Call Emma API to get volume configurations
	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *d.token.AccessToken)
	configs, response, err := d.apiClient.VolumesConfigurationsAPI.GetSystemVolumeConfigs(auth).Execute()

	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read volume configurations, got error: %s",
				tools.ExtractErrorMessage(response)))
		return
	}

	// Filter configurations by data_center_id and convert to list
	configList, diags := convertVolumeConfigsToList(ctx, configs, dataCenterId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Configurations = configList

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Helper function to get attribute types for configuration nested object
func (o volumeConfigurationModel) attrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"provider_id":      types.Int64Type,
		"provider_name":    types.StringType,
		"location_id":      types.Int64Type,
		"location_name":    types.StringType,
		"data_center_id":   types.StringType,
		"data_center_name": types.StringType,
		"volume_gb":        types.Int64Type,
		"volume_type":      types.StringType,
		"price_per_month":  types.Float64Type,
	}
}

// convertVolumeConfigsToList converts Emma API volume configurations response to Terraform list
// Filters configurations by the specified data_center_id
func convertVolumeConfigsToList(ctx context.Context, configs *emmaSdk.GetSystemVolumeConfigs200Response, dataCenterId string) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	var configModels []volumeConfigurationModel

	// Check if configs is nil or has no content
	if configs == nil || configs.Content == nil {
		// Return empty list
		emptyList, listDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: volumeConfigurationModel{}.attrTypes()}, []volumeConfigurationModel{})
		diags.Append(listDiags...)
		return emptyList, diags
	}

	// Convert each volume configuration that matches the data_center_id
	for _, config := range configs.Content {
		// Filter by data_center_id
		if config.DataCenterId == nil || *config.DataCenterId != dataCenterId {
			continue
		}

		configModel := volumeConfigurationModel{}

		if config.ProviderId != nil {
			configModel.ProviderId = types.Int64Value(int64(*config.ProviderId))
		} else {
			configModel.ProviderId = types.Int64Null()
		}

		if config.ProviderName != nil {
			configModel.ProviderName = types.StringValue(*config.ProviderName)
		} else {
			configModel.ProviderName = types.StringNull()
		}

		if config.LocationId != nil {
			configModel.LocationId = types.Int64Value(int64(*config.LocationId))
		} else {
			configModel.LocationId = types.Int64Null()
		}

		if config.LocationName != nil {
			configModel.LocationName = types.StringValue(*config.LocationName)
		} else {
			configModel.LocationName = types.StringNull()
		}

		if config.DataCenterId != nil {
			configModel.DataCenterId = types.StringValue(*config.DataCenterId)
		} else {
			configModel.DataCenterId = types.StringNull()
		}

		if config.DataCenterName != nil {
			configModel.DataCenterName = types.StringValue(*config.DataCenterName)
		} else {
			configModel.DataCenterName = types.StringNull()
		}

		if config.VolumeGb != nil {
			configModel.VolumeGb = types.Int64Value(int64(*config.VolumeGb))
		} else {
			configModel.VolumeGb = types.Int64Null()
		}

		if config.VolumeType != nil {
			configModel.VolumeType = types.StringValue(*config.VolumeType)
		} else {
			configModel.VolumeType = types.StringNull()
		}

		// Extract price from cost object
		if config.Cost != nil && config.Cost.PricePerUnit != nil {
			configModel.PricePerMonth = types.Float64Value(float64(*config.Cost.PricePerUnit))
		} else {
			configModel.PricePerMonth = types.Float64Null()
		}

		configModels = append(configModels, configModel)
	}

	// Convert to Terraform list
	configList, listDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: volumeConfigurationModel{}.attrTypes()}, configModels)
	diags.Append(listDiags...)

	return configList, diags
}
