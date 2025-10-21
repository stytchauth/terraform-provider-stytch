package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/publictokens"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                 = &publicTokenResource{}
	_ resource.ResourceWithConfigure    = &publicTokenResource{}
	_ resource.ResourceWithImportState  = &publicTokenResource{}
	_ resource.ResourceWithUpgradeState = &publicTokenResource{}
)

func NewPublicTokenResource() resource.Resource {
	return &publicTokenResource{}
}

type publicTokenResource struct {
	client *api.API
}

type publicTokenModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectSlug     types.String `tfsdk:"project_slug"`
	EnvironmentSlug types.String `tfsdk:"environment_slug"`
	PublicToken     types.String `tfsdk:"public_token"`
	CreatedAt       types.String `tfsdk:"created_at"`
}

type publicTokenResourceModelV0 struct {
	ProjectID   types.String `tfsdk:"project_id"`
	PublicToken types.String `tfsdk:"public_token"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

var publicTokenResourceLegacySchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"project_id": schema.StringAttribute{
			Required: true,
		},
		"public_token": schema.StringAttribute{
			Computed: true,
		},
		"created_at": schema.StringAttribute{
			Computed: true,
		},
	},
}

func (r *publicTokenResource) Configure(
	ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	// Add a nil check when handling ProviderData because Terraform sets that data after it calls the
	// ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.API)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *api.API (stytch-management-go client), got: %T. Please report "+
				"this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *publicTokenResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &publicTokenResourceLegacySchema,
			StateUpgrader: r.upgradePublicTokenStateV0ToV1,
		},
	}
}

func (r *publicTokenResource) upgradePublicTokenStateV0ToV1(
	ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse,
) {
	if req.State == nil {
		resp.Diagnostics.AddError(
			"Missing prior state",
			"Legacy public token state upgrade requires existing state data, but none was provided.",
		)
		return
	}

	var prior publicTokenResourceModelV0
	diags := req.State.Get(ctx, &prior)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectSlug, environmentSlug, diags := utils.ResolveLegacyProjectAndEnvironment(
		ctx, r.client, prior.ProjectID.ValueString(),
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	publicTokenValue := prior.PublicToken.ValueString()
	if publicTokenValue == "" {
		resp.Diagnostics.AddError(
			"Missing public token",
			"The stored state did not contain a public token value, so it cannot be upgraded automatically.",
		)
		return
	}

	createdAt := prior.CreatedAt.ValueString()
	tokenResp, err := r.client.PublicTokens.Get(ctx, publictokens.GetRequest{
		ProjectSlug:     projectSlug,
		EnvironmentSlug: environmentSlug,
		PublicToken:     publicTokenValue,
	})
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Unable to verify public token",
			fmt.Sprintf("The token lookup failed during migration: %s. Existing state values will be retained.", err.Error()),
		)
	} else if tokenResp != nil && !tokenResp.PublicToken.CreatedAt.IsZero() {
		publicTokenValue = tokenResp.PublicToken.PublicToken
		createdAt = tokenResp.PublicToken.CreatedAt.Format(time.RFC3339)
	}

	if createdAt == "" {
		createdAt = time.Now().Format(time.RFC3339)
	}

	newState := publicTokenModel{
		ID:              types.StringValue(fmt.Sprintf("%s.%s.%s", projectSlug, environmentSlug, publicTokenValue)),
		ProjectSlug:     types.StringValue(projectSlug),
		EnvironmentSlug: types.StringValue(environmentSlug),
		PublicToken:     types.StringValue(publicTokenValue),
		CreatedAt:       types.StringValue(createdAt),
	}

	diags = resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
}

// Metadata returns the resource type name.
func (r *publicTokenResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_public_token"
}

// Schema defines the schema for the resource.
func (r *publicTokenResource) Schema(
	_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "A public token for an environment within a Stytch project, used for SDK authentication and OAuth integrations.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management (format: project_slug.environment_slug.public_token).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"public_token": schema.StringAttribute{
				Computed:    true,
				Description: "The public token value, which also serves as part of the unique identifier for the token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project to which the public token belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to which the public token belongs.",
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
func (r *publicTokenResource) Create(
	ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse,
) {
	var plan publicTokenModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Creating public token")

	createResp, err := r.client.PublicTokens.Create(ctx, publictokens.CreateRequest{
		ProjectSlug:     plan.ProjectSlug.ValueString(),
		EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create public token", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "public_token", createResp.PublicToken.PublicToken)
	tflog.Info(ctx, "Created public token")

	plan.ID = types.StringValue(fmt.Sprintf("%s.%s.%s", plan.ProjectSlug.ValueString(), plan.EnvironmentSlug.ValueString(), createResp.PublicToken.PublicToken))
	plan.PublicToken = types.StringValue(createResp.PublicToken.PublicToken)
	plan.CreatedAt = types.StringValue(createResp.PublicToken.CreatedAt.Format(time.RFC3339))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *publicTokenResource) Read(
	ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state
	var state publicTokenModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "public_token", state.PublicToken.ValueString())
	tflog.Info(ctx, "Reading public token")

	// We call Get here just to verify the public token still exists, but there is no state to update.
	_, err := r.client.PublicTokens.Get(ctx, publictokens.GetRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		PublicToken:     state.PublicToken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get public token", err.Error())
		return
	}

	tflog.Info(ctx, "Read public token")

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *publicTokenResource) Update(
	_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	resp.Diagnostics.AddError("Update not allowed",
		"Updating this resource is not supported. Please delete and recreate the resource.")
	//nolint:staticcheck
	return // Needed so that the Semgrep rule can enforce AddError followed by return.
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *publicTokenResource) Delete(
	ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	var state publicTokenModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "public_token", state.PublicToken.ValueString())
	tflog.Info(ctx, "Deleting public token")

	_, err := r.client.PublicTokens.Delete(ctx, publictokens.DeleteRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		PublicToken:     state.PublicToken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete public token", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted public token")
}

// ImportState imports an existing public token into Terraform state.
func (r *publicTokenResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	ctx = tflog.SetField(ctx, "import_id", req.ID)
	tflog.Info(ctx, "Importing public token")

	// Import ID format: project_slug.environment_slug.public_token
	parts := strings.Split(req.ID, ".")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("Expected import ID format: project_slug.environment_slug.public_token, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_slug"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("public_token"), parts[2])...)
}
