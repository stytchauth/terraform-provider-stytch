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
	_ resource.Resource                = &consumerSDKConfigResource{}
	_ resource.ResourceWithConfigure   = &consumerSDKConfigResource{}
	_ resource.ResourceWithImportState = &consumerSDKConfigResource{}
)

func NewConsumerSDKConfigResource() resource.Resource {
	return &consumerSDKConfigResource{}
}

type consumerSDKConfigResource struct {
	client *api.API
}

type consumerSDKConfigModel struct {
	ProjectID   types.String                `tfsdk:"project_id"`
	LastUpdated types.String                `tfsdk:"last_updated"`
	Config      consumerSDKConfigInnerModel `tfsdk:"config"`
}

type consumerSDKConfigInnerModel struct {
	Basic         consumerSDKConfigBasicModel         `tfsdk:"basic"`
	Sessions      consumerSDKConfigSessionsModel      `tfsdk:"sessions"`
	MagicLinks    consumerSDKConfigMagicLinksModel    `tfsdk:"magic_links"`
	OTPs          consumerSDKConfigOTPsModel          `tfsdk:"otps"`
	OAuth         consumerSDKConfigOAuthModel         `tfsdk:"oauth"`
	TOTPs         consumerSDKConfigTOTPsModel         `tfsdk:"totps"`
	WebAuthn      consumerSDKConfigWebAuthnModel      `tfsdk:"webauthn"`
	CryptoWallets consumerSDKConfigCryptoWalletsModel `tfsdk:"crypto_wallets"`
	DFPPA         consumerSDKConfigDFPPAModel         `tfsdk:"dfppa"`
	Biometrics    consumerSDKConfigBiometricsModel    `tfsdk:"biometrics"`
	Passwords     consumerSDKConfigPasswordsModel     `tfsdk:"passwords"`
}

type consumerSDKConfigBasicModel struct {
	Enabled        types.Bool     `tfsdk:"enabled"`
	CreateNewUsers types.Bool     `tfsdk:"create_new_users"`
	Domains        []types.String `tfsdk:"domains"`
	BundleIDs      []types.String `tfsdk:"bundle_ids"`
}

type consumerSDKConfigSessionsModel struct {
	MaxSessionDurationMinutes types.Int32 `tfsdk:"max_session_duration_minutes"`
}

type consumerSDKConfigMagicLinksModel struct {
	LoginOrCreateEnabled types.Bool `tfsdk:"login_or_create_enabled"`
	SendEnabled          types.Bool `tfsdk:"send_enabled"`
	PKCERequired         types.Bool `tfsdk:"pkce_required"`
}

type consumerSDKConfigOTPsModel struct {
	SMSLoginOrCreateEnabled      types.Bool               `tfsdk:"sms_login_or_create_enabled"`
	WhatsAppLoginOrCreateEnabled types.Bool               `tfsdk:"whatsapp_login_or_create_enabled"`
	EmailLoginOrCreateEnabled    types.Bool               `tfsdk:"email_login_or_create_enabled"`
	SMSSendEnabled               types.Bool               `tfsdk:"sms_send_enabled"`
	WhatsAppSendEnabled          types.Bool               `tfsdk:"whatsapp_send_enabled"`
	EmailSendEnabled             types.Bool               `tfsdk:"email_send_enabled"`
	SMSAutofillMetadata          []sdkSMSAutofillMetadata `tfsdk:"sms_autofill_metadata"`
}

type consumerSDKConfigOAuthModel struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	PKCERequired types.Bool `tfsdk:"pkce_required"`
}

type consumerSDKConfigTOTPsModel struct {
	Enabled     types.Bool `tfsdk:"enabled"`
	CreateTOTPs types.Bool `tfsdk:"create_totps"`
}

type consumerSDKConfigWebAuthnModel struct {
	Enabled         types.Bool `tfsdk:"enabled"`
	CreateWebAuthns types.Bool `tfsdk:"create_webauthns"`
}

type consumerSDKConfigCryptoWalletsModel struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	SIWERequired types.Bool `tfsdk:"siwe_required"`
}

type consumerSDKConfigDFPPAModel struct {
	Enabled              types.String `tfsdk:"enabled"`
	OnChallenge          types.String `tfsdk:"on_challenge"`
	LookupTimeoutSeconds types.Int32  `tfsdk:"lookup_timeout_seconds"`
}

