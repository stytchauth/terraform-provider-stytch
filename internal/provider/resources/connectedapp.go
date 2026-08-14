package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-go/v18/stytch"
	"github.com/stytchauth/stytch-go/v18/stytch/consumer/connectedapps"
	capclients "github.com/stytchauth/stytch-go/v18/stytch/consumer/connectedapps/clients"
	"github.com/stytchauth/stytch-go/v18/stytch/consumer/stytchapi"
	"github.com/stytchauth/stytch-go/v18/stytch/stytcherror"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/clients"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/projectapi"
)

var (
	_ resource.Resource                   = &connectedAppResource{}
	_ resource.ResourceWithConfigure      = &connectedAppResource{}
	_ resource.ResourceWithImportState    = &connectedAppResource{}
	_ resource.ResourceWithValidateConfig = &connectedAppResource{}
)

func NewConnectedAppResource() resource.Resource {
	return &connectedAppResource{}
}

type connectedAppResource struct {
	projectAPI *projectapi.Factory
}

type connectedAppModel struct {
	ID                            types.String `tfsdk:"id"`
	ProjectSlug                   types.String `tfsdk:"project_slug"`
	EnvironmentSlug               types.String `tfsdk:"environment_slug"`
	ProjectSecret                 types.String `tfsdk:"project_secret"`
	ClientType                    types.String `tfsdk:"client_type"`
	ClientName                    types.String `tfsdk:"client_name"`
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
	ClientID                      types.String `tfsdk:"client_id"`
	ClientSecret                  types.String `tfsdk:"client_secret"`
	ClientSecretLastFour          types.String `tfsdk:"client_secret_last_four"`
	Status                        types.String `tfsdk:"status"`
	CreationMethod                types.String `tfsdk:"creation_method"`
	ClientIDMetadataURL           types.String `tfsdk:"client_id_metadata_url"`
	LastUpdated                   types.String `tfsdk:"last_updated"`
}

func (r *connectedAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerClients, ok := req.ProviderData.(*clients.Clients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *clients.Clients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.projectAPI = providerClients.ProjectAPI
}

func (r *connectedAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_app"
}

func (r *connectedAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Connected App (OAuth/OIDC client) in an environment. Managed through the project-level Stytch API: " +
			"authentication uses a project secret for the environment - create one with the stytch_secret resource. " +
			"Importing requires that secret in the STYTCH_IMPORT_PROJECT_SECRET environment variable, because Terraform " +
			"provides no configuration values during import - and so does the first plan afterwards, which refreshes from " +
			"a state that does not yet carry project_secret. " +
			"After importing a client whose URLs are managed by stytch_connected_app_redirect_url resources, either declare " +
			"redirect_urls and post_logout_redirect_urls in configuration or remove them from state before the next apply; " +
			"otherwise the first apply plans their removal.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management (format: project_slug.environment_slug.client_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project to which the connected app belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to which the connected app belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "A project secret for the environment, used to authenticate against the project-level Stytch API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_type": schema.StringAttribute{
				Required: true,
				Description: "The type of connected app: one of first_party, first_party_public, third_party, or " +
					"third_party_public. Cannot be changed after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("first_party", "first_party_public", "third_party", "third_party_public"),
				},
			},
			"client_name": schema.StringAttribute{
				Required:    true,
				Description: "A human-readable name for the connected app.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_description": schema.StringAttribute{
				Optional:    true,
				Description: "A human-readable description for the connected app.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"redirect_urls": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Redirect URI values for OAuth authorization flows. Manage these either here or via " +
					"stytch_connected_app_redirect_url resources, never both for the same client.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"post_logout_redirect_urls": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Redirect URI values for OIDC logout flows. Manage these either here or via " +
					"stytch_connected_app_redirect_url resources, never both for the same client.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"full_access_allowed": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "First-party clients only: whether an authorization token granted to this client can be " +
					"exchanged for a full Stytch session. Removing this from configuration keeps the current value rather " +
					"than restoring the server default.",
			},
			"bypass_consent_for_offline_access": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "First-party clients only: whether the client can skip explicit user consent for the " +
					"offline_access scope. Removing this from configuration keeps the current value rather than restoring " +
					"the server default.",
			},
			"access_token_expiry_minutes": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Description: "The number of minutes before the access token expires. Defaults to 60. Removing this from " +
					"configuration keeps the current value rather than restoring the default.",
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
				},
			},
			"access_token_custom_audience": schema.StringAttribute{
				Optional:    true,
				Description: "The custom audience for the access token.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"access_token_template_content": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "The content of the access token custom claims template, as a JSON object string. Defaults " +
					"to an empty object. Removing this from configuration keeps the current value rather than restoring the default.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"id_token_template_content": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "The content of the ID token custom claims template, as a JSON object string. Defaults to an " +
					"empty object. Removing this from configuration keeps the current value rather than restoring the default.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"logo_url": schema.StringAttribute{
				Optional:    true,
				Description: "The logo URL of the connected app.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"client_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the connected app client.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_secret": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
				Description: "The client secret. Only available for confidential client types, and only at creation time: " +
					"the API never returns it again, so it is null for imported resources.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_secret_last_four": schema.StringAttribute{
				Computed:    true,
				Description: "The last four characters of the client secret.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The status of the connected app.",
			},
			"creation_method": schema.StringAttribute{
				Computed:    true,
				Description: "How the connected app was created.",
			},
			"client_id_metadata_url": schema.StringAttribute{
				Computed:    true,
				Description: "The client ID metadata URL of the connected app.",
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the last Terraform update.",
			},
		},
	}
}

