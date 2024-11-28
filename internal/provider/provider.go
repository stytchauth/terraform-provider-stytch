// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/pkg/api"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/resources"
)

// Ensure StytchProvider satisfies various provider interfaces.
var (
	_ provider.Provider              = &StytchProvider{}
	_ provider.ProviderWithFunctions = &StytchProvider{}
)

// StytchProvider defines the provider implementation.
type StytchProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// StytchProviderModel describes the provider data model.
type StytchProviderModel struct {
	WorkspaceKeyID     types.String `tfsdk:"workspace_key_id"`
	WorkspaceKeySecret types.String `tfsdk:"workspace_key_secret"`
	BaseURI            types.String `tfsdk:"base_uri"`
}

func (p *StytchProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "stytch"
	resp.Version = p.version
}

func (p *StytchProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Stytch to configure your workspace, including projects, redirect URLs, email templates and more.",
		Attributes: map[string]schema.Attribute{
			"workspace_key_id": schema.StringAttribute{
				Description: "The key ID for a workspace management key obtained from the Stytch workspace management page",
				Optional:    true,
			},
			"workspace_key_secret": schema.StringAttribute{
				Description: "The key secret corresponding for the workspace key obtained from the Stytch workspace management page",
				Optional:    true,
				Sensitive:   true,
			},
			"base_uri": schema.StringAttribute{
				Description: "Base URI override to use instead of Stytch's API. This is used for internal testing only.",
				Optional:    true,
			},
		},
	}
}

func (p *StytchProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring the Stytch provider")

	var config StytchProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	// If practitioner provided a configuration value for any of the attributes, it must be a known value.

	if config.WorkspaceKeyID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("workspace_key_id"),
			"Unknown workspace key ID",
			"The provider cannot create the Stytch management client as there is an unknown configuration value for the Stytch Workspace Key ID. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STYTCH_WORKSPACE_KEY_ID environment variable.",
		)
	}
	if config.WorkspaceKeySecret.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("workspace_key_secret"),
			"Unknown workspace key secret",
			"The provider cannot create the Stytch management client as there is an unknown configuration value for the Stytch Workspace Key Secret. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STYTCH_WORKSPACE_KEY_SECRET environment variable.",
		)
	}

	if config.BaseURI.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_uri"),
			"Unknown base URI",
			"The provider cannot create the Stytch management client as there is an unknown configuration value for the Stytch Base URI. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STYTCH_WORKSPACE_BASE_URI environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Now we default values to environment variables, but override with Terraform configuration values if set.

	workspaceKeyID := os.Getenv("STYTCH_WORKSPACE_KEY_ID")
	workspaceKeySecret := os.Getenv("STYTCH_WORKSPACE_KEY_SECRET")
	baseURI := os.Getenv("STYTCH_WORKSPACE_BASE_URI")

	if !config.WorkspaceKeyID.IsNull() {
		workspaceKeyID = config.WorkspaceKeyID.ValueString()
	}
	if !config.WorkspaceKeySecret.IsNull() {
		workspaceKeySecret = config.WorkspaceKeySecret.ValueString()
	}
	if !config.BaseURI.IsNull() {
		baseURI = config.BaseURI.ValueString()
	}

	// Now we make sure the keyID and secret are not empty strings
	if workspaceKeyID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("workspace_key_id"),
			"Missing workspace key ID",
			"The provider cannot create the Stytch management client as there is a missing or empty value for the Stytch Workspace Key ID. "+
				"Set the value in the configuration or use the STYTCH_WORKSPACE_KEY_ID environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}
	if workspaceKeySecret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("workspace_key_secret"),
			"Missing workspace key secret",
			"The provider cannot create the Stytch management client as there is a missing or empty value for the Stytch Workspace Key Secret. "+
				"Set the value in the configuration or use the STYTCH_WORKSPACE_KEY_SECRET environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "workspace_key_id", workspaceKeyID)
	ctx = tflog.SetField(ctx, "workspace_key_secret", workspaceKeySecret)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "workspace_key_secret")

	tflog.Debug(ctx, "Creating stytch-management-go client")

	// Now we're finally ready to create the stytch-management-go client.
	var opts []api.APIOption
	if baseURI != "" {
		ctx = tflog.SetField(ctx, "base_uri", baseURI)
		opts = append(opts, api.WithBaseURI(baseURI))
	}
	client := api.NewClient(workspaceKeyID, workspaceKeySecret, opts...)

	// Make the client available to the provider.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Stytch provider configured", map[string]any{"success": true})
}

func (p *StytchProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewB2BSDKConfigResource,
		resources.NewConsumerSDKConfigResource,
		resources.NewEmailTemplateResource,
		resources.NewPasswordConfigResource,
		resources.NewProjectResource,
		resources.NewPublicTokenResource,
		resources.NewRBACPolicyResource,
		resources.NewRedirectURLResource,
		resources.NewSecretResource,
	}
}

func (p *StytchProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}

func (p *StytchProvider) Functions(ctx context.Context) []func() function.Function {
	return nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &StytchProvider{
			version: version,
		}
	}
}
