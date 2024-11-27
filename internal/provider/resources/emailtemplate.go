package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/pkg/api"
	"github.com/stytchauth/stytch-management-go/pkg/models/emailtemplates"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &emailTemplateResource{}
	_ resource.ResourceWithConfigure   = &emailTemplateResource{}
	_ resource.ResourceWithImportState = &emailTemplateResource{}
)

func NewEmailTemplateResource() resource.Resource {
	return &emailTemplateResource{}
}

type emailTemplateResource struct {
	client *api.API
}

type emailTemplateModel struct {
	ID                      types.String `tfsdk:"id"`
	LiveProjectID           types.String `tfsdk:"live_project_id"`
	LastUpdated             types.String `tfsdk:"last_updated"`
	TemplateID              types.String `tfsdk:"template_id"`
	Name                    types.String `tfsdk:"name"`
	SenderInformation       types.Object `tfsdk:"sender_information"`
	PrebuiltCustomization   types.Object `tfsdk:"prebuilt_customization"`
	CustomHTMLCustomization types.Object `tfsdk:"custom_html_customization"`
}

type emailTemplateSenderInformationModel struct {
	FromLocalPart    types.String `tfsdk:"from_local_part"`
	FromDomain       types.String `tfsdk:"from_domain"`
	FromName         types.String `tfsdk:"from_name"`
	ReplyToLocalPart types.String `tfsdk:"reply_to_local_part"`
	ReplyToName      types.String `tfsdk:"reply_to_name"`
}

func (m emailTemplateSenderInformationModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"from_local_part":     types.StringType,
		"from_domain":         types.StringType,
		"from_name":           types.StringType,
		"reply_to_local_part": types.StringType,
		"reply_to_name":       types.StringType,
	}
}

func emailTemplateSenderInformationModelFromEmailTemplate(e emailtemplates.EmailTemplate) emailTemplateSenderInformationModel {
	var senderInformation emailTemplateSenderInformationModel
	if e.SenderInformation != nil {
		if e.SenderInformation.FromLocalPart != nil && *e.SenderInformation.FromLocalPart != "" {
			senderInformation.FromLocalPart = types.StringValue(*e.SenderInformation.FromLocalPart)
		}
		if e.SenderInformation.FromDomain != nil && *e.SenderInformation.FromDomain != "" {
			senderInformation.FromDomain = types.StringValue(*e.SenderInformation.FromDomain)
		}
		if e.SenderInformation.FromName != nil && *e.SenderInformation.FromName != "" {
			senderInformation.FromName = types.StringValue(*e.SenderInformation.FromName)
		}
		if e.SenderInformation.ReplyToLocalPart != nil && *e.SenderInformation.ReplyToLocalPart != "" {
			senderInformation.ReplyToLocalPart = types.StringValue(*e.SenderInformation.ReplyToLocalPart)
		}
		if e.SenderInformation.ReplyToName != nil && *e.SenderInformation.ReplyToName != "" {
			senderInformation.ReplyToName = types.StringValue(*e.SenderInformation.ReplyToName)
		}
	}
	return senderInformation
}

type emailTemplatePrebuiltCustomizationModel struct {
	ButtonBorderRadius types.Float32 `tfsdk:"button_border_radius"`
	ButtonColor        types.String  `tfsdk:"button_color"`
	ButtonTextColor    types.String  `tfsdk:"button_text_color"`
	FontFamily         types.String  `tfsdk:"font_family"`
	TextAlignment      types.String  `tfsdk:"text_alignment"`
}

func (m emailTemplatePrebuiltCustomizationModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"button_border_radius": types.Float32Type,
		"button_color":         types.StringType,
		"button_text_color":    types.StringType,
		"font_family":          types.StringType,
		"text_alignment":       types.StringType,
	}
}

