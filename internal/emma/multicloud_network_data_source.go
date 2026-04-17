package emma

import (
	"context"
	"fmt"
	emmaSdk "github.com/emma-community/emma-go-sdk"
	"github.com/emma-community/terraform-provider-emma/tools"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &multicloudNetworkDataSource{}

func NewMulticloudNetworkDataSource() datasource.DataSource {
	return &multicloudNetworkDataSource{}
}

// multicloudNetworkDataSource defines the data source implementation.
type multicloudNetworkDataSource struct {
	apiClient *emmaSdk.APIClient
	token     *emmaSdk.Token
}

// multicloudNetworkDataSourceModel describes the data source data model.
type multicloudNetworkDataSourceModel struct {
	Networks            types.List `tfsdk:"networks"`
	CrossRegionConnects types.List `tfsdk:"cross_region_connects"`
}

func (d *multicloudNetworkDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_multicloud_network"
}

func (d *multicloudNetworkDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about the multicloud network configuration including direct connect networks and cross-region connections.",
		Attributes: map[string]schema.Attribute{
			"networks": schema.ListNestedAttribute{
				Description: "List of direct connect networks",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Network ID",
							Computed:    true,
						},
						"macro_region": schema.StringAttribute{
							Description: "Macro region of the network",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Operational status of the network",
							Computed:    true,
						},
					},
				},
			},
			"cross_region_connects": schema.ListNestedAttribute{
				Description: "List of cross-region connections",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Cross-region connection ID",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Operational status of the cross-region connection",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *multicloudNetworkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *multicloudNetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data multicloudNetworkDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Read multicloud network")

	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *d.token.AccessToken)
	mcn, response, err := d.apiClient.NetworkAPI.GetMultiCloudNetworks(auth).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read multicloud network, got error: %s",
				tools.ExtractErrorMessage(response)))
		return
	}

	networkAttrTypes := map[string]attr.Type{
		"id":           types.StringType,
		"macro_region": types.StringType,
		"status":       types.StringType,
	}

	crossRegionAttrTypes := map[string]attr.Type{
		"id":     types.StringType,
		"status": types.StringType,
	}

	if mcn.DirectConnect != nil {
		var networkObjects []attr.Value
		for _, n := range mcn.DirectConnect.Networks {
			nId := ""
			if n.Id != nil {
				nId = *n.Id
			}
			nMacroRegion := ""
			if n.MacroRegion != nil {
				nMacroRegion = *n.MacroRegion
			}
			nStatus := ""
			if n.Status != nil {
				nStatus = *n.Status
			}
			obj, diags := types.ObjectValue(networkAttrTypes, map[string]attr.Value{
				"id":           types.StringValue(nId),
				"macro_region": types.StringValue(nMacroRegion),
				"status":       types.StringValue(nStatus),
			})
			resp.Diagnostics.Append(diags...)
			networkObjects = append(networkObjects, obj)
		}
		networksList, diags := types.ListValue(types.ObjectType{AttrTypes: networkAttrTypes}, networkObjects)
		resp.Diagnostics.Append(diags...)
		data.Networks = networksList

		var crossRegionObjects []attr.Value
		for _, cr := range mcn.DirectConnect.CrossRegionConnects {
			crId := ""
			if cr.Id != nil {
				crId = *cr.Id
			}
			crStatus := ""
			if cr.Status != nil {
				crStatus = *cr.Status
			}
			obj, diags := types.ObjectValue(crossRegionAttrTypes, map[string]attr.Value{
				"id":     types.StringValue(crId),
				"status": types.StringValue(crStatus),
			})
			resp.Diagnostics.Append(diags...)
			crossRegionObjects = append(crossRegionObjects, obj)
		}
		crossRegionList, diags := types.ListValue(types.ObjectType{AttrTypes: crossRegionAttrTypes}, crossRegionObjects)
		resp.Diagnostics.Append(diags...)
		data.CrossRegionConnects = crossRegionList
	} else {
		networksList, diags := types.ListValue(types.ObjectType{AttrTypes: networkAttrTypes}, []attr.Value{})
		resp.Diagnostics.Append(diags...)
		data.Networks = networksList
		crossRegionList, diags := types.ListValue(types.ObjectType{AttrTypes: crossRegionAttrTypes}, []attr.Value{})
		resp.Diagnostics.Append(diags...)
		data.CrossRegionConnects = crossRegionList
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
