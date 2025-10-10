package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/projects"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/rbacpolicy"
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
	ID              types.String `tfsdk:"id"`
	ProjectSlug     types.String `tfsdk:"project_slug"`
	EnvironmentSlug types.String `tfsdk:"environment_slug"`
	LastUpdated     types.String `tfsdk:"last_updated"`
	// B2B-only fields
	StytchMember    types.Object `tfsdk:"stytch_member"`
	StytchAdmin     types.Object `tfsdk:"stytch_admin"`
	StytchResources types.Set    `tfsdk:"stytch_resources"`
	// Consumer-only field
	StytchUser types.Object `tfsdk:"stytch_user"`
	// Shared fields
	CustomRoles     types.Set `tfsdk:"custom_roles"`
	CustomResources types.Set `tfsdk:"custom_resources"`
	CustomScopes    types.Set `tfsdk:"custom_scopes"`
}

// toPolicy converts the Terraform model to an RBAC policy for API requests.
// It uses toDefaultRole() for default roles (stytch_member, stytch_admin, stytch_user) which returns
// a DefaultRole containing only permissions, as role_id and description cannot be customized.
// It uses toRole() for custom roles which returns a full Role with role_id, description, and permissions.
func (m rbacPolicyModel) toPolicy(ctx context.Context) (rbacpolicy.Policy, diag.Diagnostics) {
	var diags diag.Diagnostics
	var policy rbacpolicy.Policy

	// B2B fields - use rbacPolicyDefaultRoleModel and toDefaultRole() to create DefaultRole (only permissions)
	if !m.StytchMember.IsUnknown() && !m.StytchMember.IsNull() {
		var stytchMember rbacPolicyDefaultRoleModel
		diags.Append(m.StytchMember.As(ctx, &stytchMember, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		role := stytchMember.toDefaultRole()
		policy.StytchMember = &role
	}

	if !m.StytchAdmin.IsUnknown() && !m.StytchAdmin.IsNull() {
		var stytchAdmin rbacPolicyDefaultRoleModel
		diags.Append(m.StytchAdmin.As(ctx, &stytchAdmin, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		role := stytchAdmin.toDefaultRole()
		policy.StytchAdmin = &role
	}

	// Consumer field - use rbacPolicyDefaultRoleModel and toDefaultRole() for default roles
	if !m.StytchUser.IsUnknown() && !m.StytchUser.IsNull() {
		var stytchUser rbacPolicyDefaultRoleModel
		diags.Append(m.StytchUser.As(ctx, &stytchUser, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		role := stytchUser.toDefaultRole()
		policy.StytchUser = &role
	}

	// StytchResources can't be set by the provisioner, so we ignore it here.

	// Shared fields
	if !m.CustomRoles.IsUnknown() && !m.CustomRoles.IsNull() {
		var customRoles []rbacPolicyRoleModel
		diags.Append(m.CustomRoles.ElementsAs(ctx, &customRoles, true)...)

		policy.CustomRoles = make([]rbacpolicy.Role, len(customRoles))
		for i, roleModel := range customRoles {
			policy.CustomRoles[i] = roleModel.toRole()
		}
	}

	if !m.CustomResources.IsUnknown() && !m.CustomResources.IsNull() {
		var customResources []rbacPolicyResourceModel
		diags.Append(m.CustomResources.ElementsAs(ctx, &customResources, true)...)

		policy.CustomResources = make([]rbacpolicy.Resource, len(customResources))
		for i, resourceModel := range customResources {
			policy.CustomResources[i] = resourceModel.toResource()
		}
	}

	if !m.CustomScopes.IsUnknown() && !m.CustomScopes.IsNull() {
		var customScopes []rbacPolicyScopeModel
		diags.Append(m.CustomScopes.ElementsAs(ctx, &customScopes, true)...)

		policy.CustomScopes = make([]rbacpolicy.Scope, len(customScopes))
		for i, scopeModel := range customScopes {
			policy.CustomScopes[i] = scopeModel.toScope()
		}
	}

	return policy, diags
}

// reloadFromPolicy refreshes the Terraform state from an RBAC policy returned by the API.
// This method is used after Create, Read, and Update operations to ensure the state
// reflects the actual API response, including computed values like stytch_resources.
func (m *rbacPolicyModel) reloadFromPolicy(ctx context.Context, p rbacpolicy.Policy) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(m.ProjectSlug.ValueString() + "." + m.EnvironmentSlug.ValueString())

	// B2B fields
	if p.StytchMember != nil {
		stytchMember, diag := types.ObjectValueFrom(ctx, rbacPolicyDefaultRoleModel{}.AttributeTypes(), rbacPolicyDefaultRoleModelFrom(*p.StytchMember))
		diags = append(diags, diag...)
		m.StytchMember = stytchMember
	} else {
		m.StytchMember = types.ObjectNull(rbacPolicyDefaultRoleModel{}.AttributeTypes())
	}

	if p.StytchAdmin != nil {
		stytchAdmin, diag := types.ObjectValueFrom(ctx, rbacPolicyDefaultRoleModel{}.AttributeTypes(), rbacPolicyDefaultRoleModelFrom(*p.StytchAdmin))
		diags = append(diags, diag...)
		m.StytchAdmin = stytchAdmin
	} else {
		m.StytchAdmin = types.ObjectNull(rbacPolicyDefaultRoleModel{}.AttributeTypes())
	}

	stytchResources := make([]rbacPolicyResourceModel, len(p.StytchResources))
	for i, r := range p.StytchResources {
		stytchResources[i] = rbacPolicyResourceModelFrom(r)
	}
	stytchResourceSet, diag := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: rbacPolicyResourceModel{}.AttributeTypes()}, stytchResources)
	diags = append(diags, diag...)
	m.StytchResources = stytchResourceSet

	// Consumer field
	if p.StytchUser != nil {
		stytchUser, diag := types.ObjectValueFrom(ctx, rbacPolicyDefaultRoleModel{}.AttributeTypes(), rbacPolicyDefaultRoleModelFrom(*p.StytchUser))
		diags = append(diags, diag...)
		m.StytchUser = stytchUser
	} else {
		m.StytchUser = types.ObjectNull(rbacPolicyDefaultRoleModel{}.AttributeTypes())
	}

	// Shared fields
	customRoles := make([]rbacPolicyRoleModel, len(p.CustomRoles))
	for i, r := range p.CustomRoles {
		customRoles[i] = rbacPolicyRoleModelFrom(r)
	}
	customRoleSet, diag := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: rbacPolicyRoleModel{}.AttributeTypes()}, customRoles)
	diags = append(diags, diag...)
	m.CustomRoles = customRoleSet

	customResources := make([]rbacPolicyResourceModel, len(p.CustomResources))
	for i, r := range p.CustomResources {
		customResources[i] = rbacPolicyResourceModelFrom(r)
	}
	customResourceSet, diag := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: rbacPolicyResourceModel{}.AttributeTypes()}, customResources)
	diags = append(diags, diag...)
	m.CustomResources = customResourceSet

	customScopes := make([]rbacPolicyScopeModel, len(p.CustomScopes))
	for i, s := range p.CustomScopes {
		customScopes[i] = rbacPolicyScopeModelFrom(s)
	}
	customScopeSet, diag := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: rbacPolicyScopeModel{}.AttributeTypes()}, customScopes)
	diags = append(diags, diag...)
	m.CustomScopes = customScopeSet

	return diags
}

