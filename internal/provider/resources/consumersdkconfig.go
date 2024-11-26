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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
	Basic         consumerSDKConfigBasicModel `tfsdk:"basic"`
	Sessions      types.Object                `tfsdk:"sessions"`
	MagicLinks    types.Object                `tfsdk:"magic_links"`
	OTPs          types.Object                `tfsdk:"otps"`
	OAuth         types.Object                `tfsdk:"oauth"`
	TOTPs         types.Object                `tfsdk:"totps"`
	WebAuthn      types.Object                `tfsdk:"webauthn"`
	CryptoWallets types.Object                `tfsdk:"crypto_wallets"`
	DFPPA         types.Object                `tfsdk:"dfppa"`
	Biometrics    types.Object                `tfsdk:"biometrics"`
	Passwords     types.Object                `tfsdk:"passwords"`
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

func (m consumerSDKConfigSessionsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"max_session_duration_minutes": types.Int32Type,
	}
}

func consumerSDKConfigSessionsModelFromSDKConfig(c sdk.ConsumerSessionsConfig) consumerSDKConfigSessionsModel {
	return consumerSDKConfigSessionsModel{
		MaxSessionDurationMinutes: types.Int32Value(c.MaxSessionDurationMinutes),
	}
}

type consumerSDKConfigMagicLinksModel struct {
	LoginOrCreateEnabled types.Bool `tfsdk:"login_or_create_enabled"`
	SendEnabled          types.Bool `tfsdk:"send_enabled"`
	PKCERequired         types.Bool `tfsdk:"pkce_required"`
}

func (m consumerSDKConfigMagicLinksModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"login_or_create_enabled": types.BoolType,
		"send_enabled":            types.BoolType,
		"pkce_required":           types.BoolType,
	}
}

func consumerSDKConfigMagicLinksModelFromSDKConfig(c sdk.ConsumerMagicLinksConfig) consumerSDKConfigMagicLinksModel {
	return consumerSDKConfigMagicLinksModel{
		LoginOrCreateEnabled: types.BoolValue(c.LoginOrCreateEnabled),
		SendEnabled:          types.BoolValue(c.SendEnabled),
		PKCERequired:         types.BoolValue(c.PKCERequired),
	}
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

func (m consumerSDKConfigOTPsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"sms_login_or_create_enabled":      types.BoolType,
		"whatsapp_login_or_create_enabled": types.BoolType,
		"email_login_or_create_enabled":    types.BoolType,
		"sms_send_enabled":                 types.BoolType,
		"whatsapp_send_enabled":            types.BoolType,
		"email_send_enabled":               types.BoolType,
		"sms_autofill_metadata": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: sdkSMSAutofillMetadata{}.AttributeTypes(),
			},
		},
	}
}

func consumerSDKConfigOTPsModelFromSDKConfig(c sdk.ConsumerOTPsConfig) consumerSDKConfigOTPsModel {
	metadata := make([]sdkSMSAutofillMetadata, len(c.SMSAutofillMetadata))
	for i, m := range c.SMSAutofillMetadata {
		metadata[i] = sdkSMSAutofillMetadata{
			MetadataType:  types.StringValue(m.MetadataType),
			MetadataValue: types.StringValue(m.MetadataValue),
			BundleID:      types.StringValue(m.BundleID),
		}
	}
	return consumerSDKConfigOTPsModel{
		SMSLoginOrCreateEnabled:      types.BoolValue(c.SMSLoginOrCreateEnabled),
		WhatsAppLoginOrCreateEnabled: types.BoolValue(c.WhatsAppLoginOrCreateEnabled),
		EmailLoginOrCreateEnabled:    types.BoolValue(c.EmailLoginOrCreateEnabled),
		SMSSendEnabled:               types.BoolValue(c.SMSSendEnabled),
		WhatsAppSendEnabled:          types.BoolValue(c.WhatsAppSendEnabled),
		EmailSendEnabled:             types.BoolValue(c.EmailSendEnabled),
		SMSAutofillMetadata:          metadata,
	}
}

