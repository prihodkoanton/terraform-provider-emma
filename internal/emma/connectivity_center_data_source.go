package emma

import (
	"context"
	"fmt"
	emmaSdk "github.com/emma-community/emma-go-sdk"
	"github.com/emma-community/terraform-provider-emma/tools"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &connectivityCenterDataSource{}

func NewConnectivityCenterDataSource() datasource.DataSource {
	return &connectivityCenterDataSource{}
}

// connectivityCenterDataSource defines the data source implementation.
type connectivityCenterDataSource struct {
	apiClient *emmaSdk.APIClient
	token     *emmaSdk.Token
}

// connectivityCenterDataSourceModel describes the data source data model.
type connectivityCenterDataSourceModel struct {
	Id          types.Int64  `tfsdk:"id"`
	Location    types.String `tfsdk:"location"`
	MacroRegion types.String `tfsdk:"macro_region"`
	NetworkType types.String `tfsdk:"network_type"`
}

func (d *connectivityCenterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connectivity_center"
}

func (d *connectivityCenterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about connectivity centers available for multicloud network connections.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "ID of the connectivity center",
				Computed:    true,
			},
			"location": schema.StringAttribute{
				Description: "Location of the connectivity center",
				Required:    true,
			},
			"macro_region": schema.StringAttribute{
				Description: "Macro region of the connectivity center",
				Computed:    true,
			},
			"network_type": schema.StringAttribute{
				Description: "Network type of the connectivity center",
				Computed:    true,
			},
		},
	}
}

func (d *connectivityCenterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *connectivityCenterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data connectivityCenterDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Read connectivity center")

	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *d.token.AccessToken)
	centers, response, err := d.apiClient.NetworkAPI.ListConnectivityCenters(auth).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read connectivity centers, got error: %s",
				tools.ExtractErrorMessage(response)))
		return
	}

	locationFilter := data.Location.ValueString()
	var found *emmaSdk.ConnectivityCenter
	for i := range centers {
		if centers[i].Location == locationFilter {
			found = &centers[i]
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Connectivity center with location '%s' not found", locationFilter))
		return
	}

	data.Id = types.Int64Value(int64(found.Id))
	data.Location = types.StringValue(found.Location)
	data.MacroRegion = types.StringValue(found.MacroRegion)
	data.NetworkType = types.StringValue(found.NetworkType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
