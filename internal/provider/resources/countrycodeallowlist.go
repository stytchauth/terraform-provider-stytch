package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v2/pkg/api"
	"github.com/stytchauth/stytch-management-go/v2/pkg/models/countrycodeallowlist"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &countryCodeAllowlistResource{}
	_ resource.ResourceWithConfigure   = &countryCodeAllowlistResource{}
	_ resource.ResourceWithImportState = &countryCodeAllowlistResource{}
)

func NewCountryCodeAllowlistResource() resource.Resource {
	return &countryCodeAllowlistResource{}
}

type countryCodeAllowlistResource struct {
	client *api.API
}

type countryCodeAllowlistModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectID      types.String `tfsdk:"project_id"`
	DeliveryMethod types.String `tfsdk:"delivery_method"`
	CountryCodes   types.List   `tfsdk:"country_codes"`
	LastUpdated    types.String `tfsdk:"last_updated"`
}

// Configure sets provider-level data for the resource.
func (r *countryCodeAllowlistResource) Configure(
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
func (r *countryCodeAllowlistResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_country_code_allowlist"
}

// Schema defines the schema for the resource.
func (r *countryCodeAllowlistResource) Schema(
	ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Resource for managing country code allowlists.",
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
			"delivery_method": schema.StringAttribute{
				Description: "The delivery method for the country code allowlist.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(toStrings(countrycodeallowlist.DeliveryMethods())...),
				},
			},
			"country_codes": schema.ListAttribute{
				Description: "List of country codes to allow.",
				Required:    true,
				ElementType: types.StringType,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update.",
				Computed:    true,
			},
		},
	}
}

func (r *countryCodeAllowlistResource) setCountryCodeAllowlist(
	ctx context.Context, plan countryCodeAllowlistModel, countryCodes []string,
) error {
	var err error
	if plan.DeliveryMethod.ValueString() == string(countrycodeallowlist.DeliveryMethodSMS) {
		_, err = r.client.CountryCodeAllowlist.SetAllowedSMSCountryCodes(ctx,
			&countrycodeallowlist.SetAllowedSMSCountryCodesRequest{
				ProjectID:    plan.ProjectID.ValueString(),
				CountryCodes: countryCodes,
			})
	} else {
		_, err = r.client.CountryCodeAllowlist.SetAllowedWhatsAppCountryCodes(ctx,
			&countrycodeallowlist.SetAllowedWhatsAppCountryCodesRequest{
				ProjectID:    plan.ProjectID.ValueString(),
				CountryCodes: countryCodes,
			})
	}

	return err
}

// Create creates the resource and sets the initial Terraform state.
func (r *countryCodeAllowlistResource) Create(
	ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse,
) {
	var plan countryCodeAllowlistModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	tflog.Info(ctx, "Creating country code allowlist")

	countryCodes := make([]string, 0, len(plan.CountryCodes.Elements()))
	diags = plan.CountryCodes.ElementsAs(ctx, &countryCodes, false)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	err := r.setCountryCodeAllowlist(ctx, plan, countryCodes)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create country code allowlist", err.Error())
		return
	}

	tflog.Info(ctx, "Country code allowlist created")

	plan.ID = types.StringValue(plan.ProjectID.ValueString() + "." + plan.DeliveryMethod.ValueString())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *countryCodeAllowlistResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the current state.
	var state countryCodeAllowlistModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	tflog.Info(ctx, "Reading country code allowlist")

	var countryCodes []string
	if state.DeliveryMethod.ValueString() == string(countrycodeallowlist.DeliveryMethodSMS) {
		getResp, err := r.client.CountryCodeAllowlist.GetAllowedSMSCountryCodes(ctx,
			&countrycodeallowlist.GetAllowedSMSCountryCodesRequest{
				ProjectID: state.ProjectID.ValueString(),
			})
		if err != nil {
			resp.Diagnostics.AddError("Failed to read SMS country code allowlist", err.Error())
			return
		}
		countryCodes = getResp.CountryCodes
	} else {
		getResp, err := r.client.CountryCodeAllowlist.GetAllowedWhatsAppCountryCodes(ctx,
			&countrycodeallowlist.GetAllowedWhatsAppCountryCodesRequest{
				ProjectID: state.ProjectID.ValueString(),
			})
		if err != nil {
			resp.Diagnostics.AddError("Failed to read WhatsApp country code allowlist", err.Error())
			return
		}
		countryCodes = getResp.CountryCodes
	}

	state.CountryCodes, diags = types.ListValueFrom(ctx, types.StringType, countryCodes)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	tflog.Info(ctx, "Read country code allowlist")
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *countryCodeAllowlistResource) Update(
	ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	var plan countryCodeAllowlistModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	tflog.Info(ctx, "Updating country code allowlist")

	countryCodes := make([]string, 0, len(plan.CountryCodes.Elements()))
	diags = plan.CountryCodes.ElementsAs(ctx, &countryCodes, false)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	err := r.setCountryCodeAllowlist(ctx, plan, countryCodes)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update country code allowlist", err.Error())
		return
	}

	tflog.Info(ctx, "Country code allowlist updated")

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *countryCodeAllowlistResource) Delete(
	ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	// Get the current state.
	var state countryCodeAllowlistModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	tflog.Info(ctx, "Setting country code allowlist to default value")

	err := r.setCountryCodeAllowlist(ctx, state, countrycodeallowlist.DefaultCountryCodes)
	if err != nil {
		resp.Diagnostics.AddError("Failed to reset country code allowlist", err.Error())
		return
	}

	// No need to update the state since the resource is being deleted.
	tflog.Info(ctx, "Reset country code allowlist to default state")
}

func (r *countryCodeAllowlistResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	parts := strings.Split(req.ID, ".")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "The ID must be in the format <project_id>.<delivery_method>")
		return
	}

	ctx = tflog.SetField(ctx, "id", req.ID)
	ctx = tflog.SetField(ctx, "project_id", parts[0])
	ctx = tflog.SetField(ctx, "delivery_method", parts[1])
	tflog.Info(ctx, "Importing country code allowlist")
	resp.State.SetAttribute(ctx, path.Root("id"), req.ID)
	resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("delivery_method"), parts[1])
}
