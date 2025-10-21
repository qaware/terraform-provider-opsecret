// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	// get the secret reference from input and try to resolve it directly
	secretReference := state.ID.ValueString()
	resolvedReferenceValue, err := d.client.Secrets().Resolve(ctx, secretReference)

	// references pointing to files cannot be resolved directly and need to be resolved step by step
	if err != nil && err.Error() == "error resolving secret reference: unable to retrieve file content, currently only text files are supported" {
		if rawValue, err2 := d.resolveFileContentByReference(ctx, secretReference); err2 != nil {
			err = err2
		} else {
			state.Value = types.StringValue(strings.TrimSpace(base64.StdEncoding.EncodeToString(rawValue)))
		}
	} else {
		state.Value = types.StringValue(resolvedReferenceValue)
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read secret reference",
			err.Error(),
		)
		return
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// resolves the given secret reference by resolving each reference part step by step,
// returning the file content bytes and nil or nil and an error object if something goes wrong
func (d *secretReferenceDataSource) resolveFileContentByReference(ctx context.Context, secretReference string) ([]byte, error) {
	// skip the op:// prefix and split the remaining path on each /
	pathElements := strings.Split(secretReference[5:], "/")
	vaultName := pathElements[0]
	itemName := pathElements[1]
	fileName := pathElements[2]

	// get the vault ID by its name
	vaultId, err := d.getVaultId(ctx, vaultName)
	if err != nil {
		return nil, err
	}

	// get the item ID by its name
	itemId, err := d.getItemId(ctx, vaultId, itemName)
	if err != nil {
		return nil, err
	}

	// get the file contents by its name
	fileContents, err := d.getFileByName(ctx, vaultId, itemId, fileName)
	if err != nil {
		return nil, err
	}

	return fileContents, nil
}

// searches all available vaults, matching by given vault name
// returns the vault ID and nil on match, empty string and an error object otherwise
func (d *secretReferenceDataSource) getVaultId(ctx context.Context, vaultName string) (string, error) {
	vaults, err := d.client.Vaults().List(ctx)
	if err != nil {
		return "", err
	}
	for _, vault := range vaults {
		if vault.Title == vaultName {
			return vault.ID, nil
		}
	}
	return "", fmt.Errorf("vault '%s' not found", vaultName)
}

// searches all available items in the given vault, matching by given item name
// returns the item ID and nil on match, empty string and an error object otherwise
func (d *secretReferenceDataSource) getItemId(ctx context.Context, vaultId string, fileName string) (string, error) {
	items, err := d.client.Items().List(ctx, vaultId)
	if err != nil {
		return "", err
	}
	for _, item := range items {
		if item.Title == fileName {
			return item.ID, nil
		}
	}
	return "", fmt.Errorf("item '%s' not found", fileName)
}

// searches all available file attachments in the given item, matching by given file name
// returns the file content bytes and nil on match, nil and an error object otherwise
func (d *secretReferenceDataSource) getFileByName(ctx context.Context, vaultId string, itemId string, fileName string) ([]byte, error) {
	itemDetails, err := d.client.Items().Get(ctx, vaultId, itemId)
	if err != nil {
		return nil, err
	}
	for _, fileAttachment := range itemDetails.Files {
		if fileAttachment.Attributes.Name == fileName {
			fileBytes, err := d.client.Items().Files().Read(ctx, vaultId, itemId, fileAttachment.Attributes)
			if err != nil {
				return nil, err
			}
			return fileBytes, nil
		}
	}
	return nil, fmt.Errorf("file '%s' not found", fileName)
}