func isNotFound(err error) bool {
	var stytchErr stytcherror.Error
	if errors.As(err, &stytchErr) {
		return stytchErr.StatusCode == 404
	}
	return false
}

func parseConnectedAppImportID(id string) (projectSlug, environmentSlug, clientID string, err error) {
	parts := strings.SplitN(id, ".", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("the ID must be in the format <project_slug>.<environment_slug>.<client_id>, got %q", id)
	}
	return parts[0], parts[1], parts[2], nil
}

func isFirstPartyClient(clientType string) bool {
	return strings.HasPrefix(clientType, "first_party")
}

func connectedAppBody(app connectedapps.ConnectedApp) map[string]any {
	redirectURLs := app.RedirectURLs
	if redirectURLs == nil {
		redirectURLs = []string{}
	}
	postLogoutRedirectURLs := app.PostLogoutRedirectURLs
	if postLogoutRedirectURLs == nil {
		postLogoutRedirectURLs = []string{}
	}
	body := map[string]any{
		"client_name":                   app.ClientName,
		"client_description":            app.ClientDescription,
		"redirect_urls":                 redirectURLs,
		"post_logout_redirect_urls":     postLogoutRedirectURLs,
		"access_token_expiry_minutes":   app.AccessTokenExpiryMinutes,
		"access_token_custom_audience":  app.AccessTokenCustomAudience,
		"access_token_template_content": app.AccessTokenTemplateContent,
		"id_token_template_content":     app.IDTokenTemplateContent,
		"logo_url":                      app.LogoURL,
	}
	if isFirstPartyClient(app.ClientType) {
		body["full_access_allowed"] = app.FullAccessAllowed
		body["bypass_consent_for_offline_access"] = app.BypassConsentForOfflineAccess
	}
	return body
}

// The generated SDK Update marshals with omitempty, which silently drops
// false booleans and empty strings, so fields could never be cleared. This
// sends the exact body it is given instead.
func putConnectedApp(ctx context.Context, c stytch.Client, clientID string, body map[string]any) (*capclients.UpdateResponse, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	var retVal capclients.UpdateResponse
	err = c.NewRequest(ctx, stytch.RequestParams{
		Method:  "PUT",
		Path:    fmt.Sprintf("/v1/connected_apps/clients/%s", url.PathEscape(clientID)),
		Body:    jsonBody,
		V:       &retVal,
		Headers: map[string][]string{},
	})
	if err != nil {
		return nil, err
	}
	return &retVal, nil
}

// The conversion cannot fail for a []string into a set of strings, so the
// diagnostics are dropped rather than threaded through every caller.
func setFromStrings(ctx context.Context, values []string) types.Set {
	if len(values) == 0 {
		return types.SetNull(types.StringType)
	}
	set, _ := types.SetValueFrom(ctx, types.StringType, slices.Clone(values))
	return set
}

func stringsFromSet(set types.Set) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	values := make([]string, 0, len(set.Elements()))
	for _, element := range set.Elements() {
		if str, ok := element.(types.String); ok {
			values = append(values, str.ValueString())
		}
	}
	return values
}

