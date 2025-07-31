// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"github.com/1password/onepassword-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"os"
)

// Ensure OPSecretReferenceProvider satisfies various provider interfaces.
var _ provider.Provider = &OPSecretReferenceProvider{}
var _ provider.ProviderWithFunctions = &OPSecretReferenceProvider{}
var _ provider.ProviderWithEphemeralResources = &OPSecretReferenceProvider{}

// OPSecretReferenceProvider defines the provider implementation.
type OPSecretReferenceProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// OPSecretReferenceProviderModel describes the provider data model.
type OPSecretReferenceProviderModel struct {
	ServiceAccountToken types.String `tfsdk:"service_account_token"`
}

func (p *OPSecretReferenceProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "opsecret"
	resp.Version = p.version
}

func (p *OPSecretReferenceProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"service_account_token": schema.StringAttribute{
				MarkdownDescription: "Token for the Onepassword service account.<br>If not provided directly the OP_SERVICE_ACCOUNT_TOKEN environment variable will be used instead.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *OPSecretReferenceProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config OPSecretReferenceProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	token := ""
	envToken := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")
	if (config.ServiceAccountToken.IsUnknown() || config.ServiceAccountToken.ValueString() == "") && envToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("service_account_token"),
			"Unknown or missing Service Account Token",
			"The provider cannot create the Onepassword API client as the service account token is missing. "+
				"Either set the value statically in the configuration, or use the OP_SERVICE_ACCOUNT_TOKEN environment variable.",
		)
	}

	if !config.ServiceAccountToken.IsUnknown() && config.ServiceAccountToken.ValueString() != "" {
		token = config.ServiceAccountToken.String()
	} else {
		token = envToken
	}
	client, err := onepassword.NewClient(
		ctx,
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo("Onepassword secret terraform provider", "v0.0.1"),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed creating onepassword client", err.Error())
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *OPSecretReferenceProvider) Resources(ctx context.Context) []func() resource.Resource {
	return nil
}

func (p *OPSecretReferenceProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return nil
}

func (p *OPSecretReferenceProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSecretReferenceDataSource,
	}
}

func (p *OPSecretReferenceProvider) Functions(ctx context.Context) []func() function.Function {
	return nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OPSecretReferenceProvider{
			version: version,
		}
	}
}
