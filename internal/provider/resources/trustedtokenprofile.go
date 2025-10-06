package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/trustedtokenprofiles"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &trustedTokenProfileResource{}
	_ resource.ResourceWithConfigure   = &trustedTokenProfileResource{}
	_ resource.ResourceWithImportState = &trustedTokenProfileResource{}
)

func NewTrustedTokenProfileResource() resource.Resource {
	return &trustedTokenProfileResource{}
}

type trustedTokenProfileResource struct {
	client *api.API
}

type trustedTokenProfileModel struct {
	ID                   types.String `tfsdk:"id"`
	ProjectSlug          types.String `tfsdk:"project_slug"`
	EnvironmentSlug      types.String `tfsdk:"environment_slug"`
	ProfileID            types.String `tfsdk:"profile_id"`
	Name                 types.String `tfsdk:"name"`
	Audience             types.String `tfsdk:"audience"`
	Issuer               types.String `tfsdk:"issuer"`
	JwksURL              types.String `tfsdk:"jwks_url"`
	AttributeMappingJSON types.String `tfsdk:"attribute_mapping_json"`
	PEMFiles             types.Set    `tfsdk:"pem_files"`
	PublicKeyType        types.String `tfsdk:"public_key_type"`
	CanJITProvision      types.Bool   `tfsdk:"can_jit_provision"`
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
func (r *trustedTokenProfileResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
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
func (r *trustedTokenProfileResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_trusted_token_profile"
}

// Schema defines the schema for the resource.
func (r *trustedTokenProfileResource) Schema(
	ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Resource for managing trusted token profiles.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management (format: project_slug.environment_slug.profile_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project to which the trusted token profile belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to which the trusted token profile belongs.",
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
				Description: "The JWKS URL for the trusted token profile (required when public_key_type is JWK).",
			},
			"attribute_mapping_json": schema.StringAttribute{
				Optional:    true,
				Description: "The attribute mapping as a JSON object where keys and values are strings.",
			},
			"pem_files": schema.SetNestedAttribute{
				Optional:    true,
				Description: "Set of PEM files associated with the trusted token profile (required when public_key_type is PEM).",
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
				Description: "The type of public key. Valid values: JWK, PEM.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(trustedtokenprofiles.PublicKeyTypeJWK),
						string(trustedtokenprofiles.PublicKeyTypePEM),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"can_jit_provision": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the trusted token profile can be provisioned just-in-time.",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update.",
				Computed:    true,
			},
		},
	}
}

func (ttp *trustedTokenProfileModel) refreshFromTrustedTokenProfile(r trustedtokenprofiles.TrustedTokenProfile) diag.Diagnostics {
	var diags diag.Diagnostics

	ttp.ProfileID = types.StringValue(r.ID)
	ttp.Name = types.StringValue(r.Name)
	ttp.Audience = types.StringValue(r.Audience)
	ttp.Issuer = types.StringValue(r.Issuer)
	ttp.PublicKeyType = types.StringValue(string(r.PublicKeyType))
	ttp.CanJITProvision = types.BoolValue(r.CanJITProvision)

	if r.JwksURL != "" {
		ttp.JwksURL = types.StringValue(r.JwksURL)
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

	return diags
}

// extractPEMFilesFromPlan extracts PEM file content from the plan.
func extractPEMFilesFromPlan(plan trustedTokenProfileModel) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	var pemFiles []string

	if !plan.PEMFiles.IsNull() && !plan.PEMFiles.IsUnknown() {
		for _, elem := range plan.PEMFiles.Elements() {
			if obj, ok := elem.(types.Object); ok {
				if publicKeyAttr, ok := obj.Attributes()["public_key"]; ok {
					if publicKey, ok := publicKeyAttr.(types.String); ok && !publicKey.IsNull() && !publicKey.IsUnknown() {
						pemFiles = append(pemFiles, publicKey.ValueString())
					}
				} else {
					diags.AddError("Invalid PEM file", "public_key is required")
				}
			}
		}
	}

	return pemFiles, diags
}

// extractPEMFilesFromState extracts PEM file content and IDs from the state.
func extractPEMFilesFromState(state trustedTokenProfileModel) (map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	pemFileMap := make(map[string]string) // public_key -> pem_file_id

	if !state.PEMFiles.IsNull() && !state.PEMFiles.IsUnknown() {
		for _, elem := range state.PEMFiles.Elements() {
			if obj, ok := elem.(types.Object); ok {
				attrs := obj.Attributes()

				if publicKeyAttr, ok := attrs["public_key"]; ok {
					if publicKey, ok := publicKeyAttr.(types.String); ok && !publicKey.IsNull() && !publicKey.IsUnknown() {
						if pemFileIDAttr, ok := attrs["pem_file_id"]; ok {
							if pemFileID, ok := pemFileIDAttr.(types.String); ok && !pemFileID.IsNull() && !pemFileID.IsUnknown() {
								pemFileMap[publicKey.ValueString()] = pemFileID.ValueString()
							}
						}
					}
				} else {
					diags.AddError("Invalid PEM file", "public_key is required")
				}
			}
		}
	}

	return pemFileMap, diags
}

