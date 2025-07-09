package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v2/pkg/api"
	"github.com/stytchauth/stytch-management-go/v2/pkg/models/trustedtokenprofiles"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &trustedTokenProfilesResource{}
	_ resource.ResourceWithConfigure   = &trustedTokenProfilesResource{}
	_ resource.ResourceWithImportState = &trustedTokenProfilesResource{}
)

func NewTrustedTokenProfilesResource() resource.Resource {
	return &trustedTokenProfilesResource{}
}

type trustedTokenProfilesResource struct {
	client *api.API
}

type trustedTokenProfilesModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	ProfileID        types.String `tfsdk:"profile_id"`
	Name             types.String `tfsdk:"name"`
	Audience         types.String `tfsdk:"audience"`
	Issuer           types.String `tfsdk:"issuer"`
	JwksUrl          types.String `tfsdk:"jwks_url"`
	AttributeMapping types.Map    `tfsdk:"attribute_mapping"`
	PemFiles         types.List   `tfsdk:"pem_files"`
	PublicKeyType    types.String `tfsdk:"public_key_type"`
	LastUpdated      types.String `tfsdk:"last_updated"`
}

type pemFileModel struct {
	PemFileID types.String `tfsdk:"pem_file_id"`
	PublicKey types.String `tfsdk:"public_key"`
}

func (m pemFileModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"pem_file_id": types.StringType,
		"public_key":  types.StringType,
	}
}

