package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v2/pkg/api"
	"github.com/stytchauth/stytch-management-go/v2/pkg/models/projects"
	"github.com/stytchauth/stytch-management-go/v2/pkg/models/sdk"
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
	ID          types.String `tfsdk:"id"`
	ProjectID   types.String `tfsdk:"project_id"`
	LastUpdated types.String `tfsdk:"last_updated"`
	// A pointer is required here for ImportState to work since the initial import
	// will set a nil value until Read is called.
	Config *b2bSDKConfigInnerModel `tfsdk:"config"`
}

type b2bSDKConfigInnerModel struct {
	Basic      b2bSDKConfigBasicModel `tfsdk:"basic"`
	Sessions   types.Object           `tfsdk:"sessions"`
	MagicLinks types.Object           `tfsdk:"magic_links"`
	OAuth      types.Object           `tfsdk:"oauth"`
	TOTPs      types.Object           `tfsdk:"totps"`
	SSO        types.Object           `tfsdk:"sso"`
	OTPs       types.Object           `tfsdk:"otps"`
	DFPPA      types.Object           `tfsdk:"dfppa"`
	Passwords  types.Object           `tfsdk:"passwords"`
	Cookies    types.Object           `tfsdk:"cookies"`
}

type b2bSDKConfigBasicModel struct {
	Enabled                 types.Bool `tfsdk:"enabled"`
	CreateNewMembers        types.Bool `tfsdk:"create_new_members"`
	AllowSelfOnboarding     types.Bool `tfsdk:"allow_self_onboarding"`
	EnableMemberPermissions types.Bool `tfsdk:"enable_member_permissions"`
	Domains                 types.Set  `tfsdk:"domains"`
	BundleIDs               types.Set  `tfsdk:"bundle_ids"`
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

func b2bSDKConfigAuthorizedDomainModelFromSDKConfig(d []sdk.AuthorizedB2BDomain) []b2bSDKConfigAuthorizedDomainModel {
	m := make([]b2bSDKConfigAuthorizedDomainModel, len(d))
	for i, d := range d {
		m[i] = b2bSDKConfigAuthorizedDomainModel{
			Domain:      types.StringValue(d.Domain),
			SlugPattern: types.StringValue(d.SlugPattern),
		}
	}
	return m
}

type b2bSDKConfigSessionsModel struct {
	MaxSessionDurationMinutes types.Int32 `tfsdk:"max_session_duration_minutes"`
}

func (m b2bSDKConfigSessionsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"max_session_duration_minutes": types.Int32Type,
	}
}

