package resources

import (
	"context"
	"encoding/json"
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
	ID                   types.String `tfsdk:"id"`
	ProjectID            types.String `tfsdk:"project_id"`
	ProfileID            types.String `tfsdk:"profile_id"`
	Name                 types.String `tfsdk:"name"`
	Audience             types.String `tfsdk:"audience"`
	Issuer               types.String `tfsdk:"issuer"`
	JwksURL              types.String `tfsdk:"jwks_url"`
	AttributeMappingJSON types.String `tfsdk:"attribute_mapping_json"`
	PEMFiles             types.Set    `tfsdk:"pem_files"`
	PublicKeyType        types.String `tfsdk:"public_key_type"`
	LastUpdated          types.String `tfsdk:"last_updated"`
}

type pemFileModel struct {
	PEMFileID types.String `tfsdk:"pem_file_id"`
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
			"attribute_mapping_json": schema.StringAttribute{
				Optional:    true,
				Description: "The attribute mapping as a JSON string. Keys must be strings, values can be strings, numbers, booleans, or nested objects.",
			},
			"pem_files": schema.SetNestedAttribute{
				Optional:    true,
				Description: "Set of PEM files associated with the trusted token profile.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"pem_file_id": schema.StringAttribute{
							Computed:    true,
							Description: "The unique identifier for the PEM file.",
						},
						"public_key": schema.StringAttribute{
							Required:    true,
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

func (ttp *trustedTokenProfilesModel) refreshFromTrustedTokenProfile(ctx context.Context, r trustedtokenprofiles.TrustedTokenProfile) diag.Diagnostics {
	var diags diag.Diagnostics

	ttp.ID = types.StringValue(r.ID)
	ttp.ProfileID = types.StringValue(r.ID)
	ttp.Name = types.StringValue(r.Name)
	ttp.Audience = types.StringValue(r.Audience)
	ttp.Issuer = types.StringValue(r.Issuer)
	ttp.PublicKeyType = types.StringValue(r.PublicKeyType)

	if r.JwksURL != nil {
		ttp.JwksURL = types.StringValue(*r.JwksURL)
	} else {
		ttp.JwksURL = types.StringNull()
	}

	if len(r.AttributeMapping) > 0 {
		// Convert the map to JSON string
		jsonBytes, err := json.Marshal(r.AttributeMapping)
		if err != nil {
			diags.AddError("Failed to marshal attribute mapping", err.Error())
			return diags
		}
		ttp.AttributeMappingJSON = types.StringValue(string(jsonBytes))
	} else {
		ttp.AttributeMappingJSON = types.StringNull()
	}

	if len(r.PEMFiles) > 0 {
		ttp.PEMFiles = types.SetValueMust(types.ObjectType{AttrTypes: pemFileModel{}.AttributeTypes()},
			func() []attr.Value {
				values := make([]attr.Value, len(r.PEMFiles))
				for i, pemFile := range r.PEMFiles {
					values[i] = types.ObjectValueMust(pemFileModel{}.AttributeTypes(), map[string]attr.Value{
						"pem_file_id": types.StringValue(pemFile.ID),
						"public_key":  types.StringValue(pemFile.PublicKey),
					})
				}
				return values
			}())
	} else {
		ttp.PEMFiles = types.SetNull(types.ObjectType{AttrTypes: pemFileModel{}.AttributeTypes()})
	}

	ttp.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	return diags
}

// extractPEMFilesFromPlan extracts PEM file content from the plan
func extractPEMFilesFromPlan(ctx context.Context, plan trustedTokenProfilesModel) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	var pemFiles []string

	if !plan.PEMFiles.IsNull() && !plan.PEMFiles.IsUnknown() {
		// For SetNestedAttribute, we need to iterate through the set elements
		for _, elem := range plan.PEMFiles.Elements() {
			if obj, ok := elem.(types.Object); ok {
				// Extract the public_key from the object
				if publicKeyAttr, ok := obj.Attributes()["public_key"]; ok {
					if publicKey, ok := publicKeyAttr.(types.String); ok && !publicKey.IsNull() && !publicKey.IsUnknown() {
						pemFiles = append(pemFiles, publicKey.ValueString())
					}
				}
			}
		}
	}

	return pemFiles, diags
}

// extractPEMFilesFromState extracts PEM file content and IDs from the state
func extractPEMFilesFromState(ctx context.Context, state trustedTokenProfilesModel) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	pemFileMap := make(map[string]string) // public_key -> pem_file_id

	if !state.PEMFiles.IsNull() && !state.PEMFiles.IsUnknown() {
		// For SetNestedAttribute, we need to iterate through the set elements
		for _, elem := range state.PEMFiles.Elements() {
			if obj, ok := elem.(types.Object); ok {
				attrs := obj.Attributes()

				// Extract public_key and pem_file_id from the object
				if publicKeyAttr, ok := attrs["public_key"]; ok {
					if publicKey, ok := publicKeyAttr.(types.String); ok && !publicKey.IsNull() && !publicKey.IsUnknown() {
						if pemFileIDAttr, ok := attrs["pem_file_id"]; ok {
							if pemFileID, ok := pemFileIDAttr.(types.String); ok && !pemFileID.IsNull() && !pemFileID.IsUnknown() {
								pemFileMap[publicKey.ValueString()] = pemFileID.ValueString()
							}
						}
					}
				}
			}
		}
	}

	return pemFileMap, diags
}

