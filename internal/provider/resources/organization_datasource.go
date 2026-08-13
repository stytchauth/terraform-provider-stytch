package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-go/v18/stytch/b2b/organizations"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/clients"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/projectapi"
)

var (
	_ datasource.DataSource                     = &organizationDataSource{}
	_ datasource.DataSourceWithConfigure        = &organizationDataSource{}
	_ datasource.DataSourceWithConfigValidators = &organizationDataSource{}
)

func NewOrganizationDataSource() datasource.DataSource {
	return &organizationDataSource{}
}

type organizationDataSource struct {
	projectAPI *projectapi.Factory
}

type organizationDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	ProjectSlug            types.String `tfsdk:"project_slug"`
	EnvironmentSlug        types.String `tfsdk:"environment_slug"`
	ProjectSecret          types.String `tfsdk:"project_secret"`
	OrganizationID         types.String `tfsdk:"organization_id"`
	OrganizationSlug       types.String `tfsdk:"organization_slug"`
	OrganizationExternalID types.String `tfsdk:"organization_external_id"`
	OrganizationName       types.String `tfsdk:"organization_name"`
	TrustedMetadata        types.String `tfsdk:"trusted_metadata"`
}

func (d *organizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerClients, ok := req.ProviderData.(*clients.Clients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *clients.Clients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.projectAPI = providerClients.ProjectAPI
}

func (d *organizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *organizationDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(
			path.MatchRoot("organization_id"),
			path.MatchRoot("organization_slug"),
			path.MatchRoot("organization_external_id"),
		),
	}
}

func (d *organizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a B2B organization by ID, slug, or external ID (exactly one must be set). Authentication " +
			"uses a project secret for the environment - create one with the stytch_secret resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "A computed ID field used for Terraform resource management (format: project_slug.environment_slug.organization_id).",
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project to which the organization belongs.",
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to which the organization belongs.",
			},
			"project_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "A project secret for the environment, used to authenticate against the project-level Stytch API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"organization_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The unique ID of the organization to look up.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"organization_slug": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The slug of the organization to look up.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"organization_external_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The external ID of the organization to look up.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"organization_name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the organization.",
			},
			"trusted_metadata": schema.StringAttribute{
				Computed:    true,
				Description: "The organization's full trusted_metadata object as a JSON document.",
			},
		},
	}
}

func (d *organizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config organizationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The Get endpoint accepts an organization ID, slug, or external ID in the
	// same field.
	identifier := config.OrganizationID.ValueString()
	if identifier == "" {
		identifier = config.OrganizationSlug.ValueString()
	}
	if identifier == "" {
		identifier = config.OrganizationExternalID.ValueString()
	}

	ctx = tflog.SetField(ctx, "project_slug", config.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", config.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "organization", identifier)
	tflog.Info(ctx, "Looking up organization")

	client, err := d.projectAPI.ForB2BEnvironment(ctx, config.ProjectSlug.ValueString(), config.EnvironmentSlug.ValueString(), config.ProjectSecret.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to build project API client", err.Error())
		return
	}

	getResp, err := client.Organizations.Get(ctx, &organizations.GetParams{OrganizationID: identifier})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get organization", err.Error())
		return
	}
	org := getResp.Organization

	metadata, err := json.Marshal(org.TrustedMetadata)
	if err != nil {
		resp.Diagnostics.AddError("Failed to encode trusted_metadata", err.Error())
		return
	}

	config.ID = types.StringValue(fmt.Sprintf("%s.%s.%s", config.ProjectSlug.ValueString(), config.EnvironmentSlug.ValueString(), org.OrganizationID))
	config.OrganizationID = types.StringValue(org.OrganizationID)
	config.OrganizationSlug = types.StringValue(org.OrganizationSlug)
	config.OrganizationExternalID = optionalString(org.OrganizationExternalID)
	config.OrganizationName = types.StringValue(org.OrganizationName)
	config.TrustedMetadata = types.StringValue(string(metadata))

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
