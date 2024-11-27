package resources

import (
	"context"
	"fmt"
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
	"github.com/stytchauth/stytch-management-go/pkg/api"
	"github.com/stytchauth/stytch-management-go/pkg/models/projects"
	"github.com/stytchauth/stytch-management-go/pkg/models/rbacpolicy"
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
	ProjectID       types.String `tfsdk:"project_id"`
	LastUpdated     types.String `tfsdk:"last_updated"`
	StytchMember    types.Object `tfsdk:"stytch_member"`
	StytchAdmin     types.Object `tfsdk:"stytch_admin"`
	StytchResources types.Set    `tfsdk:"stytch_resources"`
	CustomRoles     types.Set    `tfsdk:"custom_roles"`
	CustomResources types.Set    `tfsdk:"custom_resources"`
}

func (m rbacPolicyModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":            types.StringType,
		"project_id":    types.StringType,
		"last_updated":  types.StringType,
		"stytch_member": types.ObjectType{AttrTypes: rbacPolicyRoleModel{}.AttributeTypes()},
		"stytch_admin":  types.ObjectType{AttrTypes: rbacPolicyRoleModel{}.AttributeTypes()},
		"stytch_resources": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: rbacPolicyResourceModel{}.AttributeTypes(),
			},
		},
		"custom_roles": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: rbacPolicyRoleModel{}.AttributeTypes(),
			},
		},
		"custom_resources": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: rbacPolicyResourceModel{}.AttributeTypes(),
			},
		},
	}
}

