package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/environments"
	migrationprojects "github.com/stytchauth/stytch-management-go/v3/pkg/models/migration/projects"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/projects"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                 = &projectResource{}
	_ resource.ResourceWithConfigure    = &projectResource{}
	_ resource.ResourceWithImportState  = &projectResource{}
	_ resource.ResourceWithUpgradeState = &projectResource{}
)

func NewProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct {
	client *api.API
}

type projectResourceModelV0 struct {
	LiveProjectID types.String `tfsdk:"live_project_id"`
	TestProjectID types.String `tfsdk:"test_project_id"`
	Name          types.String `tfsdk:"name"`
	Vertical      types.String `tfsdk:"vertical"`
}

var projectResourceLegacySchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"live_project_id": schema.StringAttribute{
			Computed: true,
		},
		"test_project_id": schema.StringAttribute{
			Computed: true,
		},
		"name": schema.StringAttribute{
			Optional: true,
		},
		"vertical": schema.StringAttribute{
			Optional: true,
		},
	},
}

// NOTE: This struct is almost identical to that of the environment resource.
// See internal/provider/resources/environment.go environmentResourceModel.
type environmentModel struct {
	EnvironmentSlug                                        types.String `tfsdk:"environment_slug"`
	ProjectID                                              types.String `tfsdk:"project_id"`
	Name                                                   types.String `tfsdk:"name"`
	OAuthCallbackID                                        types.String `tfsdk:"oauth_callback_id"`
	CrossOrgPasswordsEnabled                               types.Bool   `tfsdk:"cross_org_passwords_enabled"`
	UserImpersonationEnabled                               types.Bool   `tfsdk:"user_impersonation_enabled"`
	ZeroDowntimeSessionMigrationURL                        types.String `tfsdk:"zero_downtime_session_migration_url"`
	UserLockSelfServeEnabled                               types.Bool   `tfsdk:"user_lock_self_serve_enabled"`
	UserLockThreshold                                      types.Int32  `tfsdk:"user_lock_threshold"`
	UserLockTTL                                            types.Int32  `tfsdk:"user_lock_ttl"`
	IDPAuthorizationURL                                    types.String `tfsdk:"idp_authorization_url"`
	IDPDynamicClientRegistrationEnabled                    types.Bool   `tfsdk:"idp_dynamic_client_registration_enabled"`
	IDPDynamicClientRegistrationAccessTokenTemplateContent types.String `tfsdk:"idp_dynamic_client_registration_access_token_template_content"`
	CreatedAt                                              types.String `tfsdk:"created_at"`
}

var environmentAttributeTypes = map[string]attr.Type{
	"environment_slug":                        types.StringType,
	"project_id":                              types.StringType,
	"name":                                    types.StringType,
	"oauth_callback_id":                       types.StringType,
	"cross_org_passwords_enabled":             types.BoolType,
	"user_impersonation_enabled":              types.BoolType,
	"zero_downtime_session_migration_url":     types.StringType,
	"user_lock_self_serve_enabled":            types.BoolType,
	"user_lock_threshold":                     types.Int32Type,
	"user_lock_ttl":                           types.Int32Type,
	"idp_authorization_url":                   types.StringType,
	"idp_dynamic_client_registration_enabled": types.BoolType,
	"idp_dynamic_client_registration_access_token_template_content": types.StringType,
	"created_at": types.StringType,
}

type projectModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectSlug     types.String `tfsdk:"project_slug"`
	Name            types.String `tfsdk:"name"`
	Vertical        types.String `tfsdk:"vertical"`
	LiveEnvironment types.Object `tfsdk:"live_environment"`
	CreatedAt       types.String `tfsdk:"created_at"`
	LastUpdated     types.String `tfsdk:"last_updated"`
}

func (r *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.API)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *api.API (stytch-management-go client), got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *projectResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &projectResourceLegacySchema,
			StateUpgrader: r.upgradeProjectStateV0ToV1,
		},
	}
}

