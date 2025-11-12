package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/redirecturls"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                 = &redirectURLResource{}
	_ resource.ResourceWithConfigure    = &redirectURLResource{}
	_ resource.ResourceWithImportState  = &redirectURLResource{}
	_ resource.ResourceWithUpgradeState = &redirectURLResource{}
)

func NewRedirectURLResource() resource.Resource {
	return &redirectURLResource{}
}

type redirectURLResource struct {
	client *api.API
}

type redirectURLModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectSlug     types.String `tfsdk:"project_slug"`
	EnvironmentSlug types.String `tfsdk:"environment_slug"`
	LastUpdated     types.String `tfsdk:"last_updated"`
	URL             types.String `tfsdk:"url"`
	ValidTypes      types.Set    `tfsdk:"valid_types"`
}

type redirectURLResourceModelV0 struct {
	ProjectID   types.String `tfsdk:"project_id"`
	LastUpdated types.String `tfsdk:"last_updated"`
	URL         types.String `tfsdk:"url"`
	ValidTypes  types.Set    `tfsdk:"valid_types"`
}

var redirectURLResourceLegacySchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"project_id": schema.StringAttribute{
			Required: true,
		},
		"last_updated": schema.StringAttribute{
			Computed: true,
		},
		"url": schema.StringAttribute{
			Required: true,
		},
		"valid_types": schema.SetNestedAttribute{
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required: true,
					},
					"is_default": schema.BoolAttribute{
						Required: true,
					},
				},
			},
			Required: true,
		},
	},
}

// updateModelFromAPI updates the model with values from the API response
func (r *redirectURLResource) updateModelFromAPI(model *redirectURLModel, redirectURL redirecturls.RedirectURL) {
	model.ID = types.StringValue(fmt.Sprintf("%s.%s.%s", model.ProjectSlug.ValueString(), model.EnvironmentSlug.ValueString(), model.URL.ValueString()))
	model.URL = types.StringValue(redirectURL.URL)
	if len(redirectURL.ValidTypes) > 0 {
		model.ValidTypes = types.SetValueMust(types.ObjectType{AttrTypes: redirectURLTypeModel{}.AttributeTypes()},
			func() []attr.Value {
				values := make([]attr.Value, len(redirectURL.ValidTypes))
				for i, vt := range redirectURL.ValidTypes {
					values[i] = types.ObjectValueMust(redirectURLTypeModel{}.AttributeTypes(), map[string]attr.Value{
						"type":       types.StringValue(string(vt.Type)),
						"is_default": types.BoolValue(vt.IsDefault),
					})
				}
				return values
			}())
	} else {
		model.ValidTypes = types.SetNull(types.ObjectType{AttrTypes: redirectURLTypeModel{}.AttributeTypes()})
	}
}

type redirectURLTypeModel struct {
	Type      types.String `tfsdk:"type"`
	IsDefault types.Bool   `tfsdk:"is_default"`
}

func (m redirectURLTypeModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":       types.StringType,
		"is_default": types.BoolType,
	}
}

func (r *redirectURLResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *redirectURLResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &redirectURLResourceLegacySchema,
			StateUpgrader: r.upgradeRedirectURLStateV0ToV1,
		},
	}
}

func (r *redirectURLResource) upgradeRedirectURLStateV0ToV1(
	ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse,
) {
	if req.State == nil {
		resp.Diagnostics.AddError(
			"Missing prior state",
			"Legacy redirect URL state upgrade requires existing state data, but none was provided.",
		)
		return
	}

	var prior redirectURLResourceModelV0
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

	redirectResp, err := r.client.RedirectURLs.Get(ctx, redirecturls.GetRequest{
		ProjectSlug:     projectSlug,
		EnvironmentSlug: environmentSlug,
		URL:             prior.URL.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to retrieve redirect URL",
			err.Error(),
		)
		return
	}

	newState := redirectURLModel{
		ProjectSlug:     types.StringValue(projectSlug),
		EnvironmentSlug: types.StringValue(environmentSlug),
		URL:             prior.URL,
		ValidTypes:      prior.ValidTypes,
	}
	r.updateModelFromAPI(&newState, redirectResp.RedirectURL)
	newState.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
}

// Metadata returns the resource type name.
func (r *redirectURLResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redirect_url"
}

