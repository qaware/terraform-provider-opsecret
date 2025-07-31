// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &secretReferenceDataSource{}
	_ datasource.DataSourceWithConfigure = &secretReferenceDataSource{}
)

func NewSecretReferenceDataSource() datasource.DataSource {
	return &secretReferenceDataSource{}
}

type secretReferenceDataSource struct {
	client *onepassword.Client
}

type secretReferenceDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	Value types.String `tfsdk:"value"`
}

func (d *secretReferenceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*onepassword.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *onepassword.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *secretReferenceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_reference"
}

func (d *secretReferenceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The 1Password secret reference.<br>See https://developer.1password.com/docs/cli/secret-reference-syntax/ for details.",
			},
			"value": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The resolved secret value.",
			},
		},
	}
}

func (d *secretReferenceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state secretReferenceDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	resolve, err := d.client.Secrets().Resolve(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read secret reference",
			err.Error(),
		)
		return
	}

	state.Value = types.StringValue(resolve)

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
