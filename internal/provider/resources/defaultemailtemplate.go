package resources

import (
	"context"
	"fmt"
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
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/emailtemplates"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &defaultEmailTemplateResource{}
	_ resource.ResourceWithConfigure   = &defaultEmailTemplateResource{}
	_ resource.ResourceWithImportState = &defaultEmailTemplateResource{}
)

// NewDefaultEmailTemplateResource is a helper function to simplify the provider implementation.
func NewDefaultEmailTemplateResource() resource.Resource {
	return &defaultEmailTemplateResource{}
}

type defaultEmailTemplateResource struct {
	client *api.API
}

type defaultEmailTemplateModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectSlug       types.String `tfsdk:"project_slug"`
	LastUpdated       types.String `tfsdk:"last_updated"`
	EmailTemplateType types.String `tfsdk:"email_template_type"`
	TemplateID        types.String `tfsdk:"template_id"`
}

func (r *defaultEmailTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_default_email_template"
}

func (r *defaultEmailTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.API)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *api.API, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *defaultEmailTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the default email template for a specific template type in a project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the default email template mapping in the format `project_slug.email_template_type`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project for which to set the default email template",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the last Terraform update of the default email template.",
			},
			"email_template_type": schema.StringAttribute{
				Required:    true,
				Description: "The email template type for which to set the default template. Valid values: LOGIN, SIGNUP, INVITE, RESET_PASSWORD, ONE_TIME_PASSCODE, ONE_TIME_PASSCODE_SIGNUP, VERIFY_EMAIL_PASSWORD_RESET, UNLOCK, PREBUILT. Note that the PREBUILT type's default cannot be unset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(toStrings(emailtemplates.TemplateTypes())...),
				},
			},
			"template_id": schema.StringAttribute{
				Required:    true,
				Description: "The template ID of the email template to set as the default for this template type",
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *defaultEmailTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan defaultEmailTemplateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "email_template_type", plan.EmailTemplateType.ValueString())
	tflog.Info(ctx, "Setting default email template")

	_, err := r.client.EmailTemplates.SetDefault(ctx, emailtemplates.SetDefaultRequest{
		ProjectSlug:       plan.ProjectSlug.ValueString(),
		EmailTemplateType: emailtemplates.TemplateType(plan.EmailTemplateType.ValueString()),
		TemplateID:        plan.TemplateID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set default email template", err.Error())
		return
	}

	tflog.Info(ctx, "Set default email template")

	plan.ID = types.StringValue(fmt.Sprintf("%s.%s", plan.ProjectSlug.ValueString(), plan.EmailTemplateType.ValueString()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *defaultEmailTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state defaultEmailTemplateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "email_template_type", state.EmailTemplateType.ValueString())
	tflog.Info(ctx, "Reading default email template")

	getResp, err := r.client.EmailTemplates.GetDefault(ctx, emailtemplates.GetDefaultRequest{
		ProjectSlug:       state.ProjectSlug.ValueString(),
		EmailTemplateType: emailtemplates.TemplateType(state.EmailTemplateType.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get default email template", err.Error())
		return
	}

	tflog.Info(ctx, "Read default email template")

	state.TemplateID = types.StringValue(getResp.TemplateID)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *defaultEmailTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan defaultEmailTemplateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "email_template_type", plan.EmailTemplateType.ValueString())
	tflog.Info(ctx, "Updating default email template")

	_, err := r.client.EmailTemplates.SetDefault(ctx, emailtemplates.SetDefaultRequest{
		ProjectSlug:       plan.ProjectSlug.ValueString(),
		EmailTemplateType: emailtemplates.TemplateType(plan.EmailTemplateType.ValueString()),
		TemplateID:        plan.TemplateID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set default email template", err.Error())
		return
	}

	tflog.Info(ctx, "Updated default email template")

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *defaultEmailTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state defaultEmailTemplateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "email_template_type", state.EmailTemplateType.ValueString())
	tflog.Info(ctx, "Unsetting default email template")

	_, err := r.client.EmailTemplates.UnsetDefault(ctx, emailtemplates.UnsetDefaultRequest{
		ProjectSlug:       state.ProjectSlug.ValueString(),
		EmailTemplateType: emailtemplates.TemplateType(state.EmailTemplateType.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to unset default email template", err.Error())
		return
	}

	tflog.Info(ctx, "Unset default email template")
}

func (r *defaultEmailTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: project_slug.email_template_type
	idParts := types.StringValue(req.ID).ValueString()
	parts := types.StringValue(req.ID).ValueString()

	// Split by "."
	dotIndex := len(parts) - 1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '.' {
			dotIndex = i
			break
		}
	}

	if dotIndex == -1 || dotIndex == 0 || dotIndex == len(parts)-1 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'project_slug.email_template_type', got: %s", req.ID),
		)
		return
	}

	projectSlug := parts[:dotIndex]
	emailTemplateType := parts[dotIndex+1:]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), projectSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("email_template_type"), emailTemplateType)...)
}