// Schema defines the schema for the resource.
func (r *redirectURLResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "A redirect URL for an environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management (format: project_slug.environment_slug.url).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project to which the redirect URL belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to which the redirect URL belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The URL to redirect to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"valid_types": schema.SetNestedAttribute{
				Description: "The set of valid types for the redirect URL.",
				Required:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "The type of the redirect URL.",
							Validators: []validator.String{
								stringvalidator.OneOf(toStrings(redirecturls.RedirectURLTypes())...),
							},
						},
						"is_default": schema.BoolAttribute{
							Required:    true,
							Description: "Whether or not this is the default redirect URL for the given type.",
						},
					},
				},
			},
		},
	}
}

func (m redirectURLModel) toValidTypes() []redirecturls.URLType {
	var validTypes []redirecturls.URLType

	if m.ValidTypes.IsNull() || m.ValidTypes.IsUnknown() {
		return validTypes
	}

	for _, elem := range m.ValidTypes.Elements() {
		if obj, ok := elem.(types.Object); ok {
			attrs := obj.Attributes()

			typeAttr, isTypeStr := attrs["type"].(types.String)
			isDefaultAttr, isDefaultBool := attrs["is_default"].(types.Bool)
			if !isTypeStr || !isDefaultBool {
				continue
			}

			validTypes = append(validTypes, redirecturls.URLType{
				Type:      redirecturls.RedirectURLType(typeAttr.ValueString()),
				IsDefault: isDefaultAttr.ValueBool(),
			})
		}
	}

	return validTypes
}

// Create creates the resource and sets the initial Terraform state.
func (r *redirectURLResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan redirectURLModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "url", plan.URL.ValueString())
	tflog.Info(ctx, "Creating redirect URL")

	createResp, err := r.client.RedirectURLs.Create(ctx, redirecturls.CreateRequest{
		ProjectSlug:     plan.ProjectSlug.ValueString(),
		EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
		URL:             plan.URL.ValueString(),
		ValidTypes:      plan.toValidTypes(),
		// We explicitly disable default promotion logic because if the terraform provisioner specified that a redirect URL
		// is *not* the default for a given type, if the API tries to override it to true, it will result in a provider
		// inconsistency error.
		DoNotPromoteDefaults: ptr(true),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create redirect URL", err.Error())
		return
	}

	tflog.Info(ctx, "Created redirect URL")

	r.updateModelFromAPI(&plan, createResp.RedirectURL)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *redirectURLResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the current state
	var state redirectURLModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "url", state.URL.ValueString())
	tflog.Info(ctx, "Reading redirect URL")

	getResp, err := r.client.RedirectURLs.Get(ctx, redirecturls.GetRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		URL:             state.URL.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get redirect URL", err.Error())
		return
	}

	tflog.Info(ctx, "Read redirect URL")

	r.updateModelFromAPI(&state, getResp.RedirectURL)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *redirectURLResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan redirectURLModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "url", plan.URL.ValueString())
	tflog.Info(ctx, "Updating redirect URL")

	updateResp, err := r.client.RedirectURLs.Update(ctx, redirecturls.UpdateRequest{
		ProjectSlug:     plan.ProjectSlug.ValueString(),
		EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
		URL:             plan.URL.ValueString(),
		ValidTypes:      plan.toValidTypes(),
		// We explicitly disable default promotion logic because if the terraform provisioner specified that a redirect URL
		// is *not* the default for a given type, if the API tries to override it to true, it will result in a provider
		// inconsistency error.
		DoNotPromoteDefaults: ptr(true),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update redirect URL", err.Error())
		return
	}

	tflog.Info(ctx, "Updated redirect URL")

	r.updateModelFromAPI(&plan, updateResp.RedirectURL)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *redirectURLResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state redirectURLModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "url", state.URL.ValueString())
	tflog.Info(ctx, "Deleting redirect URL")

	_, err := r.client.RedirectURLs.Delete(ctx, redirecturls.DeleteRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		URL:             state.URL.ValueString(),
		// We explicitly disable default promotion logic because if the terraform provisioner specified that a redirect URL
		// is *not* the default for a given type, if the API tries to override it to true, it will result in a provider
		// inconsistency error.
		DoNotPromoteDefaults: ptr(true),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete redirect URL", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted redirect URL")
}

func (r *redirectURLResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: project_slug.environment_slug.url
	parts := strings.SplitN(req.ID, ".", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "The ID must be in the format <project_slug>.<environment_slug>.<url>")
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", parts[0])
	ctx = tflog.SetField(ctx, "environment_slug", parts[1])
	ctx = tflog.SetField(ctx, "url", parts[2])
	tflog.Info(ctx, "Importing redirect URL")

	resp.State.SetAttribute(ctx, path.Root("project_slug"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("environment_slug"), parts[1])
	resp.State.SetAttribute(ctx, path.Root("url"), parts[2])
}
