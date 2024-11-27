// Copyright (c) HashiCorp, Inc.

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/pkg/api"
	"github.com/stytchauth/stytch-management-go/pkg/models/publictokens"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &publicTokenResource{}
	_ resource.ResourceWithConfigure = &publicTokenResource{}
)

func NewPublicTokenResource() resource.Resource {
	return &publicTokenResource{}
}

type publicTokenResource struct {
	client *api.API
}

type publicTokenModel struct {
	ProjectID   types.String `tfsdk:"project_id"`
	PublicToken types.String `tfsdk:"public_token"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func (r *publicTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *publicTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_token"
}

// Schema defines the schema for the resource.
func (r *publicTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A public token used for SDK authentication and OAuth integrations.",
		Attributes: map[string]schema.Attribute{
			"public_token": schema.StringAttribute{
				Computed:    true,
				Description: "The public token value. This is a unique ID which is also the identifier for the token.",
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
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The ISO-8601 timestamp when the public token was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *publicTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan publicTokenModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	tflog.Info(ctx, "Creating public token")

	createResp, err := r.client.PublicTokens.Create(ctx, publictokens.CreateRequest{
		ProjectID: plan.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create public token", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "public_token", createResp.PublicToken.PublicToken)
	tflog.Info(ctx, "Public token created")

	plan.PublicToken = types.StringValue(createResp.PublicToken.PublicToken)
	plan.CreatedAt = types.StringValue(createResp.PublicToken.CreatedAt.Format(time.RFC3339))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *publicTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the current state
	var state publicTokenModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	tflog.Info(ctx, "Reading public token")

	// We call GetAll here just to verify the public token still exists, but there's no state to update.
	// This is because public tokens kind of *are* their identifier... so if you know the identifier,
	// you know the public token.
	getResp, err := r.client.PublicTokens.GetAll(ctx, publictokens.GetAllRequest{
		ProjectID: state.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get public token", err.Error())
	}

	found := false
	for _, publicToken := range getResp.PublicTokens {
		if publicToken.PublicToken == state.PublicToken.ValueString() {
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError("Public token not found", "The public token could not be found.")
		return
	}

	ctx = tflog.SetField(ctx, "public_token", state.PublicToken.ValueString())
	tflog.Info(ctx, "Public token found in project")

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *publicTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not allowed", "Updating this resource is not supported. Please delete and recreate the resource.")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *publicTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state publicTokenModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	tflog.Info(ctx, "Deleting public token")

	_, err := r.client.PublicTokens.Delete(ctx, publictokens.DeleteRequest{
		ProjectID:   state.ProjectID.ValueString(),
		PublicToken: state.PublicToken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete public token", err.Error())
		return
	}

	tflog.Info(ctx, "Public token deleted")
}
