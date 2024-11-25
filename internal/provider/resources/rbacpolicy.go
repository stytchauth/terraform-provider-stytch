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
	_ resource.Resource                = &rbacPolicyResource{}
	_ resource.ResourceWithConfigure   = &rbacPolicyResource{}
	_ resource.ResourceWithImportState = &rbacPolicyResource{}
)

func NewRBACPolicyResource() resource.Resource {
	return &rbacPolicyResource{}
}

type rbacPolicyResource struct {
	client *api.API
}

type rbacPolicyModel struct {
	ProjectID       types.String              `tfsdk:"project_id"`
	LastUpdated     types.String              `tfsdk:"last_updated"`
	StytchMember    rbacPolicyRoleModel       `tfsdk:"stytch_member"`
	StytchAdmin     rbacPolicyRoleModel       `tfsdk:"stytch_admin"`
	StytchResources []rbacPolicyResourceModel `tfsdk:"stytch_resources"`
	CustomRoles     []rbacPolicyRoleModel     `tfsdk:"custom_roles"`
	CustomResources []rbacPolicyResourceModel `tfsdk:"custom_resources"`
}

type rbacPolicyRoleModel struct {
	RoleID      types.String                `tfsdk:"role_id"`
	Description types.String                `tfsdk:"description"`
	Permissions []rbacPolicyPermissionModel `tfsdk:"permissions"`
}

type rbacPolicyResourceModel struct {
	ResourceID       types.String   `tfsdk:"resource_id"`
	Description      types.String   `tfsdk:"description"`
	AvailableActions []types.String `tfsdk:"available_actions"`
}

type rbacPolicyPermissionModel struct {
	ResourceID types.String   `tfsdk:"resource_id"`
	Actions    []types.String `tfsdk:"actions"`
}

func (r *rbacPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *rbacPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rbac_policy"
}

var roleAttributes = map[string]schema.Attribute{
	"role_id": schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "A human-readable name that is unique within the project",
	},
	"description": schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "A description of the role",
	},
	"permissions": schema.ListNestedAttribute{
		Optional: true,
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"resource_id": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The ID of the resource that the role can perform actions on.",
				},
				"actions": schema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "An array of actions that the role can perform on the given resource",
				},
			},
		},
	},
}

var resourceAttributes = map[string]schema.Attribute{
	"resource_id": schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "A human-readable name that is unique within the project",
	},
	"description": schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "A description of the resource",
	},
	"available_actions": schema.ListAttribute{
		Optional:    true,
		Computed:    true,
		ElementType: types.StringType,
		Description: "The actions that can be granted for this resource",
	},
}

// Schema defines the schema for the resource.
func (r *rbacPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier for the project.",
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"stytch_member": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The default role given to members within the project",
				Attributes:  roleAttributes,
			},
			"stytch_admin": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The role assigned to admins within an organization",
				Attributes:  roleAttributes,
			},
			"stytch_resources": schema.ListNestedAttribute{
				Computed: true,
				Description: "StytchResources consists of resources created by Stytch that always exist. " +
					"This field will be returned in relevant Policy objects but can never be overridden or deleted.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: resourceAttributes,
				},
			},
			"custom_roles": schema.ListNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Additional roles that exist within the project beyond the stytch_member or stytch_admin roles",
				NestedObject: schema.NestedAttributeObject{
					Attributes: roleAttributes,
				},
			},
			"custom_resources": schema.ListNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Resources that exist within the project beyond those defined within the stytch_resources",
				NestedObject: schema.NestedAttributeObject{
					Attributes: resourceAttributes,
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *rbacPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan rbacPolicyModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Generate API request body from plan and call r.client.RBAC.SetPolicy

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *rbacPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the current state
	var state rbacPolicyModel
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
func (r *rbacPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan rbacPolicyModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Generate API request body from plan and call r.client.RBAC.SetPolicy

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *rbacPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state rbacPolicyModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In this case, deleting a password config just means no longer tracaking its state in terraform.
	// We don't have to actually make a delete call within the API.
	// Would it make sense to set some sort of policy "defaults" again?
}

func (r *rbacPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("project_id"), req, resp)
}
