package resources

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-go/v18/stytch/consumer/connectedapps"
	capclients "github.com/stytchauth/stytch-go/v18/stytch/consumer/connectedapps/clients"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/clients"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/projectapi"
)

const (
	redirectURLTypeAuthorization = "authorization"
	redirectURLTypePostLogout    = "post_logout"
)

var (
	_ resource.Resource                = &connectedAppRedirectURLResource{}
	_ resource.ResourceWithConfigure   = &connectedAppRedirectURLResource{}
	_ resource.ResourceWithImportState = &connectedAppRedirectURLResource{}
)

func NewConnectedAppRedirectURLResource() resource.Resource {
	return &connectedAppRedirectURLResource{}
}

type connectedAppRedirectURLResource struct {
	projectAPI *projectapi.Factory
}

type connectedAppRedirectURLModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectSlug     types.String `tfsdk:"project_slug"`
	EnvironmentSlug types.String `tfsdk:"environment_slug"`
	ProjectSecret   types.String `tfsdk:"project_secret"`
	ClientID        types.String `tfsdk:"client_id"`
	URL             types.String `tfsdk:"url"`
	Type            types.String `tfsdk:"type"`
	LastUpdated     types.String `tfsdk:"last_updated"`
}

func (r *connectedAppRedirectURLResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *connectedAppRedirectURLResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_connected_app_redirect_url"
}

func (r *connectedAppRedirectURLResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A single redirect URL on a Connected App, managed additively: the provider reads the client, adds or " +
			"removes this one URL, and writes the client back. Authentication uses a project secret for the environment - " +
			"create one with the stytch_secret resource; importing requires that secret in the STYTCH_IMPORT_PROJECT_SECRET " +
			"environment variable, because Terraform provides no configuration values during import - and so does the first " +
			"plan afterwards, which refreshes from a state that does not yet carry project_secret. " +
			"Concurrent applies within one run are serialized per client. " +
			"Do not also manage the same client's URL arrays via the stytch_connected_app attributes, and avoid concurrent " +
			"out-of-band edits (the API offers no compare-and-swap).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management (format: project_slug.environment_slug.client_id.type.url). " +
					"Because the ID is split on dots with the url as the final segment, a client_id containing a dot cannot be imported.",
				Computed: true,
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
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the connected app client.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The redirect URL value.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(redirectURLTypeAuthorization),
				Description: "Which array the URL joins: authorization (OAuth authorization flows) or post_logout " +
					"(OIDC logout flows). Defaults to authorization.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(redirectURLTypeAuthorization, redirectURLTypePostLogout),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the last Terraform update.",
			},
		},
	}
}

func appendURL(urls []string, u string) []string {
	if slices.Contains(urls, u) {
		return urls
	}
	return append(slices.Clone(urls), u)
}

func removeURL(urls []string, u string) []string {
	result := make([]string, 0, len(urls))
	for _, existing := range urls {
		if existing != u {
			result = append(result, existing)
		}
	}
	return result
}

func parseConnectedAppRedirectURLImportID(id string) (projectSlug, environmentSlug, clientID, urlType, u string, err error) {
	parts := strings.SplitN(id, ".", 5)
	if len(parts) != 5 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[4] == "" {
		return "", "", "", "", "", fmt.Errorf("the ID must be in the format <project_slug>.<environment_slug>.<client_id>.<type>.<url>, got %q", id)
	}
	if parts[3] != redirectURLTypeAuthorization && parts[3] != redirectURLTypePostLogout {
		return "", "", "", "", "", fmt.Errorf("the type segment must be %q or %q, got %q", redirectURLTypeAuthorization, redirectURLTypePostLogout, parts[3])
	}
	return parts[0], parts[1], parts[2], parts[3], parts[4], nil
}

func urlsForType(app connectedapps.ConnectedApp, urlType string) []string {
	if urlType == redirectURLTypePostLogout {
		return app.PostLogoutRedirectURLs
	}
	return app.RedirectURLs
}

func bodyKeyForType(urlType string) string {
	if urlType == redirectURLTypePostLogout {
		return "post_logout_redirect_urls"
	}
	return "redirect_urls"
}

func (m *connectedAppRedirectURLModel) setID() {
	m.ID = types.StringValue(fmt.Sprintf("%s.%s.%s.%s.%s",
		m.ProjectSlug.ValueString(), m.EnvironmentSlug.ValueString(), m.ClientID.ValueString(), m.Type.ValueString(), m.URL.ValueString()))
}

func (r *connectedAppRedirectURLResource) mutate(ctx context.Context, m connectedAppRedirectURLModel, change func([]string) []string) error {
	unlock := r.projectAPI.Lock(projectapi.LockKeyConnectedApp(m.ClientID.ValueString()))
	defer unlock()

	client, err := r.projectAPI.ForEnvironment(ctx, m.ProjectSlug.ValueString(), m.EnvironmentSlug.ValueString(), m.ProjectSecret.ValueString())
	if err != nil {
		return err
	}

	getResp, err := client.ConnectedApp.Clients.Get(ctx, &capclients.GetParams{ClientID: m.ClientID.ValueString()})
	if err != nil {
		return err
	}
	app := getResp.ConnectedApp

	body := connectedAppBody(app)
	updated := change(urlsForType(app, m.Type.ValueString()))
	if updated == nil {
		updated = []string{}
	}
	body[bodyKeyForType(m.Type.ValueString())] = updated

	_, err = putConnectedApp(ctx, client.ConnectedApp.C, m.ClientID.ValueString(), body)
	return err
}

func (r *connectedAppRedirectURLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan connectedAppRedirectURLModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "client_id", plan.ClientID.ValueString())
	ctx = tflog.SetField(ctx, "url", plan.URL.ValueString())
	tflog.Info(ctx, "Adding connected app redirect URL")

	if err := r.mutate(ctx, plan, func(urls []string) []string {
		return appendURL(urls, plan.URL.ValueString())
	}); err != nil {
		resp.Diagnostics.AddError("Failed to add connected app redirect URL", err.Error())
		return
	}

	tflog.Info(ctx, "Added connected app redirect URL")

	plan.setID()
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *connectedAppRedirectURLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state connectedAppRedirectURLModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.projectAPI.ForEnvironment(ctx, state.ProjectSlug.ValueString(), state.EnvironmentSlug.ValueString(), state.ProjectSecret.ValueString())
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

	if !slices.Contains(urlsForType(getResp.ConnectedApp, state.Type.ValueString()), state.URL.ValueString()) {
		resp.State.RemoveResource(ctx)
		return
	}

	state.setID()
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *connectedAppRedirectURLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan connectedAppRedirectURLModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.setID()
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *connectedAppRedirectURLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state connectedAppRedirectURLModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "client_id", state.ClientID.ValueString())
	ctx = tflog.SetField(ctx, "url", state.URL.ValueString())
	tflog.Info(ctx, "Removing connected app redirect URL")

	err := r.mutate(ctx, state, func(urls []string) []string {
		return removeURL(urls, state.URL.ValueString())
	})
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Failed to remove connected app redirect URL", err.Error())
		return
	}

	tflog.Info(ctx, "Removed connected app redirect URL")
}

func (r *connectedAppRedirectURLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectSlug, environmentSlug, clientID, urlType, u, err := parseConnectedAppRedirectURLImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), projectSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_slug"), environmentSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_id"), clientID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), urlType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("url"), u)...)
}
