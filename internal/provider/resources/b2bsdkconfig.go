package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/pkg/api"
	"github.com/stytchauth/stytch-management-go/pkg/models/projects"
	"github.com/stytchauth/stytch-management-go/pkg/models/sdk"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &b2bSDKConfigResource{}
	_ resource.ResourceWithConfigure   = &b2bSDKConfigResource{}
	_ resource.ResourceWithImportState = &b2bSDKConfigResource{}
)

func NewB2BSDKConfigResource() resource.Resource {
	return &b2bSDKConfigResource{}
}

type b2bSDKConfigResource struct {
	client *api.API
}

type b2bSDKConfigModel struct {
	ProjectID   types.String `tfsdk:"project_id"`
	LastUpdated types.String `tfsdk:"last_updated"`
	Config      types.Object `tfsdk:"config"`
}

func (m b2bSDKConfigModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"project_id":   types.StringType,
		"last_updated": types.StringType,
		"config":       types.ObjectType{AttrTypes: b2bSDKConfigInnerModel{}.AttributeTypes()},
	}
}

type b2bSDKConfigInnerModel struct {
	Basic      types.Object `tfsdk:"basic"`
	Sessions   types.Object `tfsdk:"sessions"`
	MagicLinks types.Object `tfsdk:"magic_links"`
	OAuth      types.Object `tfsdk:"oauth"`
	TOTPs      types.Object `tfsdk:"totps"`
	SSO        types.Object `tfsdk:"sso"`
	OTPs       types.Object `tfsdk:"otps"`
	DFPPA      types.Object `tfsdk:"dfppa"`
	Passwords  types.Object `tfsdk:"passwords"`
}

func (m b2bSDKConfigInnerModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"basic":       types.ObjectType{AttrTypes: b2bSDKConfigBasicModel{}.AttributeTypes()},
		"sessions":    types.ObjectType{AttrTypes: b2bSDKConfigSessionsModel{}.AttributeTypes()},
		"magic_links": types.ObjectType{AttrTypes: b2bSDKConfigMagicLinksModel{}.AttributeTypes()},
		"oauth":       types.ObjectType{AttrTypes: b2bSDKConfigOAuthModel{}.AttributeTypes()},
		"totps":       types.ObjectType{AttrTypes: b2bSDKConfigTOTPsModel{}.AttributeTypes()},
		"sso":         types.ObjectType{AttrTypes: b2bSDKConfigSSOModel{}.AttributeTypes()},
		"otps":        types.ObjectType{AttrTypes: b2bSDKConfigOTPsModel{}.AttributeTypes()},
		"dfppa":       types.ObjectType{AttrTypes: b2bSDKConfigDFPPAModel{}.AttributeTypes()},
		"passwords":   types.ObjectType{AttrTypes: b2bSDKConfigPasswordsModel{}.AttributeTypes()},
	}
}

type b2bSDKConfigBasicModel struct {
	Enabled                 types.Bool                          `tfsdk:"enabled"`
	CreateNewMembers        types.Bool                          `tfsdk:"create_new_members"`
	AllowSelfOnboarding     types.Bool                          `tfsdk:"allow_self_onboarding"`
	EnableMemberPermissions types.Bool                          `tfsdk:"enable_member_permissions"`
	Domains                 []b2bSDKConfigAuthorizedDomainModel `tfsdk:"domains"`
	BundleIDs               []types.String                      `tfsdk:"bundle_ids"`
}

func (m b2bSDKConfigBasicModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":                   types.BoolType,
		"create_new_members":        types.BoolType,
		"allow_self_onboarding":     types.BoolType,
		"enable_member_permissions": types.BoolType,
		"domains":                   types.ListType{ElemType: types.ObjectType{AttrTypes: b2bSDKConfigAuthorizedDomainModel{}.AttributeTypes()}},
		"bundle_ids":                types.ListType{ElemType: types.StringType},
	}
}

type b2bSDKConfigAuthorizedDomainModel struct {
	Domain      types.String `tfsdk:"domain"`
	SlugPattern types.String `tfsdk:"slug_pattern"`
}

func (m b2bSDKConfigAuthorizedDomainModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"domain":       types.StringType,
		"slug_pattern": types.StringType,
	}
}

type b2bSDKConfigSessionsModel struct {
	MaxSessionDurationMinutes types.Int32 `tfsdk:"max_session_duration_minutes"`
}

func (m b2bSDKConfigSessionsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"max_session_duration_minutes": types.Int32Type,
	}
}

type b2bSDKConfigMagicLinksModel struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	PKCERequired types.Bool `tfsdk:"pkce_required"`
}

func (m b2bSDKConfigMagicLinksModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":       types.BoolType,
		"pkce_required": types.BoolType,
	}
}

type b2bSDKConfigOAuthModel struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	PKCERequired types.Bool `tfsdk:"pkce_required"`
}

