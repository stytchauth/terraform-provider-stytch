package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stytchauth/stytch-management-go/pkg/api"
	"github.com/stytchauth/stytch-management-go/pkg/models/projects"
	"github.com/stytchauth/stytch-management-go/pkg/models/sdk"
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

func (m consumerSDKConfigModel) toSDKConfig() sdk.ConsumerConfig {
	c := sdk.ConsumerConfig{
		Basic: &sdk.ConsumerBasicConfig{
			Enabled:        m.Config.Basic.Enabled.ValueBool(),
			CreateNewUsers: m.Config.Basic.CreateNewUsers.ValueBool(),
			Domains:        make([]string, len(m.Config.Basic.Domains)),
			BundleIDs:      make([]string, len(m.Config.Basic.BundleIDs)),
		},
	}
	for i, d := range m.Config.Basic.Domains {
		c.Basic.Domains[i] = d.ValueString()
	}
	for i, b := range m.Config.Basic.BundleIDs {
		c.Basic.BundleIDs[i] = b.ValueString()
	}

	if !m.Config.Sessions.MaxSessionDurationMinutes.IsUnknown() {
		c.Sessions = &sdk.ConsumerSessionsConfig{
			MaxSessionDurationMinutes: m.Config.Sessions.MaxSessionDurationMinutes.ValueInt32(),
		}
	}

	if !m.Config.MagicLinks.LoginOrCreateEnabled.IsUnknown() ||
		!m.Config.MagicLinks.SendEnabled.IsUnknown() ||
		!m.Config.MagicLinks.PKCERequired.IsUnknown() {
		c.MagicLinks = &sdk.ConsumerMagicLinksConfig{
			LoginOrCreateEnabled: m.Config.MagicLinks.LoginOrCreateEnabled.ValueBool(),
			SendEnabled:          m.Config.MagicLinks.SendEnabled.ValueBool(),
			PKCERequired:         m.Config.MagicLinks.PKCERequired.ValueBool(),
		}
	}

	if !m.Config.OTPs.SMSLoginOrCreateEnabled.IsUnknown() ||
		!m.Config.OTPs.WhatsAppLoginOrCreateEnabled.IsUnknown() ||
		!m.Config.OTPs.EmailLoginOrCreateEnabled.IsUnknown() ||
		!m.Config.OTPs.SMSSendEnabled.IsUnknown() ||
		!m.Config.OTPs.WhatsAppSendEnabled.IsUnknown() ||
		!m.Config.OTPs.EmailSendEnabled.IsUnknown() ||
		len(m.Config.OTPs.SMSAutofillMetadata) > 0 {
		c.OTPs = &sdk.ConsumerOTPsConfig{
			SMSLoginOrCreateEnabled:      m.Config.OTPs.SMSLoginOrCreateEnabled.ValueBool(),
			WhatsAppLoginOrCreateEnabled: m.Config.OTPs.WhatsAppLoginOrCreateEnabled.ValueBool(),
			EmailLoginOrCreateEnabled:    m.Config.OTPs.EmailLoginOrCreateEnabled.ValueBool(),
			SMSSendEnabled:               m.Config.OTPs.SMSSendEnabled.ValueBool(),
			WhatsAppSendEnabled:          m.Config.OTPs.WhatsAppSendEnabled.ValueBool(),
			EmailSendEnabled:             m.Config.OTPs.EmailSendEnabled.ValueBool(),
			SMSAutofillMetadata:          make([]sdk.SMSAutofillMetadata, len(m.Config.OTPs.SMSAutofillMetadata)),
		}
		for i, m := range m.Config.OTPs.SMSAutofillMetadata {
			c.OTPs.SMSAutofillMetadata[i] = sdk.SMSAutofillMetadata{
				MetadataType:  m.MetadataType.ValueString(),
				MetadataValue: m.MetadataValue.ValueString(),
				BundleID:      m.BundleID.ValueString(),
			}
		}
	}

	if !m.Config.OAuth.Enabled.IsUnknown() ||
		!m.Config.OAuth.PKCERequired.IsUnknown() {
		c.OAuth = &sdk.ConsumerOAuthConfig{
			Enabled:      m.Config.OAuth.Enabled.ValueBool(),
			PKCERequired: m.Config.OAuth.PKCERequired.ValueBool(),
		}
	}

	if !m.Config.TOTPs.Enabled.IsUnknown() ||
		!m.Config.TOTPs.CreateTOTPs.IsUnknown() {
		c.TOTPs = &sdk.ConsumerTOTPsConfig{
			Enabled:     m.Config.TOTPs.Enabled.ValueBool(),
			CreateTOTPs: m.Config.TOTPs.CreateTOTPs.ValueBool(),
		}
	}

	if !m.Config.WebAuthn.Enabled.IsUnknown() ||
		!m.Config.WebAuthn.CreateWebAuthns.IsUnknown() {
		c.WebAuthn = &sdk.ConsumerWebAuthnConfig{
			Enabled:         m.Config.WebAuthn.Enabled.ValueBool(),
			CreateWebAuthns: m.Config.WebAuthn.CreateWebAuthns.ValueBool(),
		}
	}

	if !m.Config.CryptoWallets.Enabled.IsUnknown() ||
		!m.Config.CryptoWallets.SIWERequired.IsUnknown() {
		c.CryptoWallets = &sdk.ConsumerCryptoWalletsConfig{
			Enabled:      m.Config.CryptoWallets.Enabled.ValueBool(),
			SIWERequired: m.Config.CryptoWallets.SIWERequired.ValueBool(),
		}
	}

	if !m.Config.DFPPA.Enabled.IsUnknown() ||
		!m.Config.DFPPA.OnChallenge.IsUnknown() ||
		!m.Config.DFPPA.LookupTimeoutSeconds.IsUnknown() {
		c.DFPPA = &sdk.ConsumerDFPPAConfig{
			Enabled:              sdk.DFPPASetting(m.Config.DFPPA.Enabled.ValueString()),
			OnChallenge:          sdk.DFPPAOnChallengeAction(m.Config.DFPPA.OnChallenge.ValueString()),
			LookupTimeoutSeconds: m.Config.DFPPA.LookupTimeoutSeconds.ValueInt32(),
		}
	}

	if !m.Config.Biometrics.Enabled.IsUnknown() ||
		!m.Config.Biometrics.CreateBiometricsEnabled.IsUnknown() {
		c.Biometrics = &sdk.ConsumerBiometricsConfig{
			Enabled:                 m.Config.Biometrics.Enabled.ValueBool(),
			CreateBiometricsEnabled: m.Config.Biometrics.CreateBiometricsEnabled.ValueBool(),
		}
	}

	if !m.Config.Passwords.Enabled.IsUnknown() ||
		!m.Config.Passwords.PKCERequiredForPasswordResets.IsUnknown() {
		c.Passwords = &sdk.ConsumerPasswordsConfig{
			Enabled:                       m.Config.Passwords.Enabled.ValueBool(),
			PKCERequiredForPasswordResets: m.Config.Passwords.PKCERequiredForPasswordResets.ValueBool(),
		}
	}

	return c
}