// rbacPolicyRoleModel represents a custom role with editable role_id, description, and permissions.
type rbacPolicyRoleModel struct {
	RoleID      types.String                `tfsdk:"role_id"`
	Description types.String                `tfsdk:"description"`
	Permissions []rbacPolicyPermissionModel `tfsdk:"permissions"`
}

func (m rbacPolicyRoleModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"role_id":     types.StringType,
		"description": types.StringType,
		"permissions": types.SetType{ElemType: types.ObjectType{AttrTypes: rbacPolicyPermissionModel{}.AttributeTypes()}},
	}
}

// rbacPolicyDefaultRoleModel represents a Stytch default role (stytch_member, stytch_admin, stytch_user)
// with only permissions. The role_id and description are managed by Stytch and cannot be customized.
type rbacPolicyDefaultRoleModel struct {
	Permissions []rbacPolicyPermissionModel `tfsdk:"permissions"`
}

func (m rbacPolicyDefaultRoleModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"permissions": types.SetType{ElemType: types.ObjectType{AttrTypes: rbacPolicyPermissionModel{}.AttributeTypes()}},
	}
}

// toRole converts a role model to a full Role for custom roles.
// Custom roles have editable role_id, description, and permissions.
func (m rbacPolicyRoleModel) toRole() rbacpolicy.Role {
	role := rbacpolicy.Role{
		RoleID:      m.RoleID.ValueString(),
		Description: m.Description.ValueString(),
		Permissions: make([]rbacpolicy.Permission, len(m.Permissions)),
	}
	for i, p := range m.Permissions {
		role.Permissions[i] = rbacpolicy.Permission{
			ResourceID: p.ResourceID.ValueString(),
			Actions:    make([]string, len(p.Actions)),
		}
		for j, a := range p.Actions {
			role.Permissions[i].Actions[j] = a.ValueString()
		}
	}
	return role
}