func optionalString(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func (m *connectedAppModel) updateFromAPI(ctx context.Context, app connectedapps.ConnectedApp) {
	m.ID = types.StringValue(fmt.Sprintf("%s.%s.%s", m.ProjectSlug.ValueString(), m.EnvironmentSlug.ValueString(), app.ClientID))
	m.ClientID = types.StringValue(app.ClientID)
	m.ClientType = types.StringValue(app.ClientType)
	m.ClientName = types.StringValue(app.ClientName)
	m.ClientDescription = optionalString(app.ClientDescription)
	m.FullAccessAllowed = types.BoolValue(app.FullAccessAllowed)
	m.BypassConsentForOfflineAccess = types.BoolValue(app.BypassConsentForOfflineAccess)
	m.AccessTokenExpiryMinutes = types.Int32Value(app.AccessTokenExpiryMinutes)
	m.AccessTokenCustomAudience = optionalString(app.AccessTokenCustomAudience)
	m.AccessTokenTemplateContent = optionalString(app.AccessTokenTemplateContent)
	m.IDTokenTemplateContent = optionalString(app.IDTokenTemplateContent)
	m.LogoURL = optionalString(app.LogoURL)
	m.ClientSecretLastFour = optionalString(app.ClientSecretLastFour)
	m.Status = optionalString(app.Status)
	m.CreationMethod = optionalString(app.CreationMethod)
	m.ClientIDMetadataURL = optionalString(app.ClientIDMetadataURL)
	if !m.RedirectURLs.IsNull() {
		m.RedirectURLs = setFromStrings(ctx, app.RedirectURLs)
	}
	if !m.PostLogoutRedirectURLs.IsNull() {
		m.PostLogoutRedirectURLs = setFromStrings(ctx, app.PostLogoutRedirectURLs)
	}
}

func (m *connectedAppModel) refreshArraysFromAPI(ctx context.Context, app connectedapps.ConnectedApp) {
	m.RedirectURLs = setFromStrings(ctx, app.RedirectURLs)
	m.PostLogoutRedirectURLs = setFromStrings(ctx, app.PostLogoutRedirectURLs)
}

func (r *connectedAppResource) apiClient(ctx context.Context, m connectedAppModel) (*stytchapi.API, error) {
	return r.projectAPI.ForEnvironment(ctx, m.ProjectSlug.ValueString(), m.EnvironmentSlug.ValueString(), m.ProjectSecret.ValueString())
}

func (r *connectedAppResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data connectedAppModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", data.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", data.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Validating connected app config")

	if data.ClientType.IsNull() || data.ClientType.IsUnknown() {
		return
	}
	clientType := data.ClientType.ValueString()
	if isFirstPartyClient(clientType) {
		return
	}

	for _, attribute := range []struct {
		name  string
		value types.Bool
	}{
		{"full_access_allowed", data.FullAccessAllowed},
		{"bypass_consent_for_offline_access", data.BypassConsentForOfflineAccess},
	} {
		if attribute.value.IsNull() || attribute.value.IsUnknown() || !attribute.value.ValueBool() {
			continue
		}
		resp.Diagnostics.AddAttributeError(
			path.Root(attribute.name),
			fmt.Sprintf("%s is not valid for this client type", attribute.name),
			fmt.Sprintf("%s is valid only for first-party client types (first_party, first_party_public), but client_type is %q.",
				attribute.name, clientType),
		)
	}
}

func (r *connectedAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan connectedAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Creating connected app")

	client, err := r.apiClient(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build project API client", err.Error())
		return
	}

	createResp, err := client.ConnectedApp.Clients.Create(ctx, &capclients.CreateParams{
		ClientType:                    capclients.CreateRequestClientType(plan.ClientType.ValueString()),
		ClientName:                    plan.ClientName.ValueString(),
		ClientDescription:             plan.ClientDescription.ValueString(),
		RedirectURLs:                  stringsFromSet(plan.RedirectURLs),
		PostLogoutRedirectURLs:        stringsFromSet(plan.PostLogoutRedirectURLs),
		FullAccessAllowed:             plan.FullAccessAllowed.ValueBool(),
		BypassConsentForOfflineAccess: plan.BypassConsentForOfflineAccess.ValueBool(),
		AccessTokenExpiryMinutes:      plan.AccessTokenExpiryMinutes.ValueInt32(),
		AccessTokenCustomAudience:     plan.AccessTokenCustomAudience.ValueString(),
		AccessTokenTemplateContent:    plan.AccessTokenTemplateContent.ValueString(),
		IDTokenTemplateContent:        plan.IDTokenTemplateContent.ValueString(),
		LogoURL:                       plan.LogoURL.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create connected app", err.Error())
		return
	}

	tflog.Info(ctx, "Created connected app")

	app := createResp.ConnectedApp
	plan.updateFromAPI(ctx, connectedapps.ConnectedApp{
		ClientID:                      app.ClientID,
		ClientType:                    app.ClientType,
		ClientName:                    app.ClientName,
		ClientDescription:             app.ClientDescription,
		Status:                        app.Status,
		FullAccessAllowed:             app.FullAccessAllowed,
		RedirectURLs:                  app.RedirectURLs,
		AccessTokenExpiryMinutes:      app.AccessTokenExpiryMinutes,
		AccessTokenTemplateContent:    app.AccessTokenTemplateContent,
		PostLogoutRedirectURLs:        app.PostLogoutRedirectURLs,
		BypassConsentForOfflineAccess: app.BypassConsentForOfflineAccess,
		IDTokenTemplateContent:        app.IDTokenTemplateContent,
		ClientSecretLastFour:          app.ClientSecretLastFour,
		AccessTokenCustomAudience:     app.AccessTokenCustomAudience,
		LogoURL:                       app.LogoURL,
		ClientIDMetadataURL:           app.ClientIDMetadataURL,
	})
	plan.ClientSecret = optionalString(app.ClientSecret)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *connectedAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state connectedAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "client_id", state.ClientID.ValueString())
	tflog.Info(ctx, "Reading connected app")

	client, err := r.apiClient(ctx, state)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build project API client", err.Error())
		return
	}

	getResp, err := client.ConnectedApp.Clients.Get(ctx, &capclients.GetParams{ClientID: state.ClientID.ValueString()})
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to get connected app", err.Error())
		return
	}

	imported, diags := req.Private.GetKey(ctx, "imported")
	resp.Diagnostics.Append(diags...)
	state.updateFromAPI(ctx, getResp.ConnectedApp)
	if len(imported) > 0 {
		state.refreshArraysFromAPI(ctx, getResp.ConnectedApp)
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, "imported", nil)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *connectedAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan connectedAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state connectedAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientID := state.ClientID.ValueString()
	unlock := r.projectAPI.Lock(projectapi.LockKeyConnectedApp(clientID))
	defer unlock()

	ctx = tflog.SetField(ctx, "client_id", clientID)
	tflog.Info(ctx, "Updating connected app")

	client, err := r.apiClient(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build project API client", err.Error())
		return
	}

	getResp, err := client.ConnectedApp.Clients.Get(ctx, &capclients.GetParams{ClientID: clientID})
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError(
				"Connected app no longer exists",
				fmt.Sprintf("Connected app %s was deleted outside of Terraform, so there is nothing to update. "+
					"Run terraform refresh (or terraform apply -refresh-only) to drop it from state, then apply again to recreate it.", clientID),
			)
			return
		}
		resp.Diagnostics.AddError("Failed to get connected app before update", err.Error())
		return
	}

	body := connectedAppBody(getResp.ConnectedApp)
	body["client_name"] = plan.ClientName.ValueString()
	overlayString(body, "client_description", plan.ClientDescription, state.ClientDescription)
	overlayString(body, "access_token_custom_audience", plan.AccessTokenCustomAudience, state.AccessTokenCustomAudience)
	overlayString(body, "access_token_template_content", plan.AccessTokenTemplateContent, state.AccessTokenTemplateContent)
	overlayString(body, "id_token_template_content", plan.IDTokenTemplateContent, state.IDTokenTemplateContent)
	overlayString(body, "logo_url", plan.LogoURL, state.LogoURL)
	if isFirstPartyClient(getResp.ConnectedApp.ClientType) {
		if !plan.FullAccessAllowed.IsUnknown() {
			body["full_access_allowed"] = plan.FullAccessAllowed.ValueBool()
		}
		if !plan.BypassConsentForOfflineAccess.IsUnknown() {
			body["bypass_consent_for_offline_access"] = plan.BypassConsentForOfflineAccess.ValueBool()
		}
	}
	if !plan.AccessTokenExpiryMinutes.IsUnknown() {
		body["access_token_expiry_minutes"] = plan.AccessTokenExpiryMinutes.ValueInt32()
	}
	if !plan.RedirectURLs.IsNull() {
		body["redirect_urls"] = stringsFromSet(plan.RedirectURLs)
	} else if !state.RedirectURLs.IsNull() {
		body["redirect_urls"] = []string{}
	}
	if !plan.PostLogoutRedirectURLs.IsNull() {
		body["post_logout_redirect_urls"] = stringsFromSet(plan.PostLogoutRedirectURLs)
	} else if !state.PostLogoutRedirectURLs.IsNull() {
		body["post_logout_redirect_urls"] = []string{}
	}

	updateResp, err := putConnectedApp(ctx, client.ConnectedApp.C, clientID, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update connected app", err.Error())
		return
	}

	tflog.Info(ctx, "Updated connected app")

	plan.updateFromAPI(ctx, updateResp.ConnectedApp)
	plan.ClientSecret = state.ClientSecret
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func overlayString(body map[string]any, key string, plan, state types.String) {
	if plan.IsUnknown() {
		return
	}
	if !plan.IsNull() {
		body[key] = plan.ValueString()
	} else if !state.IsNull() {
		body[key] = ""
	}
}

func (r *connectedAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state connectedAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "client_id", state.ClientID.ValueString())
	tflog.Info(ctx, "Deleting connected app")

	client, err := r.apiClient(ctx, state)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build project API client", err.Error())
		return
	}

	_, err = client.ConnectedApp.Clients.Delete(ctx, &capclients.DeleteParams{ClientID: state.ClientID.ValueString()})
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete connected app", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted connected app")
}

func (r *connectedAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectSlug, environmentSlug, clientID, err := parseConnectedAppImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), projectSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_slug"), environmentSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_id"), clientID)...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, "imported", []byte(`true`))...)
}