// extractPEMFilesFromPlanAsMap extracts PEM file content from the plan as a map for comparison
func extractPEMFilesFromPlanAsMap(ctx context.Context, plan trustedTokenProfilesModel) (map[string]bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	pemFileMap := make(map[string]bool)

	if !plan.PEMFiles.IsNull() && !plan.PEMFiles.IsUnknown() {
		// For SetNestedAttribute, we need to iterate through the set elements
		for _, elem := range plan.PEMFiles.Elements() {
			if obj, ok := elem.(types.Object); ok {
				// Extract the public_key from the object
				if publicKeyAttr, ok := obj.Attributes()["public_key"]; ok {
					if publicKey, ok := publicKeyAttr.(types.String); ok && !publicKey.IsNull() && !publicKey.IsUnknown() {
						pemFileMap[publicKey.ValueString()] = true
					}
				}
			}
		}
	}

	return pemFileMap, diags
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
	if !plan.AttributeMappingJSON.IsNull() && !plan.AttributeMappingJSON.IsUnknown() {
		jsonStr := plan.AttributeMappingJSON.ValueString()
		err := json.Unmarshal([]byte(jsonStr), &attributeMapping)
		if err != nil {
			resp.Diagnostics.AddError("Invalid JSON", "attribute_mapping_json must be valid JSON object")
			return
		}
	}

	// Extract PEM files from plan
	pemFiles, diags := extractPEMFilesFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &trustedtokenprofiles.CreateTrustedTokenProfileRequest{
		ProjectID:        plan.ProjectID.ValueString(),
		Name:             plan.Name.ValueString(),
		Audience:         plan.Audience.ValueString(),
		Issuer:           plan.Issuer.ValueString(),
		AttributeMapping: attributeMapping,
		PublicKeyType:    plan.PublicKeyType.ValueString(),
	}

	if plan.PublicKeyType.ValueString() == "jwk" {
		if !plan.JwksURL.IsNull() && !plan.JwksURL.IsUnknown() {
			jwksUrl := plan.JwksURL.ValueString()
			createReq.JwksURL = &jwksUrl
		}
		plan.PEMFiles = types.SetNull(types.ObjectType{AttrTypes: pemFileModel{}.AttributeTypes()})
	} else if plan.PublicKeyType.ValueString() == "pem" {
		if !plan.PEMFiles.IsNull() && !plan.PEMFiles.IsUnknown() {
			createReq.PEMFiles = pemFiles
		}
		plan.JwksURL = types.StringNull()
	}

	createResp, err := r.client.TrustedTokenProfiles.Create(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating trusted token profile",
			fmt.Sprintf("Could not create trusted token profile: %s", err),
		)
		return
	}

	// Now update the state with the response
	diags = plan.refreshFromTrustedTokenProfile(ctx, createResp.TrustedTokenProfile)
	resp.Diagnostics.Append(diags...)
	plan.ID = types.StringValue(plan.ProjectID.ValueString() + "." + plan.ProfileID.ValueString())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
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
	if getResp.TrustedTokenProfile.JwksURL != nil {
		state.JwksURL = types.StringValue(*getResp.TrustedTokenProfile.JwksURL)
	} else {
		state.JwksURL = types.StringNull()
	}

	// Handle attribute mapping
	if len(getResp.TrustedTokenProfile.AttributeMapping) > 0 {
		// Convert the map to JSON string
		jsonBytes, err := json.Marshal(getResp.TrustedTokenProfile.AttributeMapping)
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal attribute mapping", err.Error())
			return
		}
		state.AttributeMappingJSON = types.StringValue(string(jsonBytes))
	} else {
		state.AttributeMappingJSON = types.StringNull()
	}

	// Handle PEM files
	if len(getResp.TrustedTokenProfile.PEMFiles) > 0 {
		pemFiles := make([]pemFileModel, 0, len(getResp.TrustedTokenProfile.PEMFiles))
		for _, pemFile := range getResp.TrustedTokenProfile.PEMFiles {
			pemFiles = append(pemFiles, pemFileModel{
				PEMFileID: types.StringValue(pemFile.ID),
				PublicKey: types.StringValue(pemFile.PublicKey),
			})
		}
		state.PEMFiles = types.SetValueMust(types.ObjectType{AttrTypes: pemFileModel{}.AttributeTypes()},
			func() []attr.Value {
				values := make([]attr.Value, len(pemFiles))
				for i, pemFile := range pemFiles {
					values[i] = types.ObjectValueMust(pemFile.AttributeTypes(), map[string]attr.Value{
						"pem_file_id": pemFile.PEMFileID,
						"public_key":  pemFile.PublicKey,
					})
				}
				return values
			}())
	} else {
		state.PEMFiles = types.SetNull(types.ObjectType{AttrTypes: pemFileModel{}.AttributeTypes()})
	}

	// Set state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *trustedTokenProfilesResource) Update(
	ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	var plan, state trustedTokenProfilesModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	ctx = tflog.SetField(ctx, "profile_id", plan.ProfileID.ValueString())
	tflog.Info(ctx, "Updating trusted token profile")

	var attributeMapping map[string]interface{}
	if !plan.AttributeMappingJSON.IsNull() && !plan.AttributeMappingJSON.IsUnknown() {
		jsonStr := plan.AttributeMappingJSON.ValueString()
		err := json.Unmarshal([]byte(jsonStr), &attributeMapping)
		if err != nil {
			resp.Diagnostics.AddError("Invalid JSON", "attribute_mapping_json must be valid JSON object")
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

	if !plan.JwksURL.IsNull() && !plan.JwksURL.IsUnknown() {
		jwksUrl := plan.JwksURL.ValueString()
		updateReq.JwksURL = &jwksUrl
	}

	_, err := r.client.TrustedTokenProfiles.Update(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating trusted token profile",
			fmt.Sprintf("Could not update trusted token profile: %s", err),
		)
		return
	}

	// Handle PEM files - compare current state with desired plan
	currentPEMs, diags := extractPEMFilesFromState(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	desiredPEMs, diags := extractPEMFilesFromPlanAsMap(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Find PEM files to add (in plan but not in state)
	var pemFilesToAdd []string
	for desiredPEM := range desiredPEMs {
		if _, exists := currentPEMs[desiredPEM]; !exists {
			pemFilesToAdd = append(pemFilesToAdd, desiredPEM)
		}
	}

	// Find PEM file IDs to delete (in state but not in plan)
	var pemFileIDsToDelete []string
	for currentPEM, pemFileID := range currentPEMs {
		if !desiredPEMs[currentPEM] {
			pemFileIDsToDelete = append(pemFileIDsToDelete, pemFileID)
		}
	}

	// Add new PEM files
	for _, pemContent := range pemFilesToAdd {
		tflog.Info(ctx, "Adding PEM file", map[string]interface{}{
			"profile_id": plan.ProfileID.ValueString(),
		})
		_, err := r.client.TrustedTokenProfiles.CreatePEM(ctx, &trustedtokenprofiles.CreatePEMFileRequest{
			ProjectID: plan.ProjectID.ValueString(),
			ProfileID: plan.ProfileID.ValueString(),
			PublicKey: pemContent,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error adding PEM file",
				fmt.Sprintf("Could not add PEM file: %s", err),
			)
			return
		}
	}

	// Remove old PEM files
	for _, pemFileID := range pemFileIDsToDelete {
		tflog.Info(ctx, "Removing PEM file", map[string]interface{}{
			"profile_id":  plan.ProfileID.ValueString(),
			"pem_file_id": pemFileID,
		})
		_, err := r.client.TrustedTokenProfiles.DeletePEM(ctx, &trustedtokenprofiles.DeletePEMFileRequest{
			ProjectID: plan.ProjectID.ValueString(),
			ProfileID: plan.ProfileID.ValueString(),
			PEMFileID: pemFileID,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting PEM file",
				fmt.Sprintf("Could not delete PEM file %s: %s", pemFileID, err),
			)
			return
		}
	}

	// Get the final state to ensure we have the correct PEM file IDs
	getResp, err := r.client.TrustedTokenProfiles.Get(ctx, &trustedtokenprofiles.GetTrustedTokenProfileRequest{
		ProjectID: plan.ProjectID.ValueString(),
		ProfileID: plan.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading trusted token profile after update",
			fmt.Sprintf("Could not read trusted token profile: %s", err),
		)
		return
	}

	// Now update the state with the final response
	diags = plan.refreshFromTrustedTokenProfile(ctx, getResp.TrustedTokenProfile)
	resp.Diagnostics.Append(diags...)
	plan.ID = types.StringValue(plan.ProjectID.ValueString() + "." + plan.ProfileID.ValueString())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
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
	// Parse the import ID (format: project_id.profile_id)
	importID := req.ID
	parts := strings.Split(importID, ".")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format 'project_id.profile_id'",
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
