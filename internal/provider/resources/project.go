// Copyright (c) HashiCorp, Inc.

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/pkg/api"
	"github.com/stytchauth/stytch-management-go/pkg/models/projects"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

func NewProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct {
	client *api.API
}

type projectModel struct {
	ID                   types.String `tfsdk:"id"`
	LiveProjectID        types.String `tfsdk:"live_project_id"`
	TestProjectID        types.String `tfsdk:"test_project_id"`
	LastUpdated          types.String `tfsdk:"last_updated"`
	CreatedAt            types.String `tfsdk:"created_at"`
	Name                 types.String `tfsdk:"name"`
	Vertical             types.String `tfsdk:"vertical"`
	LiveOAuthCallbackID  types.String `tfsdk:"live_oauth_callback_id"`
	TestOAuthCallbackID  types.String `tfsdk:"test_oauth_callback_id"`
	UseCrossOrgPasswords types.Bool   `tfsdk:"use_cross_org_passwords"`
}

func (m *projectModel) refreshFromProject(p projects.Project) {
	m.ID = types.StringValue(p.LiveProjectID)
	m.LiveProjectID = types.StringValue(p.LiveProjectID)
	m.TestProjectID = types.StringValue(p.TestProjectID)
	m.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	m.CreatedAt = types.StringValue(p.CreatedAt.Format(time.RFC3339))
	m.Name = types.StringValue(p.Name)
	m.Vertical = types.StringValue(string(p.Vertical))
	m.LiveOAuthCallbackID = types.StringValue(p.LiveOAuthCallbackID)
	m.TestOAuthCallbackID = types.StringValue(p.TestOAuthCallbackID)
	m.UseCrossOrgPasswords = types.BoolValue(p.UseCrossOrgPasswords)
}

func (r *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a project within your Stytch workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"live_project_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the live project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"test_project_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the test project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the order.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The project's name.",
			},
			"vertical": schema.StringAttribute{
				Required:    true,
				Description: "The project's vertical. This cannot be changed after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The ISO-8601 timestamp when the project was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"live_oauth_callback_id": schema.StringAttribute{
				Computed:    true,
				Description: "The callback ID used in OAuth requests for the live project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"test_oauth_callback_id": schema.StringAttribute{
				Computed:    true,
				Description: "The callback ID used in OAuth requests for the test project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"use_cross_org_passwords": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether to enable cross-org passwords in a B2B project.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
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

	createResp, err := r.client.Projects.Create(ctx, projects.CreateRequest{
		ProjectName: plan.Name.ValueString(),
		Vertical:    projects.Vertical(plan.Vertical.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create project", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "live_project_id", createResp.Project.LiveProjectID)
	ctx = tflog.SetField(ctx, "test_project_id", createResp.Project.TestProjectID)
	tflog.Info(ctx, "Created project")

	plan.refreshFromProject(createResp.Project)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the current state
	var state projectModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "live_project_id", state.LiveProjectID.ValueString())
	tflog.Info(ctx, "Reading project")

	getResp, err := r.client.Projects.Get(ctx, projects.GetRequest{
		ProjectID: state.LiveProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get live project", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "test_project_id", getResp.Project.TestProjectID)
	tflog.Info(ctx, "Read project")

	state.refreshFromProject(getResp.Project)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_name", plan.Name.ValueString())
	tflog.Info(ctx, "Updating project")

	updateResp, err := r.client.Projects.Update(ctx, projects.UpdateRequest{
		ProjectID: plan.LiveProjectID.ValueString(),
		Name:      plan.Name.ValueString(),
		// UseCrossOrgPasswords: plan.UseCrossOrgPasswords.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update project", err.Error())
	}

	tflog.Info(ctx, "Updated project")

	plan.refreshFromProject(updateResp.Project)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "live_project_id", state.LiveProjectID.ValueString())
	tflog.Info(ctx, "Deleting project")

	_, err := r.client.Projects.Delete(ctx, projects.DeleteRequest{
		ProjectID: state.LiveProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete project", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted project")
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx = tflog.SetField(ctx, "live_project_id", req.ID)
	tflog.Info(ctx, "Importing project")
	resource.ImportStatePassthroughID(ctx, path.Root("live_project_id"), req, resp)
}