type consumerSDKConfigOAuthModel struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	PKCERequired types.Bool `tfsdk:"pkce_required"`
}

func (m consumerSDKConfigOAuthModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":       types.BoolType,
		"pkce_required": types.BoolType,
	}
}

func consumerSDKConfigOAuthModelFromSDKConfig(c sdk.ConsumerOAuthConfig) consumerSDKConfigOAuthModel {
	return consumerSDKConfigOAuthModel{
		Enabled:      types.BoolValue(c.Enabled),
		PKCERequired: types.BoolValue(c.PKCERequired),
	}
}

type consumerSDKConfigTOTPsModel struct {
	Enabled     types.Bool `tfsdk:"enabled"`
	CreateTOTPs types.Bool `tfsdk:"create_totps"`
}

func (m consumerSDKConfigTOTPsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":      types.BoolType,
		"create_totps": types.BoolType,
	}
}

func consumerSDKConfigTOTPsModelFromSDKConfig(c sdk.ConsumerTOTPsConfig) consumerSDKConfigTOTPsModel {
	return consumerSDKConfigTOTPsModel{
		Enabled:     types.BoolValue(c.Enabled),
		CreateTOTPs: types.BoolValue(c.CreateTOTPs),
	}
}

type consumerSDKConfigWebAuthnModel struct {
	Enabled         types.Bool `tfsdk:"enabled"`
	CreateWebAuthns types.Bool `tfsdk:"create_webauthns"`
}

func (m consumerSDKConfigWebAuthnModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":          types.BoolType,
		"create_webauthns": types.BoolType,
	}
}

func consumerSDKConfigWebAuthnModelFromSDKConfig(c sdk.ConsumerWebAuthnConfig) consumerSDKConfigWebAuthnModel {
	return consumerSDKConfigWebAuthnModel{
		Enabled:         types.BoolValue(c.Enabled),
		CreateWebAuthns: types.BoolValue(c.CreateWebAuthns),
	}
}

type consumerSDKConfigCryptoWalletsModel struct {
	Enabled      types.Bool `tfsdk:"enabled"`
	SIWERequired types.Bool `tfsdk:"siwe_required"`
}

func (m consumerSDKConfigCryptoWalletsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":       types.BoolType,
		"siwe_required": types.BoolType,
	}
}

func consumerSDKConfigCryptoWalletsModelFromSDKConfig(c sdk.ConsumerCryptoWalletsConfig) consumerSDKConfigCryptoWalletsModel {
	return consumerSDKConfigCryptoWalletsModel{
		Enabled:      types.BoolValue(c.Enabled),
		SIWERequired: types.BoolValue(c.SIWERequired),
	}
}

type consumerSDKConfigDFPPAModel struct {
	Enabled              types.String `tfsdk:"enabled"`
	OnChallenge          types.String `tfsdk:"on_challenge"`
	LookupTimeoutSeconds types.Int32  `tfsdk:"lookup_timeout_seconds"`
}

func (m consumerSDKConfigDFPPAModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":                types.StringType,
		"on_challenge":           types.StringType,
		"lookup_timeout_seconds": types.Int32Type,
	}
}

func consumerSDKConfigDFPPAModelFromSDKConfig(c sdk.ConsumerDFPPAConfig) consumerSDKConfigDFPPAModel {
	return consumerSDKConfigDFPPAModel{
		Enabled:              types.StringValue(string(c.Enabled)),
		OnChallenge:          types.StringValue(string(c.OnChallenge)),
		LookupTimeoutSeconds: types.Int32Value(c.LookupTimeoutSeconds),
	}
}

type consumerSDKConfigBiometricsModel struct {
	Enabled                 types.Bool `tfsdk:"enabled"`
	CreateBiometricsEnabled types.Bool `tfsdk:"create_biometrics_enabled"`
}

func (m consumerSDKConfigBiometricsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":                   types.BoolType,
		"create_biometrics_enabled": types.BoolType,
	}
}