// toDefaultRole converts a default role model to a DefaultRole for API requests.
// Used for Stytch default roles (stytch_member, stytch_admin, stytch_user).
func (m rbacPolicyDefaultRoleModel) toDefaultRole() rbacpolicy.DefaultRole {
	role := rbacpolicy.DefaultRole{
		Permissions: make([]rbacpolicy.Permission, len(m.Permissions)),
	}
	for i, p := range m.Permissions {
		role.Permissions[i] = rbacpolicy.Permission{
			ResourceID: p.ResourceID.ValueString(),
			Actions:    make([]string, len(p.Actions)),
		}
		for j, a := range p.Actions {
			role.Permissions[i].Actions[j] = a.ValueString()
		}
	}
	return role
}

// rbacPolicyRoleModelFrom creates a role model from a full Role (used for custom roles).
// Custom roles have role_id and description fields that are user-defined.
func rbacPolicyRoleModelFrom(r rbacpolicy.Role) rbacPolicyRoleModel {
	perms := make([]rbacPolicyPermissionModel, len(r.Permissions))
	for i, p := range r.Permissions {
		perms[i] = rbacPolicyPermissionModelFrom(p)
	}
	return rbacPolicyRoleModel{
		RoleID:      types.StringValue(r.RoleID),
		Description: types.StringValue(r.Description),
		Permissions: perms,
	}
}

// rbacPolicyDefaultRoleModelFrom creates a default role model from a DefaultRole.
// Used for Stytch default roles (stytch_member, stytch_admin, stytch_user) which only have permissions.
func rbacPolicyDefaultRoleModelFrom(r rbacpolicy.DefaultRole) rbacPolicyDefaultRoleModel {
	perms := make([]rbacPolicyPermissionModel, len(r.Permissions))
	for i, p := range r.Permissions {
		perms[i] = rbacPolicyPermissionModelFrom(p)
	}
	return rbacPolicyDefaultRoleModel{
		Permissions: perms,
	}
}

type rbacPolicyResourceModel struct {
	ResourceID       types.String   `tfsdk:"resource_id"`
	Description      types.String   `tfsdk:"description"`
	AvailableActions []types.String `tfsdk:"available_actions"`
}

func (m rbacPolicyResourceModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"resource_id":       types.StringType,
		"description":       types.StringType,
		"available_actions": types.SetType{ElemType: types.StringType},
	}
}

// toResource converts a resource model to an RBAC Resource for API requests.
func (m rbacPolicyResourceModel) toResource() rbacpolicy.Resource {
	resource := rbacpolicy.Resource{
		ResourceID:       m.ResourceID.ValueString(),
		Description:      m.Description.ValueString(),
		AvailableActions: make([]string, len(m.AvailableActions)),
	}
	for i, a := range m.AvailableActions {
		resource.AvailableActions[i] = a.ValueString()
	}
	return resource
}

// rbacPolicyResourceModelFrom creates a resource model from an RBAC Resource.
func rbacPolicyResourceModelFrom(r rbacpolicy.Resource) rbacPolicyResourceModel {
	actions := make([]types.String, len(r.AvailableActions))
	for i, a := range r.AvailableActions {
		actions[i] = types.StringValue(a)
	}
	return rbacPolicyResourceModel{
		ResourceID:       types.StringValue(r.ResourceID),
		Description:      types.StringValue(r.Description),
		AvailableActions: actions,
	}
}