type consumerSDKConfigBiometricsModel struct {
	Enabled                 types.Bool `tfsdk:"enabled"`
	CreateBiometricsEnabled types.Bool `tfsdk:"create_biometrics_enabled"`
}

type consumerSDKConfigPasswordsModel struct {
	Enabled                       types.Bool `tfsdk:"enabled"`
	PKCERequiredForPasswordResets types.Bool `tfsdk:"pkce_required_for_password_resets"`
}

func (r *consumerSDKConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *consumerSDKConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_consumer_sdk_config"
}

// Schema defines the schema for the resource.
func (r *consumerSDKConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Required: true,
				Description: "The ID of the consumer project for which to set the SDK config. " +
					"This can be either a live project ID or test project ID. " +
					"You may only specify one SDK config per project.",
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"config": schema.SingleNestedAttribute{
				Required:    true,
				Description: "The consumer project SDK configuration.",
				Attributes: map[string]schema.Attribute{
					"basic": schema.SingleNestedAttribute{
						Required:    true,
						Description: "The basic configuration for the consumer project SDK. This includes enabling the SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Required:    true,
								Description: "A boolean indicating whether the consumer project SDK is enabled. This allows the SDK to manage user and session data.",
							},
							"create_new_users": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether new users can be created with the SDK.",
							},
							"domains": schema.ListAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A list of domains authorized for use in the SDK.",
								ElementType: types.StringType,
							},
							"bundle_ids": schema.ListAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A list of bundle IDs authorized for use in the SDK.",
								ElementType: types.StringType,
							},
						},
					},
					"sessions": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The session configuration for the consumer project SDK.",
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
						Description: "The magic links configuration for the consumer project SDK.",
						Attributes: map[string]schema.Attribute{
							"login_or_create_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether login or create with magic links is enabled in the SDK.",
							},
							"send_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the magic links send endpoint is enabled in the SDK.",
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
					"otps": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The OTP configuration for the consumer project SDK.",
						Attributes: map[string]schema.Attribute{
							"sms_login_or_create_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the SMS OTP login or create endpoint is enabled in the SDK.",
							},
							"whatsapp_login_or_create_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the WhatsApp OTP login or create endpoint is enabled in the SDK.",
							},
							"email_login_or_create_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the email OTP login or create endpoint is enabled in the SDK.",
							},
							"sms_send_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the SMS OTP send endpoint is enabled in the SDK.",
							},
							"whatsapp_send_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the WhatsApp OTP send endpoint is enabled in the SDK.",
							},
							"email_send_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the email OTP send endpoint is enabled in the SDK.",
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
											Description: "The value of the metadata to use for autofill. This should be the associated domain name (for metadata type 'domain')" +
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
						},
					},
					"oauth": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The OAuth configuration for the consumer project SDK.",
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
						Description: "The TOTP configuration for the consumer project SDK.",
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
					"webauthn": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The WebAuthn configuration for the consumer project SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether WebAuthn endpoints are enabled in the SDK.",
							},
							"create_webauthns": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether WebAuthn creation is enabled in the SDK.",
							},
						},
					},
					"crypto_wallets": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The Crypto Wallets configuration for the consumer project SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether Crypto Wallets endpoints are enabled in the SDK.",
							},
							"siwe_required": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether Sign In With Ethereum is required for Crypto Wallets.",
							},
						},
					},
					"dfppa": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The Device Fingerprinting Protected Auth configuration for the consumer project SDK.",
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
					"biometrics": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The Biometrics configuration for the consumer project SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether biometrics endpoints are enabled in the SDK.",
							},
							"create_biometrics_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether biometrics creation is enabled in the SDK.",
							},
						},
					},
					"passwords": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The Passwords configuration for the consumer project SDK.",
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
func (r *consumerSDKConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan consumerSDKConfigModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Generate API request body from plan and call r.client.SDKConfig.SetConsumerConfig

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *consumerSDKConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the current state
	var state consumerSDKConfigModel
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
func (r *consumerSDKConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan consumerSDKConfigModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO: Generate API request body from plan and call r.client.SDK.SetConsumerConfig

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *consumerSDKConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state consumerSDKConfigModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In this case, deleting an SDK config just means no longer tracking its state in terraform.
	// TODO: We *should* however make a call to the API to disable the SDK config.
}

func (r *consumerSDKConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("project_id"), req, resp)
}
