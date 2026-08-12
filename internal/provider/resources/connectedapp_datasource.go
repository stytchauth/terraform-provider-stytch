package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-go/v18/stytch/consumer/connectedapps"
	capclients "github.com/stytchauth/stytch-go/v18/stytch/consumer/connectedapps/clients"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/clients"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/projectapi"
)

var (
	_ datasource.DataSource              = &connectedAppDataSource{}
	_ datasource.DataSourceWithConfigure = &connectedAppDataSource{}
)

func NewConnectedAppDataSource() datasource.DataSource {
	return &connectedAppDataSource{}
}

type connectedAppDataSource struct {
	projectAPI *projectapi.Factory
}

type connectedAppDataSourceModel struct {
	ID                            types.String `tfsdk:"id"`
	ProjectSlug                   types.String `tfsdk:"project_slug"`
	EnvironmentSlug               types.String `tfsdk:"environment_slug"`
	ProjectSecret                 types.String `tfsdk:"project_secret"`
	ClientName                    types.String `tfsdk:"client_name"`
	ClientID                      types.String `tfsdk:"client_id"`
	ClientType                    types.String `tfsdk:"client_type"`
	ClientDescription             types.String `tfsdk:"client_description"`
	RedirectURLs                  types.Set    `tfsdk:"redirect_urls"`
	PostLogoutRedirectURLs        types.Set    `tfsdk:"post_logout_redirect_urls"`
	FullAccessAllowed             types.Bool   `tfsdk:"full_access_allowed"`
	BypassConsentForOfflineAccess types.Bool   `tfsdk:"bypass_consent_for_offline_access"`
	AccessTokenExpiryMinutes      types.Int32  `tfsdk:"access_token_expiry_minutes"`
	AccessTokenCustomAudience     types.String `tfsdk:"access_token_custom_audience"`
	AccessTokenTemplateContent    types.String `tfsdk:"access_token_template_content"`
	IDTokenTemplateContent        types.String `tfsdk:"id_token_template_content"`
	LogoURL                       types.String `tfsdk:"logo_url"`
	ClientSecretLastFour          types.String `tfsdk:"client_secret_last_four"`
	Status                        types.String `tfsdk:"status"`
	CreationMethod                types.String `tfsdk:"creation_method"`
	ClientIDMetadataURL           types.String `tfsdk:"client_id_metadata_url"`
}

func (d *connectedAppDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *connectedAppDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_app"
}

func (d *connectedAppDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a Connected App by name in an environment. Names are not unique in Stytch: the lookup fails " +
			"when no client or more than one client carries the name. Authentication uses a project secret for the " +
			"environment - create one with the stytch_secret resource. The client_secret is never available (the API " +
			"returns it only at creation time).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "A computed ID field used for Terraform resource management (format: project_slug.environment_slug.client_id).",
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project to which the connected app belongs.",
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to which the connected app belongs.",
			},
			"project_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "A project secret for the environment, used to authenticate against the project-level Stytch API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_name": schema.StringAttribute{
				Required:    true,
				Description: "The exact name of the connected app to look up.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_id":                         schema.StringAttribute{Computed: true, Description: "The ID of the connected app client."},
			"client_type":                       schema.StringAttribute{Computed: true, Description: "The type of connected app."},
			"client_description":                schema.StringAttribute{Computed: true, Description: "The description of the connected app."},
			"redirect_urls":                     schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Redirect URI values for OAuth authorization flows."},
			"post_logout_redirect_urls":         schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Redirect URI values for OIDC logout flows."},
			"full_access_allowed":               schema.BoolAttribute{Computed: true, Description: "Whether tokens granted to this client can be exchanged for a full Stytch session."},
			"bypass_consent_for_offline_access": schema.BoolAttribute{Computed: true, Description: "Whether the client can skip explicit user consent for the offline_access scope."},
			"access_token_expiry_minutes":       schema.Int32Attribute{Computed: true, Description: "The number of minutes before the access token expires."},
			"access_token_custom_audience":      schema.StringAttribute{Computed: true, Description: "The custom audience for the access token."},
			"access_token_template_content":     schema.StringAttribute{Computed: true, Description: "The content of the access token custom claims template."},
			"id_token_template_content":         schema.StringAttribute{Computed: true, Description: "The content of the ID token custom claims template."},
			"logo_url":                          schema.StringAttribute{Computed: true, Description: "The logo URL of the connected app."},
			"client_secret_last_four":           schema.StringAttribute{Computed: true, Description: "The last four characters of the client secret."},
			"status":                            schema.StringAttribute{Computed: true, Description: "The status of the connected app."},
			"creation_method":                   schema.StringAttribute{Computed: true, Description: "How the connected app was created."},
			"client_id_metadata_url":            schema.StringAttribute{Computed: true, Description: "The client ID metadata URL of the connected app."},
		},
	}
}