// extractPEMFilesFromPlanAsMap extracts PEM file content from the plan as a map for comparison.
func extractPEMFilesFromPlanAsMap(plan trustedTokenProfileModel) (map[string]bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	pemFileMap := make(map[string]bool)

	if !plan.PEMFiles.IsNull() && !plan.PEMFiles.IsUnknown() {
		for _, elem := range plan.PEMFiles.Elements() {
			if obj, ok := elem.(types.Object); ok {
				if publicKeyAttr, ok := obj.Attributes()["public_key"]; ok {
					if publicKey, ok := publicKeyAttr.(types.String); ok && !publicKey.IsNull() && !publicKey.IsUnknown() {
						pemFileMap[publicKey.ValueString()] = true
					}
				} else {
					diags.AddError("Invalid PEM file", "public_key is required")
				}
			}
		}
	}

	return pemFileMap, diags
}

// Create creates the resource and sets the initial Terraform state.
func (r *trustedTokenProfileResource) Create(
	ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse,
) {
	var plan trustedTokenProfileModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	tflog.Info(ctx, "Creating trusted token profile")

	var attributeMapping map[string]string
	if !plan.AttributeMappingJSON.IsNull() && !plan.AttributeMappingJSON.IsUnknown() {
		jsonStr := plan.AttributeMappingJSON.ValueString()
		err := json.Unmarshal([]byte(jsonStr), &attributeMapping)
		if err != nil {
			resp.Diagnostics.AddError("Invalid JSON", "attribute_mapping_json must be valid JSON object with string keys and values")
			return
		}
	}

	// Extract PEM files from plan
	pemFiles, diags := extractPEMFilesFromPlan(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &trustedtokenprofiles.CreateTrustedTokenProfileRequest{
		ProjectSlug:      plan.ProjectSlug.ValueString(),
		EnvironmentSlug:  plan.EnvironmentSlug.ValueString(),
		Name:             plan.Name.ValueString(),
		Audience:         plan.Audience.ValueString(),
		Issuer:           plan.Issuer.ValueString(),
		AttributeMapping: attributeMapping,
		PublicKeyType:    trustedtokenprofiles.PublicKeyType(plan.PublicKeyType.ValueString()),
		CanJITProvision:  plan.CanJITProvision.ValueBool(),
	}

	if plan.PublicKeyType.ValueString() == string(trustedtokenprofiles.PublicKeyTypeJWK) {
		if !plan.JwksURL.IsNull() && !plan.JwksURL.IsUnknown() {
			jwksUrl := plan.JwksURL.ValueString()
			createReq.JwksURL = &jwksUrl
		}
		plan.PEMFiles = types.SetNull(types.ObjectType{AttrTypes: pemFileModel{}.AttributeTypes()})
	} else if plan.PublicKeyType.ValueString() == string(trustedtokenprofiles.PublicKeyTypePEM) {
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

	tflog.Info(ctx, "Created trusted token profile")

	// Update the state with the response
	diags = plan.refreshFromTrustedTokenProfile(createResp.TrustedTokenProfile)
	resp.Diagnostics.Append(diags...)
	plan.ID = types.StringValue(fmt.Sprintf("%s.%s.%s", plan.ProjectSlug.ValueString(), plan.EnvironmentSlug.ValueString(), plan.ProfileID.ValueString()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *trustedTokenProfileResource) Read(
	ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse,
) {
	var state trustedTokenProfileModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "profile_id", state.ProfileID.ValueString())
	tflog.Info(ctx, "Reading trusted token profile")

	getResp, err := r.client.TrustedTokenProfiles.Get(ctx, &trustedtokenprofiles.GetTrustedTokenProfileRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		ProfileID:       state.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading trusted token profile",
			fmt.Sprintf("Could not read trusted token profile: %s", err),
		)
		return
	}

	tflog.Info(ctx, "Read trusted token profile")

	// Update the state with the response data
	state.Name = types.StringValue(getResp.TrustedTokenProfile.Name)
	state.Audience = types.StringValue(getResp.TrustedTokenProfile.Audience)
	state.Issuer = types.StringValue(getResp.TrustedTokenProfile.Issuer)
	state.PublicKeyType = types.StringValue(string(getResp.TrustedTokenProfile.PublicKeyType))
	state.CanJITProvision = types.BoolValue(getResp.TrustedTokenProfile.CanJITProvision)

	// Handle JWKS URL
	if getResp.TrustedTokenProfile.JwksURL != "" {
		state.JwksURL = types.StringValue(getResp.TrustedTokenProfile.JwksURL)
	} else {
		state.JwksURL = types.StringNull()
	}

	// Handle attribute mapping
	if len(getResp.TrustedTokenProfile.AttributeMapping) > 0 {
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

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *trustedTokenProfileResource) Update(
	ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	var plan, state trustedTokenProfileModel
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

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "profile_id", plan.ProfileID.ValueString())
	tflog.Info(ctx, "Updating trusted token profile")

	var attributeMapping map[string]string
	if !plan.AttributeMappingJSON.IsNull() && !plan.AttributeMappingJSON.IsUnknown() {
		jsonStr := plan.AttributeMappingJSON.ValueString()
		err := json.Unmarshal([]byte(jsonStr), &attributeMapping)
		if err != nil {
			resp.Diagnostics.AddError("Invalid JSON", "attribute_mapping_json must be valid JSON object with string keys and values")
			return
		}
	}

	name := plan.Name.ValueString()
	audience := plan.Audience.ValueString()
	issuer := plan.Issuer.ValueString()
	canJITProvision := plan.CanJITProvision.ValueBool()

	updateReq := &trustedtokenprofiles.UpdateTrustedTokenProfileRequest{
		ProjectSlug:      plan.ProjectSlug.ValueString(),
		EnvironmentSlug:  plan.EnvironmentSlug.ValueString(),
		ProfileID:        plan.ProfileID.ValueString(),
		Name:             &name,
		Audience:         &audience,
		Issuer:           &issuer,
		AttributeMapping: attributeMapping,
		CanJITProvision:  &canJITProvision,
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
	currentPEMs, diags := extractPEMFilesFromState(state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	desiredPEMs, diags := extractPEMFilesFromPlanAsMap(plan)
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
			ProjectSlug:     plan.ProjectSlug.ValueString(),
			EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
			ProfileID:       plan.ProfileID.ValueString(),
			PublicKey:       pemContent,
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
			ProjectSlug:     plan.ProjectSlug.ValueString(),
			EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
			ProfileID:       plan.ProfileID.ValueString(),
			PEMFileID:       pemFileID,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting PEM file",
				fmt.Sprintf("Could not delete PEM file %s: %s", pemFileID, err),
			)
			return
		}
	}

	tflog.Info(ctx, "Updated trusted token profile")

	// Get the final state to ensure we have the correct PEM file IDs
	getResp, err := r.client.TrustedTokenProfiles.Get(ctx, &trustedtokenprofiles.GetTrustedTokenProfileRequest{
		ProjectSlug:     plan.ProjectSlug.ValueString(),
		EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
		ProfileID:       plan.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading trusted token profile after update",
			fmt.Sprintf("Could not read trusted token profile: %s", err),
		)
		return
	}

	// Update the state with the final response
	diags = plan.refreshFromTrustedTokenProfile(getResp.TrustedTokenProfile)
	resp.Diagnostics.Append(diags...)
	plan.ID = types.StringValue(fmt.Sprintf("%s.%s.%s", plan.ProjectSlug.ValueString(), plan.EnvironmentSlug.ValueString(), plan.ProfileID.ValueString()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *trustedTokenProfileResource) Delete(
	ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	var state trustedTokenProfileModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "profile_id", state.ProfileID.ValueString())
	tflog.Info(ctx, "Deleting trusted token profile")

	// Deleting a PEM-based trusted token profile also deletes all associated PEM files.
	_, err := r.client.TrustedTokenProfiles.Delete(ctx, &trustedtokenprofiles.DeleteTrustedTokenProfileRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		ProfileID:       state.ProfileID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting trusted token profile",
			fmt.Sprintf("Could not delete trusted token profile: %s", err),
		)
		return
	}

	tflog.Info(ctx, "Deleted trusted token profile")
}

// ImportState imports the resource into Terraform state.
func (r *trustedTokenProfileResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	ctx = tflog.SetField(ctx, "import_id", req.ID)
	tflog.Info(ctx, "Importing trusted token profile")

	// Import ID format: project_slug.environment_slug.profile_id
	parts := strings.Split(req.ID, ".")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("Expected import ID format: project_slug.environment_slug.profile_id, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_slug"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("profile_id"), parts[2])...)
}
