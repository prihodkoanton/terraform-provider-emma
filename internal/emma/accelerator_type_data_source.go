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

var _ datasource.DataSource = &acceleratorTypeDataSource{}

func NewAcceleratorTypeDataSource() datasource.DataSource {
	return &acceleratorTypeDataSource{}
}

// acceleratorTypeDataSource defines the data source implementation.
type acceleratorTypeDataSource struct {
	apiClient *emmaSdk.APIClient
	token     *emmaSdk.Token
}

// acceleratorTypeDataSourceModel describes the data source data model.
type acceleratorTypeDataSourceModel struct {
	Id              types.String `tfsdk:"id"`
	AcceleratorType types.String `tfsdk:"accelerator_type"`
}

func (d *acceleratorTypeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_accelerator_type"
}

func (d *acceleratorTypeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about available accelerator types (e.g. GPU models) that can be used with compute instances.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID of the accelerator type",
				Computed:    true,
			},
			"accelerator_type": schema.StringAttribute{
				Description: "Name of the accelerator type (e.g. NVIDIA A100, NVIDIA T4)",
				Required:    true,
			},
		},
	}
}

func (d *acceleratorTypeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *acceleratorTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data acceleratorTypeDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Read accelerator type")

	auth := context.WithValue(ctx, emmaSdk.ContextAccessToken, *d.token.AccessToken)
	acceleratorTypes, response, err := d.apiClient.AcceleratorTypesAPI.GetAcceleratorTypes(auth).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Unable to read accelerator types, got error: %s",
				tools.ExtractErrorMessage(response)))
		return
	}

	typeFilter := data.AcceleratorType.ValueString()
	var found *emmaSdk.AcceleratorType
	for i := range acceleratorTypes {
		if acceleratorTypes[i].AcceleratorType != nil && *acceleratorTypes[i].AcceleratorType == typeFilter {
			found = &acceleratorTypes[i]
			break
		}
	}

	if found == nil {
		resp.Diagnostics.AddError("Client Error",
			fmt.Sprintf("Accelerator type '%s' not found", typeFilter))
		return
	}

	if found.Id != nil {
		data.Id = types.StringValue(*found.Id)
	}
	data.AcceleratorType = types.StringValue(*found.AcceleratorType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