func (m *consumerSDKConfigModel) reloadFromSDKConfig(config sdk.ConsumerConfig) {
	// TODO
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
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"domains": schema.ListAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A list of domains authorized for use in the SDK.",
								ElementType: types.StringType,
								PlanModifiers: []planmodifier.List{
									listplanmodifier.UseStateForUnknown(),
								},
							},
							"bundle_ids": schema.ListAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A list of bundle IDs authorized for use in the SDK.",
								ElementType: types.StringType,
								PlanModifiers: []planmodifier.List{
									listplanmodifier.UseStateForUnknown(),
								},
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
								PlanModifiers: []planmodifier.Int32{
									int32planmodifier.UseStateForUnknown(),
								},
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
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"send_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the magic links send endpoint is enabled in the SDK.",
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
					"otps": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The OTP configuration for the consumer project SDK.",
						Attributes: map[string]schema.Attribute{
							"sms_login_or_create_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the SMS OTP login or create endpoint is enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"whatsapp_login_or_create_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the WhatsApp OTP login or create endpoint is enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"email_login_or_create_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the email OTP login or create endpoint is enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"sms_send_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the SMS OTP send endpoint is enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"whatsapp_send_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the WhatsApp OTP send endpoint is enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"email_send_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether the email OTP send endpoint is enabled in the SDK.",
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
											Description: "The value of the metadata to use for autofill. This should be the associated domain name (for metadata type 'domain')" +
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
						Description: "The TOTP configuration for the consumer project SDK.",
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
					"webauthn": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The WebAuthn configuration for the consumer project SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether WebAuthn endpoints are enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"create_webauthns": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether WebAuthn creation is enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
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
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"siwe_required": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether Sign In With Ethereum is required for Crypto Wallets.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
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
					"biometrics": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The Biometrics configuration for the consumer project SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether biometrics endpoints are enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"create_biometrics_enabled": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether biometrics creation is enabled in the SDK.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
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

func (r consumerSDKConfigResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data b2bSDKConfigModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	getProjectResp, err := r.client.Projects.Get(ctx, projects.GetRequest{
		ProjectID: data.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddWarning("Failed to get project for vertical check", err.Error())
		return
	}
	if getProjectResp.Project.Vertical != projects.VerticalConsumer {
		resp.Diagnostics.AddError("Invalid project vertical", "The project must be a Consumer project for this resource.")
		return
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

	setResp, err := r.client.SDK.SetConsumerConfig(ctx, sdk.SetConsumerConfigRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Config:    plan.toSDKConfig(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set Consumer SDK config", err.Error())
		return
	}

	plan.reloadFromSDKConfig(setResp.Config)
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

	getResp, err := r.client.SDK.GetConsumerConfig(ctx, sdk.GetConsumerConfigRequest{
		ProjectID: state.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get Consumer SDK config", err.Error())
	}

	state.reloadFromSDKConfig(getResp.Config)
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

	setResp, err := r.client.SDK.SetConsumerConfig(ctx, sdk.SetConsumerConfigRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Config:    plan.toSDKConfig(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set Consumer SDK config", err.Error())
		return
	}

	plan.reloadFromSDKConfig(setResp.Config)
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