func (m rbacPolicyModel) toPolicy(ctx context.Context) (rbacpolicy.Policy, diag.Diagnostics) {
	var diags diag.Diagnostics
	var policy rbacpolicy.Policy

	if !m.StytchMember.IsUnknown() {
		var stytchMember rbacPolicyRoleModel
		diags.Append(m.StytchMember.As(ctx, &stytchMember, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		policy.StytchMember = stytchMember.toRole()
	}

	if !m.StytchAdmin.IsUnknown() {
		var stytchAdmin rbacPolicyRoleModel
		diags.Append(m.StytchAdmin.As(ctx, &stytchAdmin, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		policy.StytchAdmin = stytchAdmin.toRole()
	}

	// StytchResources can't be set by the provisioner, so we ignore it here.

	if !m.CustomRoles.IsUnknown() {
		var customRoles []rbacPolicyRoleModel
		diags.Append(m.CustomRoles.ElementsAs(ctx, &customRoles, true)...)

		policy.CustomRoles = make([]rbacpolicy.Role, len(customRoles))
		for i, roleModel := range customRoles {
			policy.CustomRoles[i] = roleModel.toRole()
		}
	}

	if !m.CustomResources.IsUnknown() {
		var customResources []rbacPolicyResourceModel
		diags.Append(m.CustomResources.ElementsAs(ctx, &customResources, true)...)

		policy.CustomResources = make([]rbacpolicy.Resource, len(customResources))
		for i, resourceModel := range customResources {
			policy.CustomResources[i] = resourceModel.toResource()
		}
	}

	return policy, diags
}

func (m *rbacPolicyModel) reloadFromPolicy(ctx context.Context, p rbacpolicy.Policy) diag.Diagnostics {
	var diags diag.Diagnostics

	stytchMember, diag := types.ObjectValueFrom(ctx, rbacPolicyRoleModel{}.AttributeTypes(), rbacPolicyRoleModelFrom(p.StytchMember))
	diags = append(diags, diag...)

	stytchAdmin, diag := types.ObjectValueFrom(ctx, rbacPolicyRoleModel{}.AttributeTypes(), rbacPolicyRoleModelFrom(p.StytchAdmin))
	diags = append(diags, diag...)

	stytchResources := make([]rbacPolicyResourceModel, len(p.StytchResources))
	for i, r := range p.StytchResources {
		stytchResources[i] = rbacPolicyResourceModelFrom(r)
	}
	stytchResourceSet, diag := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: rbacPolicyResourceModel{}.AttributeTypes()}, stytchResources)
	diags = append(diags, diag...)

	customRoles := make([]rbacPolicyRoleModel, len(p.CustomRoles))
	for i, r := range p.CustomRoles {
		customRoles[i] = rbacPolicyRoleModelFrom(r)
	}
	customRoleSet, diag := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: rbacPolicyRoleModel{}.AttributeTypes()}, customRoles)
	diags = append(diags, diag...)

	customResources := make([]rbacPolicyResourceModel, len(p.CustomResources))
	for i, r := range p.CustomResources {
		customResources[i] = rbacPolicyResourceModelFrom(r)
	}
	customResourceSet, diag := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: rbacPolicyResourceModel{}.AttributeTypes()}, customResources)
	diags = append(diags, diag...)

	m.ID = m.ProjectID
	m.StytchMember = stytchMember
	m.StytchAdmin = stytchAdmin
	m.StytchResources = stytchResourceSet
	m.CustomRoles = customRoleSet
	m.CustomResources = customResourceSet

	return diags
}

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
		Description: "A human-readable name that is unique within the project",
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
		Description: "A role-based access control (RBAC) policy for a B2B project.",
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
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the order.",
				Computed:    true,
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
			"stytch_resources": schema.SetNestedAttribute{
				Computed: true,
				Description: "StytchResources consists of resources created by Stytch that always exist. " +
					"This field will be returned in relevant Policy objects but can never be overridden or deleted.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: resourceAttributes,
				},
			},
			"custom_roles": schema.SetNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Additional roles that exist within the project beyond the stytch_member or stytch_admin roles",
				NestedObject: schema.NestedAttributeObject{
					Attributes: roleAttributes,
				},
			},
			"custom_resources": schema.SetNestedAttribute{
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

func (r rbacPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data rbacPolicyModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the projectID isn't yet known, skip validation for now.
	// The plugin framework will call ValidateConfig again when all required values are known.
	if data.ProjectID.IsUnknown() {
		return
	}

	getProjectResp, err := r.client.Projects.Get(ctx, projects.GetRequest{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddWarning("Failed to get project for vertical check", err.Error())
		return
	}
	if getProjectResp.Project.Vertical != projects.VerticalB2B {
		resp.Diagnostics.AddError("Invalid project vertical", "The project must be a B2B project for this resource.")
		return
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

	// Since we're not allowed to edit certain attributes of the default stytch member
	// and the RBAC API requires setting *all* fields, we now *retrieve* the current value.
	if plan.StytchMember.IsUnknown() || plan.StytchAdmin.IsUnknown() {
		getResp, err := r.client.RBACPolicy.Get(ctx, rbacpolicy.GetRequest{
			ProjectID: plan.ProjectID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to fetch default Stytch role", "Failed to fetch default Stytch member or admin role for RBAC policy")
			return
		}

		stytchMember, diag := types.ObjectValueFrom(ctx, rbacPolicyRoleModel{}.AttributeTypes(), rbacPolicyRoleModelFrom(getResp.Policy.StytchMember))
		resp.Diagnostics.Append(diag...)
		stytchAdmin, diag := types.ObjectValueFrom(ctx, rbacPolicyRoleModel{}.AttributeTypes(), rbacPolicyRoleModelFrom(getResp.Policy.StytchAdmin))
		resp.Diagnostics.Append(diag...)

		plan.StytchMember = stytchMember
		plan.StytchAdmin = stytchAdmin
	}

	policy, diags := plan.toPolicy(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.RBACPolicy.Set(ctx, rbacpolicy.SetRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Policy:    policy,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set RBAC policy", err.Error())
		return
	}

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
	// Get the current state
	var state rbacPolicyModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := r.client.RBACPolicy.Get(ctx, rbacpolicy.GetRequest{
		ProjectID: state.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get RBAC policy", err.Error())
		return
	}

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

	// Since we're not allowed to edit certain attributes of the default stytch member
	// and the RBAC API requires setting *all* fields, we now *retrieve* the current value.
	if plan.StytchMember.IsUnknown() || plan.StytchAdmin.IsUnknown() {
		getResp, err := r.client.RBACPolicy.Get(ctx, rbacpolicy.GetRequest{
			ProjectID: plan.ProjectID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to fetch default Stytch role", "Failed to fetch default Stytch member or admin role for RBAC policy")
			return
		}

		stytchMember, diag := types.ObjectValueFrom(ctx, rbacPolicyRoleModel{}.AttributeTypes(), rbacPolicyRoleModelFrom(getResp.Policy.StytchMember))
		resp.Diagnostics.Append(diag...)
		stytchAdmin, diag := types.ObjectValueFrom(ctx, rbacPolicyRoleModel{}.AttributeTypes(), rbacPolicyRoleModelFrom(getResp.Policy.StytchAdmin))
		resp.Diagnostics.Append(diag...)

		plan.StytchMember = stytchMember
		plan.StytchAdmin = stytchAdmin
	}

	policy, diags := plan.toPolicy(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.RBACPolicy.Set(ctx, rbacpolicy.SetRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Policy:    policy,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set RBAC policy", err.Error())
		return
	}

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

	// To delete this resource, we remove all custom roles and resources
	policy, diags := state.toPolicy(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// First find the resources that need to be removed from the default roles
	resourcesToRemove := make(map[string]struct{})
	for _, resource := range policy.CustomResources {
		resourcesToRemove[resource.ResourceID] = struct{}{}
	}

	// then filter to only the resources that we want to keep
	// (ie, the non-custom resources)
	var keepPerms []rbacpolicy.Permission
	for _, perm := range policy.StytchMember.Permissions {
		if _, ok := resourcesToRemove[perm.ResourceID]; !ok {
			keepPerms = append(keepPerms, perm)
		}
	}
	policy.StytchMember.Permissions = keepPerms

	// then remove custom resource permissions from stytch_admin
	keepPerms = nil
	for _, perm := range policy.StytchAdmin.Permissions {
		if _, ok := resourcesToRemove[perm.ResourceID]; !ok {
			keepPerms = append(keepPerms, perm)
		}
	}
	policy.StytchAdmin.Permissions = keepPerms

	// Lastly, set custom roles and resources to empty
	policy.CustomRoles = []rbacpolicy.Role{}
	policy.CustomResources = []rbacpolicy.Resource{}

	_, err := r.client.RBACPolicy.Set(ctx, rbacpolicy.SetRequest{
		ProjectID: state.ProjectID.ValueString(),
		Policy:    policy,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset RBAC policy", err.Error())
		return
	}
}

func (r *rbacPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx = tflog.SetField(ctx, "project_id", req.ID)
	tflog.Info(ctx, "Importing RBAC policy")
	resource.ImportStatePassthroughID(ctx, path.Root("project_id"), req, resp)
}
