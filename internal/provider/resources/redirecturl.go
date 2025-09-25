package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/redirecturls"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &redirectURLResource{}
	_ resource.ResourceWithConfigure = &redirectURLResource{}
)

func NewRedirectURLResource() resource.Resource {
	return &redirectURLResource{}
}

type redirectURLResource struct {
	client *api.API
}

type redirectURLModel struct {
	ProjectID   types.String `tfsdk:"project_id"`
	LastUpdated types.String `tfsdk:"last_updated"`
	URL         types.String `tfsdk:"url"`
	ValidTypes  types.Set    `tfsdk:"valid_types"`
}

func (m *redirectURLModel) refreshFromRedirectURL(r redirecturls.RedirectURL) {
	m.URL = types.StringValue(r.URL)
	if len(r.ValidTypes) > 0 {
		m.ValidTypes = types.SetValueMust(types.ObjectType{AttrTypes: redirectURLTypeModel{}.AttributeTypes()},
			func() []attr.Value {
				values := make([]attr.Value, len(r.ValidTypes))
				for i, vt := range r.ValidTypes {
					values[i] = types.ObjectValueMust(redirectURLTypeModel{}.AttributeTypes(), map[string]attr.Value{
						"type":       types.StringValue(string(vt.Type)),
						"is_default": types.BoolValue(vt.IsDefault),
					})
				}
				return values
			}())
	} else {
		m.ValidTypes = types.SetNull(types.ObjectType{AttrTypes: redirectURLTypeModel{}.AttributeTypes()})
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

// Metadata returns the resource type name.
func (r *redirectURLResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redirect_url"
}

// Schema defines the schema for the resource.
func (r *redirectURLResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A redirect URL for a project.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the project to create the redirect URL for.",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the order.",
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
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "The type of the redirect URL.",
							Validators: []validator.String{
								stringvalidator.OneOf(toStrings(redirecturls.RedirectTypes())...),
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

func (m redirectURLModel) toValidTypes() []redirecturls.URLRedirectType {
	var validTypes []redirecturls.URLRedirectType

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

			validTypes = append(validTypes, redirecturls.URLRedirectType{
				Type:      redirecturls.RedirectType(typeAttr.ValueString()),
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

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	ctx = tflog.SetField(ctx, "url", plan.URL.ValueString())
	tflog.Info(ctx, "Creating redirect URL")

	redirectURL := redirecturls.RedirectURL{
		URL:        plan.URL.ValueString(),
		ValidTypes: plan.toValidTypes(),
	}

	createResp, err := r.client.RedirectURLs.Create(ctx, redirecturls.CreateRequest{
		ProjectID:   plan.ProjectID.ValueString(),
		RedirectURL: redirectURL,
		// We explicitly disable default promotion logic because if the terraform provisioner specified that a redirect URL
		// is *not* the default for a given type, if the API tries to override it to true, it will result in a provider
		// inconsistency error.
		DoNotPromoteDefaults: true,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create redirect URL", err.Error())
		return
	}

	tflog.Info(ctx, "Created redirect URL")

	plan.refreshFromRedirectURL(createResp.RedirectURL)
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

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	ctx = tflog.SetField(ctx, "url", state.URL.ValueString())
	tflog.Info(ctx, "Reading redirect URL")

	getResp, err := r.client.RedirectURLs.Get(ctx, redirecturls.GetRequest{
		ProjectID: state.ProjectID.ValueString(),
		URL:       state.URL.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get redirect URL", err.Error())
		return
	}

	tflog.Info(ctx, "Read redirect URL")

	state.refreshFromRedirectURL(getResp.RedirectURL)
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

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	ctx = tflog.SetField(ctx, "url", plan.URL.ValueString())
	tflog.Info(ctx, "Updating redirect URL")

	redirectURL := redirecturls.RedirectURL{
		URL:        plan.URL.ValueString(),
		ValidTypes: plan.toValidTypes(),
	}

	updateResp, err := r.client.RedirectURLs.Update(ctx, redirecturls.UpdateRequest{
		ProjectID:   plan.ProjectID.ValueString(),
		RedirectURL: redirectURL,
		// We explicitly disable default promotion logic because if the terraform provisioner specified that a redirect URL
		// is *not* the default for a given type, if the API tries to override it to true, it will result in a provider
		// inconsistency error.
		DoNotPromoteDefaults: true,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update redirect URL", err.Error())
		return
	}

	tflog.Info(ctx, "Updated redirect URL")

	plan.refreshFromRedirectURL(updateResp.RedirectURL)
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

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	tflog.Info(ctx, "Deleting redirect URL")

	_, err := r.client.RedirectURLs.Delete(ctx, redirecturls.DeleteRequest{
		ProjectID: state.ProjectID.ValueString(),
		URL:       state.URL.ValueString(),
		// We explicitly disable default promotion logic because if the terraform provisioner specified that a redirect URL
		// is *not* the default for a given type, if the API tries to override it to true, it will result in a provider
		// inconsistency error.
		DoNotPromoteDefaults: true,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete redirect URL", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted redirect URL")
}