// Configure sets provider-level data for the resource.
func (r *trustedTokenProfilesResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	// Add a nil check when handling ProviderData because Terraform sets that data after it calls
	// the ConfigureProvider RPC.
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
func (r *trustedTokenProfilesResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_trusted_token_profiles"
}

// Schema defines the schema for the resource.
func (r *trustedTokenProfilesResource) Schema(
	ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Resource for managing trusted token profiles.",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"profile_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the trusted token profile.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the trusted token profile.",
			},
			"audience": schema.StringAttribute{
				Required:    true,
				Description: "The audience for the trusted token profile.",
			},
			"issuer": schema.StringAttribute{
				Required:    true,
				Description: "The issuer for the trusted token profile.",
			},
			"jwks_url": schema.StringAttribute{
				Optional:    true,
				Description: "The JWKS URL for the trusted token profile.",
			},
			"attribute_mapping": schema.MapAttribute{
				Optional:    true,
				Description: "The attribute mapping for the trusted token profile.",
				ElementType: types.StringType,
			},
			"pem_files": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of PEM files associated with the trusted token profile.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"pem_file_id": schema.StringAttribute{
							Computed:    true,
							Description: "The unique identifier for the PEM file.",
						},
						"public_key": schema.StringAttribute{
							Computed:    true,
							Description: "The public key content.",
						},
					},
				},
			},
			"public_key_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of public key.",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update.",
				Computed:    true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *trustedTokenProfilesResource) Create(
	ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse,
) {
	// Get the plan from the request.
	var plan trustedTokenProfilesModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	tflog.Info(ctx, "Creating trusted token profile")

	var attributeMapping map[string]interface{}
	if !plan.AttributeMapping.IsNull() && !plan.AttributeMapping.IsUnknown() {
		diags = plan.AttributeMapping.ElementsAs(ctx, &attributeMapping, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	var pemFiles []string
	if !plan.PemFiles.IsNull() && !plan.PemFiles.IsUnknown() {
		diags = plan.PemFiles.ElementsAs(ctx, &pemFiles, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	createReq := &trustedtokenprofiles.CreateTrustedTokenProfileRequest{
		ProjectID:        plan.ProjectID.ValueString(),
		Name:             plan.Name.ValueString(),
		Audience:         plan.Audience.ValueString(),
		Issuer:           plan.Issuer.ValueString(),
		AttributeMapping: attributeMapping,
		PemFiles:         pemFiles,
		PublicKeyType:    plan.PublicKeyType.ValueString(),
	}

	if !plan.JwksUrl.IsNull() && !plan.JwksUrl.IsUnknown() {
		jwksUrl := plan.JwksUrl.ValueString()
		createReq.JwksUrl = &jwksUrl
	}

	createResp, err := r.client.TrustedTokenProfiles.Create(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating trusted token profile",
			fmt.Sprintf("Could not create trusted token profile: %s", err),
		)
		return
	}

	plan.ProfileID = types.StringValue(createResp.ProfileID)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", plan.ProjectID.ValueString(), createResp.ProfileID))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *trustedTokenProfilesResource) Read(
	ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state
	var state trustedTokenProfilesModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	ctx = tflog.SetField(ctx, "profile_id", state.ProfileID.ValueString())
	tflog.Info(ctx, "Reading trusted token profile")

	// Get the trusted token profile
	getResp, err := r.client.TrustedTokenProfiles.Get(ctx, &trustedtokenprofiles.GetTrustedTokenProfileRequest{
		ProjectID: state.ProjectID.ValueString(),
		ProfileID: state.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading trusted token profile",
			fmt.Sprintf("Could not read trusted token profile: %s", err),
		)
		return
	}

	// Update the state with the response data
	state.Name = types.StringValue(getResp.TrustedTokenProfile.Name)
	state.Audience = types.StringValue(getResp.TrustedTokenProfile.Audience)
	state.Issuer = types.StringValue(getResp.TrustedTokenProfile.Issuer)
	state.PublicKeyType = types.StringValue(getResp.TrustedTokenProfile.PublicKeyType)

	// Handle JWKS URL
	if getResp.TrustedTokenProfile.JwksUrl != nil {
		state.JwksUrl = types.StringValue(*getResp.TrustedTokenProfile.JwksUrl)
	} else {
		state.JwksUrl = types.StringNull()
	}

	// Handle attribute mapping
	if getResp.TrustedTokenProfile.AttributeMapping != nil {
		attributeMapping := make(map[string]attr.Value)
		for k, v := range getResp.TrustedTokenProfile.AttributeMapping {
			if strVal, ok := v.(string); ok {
				attributeMapping[k] = types.StringValue(strVal)
			}
		}
		state.AttributeMapping = types.MapValueMust(types.StringType, attributeMapping)
	} else {
		state.AttributeMapping = types.MapNull(types.StringType)
	}

	// Handle PEM files
	pemFiles := make([]pemFileModel, 0, len(getResp.TrustedTokenProfile.PemFiles))
	for _, pemFile := range getResp.TrustedTokenProfile.PemFiles {
		pemFiles = append(pemFiles, pemFileModel{
			PemFileID: types.StringValue(pemFile.PemFileID),
			PublicKey: types.StringValue(pemFile.PublicKey),
		})
	}
	state.PemFiles = types.ListValueMust(types.ObjectType{AttrTypes: pemFileModel{}.AttributeTypes()},
		func() []attr.Value {
			values := make([]attr.Value, len(pemFiles))
			for i, pemFile := range pemFiles {
				values[i] = types.ObjectValueMust(pemFile.AttributeTypes(), map[string]attr.Value{
					"pem_file_id": pemFile.PemFileID,
					"public_key":  pemFile.PublicKey,
				})
			}
			return values
		}())

	// Set state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *trustedTokenProfilesResource) Update(
	ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	var plan trustedTokenProfilesModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	ctx = tflog.SetField(ctx, "profile_id", plan.ProfileID.ValueString())
	tflog.Info(ctx, "Updating trusted token profile")

	var attributeMapping map[string]interface{}
	if !plan.AttributeMapping.IsNull() && !plan.AttributeMapping.IsUnknown() {
		diags = plan.AttributeMapping.ElementsAs(ctx, &attributeMapping, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	updateReq := &trustedtokenprofiles.UpdateTrustedTokenProfileRequest{
		ProjectID:        plan.ProjectID.ValueString(),
		ProfileID:        plan.ProfileID.ValueString(),
		Name:             plan.Name.ValueString(),
		Audience:         plan.Audience.ValueString(),
		Issuer:           plan.Issuer.ValueString(),
		AttributeMapping: attributeMapping,
	}

	if !plan.JwksUrl.IsNull() && !plan.JwksUrl.IsUnknown() {
		jwksUrl := plan.JwksUrl.ValueString()
		updateReq.JwksUrl = &jwksUrl
	}

	_, err := r.client.TrustedTokenProfiles.Update(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating trusted token profile",
			fmt.Sprintf("Could not update trusted token profile: %s", err),
		)
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *trustedTokenProfilesResource) Delete(
	ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	var state trustedTokenProfilesModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	ctx = tflog.SetField(ctx, "profile_id", state.ProfileID.ValueString())
	tflog.Info(ctx, "Deleting trusted token profile")

	_, err := r.client.TrustedTokenProfiles.Delete(ctx, &trustedtokenprofiles.DeleteTrustedTokenProfileRequest{
		ProjectID: state.ProjectID.ValueString(),
		ProfileID: state.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting trusted token profile",
			fmt.Sprintf("Could not delete trusted token profile: %s", err),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *trustedTokenProfilesResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	// Parse the import ID (format: project_id/profile_id)
	importID := req.ID
	parts := strings.Split(importID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'project_id/profile_id'",
		)
		return
	}

	projectID := parts[0]
	profileID := parts[1]

	// Set the imported state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), projectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), profileID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), importID)...)
}