func (m b2bSDKConfigOAuthModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":       types.BoolType,
		"pkce_required": types.BoolType,
	}
}

type b2bSDKConfigTOTPsModel struct {
	Enabled     types.Bool `tfsdk:"enabled"`
	CreateTOTPs types.Bool `tfsdk:"create_totps"`
}

func (m b2bSDKConfigTOTPsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":      types.BoolType,
		"create_totps": types.BoolType,
	}
}

type b2bSDKConfigSSOModel struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	PKCERequired types.Bool `tfsdk:"pkce_required"`
}

func (m b2bSDKConfigSSOModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":       types.BoolType,
		"pkce_required": types.BoolType,
	}
}

type b2bSDKConfigOTPsModel struct {
	SMSEnabled          types.Bool               `tfsdk:"sms_enabled"`
	SMSAutofillMetadata []sdkSMSAutofillMetadata `tfsdk:"sms_autofill_metadata"`
	EmailEnabled        types.Bool               `tfsdk:"email_enabled"`
}

func (m b2bSDKConfigOTPsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"sms_enabled":           types.BoolType,
		"sms_autofill_metadata": types.ListType{ElemType: types.ObjectType{AttrTypes: sdkSMSAutofillMetadata{}.AttributeTypes()}},
		"email_enabled":         types.BoolType,
	}
}

type b2bSDKConfigDFPPAModel struct {
	Enabled              types.String `tfsdk:"enabled"`
	OnChallenge          types.String `tfsdk:"on_challenge"`
	LookupTimeoutSeconds types.Int32  `tfsdk:"lookup_timeout_seconds"`
}

func (m b2bSDKConfigDFPPAModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":                types.StringType,
		"on_challenge":           types.StringType,
		"lookup_timeout_seconds": types.Int32Type,
	}
}

type b2bSDKConfigPasswordsModel struct {
	Enabled                       types.Bool `tfsdk:"enabled"`
	PKCERequiredForPasswordResets types.Bool `tfsdk:"pkce_required_for_password_resets"`
}

func (m b2bSDKConfigPasswordsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":                           types.BoolType,
		"pkce_required_for_password_resets": types.BoolType,
	}
}

func (m *b2bSDKConfigModel) reloadFromSDKConfig(ctx context.Context, c sdk.B2BConfig) diag.Diagnostics {
	cfg, diag := types.ObjectValueFrom(ctx, b2bSDKConfigInnerModel{}.AttributeTypes(), &m.Config)
	m.Config = cfg
	return diag
}

func (r *b2bSDKConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *b2bSDKConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_b2b_sdk_config"
}