func emailTemplatePrebuiltCustomizationModelFromEmailTemplate(e emailtemplates.EmailTemplate) emailTemplatePrebuiltCustomizationModel {
	var prebuiltCustomization emailTemplatePrebuiltCustomizationModel
	if e.PrebuiltCustomization != nil {
		if e.PrebuiltCustomization.ButtonBorderRadius != nil {
			prebuiltCustomization.ButtonBorderRadius = types.Float32Value(*e.PrebuiltCustomization.ButtonBorderRadius)
		}
		if e.PrebuiltCustomization.ButtonColor != nil {
			prebuiltCustomization.ButtonColor = types.StringValue(*e.PrebuiltCustomization.ButtonColor)
		}
		if e.PrebuiltCustomization.ButtonTextColor != nil {
			prebuiltCustomization.ButtonTextColor = types.StringValue(*e.PrebuiltCustomization.ButtonTextColor)
		}
		if e.PrebuiltCustomization.FontFamily != nil {
			prebuiltCustomization.FontFamily = types.StringValue(string(*e.PrebuiltCustomization.FontFamily))
		}
		if e.PrebuiltCustomization.TextAlignment != nil {
			prebuiltCustomization.TextAlignment = types.StringValue(string(*e.PrebuiltCustomization.TextAlignment))
		}
	}
	return prebuiltCustomization
}

type emailTemplateCustomHTMLCustomizationModel struct {
	TemplateType     types.String `tfsdk:"template_type"`
	HTMLContent      types.String `tfsdk:"html_content"`
	PlaintextContent types.String `tfsdk:"plaintext_content"`
	Subject          types.String `tfsdk:"subject"`
}

func (m emailTemplateCustomHTMLCustomizationModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"template_type":     types.StringType,
		"html_content":      types.StringType,
		"plaintext_content": types.StringType,
		"subject":           types.StringType,
	}
}

func emailTemplateCustomHTMLCustomizationModelFromEmailTemplate(e emailtemplates.EmailTemplate) emailTemplateCustomHTMLCustomizationModel {
	var customHTMLCustomization emailTemplateCustomHTMLCustomizationModel
	if e.CustomHTMLCustomization != nil {
		customHTMLCustomization.TemplateType = types.StringValue(string(e.CustomHTMLCustomization.TemplateType))

		if e.CustomHTMLCustomization.HTMLContent != nil {
			customHTMLCustomization.HTMLContent = types.StringValue(*e.CustomHTMLCustomization.HTMLContent)
		}
		if e.CustomHTMLCustomization.PlaintextContent != nil {
			customHTMLCustomization.PlaintextContent = types.StringValue(*e.CustomHTMLCustomization.PlaintextContent)
		}
		if e.CustomHTMLCustomization.Subject != nil {
			customHTMLCustomization.Subject = types.StringValue(*e.CustomHTMLCustomization.Subject)
		}
	}
	return customHTMLCustomization
}