func b2bSDKConfigSessionsModelFromSDKConfig(c sdk.B2BSessionsConfig) b2bSDKConfigSessionsModel {
	return b2bSDKConfigSessionsModel{
		MaxSessionDurationMinutes: types.Int32Value(c.MaxSessionDurationMinutes),
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

func b2bSDKConfigMagicLinksModelFromSDKConfig(c sdk.B2BMagicLinksConfig) b2bSDKConfigMagicLinksModel {
	return b2bSDKConfigMagicLinksModel{
		Enabled:      types.BoolValue(c.Enabled),
		PKCERequired: types.BoolValue(c.PKCERequired),
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

func b2bSDKConfigOAuthModelFromSDKConfig(c sdk.B2BOAuthConfig) b2bSDKConfigOAuthModel {
	return b2bSDKConfigOAuthModel{
		Enabled:      types.BoolValue(c.Enabled),
		PKCERequired: types.BoolValue(c.PKCERequired),
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

func b2bSDKConfigTOTPsModelFromSDKConfig(c sdk.B2BTOTPsConfig) b2bSDKConfigTOTPsModel {
	return b2bSDKConfigTOTPsModel{
		Enabled:     types.BoolValue(c.Enabled),
		CreateTOTPs: types.BoolValue(c.CreateTOTPs),
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

func b2bSDKConfigSSOModelFromSDKConfig(c sdk.B2BSSOConfig) b2bSDKConfigSSOModel {
	return b2bSDKConfigSSOModel{
		Enabled:      types.BoolValue(c.Enabled),
		PKCERequired: types.BoolValue(c.PKCERequired),
	}
}

type b2bSDKConfigOTPsModel struct {
	SMSEnabled          types.Bool `tfsdk:"sms_enabled"`
	SMSAutofillMetadata types.Set  `tfsdk:"sms_autofill_metadata"`
	EmailEnabled        types.Bool `tfsdk:"email_enabled"`
}

func (m b2bSDKConfigOTPsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"sms_enabled": types.BoolType,
		"sms_autofill_metadata": types.SetType{
			ElemType: types.ObjectType{
				AttrTypes: sdkSMSAutofillMetadata{}.AttributeTypes(),
			},
		},
		"email_enabled": types.BoolType,
	}
}

func b2bSDKConfigOTPsModelFromSDKConfig(ctx context.Context, c sdk.B2BOTPsConfig) (b2bSDKConfigOTPsModel, diag.Diagnostics) {
	metadata := make([]sdkSMSAutofillMetadata, len(c.SMSAutofillMetadata))
	for i, m := range c.SMSAutofillMetadata {
		metadata[i] = sdkSMSAutofillMetadata{
			MetadataType:  types.StringValue(m.MetadataType),
			MetadataValue: types.StringValue(m.MetadataValue),
			BundleID:      types.StringValue(m.BundleID),
		}
	}

	autofillMetadata, diag := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: sdkSMSAutofillMetadata{}.AttributeTypes()}, metadata)
	return b2bSDKConfigOTPsModel{
		SMSEnabled:          types.BoolValue(c.SMSEnabled),
		SMSAutofillMetadata: autofillMetadata,
		EmailEnabled:        types.BoolValue(c.EmailEnabled),
	}, diag
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

func b2bSDKConfigDFPPAModelFromSDKConfig(c sdk.B2BDFPPAConfig) b2bSDKConfigDFPPAModel {
	return b2bSDKConfigDFPPAModel{
		Enabled:              types.StringValue(string(c.Enabled)),
		OnChallenge:          types.StringValue(string(c.OnChallenge)),
		LookupTimeoutSeconds: types.Int32Value(c.LookupTimeoutSeconds),
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

func b2bSDKConfigPasswordsModelFromSDKConfig(c sdk.B2BPasswordsConfig) b2bSDKConfigPasswordsModel {
	return b2bSDKConfigPasswordsModel{
		Enabled:                       types.BoolValue(c.Enabled),
		PKCERequiredForPasswordResets: types.BoolValue(c.PKCERequiredForPasswordResets),
	}
}

type b2bSDKConfigCookiesModel struct {
	HttpOnlyCookies types.String `tfsdk:"http_only"`
}

func (m b2bSDKConfigCookiesModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"http_only": types.StringType,
	}
}

func b2bSDKConfigCookiesModelFromSDKConfig(c sdk.B2BCookiesConfig) b2bSDKConfigCookiesModel {
	return b2bSDKConfigCookiesModel{
		HttpOnlyCookies: types.StringValue(string(c.HttpOnlyCookies)),
	}
}

func (m b2bSDKConfigModel) toSDKConfig(ctx context.Context) (sdk.B2BConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	c := sdk.B2BConfig{
		Basic: &sdk.B2BBasicConfig{
			Enabled:                 m.Config.Basic.Enabled.ValueBool(),
			CreateNewMembers:        m.Config.Basic.CreateNewMembers.ValueBool(),
			AllowSelfOnboarding:     m.Config.Basic.AllowSelfOnboarding.ValueBool(),
			EnableMemberPermissions: m.Config.Basic.EnableMemberPermissions.ValueBool(),
		},
	}
	if !m.Config.Basic.Domains.IsUnknown() {
		var domains []b2bSDKConfigAuthorizedDomainModel
		diags.Append(m.Config.Basic.Domains.ElementsAs(ctx, &domains, true)...)
		c.Basic.Domains = make([]sdk.AuthorizedB2BDomain, len(domains))
		for i, d := range domains {
			c.Basic.Domains[i] = sdk.AuthorizedB2BDomain{
				Domain:      d.Domain.ValueString(),
				SlugPattern: d.SlugPattern.ValueString(),
			}
		}
	}

	if !m.Config.Basic.BundleIDs.IsUnknown() {
		var bundleIDs []string
		diags.Append(m.Config.Basic.BundleIDs.ElementsAs(ctx, &bundleIDs, true)...)
		c.Basic.BundleIDs = bundleIDs
	}

	if !m.Config.Sessions.IsUnknown() {
		var sessions b2bSDKConfigSessionsModel
		diags.Append(m.Config.Sessions.As(ctx, &sessions, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.Sessions = &sdk.B2BSessionsConfig{
			MaxSessionDurationMinutes: sessions.MaxSessionDurationMinutes.ValueInt32(),
		}
	}

	if !m.Config.MagicLinks.IsUnknown() {
		var magicLinks b2bSDKConfigMagicLinksModel
		diags.Append(m.Config.MagicLinks.As(ctx, &magicLinks, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.MagicLinks = &sdk.B2BMagicLinksConfig{
			Enabled:      magicLinks.Enabled.ValueBool(),
			PKCERequired: magicLinks.PKCERequired.ValueBool(),
		}
	}

	if !m.Config.OAuth.IsUnknown() {
		var oAuth b2bSDKConfigOAuthModel
		diags.Append(m.Config.OAuth.As(ctx, &oAuth, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.OAuth = &sdk.B2BOAuthConfig{
			Enabled:      oAuth.Enabled.ValueBool(),
			PKCERequired: oAuth.PKCERequired.ValueBool(),
		}
	}

	if !m.Config.TOTPs.IsUnknown() {
		var totps b2bSDKConfigTOTPsModel
		diags.Append(m.Config.TOTPs.As(ctx, &totps, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.TOTPs = &sdk.B2BTOTPsConfig{
			Enabled:     totps.Enabled.ValueBool(),
			CreateTOTPs: totps.CreateTOTPs.ValueBool(),
		}
	}

	if !m.Config.SSO.IsUnknown() {
		var sso b2bSDKConfigSSOModel
		diags.Append(m.Config.SSO.As(ctx, &sso, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.SSO = &sdk.B2BSSOConfig{
			Enabled:      sso.Enabled.ValueBool(),
			PKCERequired: sso.PKCERequired.ValueBool(),
		}
	}

	if !m.Config.OTPs.IsUnknown() {
		var otps b2bSDKConfigOTPsModel
		diags.Append(m.Config.OTPs.As(ctx, &otps, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)

		var smsAutofillMetadata []sdkSMSAutofillMetadata
		diags.Append(otps.SMSAutofillMetadata.ElementsAs(ctx, &smsAutofillMetadata, true)...)
		c.OTPs = &sdk.B2BOTPsConfig{
			SMSEnabled:          otps.SMSEnabled.ValueBool(),
			EmailEnabled:        otps.EmailEnabled.ValueBool(),
			SMSAutofillMetadata: make([]sdk.SMSAutofillMetadata, len(smsAutofillMetadata)),
		}
		for i, m := range smsAutofillMetadata {
			c.OTPs.SMSAutofillMetadata[i] = sdk.SMSAutofillMetadata{
				MetadataType:  m.MetadataType.ValueString(),
				MetadataValue: m.MetadataValue.ValueString(),
				BundleID:      m.BundleID.ValueString(),
			}
		}
	}

	if !m.Config.DFPPA.IsUnknown() {
		var dfppa b2bSDKConfigDFPPAModel
		diags.Append(m.Config.DFPPA.As(ctx, &dfppa, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.DFPPA = &sdk.B2BDFPPAConfig{
			Enabled:              sdk.DFPPASetting(dfppa.Enabled.ValueString()),
			OnChallenge:          sdk.DFPPAOnChallengeAction(dfppa.OnChallenge.ValueString()),
			LookupTimeoutSeconds: dfppa.LookupTimeoutSeconds.ValueInt32(),
		}
	}

	if !m.Config.Passwords.IsUnknown() {
		var passwords b2bSDKConfigPasswordsModel
		diags.Append(m.Config.Passwords.As(ctx, &passwords, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.Passwords = &sdk.B2BPasswordsConfig{
			Enabled:                       passwords.Enabled.ValueBool(),
			PKCERequiredForPasswordResets: passwords.PKCERequiredForPasswordResets.ValueBool(),
		}
	}

	if !m.Config.Cookies.IsUnknown() {
		var cookies b2bSDKConfigCookiesModel
		diags.Append(m.Config.Cookies.As(ctx, &cookies, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		c.Cookies = &sdk.B2BCookiesConfig{
			HttpOnlyCookies: sdk.HttpOnlyCookiesSetting(cookies.HttpOnlyCookies.ValueString()),
		}
	}

	return c, diags
}

func (m *b2bSDKConfigModel) reloadFromSDKConfig(ctx context.Context, c sdk.B2BConfig) diag.Diagnostics {
	var diags diag.Diagnostics

	if c.Sessions == nil {
		diags.AddError("sessions is nil", nilSDKObject)
	}
	if c.MagicLinks == nil {
		diags.AddError("magic_links is nil", nilSDKObject)
	}
	if c.OAuth == nil {
		diags.AddError("oauth is nil", nilSDKObject)
	}
	if c.TOTPs == nil {
		diags.AddError("totps is nil", nilSDKObject)
	}
	if c.SSO == nil {
		diags.AddError("sso is nil", nilSDKObject)
	}
	if c.OTPs == nil {
		diags.AddError("otps is nil", nilSDKObject)
	}
	if c.DFPPA == nil {
		diags.AddError("dfppa is nil", nilSDKObject)
	}
	if c.Passwords == nil {
		diags.AddError("passwords is nil", nilSDKObject)
	}
	if c.Cookies == nil {
		diags.AddError("cookies is nil", nilSDKObject)
	}

	if diags.HasError() {
		return diags
	}

	domains, diag := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: b2bSDKConfigAuthorizedDomainModel{}.AttributeTypes()}, b2bSDKConfigAuthorizedDomainModelFromSDKConfig(c.Basic.Domains))
	diags.Append(diag...)

	bundleIDs, diag := types.SetValueFrom(ctx, types.StringType, c.Basic.BundleIDs)
	diags.Append(diag...)

	sessions, diag := types.ObjectValueFrom(ctx, b2bSDKConfigSessionsModel{}.AttributeTypes(), b2bSDKConfigSessionsModelFromSDKConfig(*c.Sessions))
	diags.Append(diag...)

	magicLinks, diag := types.ObjectValueFrom(ctx, b2bSDKConfigMagicLinksModel{}.AttributeTypes(), b2bSDKConfigMagicLinksModelFromSDKConfig(*c.MagicLinks))
	diags.Append(diag...)

	oauth, diag := types.ObjectValueFrom(ctx, b2bSDKConfigOAuthModel{}.AttributeTypes(), b2bSDKConfigOAuthModelFromSDKConfig(*c.OAuth))
	diags.Append(diag...)

	totps, diag := types.ObjectValueFrom(ctx, b2bSDKConfigTOTPsModel{}.AttributeTypes(), b2bSDKConfigTOTPsModelFromSDKConfig(*c.TOTPs))
	diags.Append(diag...)

	sso, diag := types.ObjectValueFrom(ctx, b2bSDKConfigSSOModel{}.AttributeTypes(), b2bSDKConfigSSOModelFromSDKConfig(*c.SSO))
	diags.Append(diag...)

	otpModel, diag := b2bSDKConfigOTPsModelFromSDKConfig(ctx, *c.OTPs)
	diags.Append(diag...)
	otps, diag := types.ObjectValueFrom(ctx, b2bSDKConfigOTPsModel{}.AttributeTypes(), otpModel)
	diags.Append(diag...)

	dfppa, diag := types.ObjectValueFrom(ctx, b2bSDKConfigDFPPAModel{}.AttributeTypes(), b2bSDKConfigDFPPAModelFromSDKConfig(*c.DFPPA))
	diags.Append(diag...)

	passwords, diag := types.ObjectValueFrom(ctx, b2bSDKConfigPasswordsModel{}.AttributeTypes(), b2bSDKConfigPasswordsModelFromSDKConfig(*c.Passwords))
	diags.Append(diag...)

	cookies, diag := types.ObjectValueFrom(ctx, b2bSDKConfigCookiesModel{}.AttributeTypes(), b2bSDKConfigCookiesModelFromSDKConfig(*c.Cookies))
	diags.Append(diag...)

	cfg := b2bSDKConfigInnerModel{
		Basic: b2bSDKConfigBasicModel{
			Enabled:                 types.BoolValue(c.Basic.Enabled),
			CreateNewMembers:        types.BoolValue(c.Basic.CreateNewMembers),
			AllowSelfOnboarding:     types.BoolValue(c.Basic.AllowSelfOnboarding),
			EnableMemberPermissions: types.BoolValue(c.Basic.EnableMemberPermissions),
			Domains:                 domains,
			BundleIDs:               bundleIDs,
		},
		Sessions:   sessions,
		MagicLinks: magicLinks,
		OAuth:      oauth,
		TOTPs:      totps,
		SSO:        sso,
		OTPs:       otps,
		DFPPA:      dfppa,
		Passwords:  passwords,
		Cookies:    cookies,
	}
	m.ID = m.ProjectID
	m.Config = &cfg
	return diags
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
		Description: "Manages the configuration of your JavaScript, React Native, iOS, or Android SDKs for a B2B project",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.StringAttribute{
				Required: true,
				Description: "The ID of the B2B project for which to set the SDK config. " +
					"This can be either a live project ID or test project ID. " +
					"You may only specify one SDK config per project.",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the order.",
				Computed:    true,
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
								Optional:           true,
								Computed:           true,
								Description:        "A boolean indicating whether new members can be created with the SDK.",
								DeprecationMessage: "This field is deprecated and no longer affects SDK functionality. It will be removed in a future major release.",
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
							"domains": schema.SetNestedAttribute{
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
							"bundle_ids": schema.SetAttribute{
								Optional:    true,
								Computed:    true,
								ElementType: types.StringType,
								Description: "A list of bundle IDs authorized for use in the SDK.",
								PlanModifiers: []planmodifier.Set{
									setplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
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
							"sms_autofill_metadata": schema.SetNestedAttribute{
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
											Validators: []validator.String{
												stringvalidator.OneOf("domain", "hash"),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"dfppa": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The Device Fingerprinting Protected Auth configuration for the B2B project SDK.",
						Attributes: map[string]schema.Attribute{
							"enabled": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Description: "A boolean indicating whether Device Fingerprinting Protected Auth is enabled in the SDK.",
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
								Validators: []validator.String{
									stringvalidator.OneOf(toStrings(sdk.DFPPASettings())...),
								},
							},
							"on_challenge": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Description: "The action to take when a DFPPA 'challenge' verdict is returned.",
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
								Validators: []validator.String{
									stringvalidator.OneOf(toStrings(sdk.DFPPAOnChallengeActions())...),
								},
							},
							"lookup_timeout_seconds": schema.Int32Attribute{
								Optional:           true,
								Computed:           true,
								Description:        "How long to wait for a DFPPA lookup to complete before timing out.",
								DeprecationMessage: "This field is deprecated and no longer affects SDK functionality. It will be removed in a future major release.",
								PlanModifiers: []planmodifier.Int32{
									int32planmodifier.UseStateForUnknown(),
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
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
					},
					"cookies": schema.SingleNestedAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The Cookies configuration for the B2B project SDK.",
						Attributes: map[string]schema.Attribute{
							"http_only": schema.StringAttribute{
								Optional: true,
								Computed: true,
								Description: "Whether cookies should be set with the HttpOnly flag. HttpOnly cookies can only be set when the frontend SDK is " +
									"configured to use a custom authentication domain. Set to 'DISABLED' to disable, 'ENABLED' to enable, or " +
									"'ENFORCED' to enable and block web requests that don't use a custom authentication domain.",
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
								Validators: []validator.String{
									stringvalidator.OneOf(toStrings(sdk.HttpOnlyCookiesSettings())...),
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

	ctx = tflog.SetField(ctx, "project_id", data.ProjectID.ValueString())
	tflog.Info(ctx, "Validating B2B SDK config")

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

	tflog.Info(ctx, "B2B SDK config validated")
}

// Create creates the resource and sets the initial Terraform state.
func (r *b2bSDKConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan b2bSDKConfigModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	tflog.Info(ctx, "Creating B2B SDK config")

	cfg, diag := plan.toSDKConfig(ctx)
	resp.Diagnostics.Append(diag...)
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

	tflog.Info(ctx, "B2B SDK config created")

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

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	tflog.Info(ctx, "Reading B2B SDK config")

	getResp, err := r.client.SDK.GetB2BConfig(ctx, sdk.GetB2BConfigRequest{
		ProjectID: state.ProjectID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get B2B SDK config", err.Error())
		return
	}

	tflog.Info(ctx, "B2B SDK config read")

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

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	tflog.Info(ctx, "Updating B2B SDK config")

	cfg, diag := plan.toSDKConfig(ctx)
	resp.Diagnostics.Append(diag...)
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

	tflog.Info(ctx, "B2B SDK config updated")

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

	tflog.Info(ctx, "Deleting B2B SDK config")

	// To delete the SDK config, we set the basic config to disabled. Other fields are left as-is. Although this isn't
	// *perfect*, it's the best we can do since we don't want to define various "defaults" in different repos since they
	// would likely drift apart over time.
	// NOTE: This does cause a bit of weirdness if someone previously set some field like Sessions.MaxSessionDurationMinutes,
	// then deleted the entire config, then *recreated* it and wanted to rely on the "default" session duration value. Since
	// this value isn't "officially" endorsed anywhere, we assume that the provisioner not specifying it means they're fine
	// with any acceptable value.
	_, err := r.client.SDK.SetB2BConfig(ctx, sdk.SetB2BConfigRequest{
		ProjectID: state.ProjectID.ValueString(),
		Config: sdk.B2BConfig{
			Basic: &sdk.B2BBasicConfig{
				Enabled: false,
			},
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset B2B SDK config", err.Error())
		return
	}

	tflog.Info(ctx, "B2B SDK config deleted")
}

func (r *b2bSDKConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx = tflog.SetField(ctx, "project_id", req.ID)
	tflog.Info(ctx, "Importing B2B SDK config")
	resource.ImportStatePassthroughID(ctx, path.Root("project_id"), req, resp)
}