func consumerSDKConfigBiometricsModelFromSDKConfig(c sdk.ConsumerBiometricsConfig) consumerSDKConfigBiometricsModel {
	return consumerSDKConfigBiometricsModel{
		Enabled:                 types.BoolValue(c.Enabled),
		CreateBiometricsEnabled: types.BoolValue(c.CreateBiometricsEnabled),
	}
}

type consumerSDKConfigPasswordsModel struct {
	Enabled                       types.Bool `tfsdk:"enabled"`
	PKCERequiredForPasswordResets types.Bool `tfsdk:"pkce_required_for_password_resets"`
}

func (m consumerSDKConfigPasswordsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":                           types.BoolType,
		"pkce_required_for_password_resets": types.BoolType,
	}
}

func consumerSDKConfigPasswordsModelFromSDKConfig(c sdk.ConsumerPasswordsConfig) consumerSDKConfigPasswordsModel {
	return consumerSDKConfigPasswordsModel{
		Enabled:                       types.BoolValue(c.Enabled),
		PKCERequiredForPasswordResets: types.BoolValue(c.PKCERequiredForPasswordResets),
	}
}

func (m consumerSDKConfigModel) toSDKConfig(ctx context.Context) (sdk.ConsumerConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
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

	if !m.Config.Sessions.IsUnknown() {
		var sessions consumerSDKConfigSessionsModel
		diags.Append(m.Config.Sessions.As(ctx, &sessions, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.Sessions = &sdk.ConsumerSessionsConfig{
			MaxSessionDurationMinutes: sessions.MaxSessionDurationMinutes.ValueInt32(),
		}
	}

	if !m.Config.MagicLinks.IsUnknown() {
		var magicLinks consumerSDKConfigMagicLinksModel
		diags.Append(m.Config.MagicLinks.As(ctx, &magicLinks, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.MagicLinks = &sdk.ConsumerMagicLinksConfig{
			LoginOrCreateEnabled: magicLinks.LoginOrCreateEnabled.ValueBool(),
			SendEnabled:          magicLinks.SendEnabled.ValueBool(),
			PKCERequired:         magicLinks.PKCERequired.ValueBool(),
		}
	}

	if !m.Config.OTPs.IsUnknown() {
		var otps consumerSDKConfigOTPsModel
		diags.Append(m.Config.OTPs.As(ctx, &otps, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.OTPs = &sdk.ConsumerOTPsConfig{
			SMSLoginOrCreateEnabled:      otps.SMSLoginOrCreateEnabled.ValueBool(),
			WhatsAppLoginOrCreateEnabled: otps.WhatsAppLoginOrCreateEnabled.ValueBool(),
			EmailLoginOrCreateEnabled:    otps.EmailLoginOrCreateEnabled.ValueBool(),
			SMSSendEnabled:               otps.SMSSendEnabled.ValueBool(),
			WhatsAppSendEnabled:          otps.WhatsAppSendEnabled.ValueBool(),
			EmailSendEnabled:             otps.EmailSendEnabled.ValueBool(),
			SMSAutofillMetadata:          make([]sdk.SMSAutofillMetadata, len(otps.SMSAutofillMetadata)),
		}
		for i, m := range otps.SMSAutofillMetadata {
			c.OTPs.SMSAutofillMetadata[i] = sdk.SMSAutofillMetadata{
				MetadataType:  m.MetadataType.ValueString(),
				MetadataValue: m.MetadataValue.ValueString(),
				BundleID:      m.BundleID.ValueString(),
			}
		}
	}

	if !m.Config.OAuth.IsUnknown() {
		var oauth consumerSDKConfigOAuthModel
		diags.Append(m.Config.OAuth.As(ctx, &oauth, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.OAuth = &sdk.ConsumerOAuthConfig{
			Enabled:      oauth.Enabled.ValueBool(),
			PKCERequired: oauth.PKCERequired.ValueBool(),
		}
	}

	if !m.Config.TOTPs.IsUnknown() {
		var totps consumerSDKConfigTOTPsModel
		diags.Append(m.Config.TOTPs.As(ctx, &totps, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.TOTPs = &sdk.ConsumerTOTPsConfig{
			Enabled:     totps.Enabled.ValueBool(),
			CreateTOTPs: totps.CreateTOTPs.ValueBool(),
		}
	}

	if !m.Config.WebAuthn.IsUnknown() {
		var webAuthn consumerSDKConfigWebAuthnModel
		diags.Append(m.Config.WebAuthn.As(ctx, &webAuthn, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.WebAuthn = &sdk.ConsumerWebAuthnConfig{
			Enabled:         webAuthn.Enabled.ValueBool(),
			CreateWebAuthns: webAuthn.CreateWebAuthns.ValueBool(),
		}
	}

	if !m.Config.CryptoWallets.IsUnknown() {
		var cryptoWallets consumerSDKConfigCryptoWalletsModel
		diags.Append(m.Config.CryptoWallets.As(ctx, &cryptoWallets, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.CryptoWallets = &sdk.ConsumerCryptoWalletsConfig{
			Enabled:      cryptoWallets.Enabled.ValueBool(),
			SIWERequired: cryptoWallets.SIWERequired.ValueBool(),
		}
	}

	if !m.Config.DFPPA.IsUnknown() {
		var dfppa consumerSDKConfigDFPPAModel
		diags.Append(m.Config.DFPPA.As(ctx, &dfppa, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.DFPPA = &sdk.ConsumerDFPPAConfig{
			Enabled:              sdk.DFPPASetting(dfppa.Enabled.ValueString()),
			OnChallenge:          sdk.DFPPAOnChallengeAction(dfppa.OnChallenge.ValueString()),
			LookupTimeoutSeconds: dfppa.LookupTimeoutSeconds.ValueInt32(),
		}
	}

	if !m.Config.Biometrics.IsUnknown() {
		var biometrics consumerSDKConfigBiometricsModel
		diags.Append(m.Config.Biometrics.As(ctx, &biometrics, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.Biometrics = &sdk.ConsumerBiometricsConfig{
			Enabled:                 biometrics.Enabled.ValueBool(),
			CreateBiometricsEnabled: biometrics.CreateBiometricsEnabled.ValueBool(),
		}
	}

	if !m.Config.Passwords.IsUnknown() {
		var passwords consumerSDKConfigPasswordsModel
		diags.Append(m.Config.Passwords.As(ctx, &passwords, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.Passwords = &sdk.ConsumerPasswordsConfig{
			Enabled:                       passwords.Enabled.ValueBool(),
			PKCERequiredForPasswordResets: passwords.PKCERequiredForPasswordResets.ValueBool(),
		}
	}

	return c, diags
}

func (m *consumerSDKConfigModel) reloadFromSDKConfig(ctx context.Context, c sdk.ConsumerConfig) diag.Diagnostics {
	var diags diag.Diagnostics

	if c.Sessions == nil {
		diags.AddError("sessions is nil", nilSDKObject)
	}
	if c.MagicLinks == nil {
		diags.AddError("magic_links is nil", nilSDKObject)
	}
	if c.OTPs == nil {
		diags.AddError("otps is nil", nilSDKObject)
	}
	if c.OAuth == nil {
		diags.AddError("oauth is nil", nilSDKObject)
	}
	if c.TOTPs == nil {
		diags.AddError("totps is nil", nilSDKObject)
	}
	if c.WebAuthn == nil {
		diags.AddError("webauthn is nil", nilSDKObject)
	}
	if c.CryptoWallets == nil {
		diags.AddError("crypto_wallets is nil", nilSDKObject)
	}
	if c.DFPPA == nil {
		diags.AddError("dfppa is nil", nilSDKObject)
	}
	if c.Biometrics == nil {
		diags.AddError("biometrics is nil", nilSDKObject)
	}
	if c.Passwords == nil {
		diags.AddError("passwords is nil", nilSDKObject)
	}

	if diags.HasError() {
		return diags
	}

	sessions, diag := types.ObjectValueFrom(ctx, consumerSDKConfigSessionsModel{}.AttributeTypes(), consumerSDKConfigSessionsModelFromSDKConfig(*c.Sessions))
	diags.Append(diag...)

	magicLinks, diag := types.ObjectValueFrom(ctx, consumerSDKConfigMagicLinksModel{}.AttributeTypes(), consumerSDKConfigMagicLinksModelFromSDKConfig(*c.MagicLinks))
	diags.Append(diag...)

	otps, diag := types.ObjectValueFrom(ctx, consumerSDKConfigOTPsModel{}.AttributeTypes(), consumerSDKConfigOTPsModelFromSDKConfig(*c.OTPs))
	diags.Append(diag...)

	oauth, diag := types.ObjectValueFrom(ctx, consumerSDKConfigOAuthModel{}.AttributeTypes(), consumerSDKConfigOAuthModelFromSDKConfig(*c.OAuth))
	diags.Append(diag...)

	totps, diag := types.ObjectValueFrom(ctx, consumerSDKConfigTOTPsModel{}.AttributeTypes(), consumerSDKConfigTOTPsModelFromSDKConfig(*c.TOTPs))
	diags.Append(diag...)

	webAuthn, diag := types.ObjectValueFrom(ctx, consumerSDKConfigWebAuthnModel{}.AttributeTypes(), consumerSDKConfigWebAuthnModelFromSDKConfig(*c.WebAuthn))
	diags.Append(diag...)

	cryptoWallets, diag := types.ObjectValueFrom(ctx, consumerSDKConfigCryptoWalletsModel{}.AttributeTypes(), consumerSDKConfigCryptoWalletsModelFromSDKConfig(*c.CryptoWallets))
	diags.Append(diag...)

	dfppa, diag := types.ObjectValueFrom(ctx, consumerSDKConfigDFPPAModel{}.AttributeTypes(), consumerSDKConfigDFPPAModelFromSDKConfig(*c.DFPPA))
	diags.Append(diag...)

	biometrics, diag := types.ObjectValueFrom(ctx, consumerSDKConfigBiometricsModel{}.AttributeTypes(), consumerSDKConfigBiometricsModelFromSDKConfig(*c.Biometrics))
	diags.Append(diag...)

	passwords, diag := types.ObjectValueFrom(ctx, consumerSDKConfigPasswordsModel{}.AttributeTypes(), consumerSDKConfigPasswordsModelFromSDKConfig(*c.Passwords))
	diags.Append(diag...)

	cfg := consumerSDKConfigInnerModel{
		Basic: consumerSDKConfigBasicModel{
			Enabled:        types.BoolValue(c.Basic.Enabled),
			CreateNewUsers: types.BoolValue(c.Basic.CreateNewUsers),
			Domains:        make([]types.String, len(c.Basic.Domains)),
			BundleIDs:      make([]types.String, len(c.Basic.BundleIDs)),
		},
		Sessions:      sessions,
		MagicLinks:    magicLinks,
		OTPs:          otps,
		OAuth:         oauth,
		TOTPs:         totps,
		WebAuthn:      webAuthn,
		CryptoWallets: cryptoWallets,
		DFPPA:         dfppa,
		Biometrics:    biometrics,
		Passwords:     passwords,
	}
	for i, d := range c.Basic.Domains {
		cfg.Basic.Domains[i] = types.StringValue(d)
	}
	for i, b := range c.Basic.BundleIDs {
		cfg.Basic.BundleIDs[i] = types.StringValue(b)
	}
	m.Config = cfg
	return diags
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

func (r consumerSDKConfigResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data consumerSDKConfigModel
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

	cfg, diags := plan.toSDKConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.SDK.SetConsumerConfig(ctx, sdk.SetConsumerConfigRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Config:    cfg,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set Consumer SDK config", err.Error())
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

	diags = state.reloadFromSDKConfig(ctx, getResp.Config)
	resp.Diagnostics.Append(diags...)
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

	cfg, diags := plan.toSDKConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setResp, err := r.client.SDK.SetConsumerConfig(ctx, sdk.SetConsumerConfigRequest{
		ProjectID: plan.ProjectID.ValueString(),
		Config:    cfg,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to set Consumer SDK config", err.Error())
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