func (m emailTemplateModel) toEmailTemplate(ctx context.Context) (emailtemplates.EmailTemplate, diag.Diagnostics) {
	var diags diag.Diagnostics
	e := emailtemplates.EmailTemplate{
		TemplateID: m.TemplateID.ValueString(),
		Name:       ptr(m.Name.ValueString()),
	}

	if !m.SenderInformation.IsUnknown() && !m.SenderInformation.IsNull() {
		var senderInformation emailTemplateSenderInformationModel
		diags.Append(m.SenderInformation.As(ctx, &senderInformation, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		e.SenderInformation = &emailtemplates.SenderInformation{}
		if !senderInformation.FromLocalPart.IsUnknown() {
			e.SenderInformation.FromLocalPart = ptr(senderInformation.FromLocalPart.ValueString())
		}
		if !senderInformation.FromDomain.IsUnknown() {
			e.SenderInformation.FromDomain = ptr(senderInformation.FromDomain.ValueString())
		}
		if !senderInformation.FromName.IsUnknown() {
			e.SenderInformation.FromName = ptr(senderInformation.FromName.ValueString())
		}
		if !senderInformation.ReplyToLocalPart.IsUnknown() {
			e.SenderInformation.ReplyToLocalPart = ptr(senderInformation.ReplyToLocalPart.ValueString())
		}
		if !senderInformation.ReplyToName.IsUnknown() {
			e.SenderInformation.ReplyToName = ptr(senderInformation.ReplyToName.ValueString())
		}
	}

	if !m.PrebuiltCustomization.IsUnknown() && !m.PrebuiltCustomization.IsNull() {
		var prebuiltCustomization emailTemplatePrebuiltCustomizationModel
		diags.Append(m.PrebuiltCustomization.As(ctx, &prebuiltCustomization, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		e.PrebuiltCustomization = &emailtemplates.PrebuiltCustomization{}
		if !prebuiltCustomization.ButtonBorderRadius.IsUnknown() {
			e.PrebuiltCustomization.ButtonBorderRadius = ptr(prebuiltCustomization.ButtonBorderRadius.ValueFloat32())
		}
		if !prebuiltCustomization.ButtonColor.IsUnknown() {
			e.PrebuiltCustomization.ButtonColor = ptr(prebuiltCustomization.ButtonColor.ValueString())
		}
		if !prebuiltCustomization.ButtonTextColor.IsUnknown() {
			e.PrebuiltCustomization.ButtonTextColor = ptr(prebuiltCustomization.ButtonTextColor.ValueString())
		}
		if !prebuiltCustomization.FontFamily.IsUnknown() {
			e.PrebuiltCustomization.FontFamily = ptr(emailtemplates.FontFamily(prebuiltCustomization.FontFamily.ValueString()))
		}
		if !prebuiltCustomization.TextAlignment.IsUnknown() {
			e.PrebuiltCustomization.TextAlignment = ptr(emailtemplates.TextAlignment(prebuiltCustomization.TextAlignment.ValueString()))
		}
	}

	if !m.CustomHTMLCustomization.IsUnknown() && !m.CustomHTMLCustomization.IsNull() {
		var customHTMLCustomization emailTemplateCustomHTMLCustomizationModel
		diags.Append(m.CustomHTMLCustomization.As(ctx, &customHTMLCustomization, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)
		e.CustomHTMLCustomization = &emailtemplates.CustomHTMLCustomization{}
		if !customHTMLCustomization.TemplateType.IsUnknown() {
			e.CustomHTMLCustomization.TemplateType = emailtemplates.TemplateType(customHTMLCustomization.TemplateType.ValueString())
		}
		if !customHTMLCustomization.HTMLContent.IsUnknown() {
			e.CustomHTMLCustomization.HTMLContent = ptr(customHTMLCustomization.HTMLContent.ValueString())
		}
		if !customHTMLCustomization.PlaintextContent.IsUnknown() {
			e.CustomHTMLCustomization.PlaintextContent = ptr(customHTMLCustomization.PlaintextContent.ValueString())
		}
		if !customHTMLCustomization.Subject.IsUnknown() {
			e.CustomHTMLCustomization.Subject = ptr(customHTMLCustomization.Subject.ValueString())
		}
	}

	return e, diags
}

func (m emailTemplateModel) compareSenderInfo(ctx context.Context, newInfo emailTemplateSenderInformationModel) diag.Diagnostics {
	var diags diag.Diagnostics

	var oldInfo emailTemplateSenderInformationModel
	diags.Append(m.SenderInformation.As(ctx, &oldInfo, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})...)

	// What we're looking for:
	// - If the old value was *not* unknown
	// - and the old value was *not* null
	// - and the old value does not match the new value... that's bad.
	if (!oldInfo.FromName.IsUnknown() && !oldInfo.FromName.IsNull() && oldInfo.FromName.ValueString() != newInfo.FromName.ValueString()) ||
		(!oldInfo.FromLocalPart.IsUnknown() && !oldInfo.FromLocalPart.IsNull() && oldInfo.FromLocalPart.ValueString() != newInfo.FromLocalPart.ValueString()) ||
		(!oldInfo.FromDomain.IsUnknown() && !oldInfo.FromDomain.IsNull() && oldInfo.FromDomain.ValueString() != newInfo.FromDomain.ValueString()) ||
		(!oldInfo.ReplyToName.IsUnknown() && !oldInfo.ReplyToName.IsNull() && oldInfo.ReplyToName.ValueString() != newInfo.ReplyToName.ValueString()) ||
		(!oldInfo.ReplyToLocalPart.IsUnknown() && !oldInfo.ReplyToLocalPart.IsNull() && oldInfo.ReplyToLocalPart.ValueString() != newInfo.ReplyToLocalPart.ValueString()) {
		ctx = tflog.SetField(ctx, "old_sender_info", oldInfo)
		ctx = tflog.SetField(ctx, "new_sender_info", newInfo)
		tflog.Error(ctx, "Sender information mismatch")
		diags.AddError("Invalid SenderInformation", "SenderInformation was not updated - is the custom domain correct?")
	}

	return diags
}

func (m *emailTemplateModel) reloadFromEmailTemplate(ctx context.Context, e emailtemplates.EmailTemplate) diag.Diagnostics {
	var diags diag.Diagnostics

	if e.SenderInformation != nil {
		newSenderInfo := emailTemplateSenderInformationModelFromEmailTemplate(e)
		diags.Append(m.compareSenderInfo(ctx, newSenderInfo)...)
		senderInformation, diag := types.ObjectValueFrom(ctx, emailTemplateSenderInformationModel{}.AttributeTypes(), newSenderInfo)
		diags.Append(diag...)
		m.SenderInformation = senderInformation
	} else {
		// If m.SenderInformation *wasn't* null but nothing was returned, the provisioner supplied bad values.
		if !m.SenderInformation.IsUnknown() && !m.SenderInformation.IsNull() {
			diags.AddError("Invalid SenderInformation", "Supplied SenderInformation was invalid and could not be applied")
		}
		m.SenderInformation = types.ObjectNull(emailTemplateSenderInformationModel{}.AttributeTypes())
	}

	if e.PrebuiltCustomization != nil {
		prebuiltCustomization, diag := types.ObjectValueFrom(ctx, emailTemplatePrebuiltCustomizationModel{}.AttributeTypes(), emailTemplatePrebuiltCustomizationModelFromEmailTemplate(e))
		diags.Append(diag...)
		m.PrebuiltCustomization = prebuiltCustomization
	} else {
		m.PrebuiltCustomization = types.ObjectNull(emailTemplatePrebuiltCustomizationModel{}.AttributeTypes())
	}

	if e.CustomHTMLCustomization != nil {
		customHTMLCustomization, diag := types.ObjectValueFrom(ctx, emailTemplateCustomHTMLCustomizationModel{}.AttributeTypes(), emailTemplateCustomHTMLCustomizationModelFromEmailTemplate(e))
		diags.Append(diag...)
		m.CustomHTMLCustomization = customHTMLCustomization
	} else {
		m.CustomHTMLCustomization = types.ObjectNull(emailTemplateCustomHTMLCustomizationModel{}.AttributeTypes())
	}

	m.ID = types.StringValue(m.LiveProjectID.ValueString() + "." + m.TemplateID.ValueString())
	m.TemplateID = types.StringValue(e.TemplateID)
	if e.Name != nil {
		m.Name = types.StringValue(*e.Name)
	} else {
		m.Name = types.StringNull()
	}
	m.TemplateID = types.StringValue(e.TemplateID)

	return diags
}

func (r *emailTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *emailTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_template"
}

// Schema defines the schema for the resource.
func (r *emailTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"live_project_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier for the live project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template_id": schema.StringAttribute{
				Required: true,
				Description: "A unique identifier to use for the template – this is how you'll refer to the template when sending " +
					"emails from your project or managing this template. It can never be changed after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "A human-readable name of the template. This does not have to be unique.",
			},
			"sender_information": schema.SingleNestedAttribute{
				Optional: true,
				Computed: true,
				Description: "SenderInformation is information about the email sender, such as the reply address or rendered name. " +
					"This is an optional field for PrebuiltCustomization, but required for CustomHTMLCustomization.",
				Attributes: map[string]schema.Attribute{
					"from_local_part": schema.StringAttribute{
						Optional:    true,
						Description: "The prefix of the sender’s email address, everything before the @ symbol (eg: first.last)",
					},
					"from_domain": schema.StringAttribute{
						Optional:    true,
						Description: "The postfix of the sender’s email address, everything after the @ symbol (eg: stytch.com)",
					},
					"from_name": schema.StringAttribute{
						Optional:    true,
						Description: "The sender of the email (eg: Login)",
					},
					"reply_to_local_part": schema.StringAttribute{
						Optional:    true,
						Description: "The prefix of the reply-to email address, everything before the @ symbol (eg: first.last)",
					},
					"reply_to_name": schema.StringAttribute{
						Optional:    true,
						Description: "The sender of the reply-to email address (eg: Support)",
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"prebuilt_customization": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Customization related to prebuilt fields (such as button color) for prebuilt email templates",
				Attributes: map[string]schema.Attribute{
					"button_border_radius": schema.Float32Attribute{
						Optional:    true,
						Description: "The radius of the button border in the email body",
					},
					"button_color": schema.StringAttribute{
						Optional:    true,
						Description: "The color of the button in the email body",
					},
					"button_text_color": schema.StringAttribute{
						Optional:    true,
						Description: "The color of the text in the button in the email body",
					},
					"font_family": schema.StringAttribute{
						Optional:    true,
						Description: "The font type to be used in the email body",
					},
					"text_alignment": schema.StringAttribute{
						Optional:    true,
						Description: "The alignment of the text in the email body",
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"custom_html_customization": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Customization defined for completely custom HTML email templates",
				Attributes: map[string]schema.Attribute{
					"template_type": schema.StringAttribute{
						Optional:    true,
						Description: "The type of email template this custom HTML customization is valid for",
					},
					"html_content": schema.StringAttribute{
						Optional:    true,
						Description: "The HTML content of the email body",
					},
					"plaintext_content": schema.StringAttribute{
						Optional:    true,
						Description: "The plaintext content of the email body",
					},
					"subject": schema.StringAttribute{
						Optional:    true,
						Description: "The subject line in the email template",
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *emailTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan emailTemplateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.LiveProjectID.ValueString())
	ctx = tflog.SetField(ctx, "template_id", plan.TemplateID.ValueString())
	tflog.Info(ctx, "Creating email template")

	emailTemplate, diags := plan.toEmailTemplate(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createResp, err := r.client.EmailTemplates.Create(ctx, emailtemplates.CreateRequest{
		ProjectID:     plan.LiveProjectID.ValueString(),
		EmailTemplate: emailTemplate,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create email template", err.Error())
		return
	}

	tflog.Info(ctx, "Email template created")

	diags = plan.reloadFromEmailTemplate(ctx, createResp.EmailTemplate)
	resp.Diagnostics.Append(diags...)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *emailTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the current state
	var state emailTemplateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "live_project_id", state.LiveProjectID.ValueString())
	ctx = tflog.SetField(ctx, "template_id", state.TemplateID.ValueString())
	tflog.Info(ctx, "Reading email template")

	getResp, err := r.client.EmailTemplates.Get(ctx, emailtemplates.GetRequest{
		ProjectID:  state.LiveProjectID.ValueString(),
		TemplateID: state.TemplateID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read email template", err.Error())
		return
	}

	tflog.Info(ctx, "Read email template")

	diags = state.reloadFromEmailTemplate(ctx, getResp.EmailTemplate)
	resp.Diagnostics.Append(diags...)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *emailTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan emailTemplateModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "live_project_id", plan.LiveProjectID.ValueString())
	ctx = tflog.SetField(ctx, "template_id", plan.TemplateID.ValueString())
	tflog.Info(ctx, "Updating email template")

	emailTemplate, diags := plan.toEmailTemplate(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateResp, err := r.client.EmailTemplates.Update(ctx, emailtemplates.UpdateRequest{
		ProjectID:     plan.LiveProjectID.ValueString(),
		EmailTemplate: emailTemplate,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update email template", err.Error())
		return
	}

	tflog.Info(ctx, "Updated email template")

	diags = plan.reloadFromEmailTemplate(ctx, updateResp.EmailTemplate)
	resp.Diagnostics.Append(diags...)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *emailTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state emailTemplateModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "live_project_id", state.LiveProjectID.ValueString())
	ctx = tflog.SetField(ctx, "template_id", state.TemplateID.ValueString())
	tflog.Info(ctx, "Deleting email template")

	_, err := r.client.EmailTemplates.Delete(ctx, emailtemplates.DeleteRequest{
		ProjectID:  state.LiveProjectID.ValueString(),
		TemplateID: state.TemplateID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete email template", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted email template")
}

func (r *emailTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ".")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "The ID must be in the format <project_id>.<template_id>")
		return
	}

	ctx = tflog.SetField(ctx, "live_project_id", parts[0])
	ctx = tflog.SetField(ctx, "template_id", parts[1])
	tflog.Info(ctx, "Importing email template")
	resp.State.SetAttribute(ctx, path.Root("live_project_id"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("template_id"), parts[1])
}