type rbacPolicyScopeModel struct {
	Scope       types.String                `tfsdk:"scope"`
	Description types.String                `tfsdk:"description"`
	Permissions []rbacPolicyPermissionModel `tfsdk:"permissions"`
}

func (m rbacPolicyScopeModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"scope":       types.StringType,
		"description": types.StringType,
		"permissions": types.SetType{ElemType: types.ObjectType{AttrTypes: rbacPolicyPermissionModel{}.AttributeTypes()}},
	}
}

// toScope converts a scope model to an RBAC Scope for API requests.
func (m rbacPolicyScopeModel) toScope() rbacpolicy.Scope {
	scope := rbacpolicy.Scope{
		Scope:       m.Scope.ValueString(),
		Description: m.Description.ValueString(),
		Permissions: make([]rbacpolicy.Permission, len(m.Permissions)),
	}
	for i, p := range m.Permissions {
		scope.Permissions[i] = rbacpolicy.Permission{
			ResourceID: p.ResourceID.ValueString(),
			Actions:    make([]string, len(p.Actions)),
		}
		for j, a := range p.Actions {
			scope.Permissions[i].Actions[j] = a.ValueString()
		}
	}
	return scope
}

// rbacPolicyScopeModelFrom creates a scope model from an RBAC Scope.
func rbacPolicyScopeModelFrom(s rbacpolicy.Scope) rbacPolicyScopeModel {
	perms := make([]rbacPolicyPermissionModel, len(s.Permissions))
	for i, p := range s.Permissions {
		perms[i] = rbacPolicyPermissionModelFrom(p)
	}
	return rbacPolicyScopeModel{
		Scope:       types.StringValue(s.Scope),
		Description: types.StringValue(s.Description),
		Permissions: perms,
	}
}

type rbacPolicyPermissionModel struct {
	ResourceID types.String   `tfsdk:"resource_id"`
	Actions    []types.String `tfsdk:"actions"`
}

func (m rbacPolicyPermissionModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"resource_id": types.StringType,
		"actions":     types.SetType{ElemType: types.StringType},
	}
}

// rbacPolicyPermissionModelFrom creates a permission model from an RBAC Permission.
func rbacPolicyPermissionModelFrom(p rbacpolicy.Permission) rbacPolicyPermissionModel {
	actions := make([]types.String, len(p.Actions))
	for i, a := range p.Actions {
		actions[i] = types.StringValue(a)
	}
	return rbacPolicyPermissionModel{
		ResourceID: types.StringValue(p.ResourceID),
		Actions:    actions,
	}
}

func (r *rbacPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// defaultRoleAttributes defines the schema for Stytch default roles (stytch_member, stytch_admin, stytch_user).
// These roles only have permissions; role_id and description are managed by Stytch.
var defaultRoleAttributes = map[string]schema.Attribute{
	"permissions": schema.SetNestedAttribute{
		Optional: true,
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"resource_id": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The ID of the resource that the role can perform actions on.",
				},
				"actions": schema.SetAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "An array of actions that the role can perform on the given resource",
				},
			},
		},
	},
}

// roleAttributes defines the schema for custom roles with editable role_id, description, and permissions.
var roleAttributes = map[string]schema.Attribute{
	"role_id": schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "A human-readable name that is unique within the environment",
	},
	"description": schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "A description of the role",
	},
	"permissions": schema.SetNestedAttribute{
		Optional: true,
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"resource_id": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The ID of the resource that the role can perform actions on.",
				},
				"actions": schema.SetAttribute{
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
		Description: "A human-readable name that is unique within the environment",
	},
	"description": schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "A description of the resource",
	},
	"available_actions": schema.SetAttribute{
		Optional:    true,
		Computed:    true,
		ElementType: types.StringType,
		Description: "The actions that can be granted for this resource",
	},
}

