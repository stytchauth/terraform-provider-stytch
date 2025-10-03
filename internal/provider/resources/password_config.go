package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/passwordstrengthconfig"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                     = &passwordConfigResource{}
	_ resource.ResourceWithConfigure        = &passwordConfigResource{}
	_ resource.ResourceWithImportState      = &passwordConfigResource{}
	_ resource.ResourceWithConfigValidators = &passwordConfigResource{}
)

func NewPasswordConfigResource() resource.Resource {
	return &passwordConfigResource{}
}

type passwordConfigResource struct {
	client *api.API
}

type passwordConfigModel struct {
	ID                          types.String `tfsdk:"id"`
	ProjectSlug                 types.String `tfsdk:"project_slug"`
	EnvironmentSlug             types.String `tfsdk:"environment_slug"`
	CheckBreachOnCreation       types.Bool   `tfsdk:"check_breach_on_creation"`
	CheckBreachOnAuthentication types.Bool   `tfsdk:"check_breach_on_authentication"`
	ValidateOnAuthentication    types.Bool   `tfsdk:"validate_on_authentication"`
	ValidationPolicy            types.String `tfsdk:"validation_policy"`
	LudsMinPasswordLength       types.Int64  `tfsdk:"luds_min_password_length"`
	LudsMinPasswordComplexity   types.Int64  `tfsdk:"luds_min_password_complexity"`
	LastUpdated                 types.String `tfsdk:"last_updated"`
}

func (r *passwordConfigResource) Configure(
	ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.API)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *api.API (stytch-management-go client), got: %T. Please report "+
				"this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *passwordConfigResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_password_config"
}

