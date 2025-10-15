package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/jwttemplates"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                 = &jwtTemplateResource{}
	_ resource.ResourceWithConfigure    = &jwtTemplateResource{}
	_ resource.ResourceWithImportState  = &jwtTemplateResource{}
	_ resource.ResourceWithUpgradeState = &jwtTemplateResource{}
)

func NewJWTTemplateResource() resource.Resource {
	return &jwtTemplateResource{}
}

type jwtTemplateResource struct {
	client *api.API
}

type jwtTemplateModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectSlug     types.String `tfsdk:"project_slug"`
	EnvironmentSlug types.String `tfsdk:"environment_slug"`
	TemplateType    types.String `tfsdk:"template_type"`
	TemplateContent types.String `tfsdk:"template_content"`
	CustomAudience  types.String `tfsdk:"custom_audience"`
	LastUpdated     types.String `tfsdk:"last_updated"`
}

type jwtTemplateResourceModelV0 struct {
	ProjectID       types.String `tfsdk:"project_id"`
	TemplateType    types.String `tfsdk:"template_type"`
	TemplateContent types.String `tfsdk:"template_content"`
	CustomAudience  types.String `tfsdk:"custom_audience"`
}

var jwtTemplateResourceLegacySchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"project_id": schema.StringAttribute{
			Required: true,
		},
		"template_type": schema.StringAttribute{
			Required: true,
		},
		"template_content": schema.StringAttribute{
			Optional: true,
			Computed: true,
		},
		"custom_audience": schema.StringAttribute{
			Optional: true,
			Computed: true,
		},
	},
}

func (r *jwtTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.API)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *api.API (stytch-management-go client), got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *jwtTemplateResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &jwtTemplateResourceLegacySchema,
			StateUpgrader: r.upgradeJWTTemplateStateV0ToV1,
		},
	}
}

func (r *jwtTemplateResource) upgradeJWTTemplateStateV0ToV1(
	ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse,
) {
	if req.State == nil {
		resp.Diagnostics.AddError(
			"Missing prior state",
			"Legacy JWT template state upgrade requires existing state data, but none was provided.",
		)
		return
	}

	var prior jwtTemplateResourceModelV0
	diags := req.State.Get(ctx, &prior)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectSlug, environmentSlug, diags := resolveLegacyProjectAndEnvironment(
		ctx, r.client, prior.ProjectID.ValueString(),
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := r.client.JWTTemplates.Get(ctx, &jwttemplates.GetRequest{
		ProjectSlug:     projectSlug,
		EnvironmentSlug: environmentSlug,
		JWTTemplateType: jwttemplates.JWTTemplateType(strings.ToUpper(prior.TemplateType.ValueString())),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to retrieve JWT template", err.Error())
		return
	}

	templateType := strings.ToUpper(prior.TemplateType.ValueString())
	newState := jwtTemplateModel{
		ID:              types.StringValue(fmt.Sprintf("%s.%s.%s", projectSlug, environmentSlug, templateType)),
		ProjectSlug:     types.StringValue(projectSlug),
		EnvironmentSlug: types.StringValue(environmentSlug),
		TemplateType:    types.StringValue(templateType),
		LastUpdated:     types.StringValue(time.Now().Format(time.RFC850)),
	}

	r.updateModelFromAPI(&newState, &getResp.JWTTemplate)

	diags = resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
}

// Metadata returns the resource type name.
func (r *jwtTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jwt_template"
}

// Schema defines the schema for the resource.
func (r *jwtTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Resource for creating and managing JWT templates.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management (format: project_slug.environment_slug.template_type).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project to which the JWT template belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to which the JWT template belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of JWT template. Valid values: SESSION, M2M.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(toStrings(jwttemplates.JWTTemplateTypes())...),
				},
			},
			"template_content": schema.StringAttribute{
				Required:    true,
				Description: "The content of the JWT template.",
			},
			"custom_audience": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "An optional custom audience for the JWT template.",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update.",
				Computed:    true,
			},
		},
	}
}

// updateModelFromAPI updates the model with values from the API response
func (r *jwtTemplateResource) updateModelFromAPI(model *jwtTemplateModel, template *jwttemplates.JWTTemplate) {
	model.ID = types.StringValue(fmt.Sprintf("%s.%s.%s", model.ProjectSlug.ValueString(), model.EnvironmentSlug.ValueString(), model.TemplateType.ValueString()))
	model.TemplateContent = types.StringValue(template.TemplateContent)
	model.CustomAudience = types.StringValue(template.CustomAudience)
}

// Create creates the resource and sets the initial Terraform state.
func (r *jwtTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan jwtTemplateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "template_type", plan.TemplateType.ValueString())
	tflog.Info(ctx, "Creating JWT template")

	createResp, err := r.client.JWTTemplates.Set(ctx, &jwttemplates.SetRequest{
		ProjectSlug:     plan.ProjectSlug.ValueString(),
		EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
		JWTTemplateType: jwttemplates.JWTTemplateType(plan.TemplateType.ValueString()),
		TemplateContent: plan.TemplateContent.ValueString(),
		CustomAudience:  plan.CustomAudience.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create JWT template", err.Error())
		return
	}

	tflog.Info(ctx, "JWT template created")

	r.updateModelFromAPI(&plan, &createResp.JWTTemplate)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *jwtTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state jwtTemplateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "template_type", state.TemplateType.ValueString())
	tflog.Info(ctx, "Reading JWT template")

	getResp, err := r.client.JWTTemplates.Get(ctx, &jwttemplates.GetRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		JWTTemplateType: jwttemplates.JWTTemplateType(state.TemplateType.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read JWT template", err.Error())
		return
	}

	tflog.Info(ctx, "Read JWT template")

	r.updateModelFromAPI(&state, &getResp.JWTTemplate)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *jwtTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan jwtTemplateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "template_type", plan.TemplateType.ValueString())
	tflog.Info(ctx, "Updating JWT template")

	updateResp, err := r.client.JWTTemplates.Set(ctx, &jwttemplates.SetRequest{
		ProjectSlug:     plan.ProjectSlug.ValueString(),
		EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
		JWTTemplateType: jwttemplates.JWTTemplateType(plan.TemplateType.ValueString()),
		TemplateContent: plan.TemplateContent.ValueString(),
		CustomAudience:  plan.CustomAudience.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update JWT template", err.Error())
		return
	}

	tflog.Info(ctx, "JWT template updated")

	r.updateModelFromAPI(&plan, &updateResp.JWTTemplate)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
// Note: JWT templates cannot be deleted via API, they can only be reset to default values.
func (r *jwtTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state jwtTemplateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "template_type", state.TemplateType.ValueString())
	tflog.Info(ctx, "Deleting JWT template (resetting to default values)")

	// JWT templates cannot be deleted via API, only reset to empty/default values
	_, err := r.client.JWTTemplates.Set(ctx, &jwttemplates.SetRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		JWTTemplateType: jwttemplates.JWTTemplateType(state.TemplateType.ValueString()),
		TemplateContent: "{}",
		CustomAudience:  "",
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset JWT template to default values", err.Error())
		return
	}

	tflog.Info(ctx, "JWT template reset to default values")
}

// ImportState imports the resource into Terraform state.
func (r *jwtTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx = tflog.SetField(ctx, "import_id", req.ID)
	tflog.Info(ctx, "Importing JWT template")

	// Import ID format: project_slug.environment_slug.template_type
	parts := strings.Split(req.ID, ".")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("Expected import ID format: project_slug.environment_slug.template_type, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_slug"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_type"), parts[2])...)
}
