package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stytchauth/stytch-management-go/pkg/api"
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
	ProjectID   types.String           `tfsdk:"project_id"`
	LastUpdated types.String           `tfsdk:"last_updated"`
	Config      b2bSDKConfigInnerModel `tfsdk:"config"`
}

type b2bSDKConfigInnerModel struct {
	Basic      b2bSDKConfigBasicModel      `tfsdk:"basic"`
	Sessions   b2bSDKConfigSessionsModel   `tfsdk:"sessions"`
	MagicLinks b2bSDKConfigMagicLinksModel `tfsdk:"magic_links"`
	OAuth      b2bSDKConfigOAuthModel      `tfsdk:"oauth"`
	TOTPs      b2bSDKConfigTOTPsModel      `tfsdk:"totps"`
	SSO        b2bSDKConfigSSOModel        `tfsdk:"sso"`
	OTPs       b2bSDKConfigOTPsModel       `tfsdk:"otps"`
	DFPPA      b2bSDKConfigDFPPAModel      `tfsdk:"dfppa"`
	Passwords  b2bSDKConfigPasswordsModel  `tfsdk:"passwords"`
}

type b2bSDKConfigBasicModel struct {
	Enabled                 types.Bool                          `tfsdk:"enabled"`
	CreateNewMembers        types.Bool                          `tfsdk:"create_new_members"`
	AllowSelfOnboarding     types.Bool                          `tfsdk:"allow_self_onboarding"`
	EnableMemberPermissions types.Bool                          `tfsdk:"enable_member_permissions"`
	Domains                 []b2bSDKConfigAuthorizedDomainModel `tfsdk:"domains"`
	BundleIDs               []types.String                      `tfsdk:"bundle_ids"`
}

type b2bSDKConfigAuthorizedDomainModel struct {
	Domain      types.String `tfsdk:"domain"`
	SlugPattern types.String `tfsdk:"slug_pattern"`
}

type b2bSDKConfigSessionsModel struct {
	MaxSessionDurationMinutes types.Int32 `tfsdk:"max_session_duration_minutes"`
}

type b2bSDKConfigMagicLinksModel struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	PKCERequired types.Bool `tfsdk:"pkce_required"`
}

type b2bSDKConfigOAuthModel struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	PKCERequired types.Bool `tfsdk:"pkce_required"`
}

type b2bSDKConfigTOTPsModel struct {
	Enabled     types.Bool `tfsdk:"enabled"`
	CreateTOTPs types.Bool `tfsdk:"create_totps"`
}

type b2bSDKConfigSSOModel struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	PKCERequired types.Bool `tfsdk:"pkce_required"`
}

type b2bSDKConfigOTPsModel struct {
	SMSEnabled          types.Bool               `tfsdk:"sms_enabled"`
	SMSAutofillMetadata []sdkSMSAutofillMetadata `tfsdk:"sms_autofill_metadata"`
	EmailEnabled        types.Bool               `tfsdk:"email_enabled"`
}

type b2bSDKConfigDFPPAModel struct {
	Enabled              types.String `tfsdk:"enabled"`
	OnChallenge          types.String `tfsdk:"on_challenge"`
	LookupTimeoutSeconds types.Int32  `tfsdk:"lookup_timeout_seconds"`
}

type b2bSDKConfigPasswordsModel struct {
	Enabled                       types.Bool `tfsdk:"enabled"`
	PKCERequiredForPasswordResets types.Bool `tfsdk:"pkce_required_for_password_resets"`
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
							},
							"allow_self_onboarding": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether self-onboarding is allowed for members in the SDK.",
							},
							"enable_member_permissions": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether member permissions RBAC are enabled in the SDK.",
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
										},
										"slug_pattern": schema.StringAttribute{
											Optional: true,
											Computed: true,
											Description: "SlugPattern is the slug pattern which can be used to support authentication flows specific to each organization. An example" +
												"value here might be 'https://{{slug}}.example.com'. The value **must** include '{{slug}}' as a placeholder for the slug.",
										},
									},
								},
							},
							"bundle_ids": schema.ListAttribute{
								Optional:    true,
								Computed:    true,
								ElementType: types.StringType,
								Description: "A list of bundle IDs authorized for use in the SDK.",
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
							},
							"pkce_required": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Description: "PKCERequired is a boolean indicating whether PKCE is required for magic links. PKCE increases security by " +
									"introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow from " +
									"the same application on the device. This prevents a malicious app from intercepting a redirect and authenticating " +
									"with the users token. PKCE is enabled by default for mobile SDKs.",
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
							},
							"pkce_required": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Description: "PKCERequired is a boolean indicating whether PKCE is required for OAuth. PKCE increases security by " +
									"introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow from " +
									"the same application on the device. This prevents a malicious app from intercepting a redirect and authenticating " +
									"with the users token. PKCE is enabled by default for mobile SDKs.",
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
							},
							"create_totps": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether TOTP creation is enabled in the SDK.",
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
							},
							"pkce_required": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Description: "PKCERequired is a boolean indicating whether PKCE is required for SSO. PKCE increases security by " +
									"introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow from " +
									"the same application on the device. This prevents a malicious app from intercepting a redirect and authenticating " +
									"with the users token. PKCE is enabled by default for mobile SDKs.",
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
										},
										"metadata_value": schema.StringAttribute{
											Optional: true,
											Computed: true,
											Description: "MetadataValue is the value of the metadata to use for autofill. This should be the associated domain name (for metadata type 'domain') " +
												"or application hash (for metadata type 'hash').",
										},
										"bundle_id": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "The ID of the bundle to use for autofill. This should be the associated bundle ID.",
										},
									},
								},
							},
							"email_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the email OTP endpoints are enabled in the SDK.",
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
							},
							"on_challenge": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Description: "The action to take when a DFPPA 'challenge' verdict is returned.",
							},
							"lookup_timeout_seconds": schema.Int32Attribute{
								Optional:    true,
								Computed:    true,
								Description: "How long to wait for a DFPPA lookup to complete before timing out.",
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
							},
							"pkce_required_for_password_resets": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Description: "PKCERequiredForPasswordResets is a boolean indicating whether PKCE is required for password resets. PKCE increases " +
									"security by introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow " +
									"from the same application on the device. This prevents a malicious app from intercepting a redirect and " +
									"authenticating with the users token. PKCE is enabled by default for mobile SDKs.",
							},
						},
					},
				},
			},
		},
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

	// TODO: Generate API request body from plan and call r.client.SDKConfig.SetB2BConfig

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

	// Set refreshed state
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

	// TODO: Generate API request body from plan and call r.client.SDK.SetB2BConfig

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