func (r *passwordConfigResource) Schema(
	_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Manages password strength configuration for an environment within a Stytch project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management (format: project_slug.environment_slug).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project to which the password config belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to which the password config belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"check_breach_on_creation": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to use the HaveIBeenPwned database to detect password breaches when a user first creates their password.",
			},
			"check_breach_on_authentication": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to use the HaveIBeenPwned database to detect password breaches when a user authenticates.",
			},
			"validate_on_authentication": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to require a password reset on authentication if a user's current password no longer meets the project's current policy requirements.",
			},
			"validation_policy": schema.StringAttribute{
				Required:    true,
				Description: "The policy to use for password validation. Valid values: LUDS, ZXCVBN.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						toStrings(passwordstrengthconfig.ValidationPolicies())...),
				},
			},
			"luds_min_password_length": schema.Int64Attribute{
				Optional:    true,
				Description: "The minimum number of characters in a password if using a LUDS validation_policy. Must be between 8 and 32.",
				Validators: []validator.Int64{
					int64validator.Between(8, 32),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"luds_min_password_complexity": schema.Int64Attribute{
				Optional:    true,
				Description: "The minimum number of character types (Lowercase, Uppercase, Digits, Symbols) in a password when using a LUDS validation_policy. Must be between 1 and 4.",
				Validators: []validator.Int64{
					int64validator.Between(1, 4),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update.",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *passwordConfigResource) Create(
	ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse,
) {
	var plan passwordConfigModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Creating password config")

	setRequest := passwordstrengthconfig.SetRequest{
		ProjectSlug:                 plan.ProjectSlug.ValueString(),
		EnvironmentSlug:             plan.EnvironmentSlug.ValueString(),
		CheckBreachOnCreation:       plan.CheckBreachOnCreation.ValueBool(),
		CheckBreachOnAuthentication: plan.CheckBreachOnAuthentication.ValueBool(),
		ValidateOnAuthentication:    plan.ValidateOnAuthentication.ValueBool(),
		ValidationPolicy:            passwordstrengthconfig.ValidationPolicy(plan.ValidationPolicy.ValueString()),
	}

	if !plan.LudsMinPasswordLength.IsNull() {
		length := int(plan.LudsMinPasswordLength.ValueInt64())
		setRequest.LudsMinPasswordLength = &length
	}

	if !plan.LudsMinPasswordComplexity.IsNull() {
		complexity := int(plan.LudsMinPasswordComplexity.ValueInt64())
		setRequest.LudsMinPasswordComplexity = &complexity
	}

	setResp, err := r.client.PasswordStrengthConfig.Set(ctx, setRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create password config", err.Error())
		return
	}

	tflog.Info(ctx, "Created password config")

	plan.ID = types.StringValue(fmt.Sprintf("%s.%s", plan.ProjectSlug.ValueString(), plan.EnvironmentSlug.ValueString()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	r.updateModelFromAPI(&plan, &setResp.PasswordStrengthConfig)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *passwordConfigResource) Read(
	ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse,
) {
	var state passwordConfigModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Reading password config")

	getResp, err := r.client.PasswordStrengthConfig.Get(ctx, passwordstrengthconfig.GetRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get password config", err.Error())
		return
	}

	tflog.Info(ctx, "Read password config")

	r.updateModelFromAPI(&state, &getResp.PasswordStrengthConfig)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *passwordConfigResource) Update(
	ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	var plan passwordConfigModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Updating password config")

	setRequest := passwordstrengthconfig.SetRequest{
		ProjectSlug:                 plan.ProjectSlug.ValueString(),
		EnvironmentSlug:             plan.EnvironmentSlug.ValueString(),
		CheckBreachOnCreation:       plan.CheckBreachOnCreation.ValueBool(),
		CheckBreachOnAuthentication: plan.CheckBreachOnAuthentication.ValueBool(),
		ValidateOnAuthentication:    plan.ValidateOnAuthentication.ValueBool(),
		ValidationPolicy:            passwordstrengthconfig.ValidationPolicy(plan.ValidationPolicy.ValueString()),
	}

	if !plan.LudsMinPasswordLength.IsNull() {
		length := int(plan.LudsMinPasswordLength.ValueInt64())
		setRequest.LudsMinPasswordLength = &length
	}

	if !plan.LudsMinPasswordComplexity.IsNull() {
		complexity := int(plan.LudsMinPasswordComplexity.ValueInt64())
		setRequest.LudsMinPasswordComplexity = &complexity
	}

	setResp, err := r.client.PasswordStrengthConfig.Set(ctx, setRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update password config", err.Error())
		return
	}

	tflog.Info(ctx, "Updated password config")

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	r.updateModelFromAPI(&plan, &setResp.PasswordStrengthConfig)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
// The actual policy is not deleted, only reset to Stytch default values.
func (r *passwordConfigResource) Delete(
	ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	var state passwordConfigModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Deleting password config (resetting to defaults)")

	// Reset to default values
	setRequest := passwordstrengthconfig.SetRequest{
		ProjectSlug:                 state.ProjectSlug.ValueString(),
		EnvironmentSlug:             state.EnvironmentSlug.ValueString(),
		CheckBreachOnCreation:       true,
		CheckBreachOnAuthentication: true,
		ValidateOnAuthentication:    true,
		ValidationPolicy:            passwordstrengthconfig.ValidationPolicyZXCVBN,
	}

	_, err := r.client.PasswordStrengthConfig.Set(ctx, setRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset password config", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted password config (reset to defaults)")
}

// ImportState imports an existing password config into Terraform state.
func (r *passwordConfigResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	ctx = tflog.SetField(ctx, "import_id", req.ID)
	tflog.Info(ctx, "Importing password config")

	// Import ID format: project_slug.environment_slug
	parts := strings.Split(req.ID, ".")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("Expected import ID format: project_slug.environment_slug, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_slug"), parts[1])...)
}

// ConfigValidators returns validators for the resource configuration.
func (r *passwordConfigResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&ludsFieldsValidator{},
	}
}

// ludsFieldsValidator ensures LUDS fields are not set when using ZXCVBN policy.
type ludsFieldsValidator struct{}

func (v *ludsFieldsValidator) Description(ctx context.Context) string {
	return "Validates that LUDS fields are not set when validation_policy is ZXCVBN"
}

func (v *ludsFieldsValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates that LUDS fields are not set when validation_policy is ZXCVBN"
}

func (v *ludsFieldsValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config passwordConfigModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only validate if validation_policy is set
	if config.ValidationPolicy.IsNull() || config.ValidationPolicy.IsUnknown() {
		return
	}

	policy := config.ValidationPolicy.ValueString()
	if policy == string(passwordstrengthconfig.ValidationPolicyZXCVBN) {
		if !config.LudsMinPasswordLength.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("luds_min_password_length"),
				"Invalid Attribute Configuration",
				"luds_min_password_length cannot be set when validation_policy is ZXCVBN",
			)
		}
		if !config.LudsMinPasswordComplexity.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("luds_min_password_complexity"),
				"Invalid Attribute Configuration",
				"luds_min_password_complexity cannot be set when validation_policy is ZXCVBN",
			)
		}
	}
}

// updateModelFromAPI updates the model with values from the API response
func (r *passwordConfigResource) updateModelFromAPI(model *passwordConfigModel, config *passwordstrengthconfig.PasswordStrengthConfig) {
	model.CheckBreachOnCreation = types.BoolValue(config.CheckBreachOnCreation)
	model.CheckBreachOnAuthentication = types.BoolValue(config.CheckBreachOnAuthentication)
	model.ValidateOnAuthentication = types.BoolValue(config.ValidateOnAuthentication)
	model.ValidationPolicy = types.StringValue(string(config.ValidationPolicy))

	if config.LudsMinPasswordLength != nil {
		model.LudsMinPasswordLength = types.Int64Value(int64(*config.LudsMinPasswordLength))
	} else {
		model.LudsMinPasswordLength = types.Int64Null()
	}

	if config.LudsMinPasswordComplexity != nil {
		model.LudsMinPasswordComplexity = types.Int64Value(int64(*config.LudsMinPasswordComplexity))
	} else {
		model.LudsMinPasswordComplexity = types.Int64Null()
	}
}
