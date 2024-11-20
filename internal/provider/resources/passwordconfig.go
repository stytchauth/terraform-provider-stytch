package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stytchauth/stytch-management-go/pkg/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &passwordConfigResource{}
	_ resource.ResourceWithConfigure   = &passwordConfigResource{}
	_ resource.ResourceWithImportState = &passwordConfigResource{}
)

func NewPasswordConfigResource() resource.Resource {
	return &passwordConfigResource{}
}

type passwordConfigResource struct {
	client *api.API
}

type passwordConfigModel struct {
	ProjectID                   types.String `tfsdk:"project_id"`
	LastUpdated                 types.String `tfsdk:"last_updated"`
	CheckBreachOnCreate         types.Bool   `tfsdk:"check_breach_on_create"`
	CheckBreachOnAuthentication types.Bool   `tfsdk:"check_breach_on_authentication"`
	ValidateOnAuthentication    types.Bool   `tfsdk:"validate_on_authentication"`
	ValidationPolicy            types.String `tfsdk:"validation_policy"`
	LudsMinPasswordLength       types.Int32  `tfsdk:"luds_min_password_length"`
	LudsMinPasswordComplexity   types.Int32  `tfsdk:"luds_min_password_complexity"`
}

func (r *passwordConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *passwordConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password_config"
}

// Schema defines the schema for the resource.
func (r *passwordConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Required: true,
				Description: "The ID of the project for which to set the password config. " +
					"This can be either a live project ID or test project ID. " +
					"You may only specify one password config per project.",
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"check_breach_on_create": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to check if a user's password has been breached at the time of password creation using the HaveIBeenPwned database",
			},
			"check_breach_on_authentication": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to check if a user's password has been breached at the time of password authentication using the HaveIBeenPwned database",
			},
			"validate_on_authentication": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to validate that a password meets the project's password strength configuration at the time of authentication",
			},
			"validation_policy": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The policy to use for password strength validation, either ZXCVBN or LUDS",
			},
			"luds_min_password_length": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Description: "The minimum password length when using a LUDS validation policy. " +
					"If present, this value must be between 8 and 32.",
			},
			"luds_min_password_complexity": schema.Int32Attribute{
				Optional: true,
				Computed: true,
				Description: "The minimum number of character types (lowercase letters, uppercase letters, digits, and symbols) to require when using a LUDS validation policy. " +
					"If present, this value must be between 1 and 4.",
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *passwordConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan passwordConfigModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Generate API request body from plan and call r.client.PasswordStrengthConfig.Set

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *passwordConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the current state
	var state passwordConfigModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Get refreshed value from the API

	// Set refreshed state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *passwordConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan passwordConfigModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Generate API request body from plan and call r.client.PasswordStrengthConfig.Set

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *passwordConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state passwordConfigModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In this case, deleting a password config just means no longer tracaking its state in terraform.
	// We don't have to actually make a delete call within the API.
}

func (r *passwordConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("project_id"), req, resp)
}
