package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stytchauth/stytch-management-go/pkg/api"
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
		Attributes: map[string]schema.Attribute{
			"public_token": schema.StringAttribute{
				Computed:    true,
				Description: "The public token value. This is a unique ID which is also the identifier for the token.",
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

	// TODO: Generate API request body from plan and call r.client.PublicTokens.Create

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

	// TODO: Get refreshed value from the API

	// Set refreshed state
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

	// TODO: Call r.client.PublicTokens.Delete
}
