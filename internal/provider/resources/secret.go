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
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/secrets"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                 = &secretResource{}
	_ resource.ResourceWithConfigure    = &secretResource{}
	_ resource.ResourceWithUpgradeState = &secretResource{}
)

func NewSecretResource() resource.Resource {
	return &secretResource{}
}

type secretResource struct {
	client *api.API
}

type secretModel struct {
	ProjectSlug     types.String `tfsdk:"project_slug"`
	EnvironmentSlug types.String `tfsdk:"environment_slug"`
	SecretID        types.String `tfsdk:"secret_id"`
	CreatedAt       types.String `tfsdk:"created_at"`
	Secret          types.String `tfsdk:"secret"`
}

type secretResourceModelV0 struct {
	ProjectID types.String `tfsdk:"project_id"`
	SecretID  types.String `tfsdk:"secret_id"`
	CreatedAt types.String `tfsdk:"created_at"`
	Secret    types.String `tfsdk:"secret"`
}

var secretResourceLegacySchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"project_id": schema.StringAttribute{
			Required: true,
		},
		"secret_id": schema.StringAttribute{
			Computed: true,
		},
		"created_at": schema.StringAttribute{
			Computed: true,
		},
		"secret": schema.StringAttribute{
			Computed:  true,
			Sensitive: true,
		},
	},
}

func (r *secretResource) Configure(
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

func (r *secretResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &secretResourceLegacySchema,
			StateUpgrader: r.upgradeSecretStateV0ToV1,
		},
	}
}

func (r *secretResource) upgradeSecretStateV0ToV1(
	ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse,
) {
	if req.State == nil {
		resp.Diagnostics.AddError(
			"Missing prior state",
			"Legacy secret state upgrade requires existing state data, but none was provided.",
		)
		return
	}

	var prior secretResourceModelV0
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

	newState := secretModel{
		ProjectSlug:     types.StringValue(projectSlug),
		EnvironmentSlug: types.StringValue(environmentSlug),
		SecretID:        prior.SecretID,
		CreatedAt:       prior.CreatedAt,
		Secret:          prior.Secret,
	}

	diags = resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
}

// Metadata returns the resource type name.
func (r *secretResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

// Schema defines the schema for the resource.
func (r *secretResource) Schema(
	_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "A secret for an environment within a Stytch project, used in the Stytch API.",
		Attributes: map[string]schema.Attribute{
			"secret_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the secret.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project to which the secret belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to which the secret belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The ISO-8601 timestamp when the secret was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The secret value.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *secretResource) Create(
	ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse,
) {
	var plan secretModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Creating secret")

	createResp, err := r.client.Secrets.Create(ctx, secrets.CreateSecretRequest{
		ProjectSlug:     plan.ProjectSlug.ValueString(),
		EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create secret", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "secret_id", createResp.CreatedSecret.SecretID)
	ctx = tflog.SetField(ctx, "secret", createResp.CreatedSecret.Secret)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "secret")
	tflog.Info(ctx, "Created secret")

	plan.SecretID = types.StringValue(createResp.CreatedSecret.SecretID)
	plan.CreatedAt = types.StringValue(createResp.CreatedSecret.CreatedAt.Format(time.RFC3339))
	plan.Secret = types.StringValue(createResp.CreatedSecret.Secret)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *secretResource) Read(
	ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state
	var state secretModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Reading secret")

	// We call Get here just to verify the secret still exists, but there is no state to update.
	_, err := r.client.Secrets.Get(ctx, secrets.GetSecretRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		SecretID:        state.SecretID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get secret", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "secret_id", state.SecretID.ValueString())
	tflog.Info(ctx, "Read secret")

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *secretResource) Update(
	_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	resp.Diagnostics.AddError("Update not allowed",
		"Updating this resource is not supported. Please delete and recreate the resource.")
	//nolint:staticcheck
	return // Needed so that the Semgrep rule can enforce AddError followed by return.
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *secretResource) Delete(
	ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	var state secretModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "secret_id", state.SecretID.ValueString())
	tflog.Info(ctx, "Deleting secret")

	_, err := r.client.Secrets.Delete(ctx, secrets.DeleteSecretRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		SecretID:        state.SecretID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete secret", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted secret")
}
