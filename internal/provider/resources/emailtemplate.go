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
	LiveProjectID           types.String                              `tfsdk:"live_project_id"`
	TemplateID              types.String                              `tfsdk:"template_id"`
	LastUpdated             types.String                              `tfsdk:"last_updated"`
	Name                    types.String                              `tfsdk:"name"`
	SenderInformation       emailTemplateSenderInformationModel       `tfsdk:"sender_information"`
	PrebuiltCustomization   emailTemplatePrebuiltCustomizationModel   `tfsdk:"prebuilt_customization"`
	CustomHTMLCustomization emailTemplateCustomHTMLCustomizationModel `tfsdk:"custom_html_customization"`
}

type emailTemplateSenderInformationModel struct {
	FromLocalPart    types.String `tfsdk:"from_local_part"`
	FromDomain       types.String `tfsdk:"from_domain"`
	FromName         types.String `tfsdk:"from_name"`
	ReplyToLocalPart types.String `tfsdk:"reply_to_local_part"`
	ReplyToName      types.String `tfsdk:"reply_to_name"`
}

type emailTemplatePrebuiltCustomizationModel struct {
	ButtonBorderRadius types.Number `tfsdk:"button_border_radius"`
	ButtonColor        types.String `tfsdk:"button_color"`
	ButtonTextColor    types.String `tfsdk:"button_text_color"`
	FontFamily         types.String `tfsdk:"font_family"`
	TextAlignment      types.String `tfsdk:"text_alignment"`
}

type emailTemplateCustomHTMLCustomizationModel struct {
	TemplateType     types.String `tfsdk:"template_type"`
	HTMLContent      types.String `tfsdk:"html_content"`
	PlaintextContent types.String `tfsdk:"plaintext_content"`
	Subject          types.String `tfsdk:"subject"`
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
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Schema defines the schema for the resource.
func (r *emailTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"live_project_id": schema.StringAttribute{
				Computed:    true,
				Description: "The unique identifier for the live project.",
			},
			"template_id": schema.StringAttribute{
				Required: true,
				Description: "A unique identifier to use for the template – this is how you'll refer to the template when sending " +
					"emails from your project or managing this template. It can never be changed after creation.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "A human-readable name of the template. This does not have to be unique.",
			},
			"sender_information": schema.SingleNestedAttribute{
				Optional: true,
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
			},
			"prebuilt_customization": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Customization related to prebuilt fields (such as button color) for prebuilt email templates",
				Attributes: map[string]schema.Attribute{
					"button_border_radius": schema.NumberAttribute{
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
			},
			"custom_html_customization": schema.SingleNestedAttribute{
				Optional:    true,
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

	// TODO: Generate API request body from plan and call r.client.EmailTemplates.Create

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

	// TODO: Get refreshed value from the API

	// Set refreshed state
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

	// TODO: Generate API request body from plan and call r.client.EmailTemplates.Update

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

	// TODO: Call r.client.EmailTemplates.Delete
}

func (r *emailTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("live_project_id"), req, resp)
}