// At the API's maximum page size of 1000, this allows a million clients in one
// environment before giving up.
const maxConnectedAppSearchPages = 1000

func matchConnectedAppsByName(apps []connectedapps.ConnectedApp, name string) []connectedapps.ConnectedApp {
	var matches []connectedapps.ConnectedApp
	for _, app := range apps {
		if app.ClientName == name {
			matches = append(matches, app)
		}
	}
	return matches
}

func (d *connectedAppDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config connectedAppDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", config.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", config.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "client_name", config.ClientName.ValueString())
	tflog.Info(ctx, "Looking up connected app by name")

	client, err := d.projectAPI.ForEnvironment(ctx, config.ProjectSlug.ValueString(), config.EnvironmentSlug.ValueString(), config.ProjectSecret.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to build project API client", err.Error())
		return
	}

	name := config.ClientName.ValueString()
	var matches []connectedapps.ConnectedApp
	cursor := ""
	// A page cap in addition to the repeated-cursor guard below: that guard only
	// catches a cursor that repeats immediately, not a longer cycle.
	for page := 0; ; page++ {
		if page >= maxConnectedAppSearchPages {
			resp.Diagnostics.AddError(
				"Connected app search did not terminate",
				fmt.Sprintf("The connected app search paged %d times without exhausting its results while looking up %q in %s/%s.",
					maxConnectedAppSearchPages, name, config.ProjectSlug.ValueString(), config.EnvironmentSlug.ValueString()),
			)
			return
		}
		searchResp, err := client.ConnectedApp.Clients.Search(ctx, &capclients.SearchParams{Cursor: cursor, Limit: 1000})
		if err != nil {
			resp.Diagnostics.AddError("Failed to search connected apps", err.Error())
			return
		}
		matches = append(matches, matchConnectedAppsByName(searchResp.ConnectedApps, name)...)
		next := searchResp.ResultsMetadata.NextCursor
		if next == "" {
			break
		}
		if next == cursor {
			resp.Diagnostics.AddError(
				"Connected app search pagination did not advance",
				fmt.Sprintf("The connected app search returned the same cursor twice while looking up %q in %s/%s.", name, config.ProjectSlug.ValueString(), config.EnvironmentSlug.ValueString()),
			)
			return
		}
		cursor = next
	}

	switch len(matches) {
	case 0:
		resp.Diagnostics.AddError("Connected app not found", fmt.Sprintf("No connected app named %q exists in %s/%s.", name, config.ProjectSlug.ValueString(), config.EnvironmentSlug.ValueString()))
		return
	case 1:
	default:
		clientIDs := make([]string, 0, len(matches))
		for _, match := range matches {
			clientIDs = append(clientIDs, match.ClientID)
		}
		resp.Diagnostics.AddError(
			"Connected app name is ambiguous",
			fmt.Sprintf("%d connected apps named %q exist in %s/%s (client_ids: %s); rename the clients or manage one as a stytch_connected_app resource.", len(matches), name, config.ProjectSlug.ValueString(), config.EnvironmentSlug.ValueString(), strings.Join(clientIDs, ", ")),
		)
		return
	}

	app := matches[0]
	config.ID = types.StringValue(fmt.Sprintf("%s.%s.%s", config.ProjectSlug.ValueString(), config.EnvironmentSlug.ValueString(), app.ClientID))
	config.ClientID = types.StringValue(app.ClientID)
	config.ClientType = types.StringValue(app.ClientType)
	config.ClientDescription = optionalString(app.ClientDescription)
	config.RedirectURLs = setFromStrings(ctx, app.RedirectURLs)
	config.PostLogoutRedirectURLs = setFromStrings(ctx, app.PostLogoutRedirectURLs)
	config.FullAccessAllowed = types.BoolValue(app.FullAccessAllowed)
	config.BypassConsentForOfflineAccess = types.BoolValue(app.BypassConsentForOfflineAccess)
	config.AccessTokenExpiryMinutes = types.Int32Value(app.AccessTokenExpiryMinutes)
	config.AccessTokenCustomAudience = optionalString(app.AccessTokenCustomAudience)
	config.AccessTokenTemplateContent = optionalString(app.AccessTokenTemplateContent)
	config.IDTokenTemplateContent = optionalString(app.IDTokenTemplateContent)
	config.LogoURL = optionalString(app.LogoURL)
	config.ClientSecretLastFour = optionalString(app.ClientSecretLastFour)
	config.Status = optionalString(app.Status)
	config.CreationMethod = optionalString(app.CreationMethod)
	config.ClientIDMetadataURL = optionalString(app.ClientIDMetadataURL)

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}
