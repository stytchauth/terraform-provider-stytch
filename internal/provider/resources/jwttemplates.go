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
	"github.com/stytchauth/stytch-management-go/v2/pkg/api"
	"github.com/stytchauth/stytch-management-go/v2/pkg/models/jwttemplates"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &jwtTemplateResource{}
	_ resource.ResourceWithConfigure   = &jwtTemplateResource{}
	_ resource.ResourceWithImportState = &jwtTemplateResource{}
)

func NewJWTTemplateResource() resource.Resource {
	return &jwtTemplateResource{}
}

type jwtTemplateResource struct {
	client *api.API
}

type jwtTemplateModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectID       types.String `tfsdk:"project_id"`
	TemplateType    types.String `tfsdk:"template_type"`
	TemplateContent types.String `tfsdk:"template_content"`
	CustomAudience  types.String `tfsdk:"custom_audience"`
	LastUpdated     types.String `tfsdk:"last_updated"`
}

func (m jwtTemplateModel) toJWTTemplate() jwttemplates.JWTTemplate {
	return jwttemplates.JWTTemplate{
		TemplateType:    jwttemplates.TemplateType(m.TemplateType.ValueString()),
		TemplateContent: m.TemplateContent.ValueString(),
		CustomAudience:  m.CustomAudience.ValueString(),
	}
}

func (m *jwtTemplateModel) reloadFromJWTTemplate(jwtTemplate jwttemplates.JWTTemplate) {
	m.ID = types.StringValue(m.ProjectID.ValueString() + "." + m.TemplateType.ValueString())
	m.TemplateType = types.StringValue(string(jwtTemplate.TemplateType))
	m.TemplateContent = types.StringValue(jwtTemplate.TemplateContent)
	m.CustomAudience = types.StringValue(jwtTemplate.CustomAudience)
}

func (r *jwtTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
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

// Metadata returns the resource type name.
func (r *jwtTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_jwt_template"
}

// Schema defines the schema for the resource.
func (r *jwtTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Resource for creating and managing JWT templates.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier for the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of JWT template being created",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(toStrings(jwttemplates.TemplateTypes())...),
				},
			},
			"template_content": schema.StringAttribute{
				Required:    true,
				Description: "The content of the JWT template.",
			},
			"custom_audience": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "An optional custom audience for the JWT template",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the order.",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *jwtTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan jwtTemplateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	tflog.Info(ctx, "Creating JWT template")

	createResp, err := r.client.JWTTemplates.Set(ctx, &jwttemplates.SetRequest{
		ProjectID:   plan.ProjectID.ValueString(),
		JWTTemplate: plan.toJWTTemplate(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create JWT template", err.Error())
		return
	}

	tflog.Info(ctx, "JWT template created")

	plan.reloadFromJWTTemplate(createResp.JWTTemplate)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *jwtTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the current state
	var state jwtTemplateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	tflog.Info(ctx, "Reading JWT template")

	getResp, err := r.client.JWTTemplates.Get(ctx, &jwttemplates.GetRequest{
		ProjectID:    state.ProjectID.ValueString(),
		TemplateType: jwttemplates.TemplateType(state.TemplateType.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read JWT template", err.Error())
		return
	}

	tflog.Info(ctx, "Read JWT template")

	state.reloadFromJWTTemplate(getResp.JWTTemplate)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *jwtTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan jwtTemplateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	tflog.Info(ctx, "Updating JWT template")

	updateResp, err := r.client.JWTTemplates.Set(ctx, &jwttemplates.SetRequest{
		ProjectID:   plan.ProjectID.ValueString(),
		JWTTemplate: plan.toJWTTemplate(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update JWT template", err.Error())
		return
	}

	tflog.Info(ctx, "Updated JWT template")

	plan.reloadFromJWTTemplate(updateResp.JWTTemplate)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *jwtTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state jwtTemplateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	tflog.Info(ctx, "Setting JWT template to default value")

	_, err := r.client.JWTTemplates.Set(ctx, &jwttemplates.SetRequest{
		ProjectID: state.ProjectID.ValueString(),
		JWTTemplate: jwttemplates.JWTTemplate{
			TemplateType:    jwttemplates.TemplateType(state.TemplateType.ValueString()),
			TemplateContent: "{}",
			CustomAudience:  "",
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset JWT template state", err.Error())
		return
	}

	tflog.Info(ctx, "Reset JWT template to default state")
}

func (r *jwtTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ".")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "The ID must be in the format <project_id>.<template_type>")
		return
	}

	ctx = tflog.SetField(ctx, "project_id", parts[0])
	ctx = tflog.SetField(ctx, "template_type", parts[1])
	tflog.Info(ctx, "Importing JWT template")
	resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("template_type"), parts[1])
}