// Schema defines the schema for the resource.
func (r *b2bSDKConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Required: true,
				Description: "The ID of the B2B project for which to set the SDK config. " +
					"This can be either a live project ID or test project ID. " +
					"You may only specify one SDK config per project.",
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"config": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The B2B project SDK configuration.",
				Attributes: map[string]schema.Attribute{
					"basic": schema.SingleNestedAttribute{
						Required:    true,
						Description: "The basic configuration for the B2B project SDK. This includes enabling the SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Required:    true,
								Description: "A boolean indicating whether the B2B project SDK is enabled. This allows the SDK to manage user and session data.",
							},
							"create_new_members": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether new members can be created with the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"allow_self_onboarding": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether self-onboarding is allowed for members in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"enable_member_permissions": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether member permissions RBAC are enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"domains": schema.ListNestedAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A list of domains authorized for use in the SDK.",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"domain": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "The domain name. Stytch uses the same-origin policy to determine matches.",
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
										},
										"slug_pattern": schema.StringAttribute{
											Optional: true,
											Computed: true,
											Description: "SlugPattern is the slug pattern which can be used to support authentication flows specific to each organization. An example" +
												"value here might be 'https://{{slug}}.example.com'. The value **must** include '{{slug}}' as a placeholder for the slug.",
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
										},
									},
								},
							},
							"bundle_ids": schema.ListAttribute{
								Optional:    true,
								Computed:    true,
								ElementType: types.StringType,
								Description: "A list of bundle IDs authorized for use in the SDK.",
								PlanModifiers: []planmodifier.List{
									listplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"sessions": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The session configuration for the B2B project SDK.",
						Attributes: map[string]schema.Attribute{
							"max_session_duration_minutes": schema.Int32Attribute{
								Optional:    true,
								Computed:    true,
								Description: "The maximum session duration that can be created in minutes.",
								PlanModifiers: []planmodifier.Int32{
									int32planmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"magic_links": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The magic links configuration for the B2B project SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether magic links endpoints are enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"pkce_required": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Description: "PKCERequired is a boolean indicating whether PKCE is required for magic links. PKCE increases security by " +
									"introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow from " +
									"the same application on the device. This prevents a malicious app from intercepting a redirect and authenticating " +
									"with the users token. PKCE is enabled by default for mobile SDKs.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"oauth": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The OAuth configuration for the B2B project SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether OAuth endpoints are enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"pkce_required": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Description: "PKCERequired is a boolean indicating whether PKCE is required for OAuth. PKCE increases security by " +
									"introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow from " +
									"the same application on the device. This prevents a malicious app from intercepting a redirect and authenticating " +
									"with the users token. PKCE is enabled by default for mobile SDKs.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"totps": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The TOTPs configuration for the B2B project SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether TOTP endpoints are enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"create_totps": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether TOTP creation is enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"sso": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The SSO configuration for the B2B project SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether SSO endpoints are enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"pkce_required": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Description: "PKCERequired is a boolean indicating whether PKCE is required for SSO. PKCE increases security by " +
									"introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow from " +
									"the same application on the device. This prevents a malicious app from intercepting a redirect and authenticating " +
									"with the users token. PKCE is enabled by default for mobile SDKs.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"otps": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The OTPs configuration for the B2B project SDK.",
						Attributes: map[string]schema.Attribute{
							"sms_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the SMS OTP endpoints are enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"sms_autofill_metadata": schema.ListNestedAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A list of metadata that can be used for autofill of SMS OTPs.",
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"metadata_type": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "The type of metadata to use for autofill. This should be either 'domain' or 'hash'.",
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
										},
										"metadata_value": schema.StringAttribute{
											Optional: true,
											Computed: true,
											Description: "MetadataValue is the value of the metadata to use for autofill. This should be the associated domain name (for metadata type 'domain') " +
												"or application hash (for metadata type 'hash').",
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
										},
										"bundle_id": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "The ID of the bundle to use for autofill. This should be the associated bundle ID.",
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
										},
									},
								},
							},
							"email_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the email OTP endpoints are enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"dfppa": schema.SingleNestedAttribute{
						Optional: true,
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"enabled": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether Device Fingerprinting Protected Auth is enabled in the SDK.",
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"on_challenge": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Description: "The action to take when a DFPPA 'challenge' verdict is returned.",
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"lookup_timeout_seconds": schema.Int32Attribute{
								Optional:    true,
								Computed:    true,
								Description: "How long to wait for a DFPPA lookup to complete before timing out.",
								PlanModifiers: []planmodifier.Int32{
									int32planmodifier.UseStateForUnknown(),
								},
							},
						},
					},
					"passwords": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The passwords configuration for the B2B project SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether password endpoints are enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"pkce_required_for_password_resets": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Description: "PKCERequiredForPasswordResets is a boolean indicating whether PKCE is required for password resets. PKCE increases " +
									"security by introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow " +
									"from the same application on the device. This prevents a malicious app from intercepting a redirect and " +
									"authenticating with the users token. PKCE is enabled by default for mobile SDKs.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r b2bSDKConfigResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data b2bSDKConfigModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the projectID isn't yet known, skip validation for now.
	// The plugin framework will call ValidateConfig again when all required values are known.
	if data.ProjectID.IsUnknown() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Validating B2B SDK config for %+v", data))
	getProjectResp, err := r.client.Projects.Get(ctx, projects.GetRequest{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddWarning("Failed to get project for vertical check", err.Error())
		return
	}
	if getProjectResp.Project.Vertical != projects.VerticalB2B {
		resp.Diagnostics.AddError("Invalid project vertical", "The project must be a B2B project for this resource.")
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *b2bSDKConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan b2bSDKConfigModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var cfg sdk.B2BConfig
	diags = plan.Config.As(ctx, &cfg, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.SDK.SetB2BConfig(ctx, sdk.SetB2BConfigRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Config:    cfg,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set B2B SDK config", err.Error())
		return
	}

	diags = plan.reloadFromSDKConfig(ctx, setResp.Config)
	resp.Diagnostics.Append(diags...)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *b2bSDKConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the current state
	var state b2bSDKConfigModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Get refreshed value from the API
	getResp, err := r.client.SDK.GetB2BConfig(ctx, sdk.GetB2BConfigRequest{
		ProjectID: state.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get B2B SDK config", err.Error())
		return
	}

	diags = state.reloadFromSDKConfig(ctx, getResp.Config)
	resp.Diagnostics.Append(diags...)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *b2bSDKConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan b2bSDKConfigModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var cfg sdk.B2BConfig
	diags = plan.Config.As(ctx, &cfg, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.SDK.SetB2BConfig(ctx, sdk.SetB2BConfigRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Config:    cfg,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set B2B SDK config", err.Error())
		return
	}

	diags = plan.reloadFromSDKConfig(ctx, setResp.Config)
	resp.Diagnostics.Append(diags...)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *b2bSDKConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state b2bSDKConfigModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In this case, deleting an SDK config just means no longer tracking its state in terraform.
	// TODO: We *should* however make a call to the API to disable the SDK config.
}

func (r *b2bSDKConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("project_id"), req, resp)
}