// Schema defines the schema for the resource.
func (r *rbacPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A role-based access control (RBAC) policy for an environment. Supports both B2B and Consumer projects.\n\n" +
			"**B2B-specific fields:** stytch_member, stytch_admin, stytch_resources\n" +
			"**Consumer-specific fields:** stytch_user\n" +
			"**Shared fields:** custom_roles, custom_resources, custom_scopes\n\n" +
			"**Important:** Stytch default roles (stytch_member, stytch_admin, stytch_user) have required permissions for Stytch resources " +
			"that cannot be removed. These permissions must always be included when modifying default roles.",
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
				Description: "The slug of the project to which the RBAC policy belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to which the RBAC policy belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update.",
				Computed:    true,
			},
			"stytch_member": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Description: "**B2B only:** The default role given to members within the environment. " +
					"Default permissions for Stytch resources must be retained.",
				Attributes: defaultRoleAttributes,
			},
			"stytch_admin": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Description: "**B2B only:** The role assigned to admins within an organization. " +
					"Default permissions for Stytch resources must be retained.",
				Attributes: defaultRoleAttributes,
			},
			"stytch_resources": schema.SetNestedAttribute{
				Computed: true,
				Description: "**B2B only:** Resources created by Stytch that always exist. " +
					"This field is read-only and cannot be overridden or deleted.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: resourceAttributes,
				},
			},
			"stytch_user": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Description: "**Consumer only:** The default role given to users within the environment. " +
					"Default permissions for Stytch resources must be retained.",
				Attributes: defaultRoleAttributes,
			},
			"custom_roles": schema.SetNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Additional roles that exist within the environment beyond the default Stytch roles.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: roleAttributes,
				},
			},
			"custom_resources": schema.SetNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Resources that exist within the environment beyond those defined in stytch_resources.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: resourceAttributes,
				},
			},
			"custom_scopes": schema.SetNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Additional scopes that exist within the environment beyond those defined by default.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"scope": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "A human-readable name that is unique within the environment",
						},
						"description": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "A description of the scope",
						},
						"permissions": schema.SetNestedAttribute{
							Optional: true,
							Computed: true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"resource_id": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "The ID of the resource that the scope can perform actions on.",
									},
									"actions": schema.SetAttribute{
										Optional:    true,
										Computed:    true,
										ElementType: types.StringType,
										Description: "An array of actions that the scope can perform on the given resource",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r rbacPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data rbacPolicyModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the projectSlug isn't yet known or the provider client is not injected, skip validation for now.
	if data.ProjectSlug.IsUnknown() || r.client == nil {
		return
	}

	getProjectResp, err := r.client.Projects.Get(ctx, projects.GetRequest{
		ProjectSlug: data.ProjectSlug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddWarning("Failed to get project for vertical check", err.Error())
		return
	}

	// Validate B2B-specific fields are only used with B2B projects
	if getProjectResp.Project.Vertical == projects.VerticalB2B {
		if !data.StytchUser.IsNull() && !data.StytchUser.IsUnknown() {
			resp.Diagnostics.AddError("Invalid field for B2B project", "stytch_user field can only be used with Consumer projects")
		}
	}

	// Validate Consumer-specific fields are only used with Consumer projects
	if getProjectResp.Project.Vertical == projects.VerticalConsumer {
		if !data.StytchMember.IsNull() && !data.StytchMember.IsUnknown() {
			resp.Diagnostics.AddError("Invalid field for Consumer project", "stytch_member field can only be used with B2B projects")
		}
		if !data.StytchAdmin.IsNull() && !data.StytchAdmin.IsUnknown() {
			resp.Diagnostics.AddError("Invalid field for Consumer project", "stytch_admin field can only be used with B2B projects")
		}
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

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Creating RBAC policy")

	policy, diags := plan.toPolicy(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the policy - the API will return the full policy including defaults
	setResp, err := r.client.RBACPolicy.Set(ctx, rbacpolicy.SetRequest{
		ProjectSlug:     plan.ProjectSlug.ValueString(),
		EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
		StytchMember:    policy.StytchMember,
		StytchAdmin:     policy.StytchAdmin,
		StytchUser:      policy.StytchUser,
		CustomRoles:     policy.CustomRoles,
		CustomResources: policy.CustomResources,
		CustomScopes:    policy.CustomScopes,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set RBAC policy", err.Error())
		return
	}

	tflog.Info(ctx, "Created RBAC policy")

	// Reload from the API response to get all computed values
	diags = plan.reloadFromPolicy(ctx, setResp.Policy)
	resp.Diagnostics.Append(diags...)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *rbacPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state rbacPolicyModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Reading RBAC policy")

	getResp, err := r.client.RBACPolicy.Get(ctx, rbacpolicy.GetRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get RBAC policy", err.Error())
		return
	}

	tflog.Info(ctx, "Read RBAC policy")

	diags = state.reloadFromPolicy(ctx, getResp.Policy)
	resp.Diagnostics.Append(diags...)
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

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Updating RBAC policy")

	policy, diags := plan.toPolicy(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the policy - the API will return the full policy including defaults
	setResp, err := r.client.RBACPolicy.Set(ctx, rbacpolicy.SetRequest{
		ProjectSlug:     plan.ProjectSlug.ValueString(),
		EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
		StytchMember:    policy.StytchMember,
		StytchAdmin:     policy.StytchAdmin,
		StytchUser:      policy.StytchUser,
		CustomRoles:     policy.CustomRoles,
		CustomResources: policy.CustomResources,
		CustomScopes:    policy.CustomScopes,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set RBAC policy", err.Error())
		return
	}

	tflog.Info(ctx, "Updated RBAC policy")

	// Reload from the API response to get all computed values
	diags = plan.reloadFromPolicy(ctx, setResp.Policy)
	resp.Diagnostics.Append(diags...)
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

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Deleting RBAC policy (resetting to defaults)")

	// Get current policy to reset to defaults
	getResp, err := r.client.RBACPolicy.Get(ctx, rbacpolicy.GetRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get RBAC policy", err.Error())
		return
	}

	policy := getResp.Policy

	// Remove custom resource permissions from default roles
	resourcesToRemove := make(map[string]struct{})
	for _, resource := range policy.CustomResources {
		resourcesToRemove[resource.ResourceID] = struct{}{}
	}

	// Filter permissions for B2B roles
	if policy.StytchMember != nil {
		var keepPerms []rbacpolicy.Permission
		for _, perm := range policy.StytchMember.Permissions {
			if _, ok := resourcesToRemove[perm.ResourceID]; !ok {
				keepPerms = append(keepPerms, perm)
			}
		}
		policy.StytchMember.Permissions = keepPerms
	}

	if policy.StytchAdmin != nil {
		var keepPerms []rbacpolicy.Permission
		for _, perm := range policy.StytchAdmin.Permissions {
			if _, ok := resourcesToRemove[perm.ResourceID]; !ok {
				keepPerms = append(keepPerms, perm)
			}
		}
		policy.StytchAdmin.Permissions = keepPerms
	}

	// Filter permissions for Consumer role
	if policy.StytchUser != nil {
		var keepPerms []rbacpolicy.Permission
		for _, perm := range policy.StytchUser.Permissions {
			if _, ok := resourcesToRemove[perm.ResourceID]; !ok {
				keepPerms = append(keepPerms, perm)
			}
		}
		policy.StytchUser.Permissions = keepPerms
	}

	// Reset custom roles, resources, and scopes
	policy.CustomRoles = []rbacpolicy.Role{}
	policy.CustomResources = []rbacpolicy.Resource{}
	policy.CustomScopes = []rbacpolicy.Scope{}

	_, err = r.client.RBACPolicy.Set(ctx, rbacpolicy.SetRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		StytchMember:    policy.StytchMember,
		StytchAdmin:     policy.StytchAdmin,
		StytchUser:      policy.StytchUser,
		CustomRoles:     policy.CustomRoles,
		CustomResources: policy.CustomResources,
		CustomScopes:    policy.CustomScopes,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset RBAC policy", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted RBAC policy (reset to defaults)")
}

func (r *rbacPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: project_slug.environment_slug
	parts := strings.Split(req.ID, ".")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "The ID must be in the format <project_slug>.<environment_slug>")
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", parts[0])
	ctx = tflog.SetField(ctx, "environment_slug", parts[1])
	tflog.Info(ctx, "Importing RBAC policy")

	resp.State.SetAttribute(ctx, path.Root("project_slug"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("environment_slug"), parts[1])
}