func (r *projectResource) upgradeProjectStateV0ToV1(
	ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse,
) {
	if req.State == nil {
		resp.Diagnostics.AddError(
			"Missing prior state",
			"Legacy project state upgrade requires existing state data, but none was provided.",
		)
		return
	}

	var prior projectResourceModelV0
	diags := req.State.Get(ctx, &prior)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	liveProjectID := prior.LiveProjectID.ValueString()
	if liveProjectID == "" {
		resp.Diagnostics.AddError(
			"Missing legacy project ID",
			"The stored state did not contain a live project ID, so it cannot be upgraded automatically.",
		)
		return
	}

	migrationResp, err := r.client.V1ToV3MigrationClient.GetProject(ctx, migrationprojects.GetProjectRequest{
		ProjectID: liveProjectID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to retrieve legacy project metadata",
			err.Error(),
		)
		return
	}

	legacy := migrationResp.Project
	projectSlug := legacy.ProjectSlug
	if projectSlug == "" {
		resp.Diagnostics.AddError(
			"Missing project slug",
			"The migration endpoint did not return a project slug for the legacy project.",
		)
		return
	}

	liveEnvironmentSlug := legacy.LiveEnvironmentSlug
	if liveEnvironmentSlug == "" {
		resp.Diagnostics.AddError(
			"Missing live environment slug",
			"The migration endpoint did not return a live environment slug for the legacy project.",
		)
		return
	}

	projectResp, err := r.client.Projects.Get(ctx, projects.GetRequest{
		ProjectSlug: projectSlug,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch project details",
			err.Error(),
		)
		return
	}

	envResp, err := r.client.Environments.Get(ctx, environments.GetRequest{
		ProjectSlug:     projectSlug,
		EnvironmentSlug: liveEnvironmentSlug,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to fetch live environment details",
			err.Error(),
		)
		return
	}

	liveEnvState := refreshFromLiveEnv(envResp.Environment)
	liveEnvObj, diags := types.ObjectValueFrom(ctx, environmentAttributeTypes, liveEnvState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	lastUpdated := time.Now().Format(time.RFC850)
	createdAt := ""
	if !projectResp.Project.CreatedAt.IsZero() {
		createdAt = projectResp.Project.CreatedAt.Format(time.RFC3339)
	}

	newState := projectModel{
		ID:              types.StringValue(projectSlug),
		ProjectSlug:     types.StringValue(projectSlug),
		Name:            types.StringValue(projectResp.Project.Name),
		Vertical:        types.StringValue(string(projectResp.Project.Vertical)),
		CreatedAt:       types.StringValue(createdAt),
		LastUpdated:     types.StringValue(lastUpdated),
		LiveEnvironment: liveEnvObj,
	}

	diags = resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
}

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages a Stytch project and its live environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_slug": schema.StringAttribute{
				Description: "The immutable unique identifier for the project. If not provided, one will be generated.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The project's name.",
				Required:    true,
			},
			"vertical": schema.StringAttribute{
				Description: "The project's vertical (CONSUMER or B2B). Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(toStrings(projects.Verticals())...),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "The ISO-8601 timestamp when the project was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update for the resource.",
				Computed:    true,
			},
			"live_environment": schema.SingleNestedAttribute{
				Description: "Configuration for the project's live environment. Optional, but once created cannot be removed.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"environment_slug": schema.StringAttribute{
						Description: "The unique identifier (slug) for the live environment. Defaults to 'production'.",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("production"),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"project_id": schema.StringAttribute{
						Description: "The project ID for the live environment (used for API authentication).",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						Description: "The environment's name.",
						Required:    true,
					},
					"oauth_callback_id": schema.StringAttribute{
						Description: "The callback ID used in OAuth requests for the environment.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"cross_org_passwords_enabled": schema.BoolAttribute{
						Description: "Whether cross-org passwords are enabled for the environment. Irrelevant for Consumer projects.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"user_impersonation_enabled": schema.BoolAttribute{
						Description: "Whether user impersonation is enabled for the environment.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"zero_downtime_session_migration_url": schema.StringAttribute{
						Description: "The OIDC-compliant UserInfo endpoint for session migration.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"user_lock_self_serve_enabled": schema.BoolAttribute{
						Description: "Whether users who get locked out should automatically get an unlock email magic link.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"user_lock_threshold": schema.Int32Attribute{
						Description: "The number of failed authenticate attempts that will cause a user to be locked. Defaults to 10.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int32{
							int32planmodifier.UseStateForUnknown(),
						},
					},
					"user_lock_ttl": schema.Int32Attribute{
						Description: "The time in seconds that a user remains locked once the lock is set. Defaults to 1 hour (3600 seconds).",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Int32{
							int32planmodifier.UseStateForUnknown(),
						},
					},
					"idp_authorization_url": schema.StringAttribute{
						Description: "The OpenID Configuration endpoint for Connected Apps for the environment.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"idp_dynamic_client_registration_enabled": schema.BoolAttribute{
						Description: "Whether the project has opted in to Dynamic Client Registration (DCR) for Connected Apps.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"idp_dynamic_client_registration_access_token_template_content": schema.StringAttribute{
						Description: "The access token template to use for clients created through Dynamic Client Registration (DCR).",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"created_at": schema.StringAttribute{
						Description: "The ISO-8601 timestamp when the environment was created.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_name", plan.Name.ValueString())
	ctx = tflog.SetField(ctx, "vertical", plan.Vertical.ValueString())
	tflog.Info(ctx, "Creating project")

	// Create the project
	createProjectReq := projects.CreateRequest{
		Name:     plan.Name.ValueString(),
		Vertical: projects.Vertical(plan.Vertical.ValueString()),
	}
	if !plan.ProjectSlug.IsNull() && !plan.ProjectSlug.IsUnknown() {
		slug := plan.ProjectSlug.ValueString()
		createProjectReq.ProjectSlug = &slug
	}

	createResp, err := r.client.Projects.Create(ctx, createProjectReq)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create project", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", createResp.Project.ProjectSlug)
	tflog.Info(ctx, "Created project")

	// Build initial project state
	plan.ID = types.StringValue(createResp.Project.ProjectSlug)
	plan.ProjectSlug = types.StringValue(createResp.Project.ProjectSlug)
	plan.Name = types.StringValue(createResp.Project.Name)
	plan.Vertical = types.StringValue(string(createResp.Project.Vertical))
	plan.CreatedAt = types.StringValue(createResp.Project.CreatedAt.Format(time.RFC3339))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Create the live environment if specified
	if !plan.LiveEnvironment.IsNull() && !plan.LiveEnvironment.IsUnknown() {
		var liveEnvPlan environmentModel
		diags = plan.LiveEnvironment.As(ctx, &liveEnvPlan, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		ctx = tflog.SetField(ctx, "environment_slug", liveEnvPlan.EnvironmentSlug.ValueString())
		ctx = tflog.SetField(ctx, "environment_name", liveEnvPlan.Name.ValueString())
		tflog.Info(ctx, "Creating live environment")

		createEnvReq := buildEnvironmentCreateRequest(createResp.Project.ProjectSlug, liveEnvPlan)
		createEnvResp, err := r.client.Environments.Create(ctx, createEnvReq)
		if err != nil {
			resp.Diagnostics.AddError("Failed to create live environment", err.Error())
			return
		}

		tflog.Info(ctx, "Created live environment")

		liveEnvState := refreshFromLiveEnv(createEnvResp.Environment)

		liveEnvObj, diags := types.ObjectValueFrom(ctx, plan.LiveEnvironment.AttributeTypes(ctx), liveEnvState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.LiveEnvironment = liveEnvObj
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	tflog.Info(ctx, "Reading project")

	// Get the project
	getProjectResp, err := r.client.Projects.Get(ctx, projects.GetRequest{
		ProjectSlug: state.ProjectSlug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get project", err.Error())
		return
	}

	tflog.Info(ctx, "Read project")

	// Update the project state
	state.ID = types.StringValue(getProjectResp.Project.ProjectSlug)
	state.ProjectSlug = types.StringValue(getProjectResp.Project.ProjectSlug)
	state.Name = types.StringValue(getProjectResp.Project.Name)
	state.Vertical = types.StringValue(string(getProjectResp.Project.Vertical))
	state.CreatedAt = types.StringValue(getProjectResp.Project.CreatedAt.Format(time.RFC3339))

	// Try to discover and read the live environment
	// If state already has the environment slug, use it directly
	// Otherwise, query all environments to find the LIVE one
	var environmentSlug string
	if !state.LiveEnvironment.IsNull() && !state.LiveEnvironment.IsUnknown() {
		var liveEnvState environmentModel
		diags = state.LiveEnvironment.As(ctx, &liveEnvState, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		environmentSlug = liveEnvState.EnvironmentSlug.ValueString()
	} else {
		// Discover the live environment by listing all environments
		tflog.Info(ctx, "Discovering live environment")
		getAllEnvResp, err := r.client.Environments.GetAll(ctx, environments.GetAllRequest{
			ProjectSlug: state.ProjectSlug.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to list environments", err.Error())
			return
		}

		// Find the LIVE environment
		for _, env := range getAllEnvResp.Environments {
			if env.Type == environments.EnvironmentTypeLive {
				environmentSlug = env.EnvironmentSlug
				break
			}
		}
	}

	// If we found a live environment, fetch and populate it
	if environmentSlug != "" {
		ctx = tflog.SetField(ctx, "environment_slug", environmentSlug)
		tflog.Info(ctx, "Reading live environment")

		getEnvResp, err := r.client.Environments.Get(ctx, environments.GetRequest{
			ProjectSlug:     state.ProjectSlug.ValueString(),
			EnvironmentSlug: environmentSlug,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to get live environment", err.Error())
			return
		}

		tflog.Info(ctx, "Read live environment")

		liveEnvState := refreshFromLiveEnv(getEnvResp.Environment)

		liveEnvObj, diags := types.ObjectValueFrom(ctx, state.LiveEnvironment.AttributeTypes(ctx), liveEnvState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.LiveEnvironment = liveEnvObj
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state projectModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	tflog.Info(ctx, "Updating project")

	// Validate that live_environment cannot be removed once created
	stateHasLiveEnv := !state.LiveEnvironment.IsNull() && !state.LiveEnvironment.IsUnknown()
	planHasLiveEnv := !plan.LiveEnvironment.IsNull() && !plan.LiveEnvironment.IsUnknown()

	if stateHasLiveEnv && !planHasLiveEnv {
		resp.Diagnostics.AddError(
			"Cannot remove live_environment",
			"The live_environment cannot be removed once created. The backend does not support deleting the live environment."+
				" If you want to delete the project, delete the entire resource instead.",
		)
		return
	}

	// Update the project name if changed
	if !plan.Name.Equal(state.Name) {
		updateProjectReq := projects.UpdateRequest{
			ProjectSlug: state.ProjectSlug.ValueString(),
			Name:        ptr(plan.Name.ValueString()),
		}

		updateProjectResp, err := r.client.Projects.Update(ctx, updateProjectReq)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update project", err.Error())
			return
		}

		tflog.Info(ctx, "Updated project")

		state.Name = types.StringValue(updateProjectResp.Project.Name)
	}

	// Handle live environment changes
	if !planHasLiveEnv {
		// No live environment in plan, nothing to do (already validated it wasn't removed above)
		state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		return
	}

	var liveEnvPlan environmentModel
	diags = plan.LiveEnvironment.As(ctx, &liveEnvPlan, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If state doesn't have live environment but plan does, create it
	if !stateHasLiveEnv {
		ctx = tflog.SetField(ctx, "environment_slug", liveEnvPlan.EnvironmentSlug.ValueString())
		ctx = tflog.SetField(ctx, "environment_name", liveEnvPlan.Name.ValueString())
		tflog.Info(ctx, "Creating live environment")

		createEnvReq := buildEnvironmentCreateRequest(state.ProjectSlug.ValueString(), liveEnvPlan)
		createEnvResp, err := r.client.Environments.Create(ctx, createEnvReq)
		if err != nil {
			resp.Diagnostics.AddError("Failed to create live environment", err.Error())
			return
		}

		tflog.Info(ctx, "Created live environment")

		liveEnvState := refreshFromLiveEnv(createEnvResp.Environment)

		liveEnvObj, diags := types.ObjectValueFrom(ctx, state.LiveEnvironment.AttributeTypes(ctx), liveEnvState)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.LiveEnvironment = liveEnvObj
		state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		return
	}

	// Both state and plan have live environment, check if it changed
	var liveEnvState environmentModel
	diags = state.LiveEnvironment.As(ctx, &liveEnvState, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.LiveEnvironment.Equal(state.LiveEnvironment) {
		ctx = tflog.SetField(ctx, "environment_slug", liveEnvState.EnvironmentSlug.ValueString())
		tflog.Info(ctx, "Updating live environment")

		updateEnvReq := environments.UpdateRequest{
			ProjectSlug:     state.ProjectSlug.ValueString(),
			EnvironmentSlug: liveEnvState.EnvironmentSlug.ValueString(),
		}

		if !liveEnvPlan.Name.Equal(liveEnvState.Name) {
			updateEnvReq.Name = ptr(liveEnvPlan.Name.ValueString())
		}
		if !liveEnvPlan.CrossOrgPasswordsEnabled.IsNull() && !liveEnvPlan.CrossOrgPasswordsEnabled.Equal(liveEnvState.CrossOrgPasswordsEnabled) {
			updateEnvReq.CrossOrgPasswordsEnabled = ptr(liveEnvPlan.CrossOrgPasswordsEnabled.ValueBool())
		}
		if !liveEnvPlan.UserImpersonationEnabled.IsNull() && !liveEnvPlan.UserImpersonationEnabled.Equal(liveEnvState.UserImpersonationEnabled) {
			updateEnvReq.UserImpersonationEnabled = ptr(liveEnvPlan.UserImpersonationEnabled.ValueBool())
		}
		if !liveEnvPlan.ZeroDowntimeSessionMigrationURL.IsNull() && !liveEnvPlan.ZeroDowntimeSessionMigrationURL.Equal(liveEnvState.ZeroDowntimeSessionMigrationURL) {
			updateEnvReq.ZeroDowntimeSessionMigrationURL = ptr(liveEnvPlan.ZeroDowntimeSessionMigrationURL.ValueString())
		}
		if !liveEnvPlan.UserLockSelfServeEnabled.IsNull() && !liveEnvPlan.UserLockSelfServeEnabled.Equal(liveEnvState.UserLockSelfServeEnabled) {
			updateEnvReq.UserLockSelfServeEnabled = ptr(liveEnvPlan.UserLockSelfServeEnabled.ValueBool())
		}
		if !liveEnvPlan.UserLockThreshold.IsNull() && !liveEnvPlan.UserLockThreshold.Equal(liveEnvState.UserLockThreshold) {
			updateEnvReq.UserLockThreshold = ptr(int(liveEnvPlan.UserLockThreshold.ValueInt32()))
		}
		if !liveEnvPlan.UserLockTTL.IsNull() && !liveEnvPlan.UserLockTTL.Equal(liveEnvState.UserLockTTL) {
			updateEnvReq.UserLockTTL = ptr(int(liveEnvPlan.UserLockTTL.ValueInt32()))
		}
		if !liveEnvPlan.IDPAuthorizationURL.IsNull() && !liveEnvPlan.IDPAuthorizationURL.Equal(liveEnvState.IDPAuthorizationURL) {
			updateEnvReq.IDPAuthorizationURL = ptr(liveEnvPlan.IDPAuthorizationURL.ValueString())
		}
		if !liveEnvPlan.IDPDynamicClientRegistrationEnabled.IsNull() && !liveEnvPlan.IDPDynamicClientRegistrationEnabled.Equal(liveEnvState.IDPDynamicClientRegistrationEnabled) {
			updateEnvReq.IDPDynamicClientRegistrationEnabled = ptr(liveEnvPlan.IDPDynamicClientRegistrationEnabled.ValueBool())
		}
		if !liveEnvPlan.IDPDynamicClientRegistrationAccessTokenTemplateContent.IsNull() && !liveEnvPlan.IDPDynamicClientRegistrationAccessTokenTemplateContent.Equal(liveEnvState.IDPDynamicClientRegistrationAccessTokenTemplateContent) {
			updateEnvReq.IDPDynamicClientRegistrationAccessTokenTemplateContent = ptr(liveEnvPlan.IDPDynamicClientRegistrationAccessTokenTemplateContent.ValueString())
		}

		updateEnvResp, err := r.client.Environments.Update(ctx, updateEnvReq)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update live environment", err.Error())
			return
		}

		tflog.Info(ctx, "Updated live environment")

		liveEnvState = refreshFromLiveEnv(updateEnvResp.Environment)
	}

	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	liveEnvObj, diags := types.ObjectValueFrom(ctx, state.LiveEnvironment.AttributeTypes(ctx), liveEnvState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.LiveEnvironment = liveEnvObj

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	tflog.Info(ctx, "Deleting project")

	_, err := r.client.Projects.Delete(ctx, projects.DeleteRequest{
		ProjectSlug: state.ProjectSlug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete project", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted project")
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx = tflog.SetField(ctx, "project_slug", req.ID)
	tflog.Info(ctx, "Importing project")
	resource.ImportStatePassthroughID(ctx, path.Root("project_slug"), req, resp)
}

// Helper method to build the environment model from API response.
func refreshFromLiveEnv(env environments.Environment) environmentModel {
	return environmentModel{
		EnvironmentSlug:                     types.StringValue(env.EnvironmentSlug),
		ProjectID:                           types.StringValue(env.ProjectID),
		Name:                                types.StringValue(env.Name),
		OAuthCallbackID:                     types.StringValue(env.OAuthCallbackID),
		CrossOrgPasswordsEnabled:            types.BoolValue(env.CrossOrgPasswordsEnabled),
		UserImpersonationEnabled:            types.BoolValue(env.UserImpersonationEnabled),
		ZeroDowntimeSessionMigrationURL:     types.StringValue(env.ZeroDowntimeSessionMigrationURL),
		UserLockSelfServeEnabled:            types.BoolValue(env.UserLockSelfServeEnabled),
		UserLockThreshold:                   types.Int32Value(int32(env.UserLockThreshold)),
		UserLockTTL:                         types.Int32Value(int32(env.UserLockTTL)),
		IDPAuthorizationURL:                 types.StringValue(env.IDPAuthorizationURL),
		IDPDynamicClientRegistrationEnabled: types.BoolValue(env.IDPDynamicClientRegistrationEnabled),
		IDPDynamicClientRegistrationAccessTokenTemplateContent: types.StringValue(env.IDPDynamicClientRegistrationAccessTokenTemplateContent),
		CreatedAt: types.StringValue(env.CreatedAt.Format(time.RFC3339)),
	}
}

// Helper method to create environment request from plan.
func buildEnvironmentCreateRequest(projectSlug string, liveEnvPlan environmentModel) environments.CreateRequest {
	createEnvReq := environments.CreateRequest{
		ProjectSlug: projectSlug,
		Name:        liveEnvPlan.Name.ValueString(),
		Type:        environments.EnvironmentTypeLive,
	}
	envSlug := liveEnvPlan.EnvironmentSlug.ValueString()
	createEnvReq.EnvironmentSlug = &envSlug

	if !liveEnvPlan.CrossOrgPasswordsEnabled.IsNull() && !liveEnvPlan.CrossOrgPasswordsEnabled.IsUnknown() {
		createEnvReq.CrossOrgPasswordsEnabled = ptr(liveEnvPlan.CrossOrgPasswordsEnabled.ValueBool())
	}
	if !liveEnvPlan.UserImpersonationEnabled.IsNull() && !liveEnvPlan.UserImpersonationEnabled.IsUnknown() {
		createEnvReq.UserImpersonationEnabled = ptr(liveEnvPlan.UserImpersonationEnabled.ValueBool())
	}
	if !liveEnvPlan.ZeroDowntimeSessionMigrationURL.IsNull() && !liveEnvPlan.ZeroDowntimeSessionMigrationURL.IsUnknown() {
		createEnvReq.ZeroDowntimeSessionMigrationURL = ptr(liveEnvPlan.ZeroDowntimeSessionMigrationURL.ValueString())
	}
	if !liveEnvPlan.UserLockSelfServeEnabled.IsNull() && !liveEnvPlan.UserLockSelfServeEnabled.IsUnknown() {
		createEnvReq.UserLockSelfServeEnabled = ptr(liveEnvPlan.UserLockSelfServeEnabled.ValueBool())
	}
	if !liveEnvPlan.UserLockThreshold.IsNull() && !liveEnvPlan.UserLockThreshold.IsUnknown() {
		createEnvReq.UserLockThreshold = ptr(int(liveEnvPlan.UserLockThreshold.ValueInt32()))
	}
	if !liveEnvPlan.UserLockTTL.IsNull() && !liveEnvPlan.UserLockTTL.IsUnknown() {
		createEnvReq.UserLockTTL = ptr(int(liveEnvPlan.UserLockTTL.ValueInt32()))
	}
	if !liveEnvPlan.IDPAuthorizationURL.IsNull() && !liveEnvPlan.IDPAuthorizationURL.IsUnknown() {
		createEnvReq.IDPAuthorizationURL = ptr(liveEnvPlan.IDPAuthorizationURL.ValueString())
	}
	if !liveEnvPlan.IDPDynamicClientRegistrationEnabled.IsNull() && !liveEnvPlan.IDPDynamicClientRegistrationEnabled.IsUnknown() {
		createEnvReq.IDPDynamicClientRegistrationEnabled = ptr(liveEnvPlan.IDPDynamicClientRegistrationEnabled.ValueBool())
	}
	if !liveEnvPlan.IDPDynamicClientRegistrationAccessTokenTemplateContent.IsNull() && !liveEnvPlan.IDPDynamicClientRegistrationAccessTokenTemplateContent.IsUnknown() {
		createEnvReq.IDPDynamicClientRegistrationAccessTokenTemplateContent = ptr(liveEnvPlan.IDPDynamicClientRegistrationAccessTokenTemplateContent.ValueString())
	}

	return createEnvReq
}
