package resources

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/eventlogstreaming"
)

var (
	_ resource.Resource                = &eventLogStreamingResource{}
	_ resource.ResourceWithConfigure   = &eventLogStreamingResource{}
	_ resource.ResourceWithImportState = &eventLogStreamingResource{}
)

// preserveSensitiveValuePlanModifier is a plan modifier that preserves sensitive values
// during refresh operations to prevent drift detection issues.
type preserveSensitiveValuePlanModifier struct{}

func (m preserveSensitiveValuePlanModifier) Description(ctx context.Context) string {
	return "Preserves sensitive values during refresh operations"
}

func (m preserveSensitiveValuePlanModifier) MarkdownDescription(ctx context.Context) string {
	return "Preserves sensitive values during refresh operations"
}

func (m preserveSensitiveValuePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the plan value is unknown (during refresh), preserve the state value
	if req.PlanValue.IsUnknown() {
		resp.PlanValue = req.StateValue
		return
	}

	// If the plan value is null but state has a value, preserve the state value
	if req.PlanValue.IsNull() && !req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
		return
	}

	// Otherwise, use the plan value (user explicitly set it)
	resp.PlanValue = req.PlanValue
}

func NewEventLogStreamingResource() resource.Resource {
	return &eventLogStreamingResource{}
}

type eventLogStreamingResource struct {
	client *api.API
}

type eventLogStreamingModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	DestinationType   types.String `tfsdk:"destination_type"`
	DatadogConfig     types.Object `tfsdk:"datadog_config"`
	GrafanaLokiConfig types.Object `tfsdk:"grafana_loki_config"`
	StreamingStatus   types.String `tfsdk:"streaming_status"`
	LastUpdated       types.String `tfsdk:"last_updated"`
}

type eventLogStreamingDatadogConfigModel struct {
	Site   types.String `tfsdk:"site"`
	ApiKey types.String `tfsdk:"api_key"`
}

func (m eventLogStreamingDatadogConfigModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"site":    types.StringType,
		"api_key": types.StringType,
	}
}

type eventLogStreamingGrafanaLokiConfigModel struct {
	Hostname types.String `tfsdk:"hostname"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (m eventLogStreamingGrafanaLokiConfigModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"hostname": types.StringType,
		"username": types.StringType,
		"password": types.StringType,
	}
}

func (r *eventLogStreamingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *eventLogStreamingResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_event_log_streaming"
}

// Schema defines the schema for the resource.
func (r *eventLogStreamingResource) Schema(
	ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "Resource for managing event log streaming.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the resource.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Required:    true,
				Description: "The unique identifier for the project.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"destination_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of destination to send events to.",
				Validators: []validator.String{
					stringvalidator.OneOf(toStrings(eventlogstreaming.DestinationTypes())...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"streaming_status": schema.StringAttribute{
				Computed:    true,
				Description: "The status of streaming for this project and destination.",
				Validators: []validator.String{
					stringvalidator.OneOf(toStrings(eventlogstreaming.StreamingStatuses())...),
				},
			},
			"datadog_config": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "The configuration for the Datadog destination to send events to.",
				Attributes: map[string]schema.Attribute{
					"site": schema.StringAttribute{
						Required:    true,
						Description: "The site of the Datadog account.",
						Validators: []validator.String{
							stringvalidator.OneOf(toStrings(eventlogstreaming.DatadogSites())...),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"api_key": schema.StringAttribute{
						Required:    true,
						Description: "The API key for the Datadog account.",
						Sensitive:   true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(32, 32),
							stringvalidator.RegexMatches(regexp.MustCompile(`^[a-f0-9]+$`), "must be a hex string"),
						},
						PlanModifiers: []planmodifier.String{
							preserveSensitiveValuePlanModifier{},
						},
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"grafana_loki_config": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "The configuration for the Grafana Loki destination to send events to.",
				Attributes: map[string]schema.Attribute{
					"hostname": schema.StringAttribute{
						Required:    true,
						Description: "The hostname of the Grafana Loki instance. Custom protocols and paths are not supported.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"username": schema.StringAttribute{
						Required:    true,
						Description: "The username for the Grafana Loki instance.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"password": schema.StringAttribute{
						Required:    true,
						Description: "The password for the Grafana Loki instance.",
						Sensitive:   true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							preserveSensitiveValuePlanModifier{},
						},
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *eventLogStreamingResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data eventLogStreamingModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", data.ProjectID.ValueString())
	tflog.Info(ctx, "Validating event log streaming config")

	switch data.DestinationType.ValueString() {
	case string(eventlogstreaming.DestinationTypeDatadog):

		if data.DatadogConfig.IsNull() || data.DatadogConfig.IsUnknown() {
			resp.Diagnostics.AddError("Datadog config is required", "The datadog_config block is required when destination_type is set to DATADOG.")
			return
		}

		if !data.GrafanaLokiConfig.IsNull() {
			resp.Diagnostics.AddError("Invalid configuration", "The grafana_loki_config block is not allowed when destination_type is set to DATADOG.")
			return
		}

	case string(eventlogstreaming.DestinationTypeGrafanaLoki):

		if data.GrafanaLokiConfig.IsNull() || data.GrafanaLokiConfig.IsUnknown() {
			resp.Diagnostics.AddError("Grafana Loki config is required", "The grafana_loki_config block is required when destination_type is set to GRAFANA_LOKI.")
			return
		}

		if !data.DatadogConfig.IsNull() {
			resp.Diagnostics.AddError("Invalid configuration", "The datadog_config block is not allowed when destination_type is set to GRAFANA_LOKI.")
			return
		}
	default:
		resp.Diagnostics.AddError("Invalid destination type", "Unsupported destination type: %")
		return
	}
}

func (m eventLogStreamingModel) toDestinationConfig(ctx context.Context) (eventlogstreaming.DestinationConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	var destinationConfig eventlogstreaming.DestinationConfig

	switch m.DestinationType.ValueString() {
	case string(eventlogstreaming.DestinationTypeDatadog):

		var datadogConfig eventLogStreamingDatadogConfigModel
		diags.Append(m.DatadogConfig.As(ctx, &datadogConfig, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    false,
			UnhandledUnknownAsEmpty: false,
		})...)

		destinationConfig.Datadog = &eventlogstreaming.DatadogConfig{
			Site:   eventlogstreaming.DatadogSite(datadogConfig.Site.ValueString()),
			APIKey: datadogConfig.ApiKey.ValueString(),
		}

	case string(eventlogstreaming.DestinationTypeGrafanaLoki):
		var grafanaLokiConfig eventLogStreamingGrafanaLokiConfigModel
		diags.Append(m.GrafanaLokiConfig.As(ctx, &grafanaLokiConfig, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    false,
			UnhandledUnknownAsEmpty: false,
		})...)

		destinationConfig.GrafanaLoki = &eventlogstreaming.GrafanaLokiConfig{
			Hostname: grafanaLokiConfig.Hostname.ValueString(),
			Username: grafanaLokiConfig.Username.ValueString(),
			Password: grafanaLokiConfig.Password.ValueString(),
		}
	}

	return destinationConfig, diags
}

func (m *eventLogStreamingModel) refreshFromEventLogStreaming(ctx context.Context, r eventlogstreaming.EventLogStreamingConfig) diag.Diagnostics {
	var diags diag.Diagnostics

	m.StreamingStatus = types.StringValue(string(r.StreamingStatus))

	var diag diag.Diagnostics

	switch r.DestinationType {
	case eventlogstreaming.DestinationTypeDatadog:
		m.DatadogConfig, diag = types.ObjectValueFrom(ctx, eventLogStreamingDatadogConfigModel{}.AttributeTypes(), eventLogStreamingDatadogConfigModel{
			Site:   types.StringValue(string(r.DestinationConfig.Datadog.Site)),
			ApiKey: types.StringValue(r.DestinationConfig.Datadog.APIKey),
		})
		diags.Append(diag...)
		m.GrafanaLokiConfig = types.ObjectNull(eventLogStreamingGrafanaLokiConfigModel{}.AttributeTypes())
	case eventlogstreaming.DestinationTypeGrafanaLoki:
		m.GrafanaLokiConfig, diag = types.ObjectValueFrom(ctx, eventLogStreamingGrafanaLokiConfigModel{}.AttributeTypes(), eventLogStreamingGrafanaLokiConfigModel{
			Hostname: types.StringValue(r.DestinationConfig.GrafanaLoki.Hostname),
			Username: types.StringValue(r.DestinationConfig.GrafanaLoki.Username),
			Password: types.StringValue(r.DestinationConfig.GrafanaLoki.Password),
		})
		diags.Append(diag...)
		m.DatadogConfig = types.ObjectNull(eventLogStreamingDatadogConfigModel{}.AttributeTypes())
	default:
		m.DatadogConfig = types.ObjectNull(eventLogStreamingDatadogConfigModel{}.AttributeTypes())
		m.GrafanaLokiConfig = types.ObjectNull(eventLogStreamingGrafanaLokiConfigModel{}.AttributeTypes())
	}

	return diags
}

func (m *eventLogStreamingModel) refreshFromMaskedEventLogStreaming(ctx context.Context, r eventlogstreaming.EventLogStreamingConfigMasked) diag.Diagnostics {
	var diags diag.Diagnostics

	m.StreamingStatus = types.StringValue(string(r.StreamingStatus))

	var diag diag.Diagnostics

	switch r.DestinationType {
	case eventlogstreaming.DestinationTypeDatadog:
		// For masked responses, we only get non-sensitive data
		// Sensitive fields will be preserved by the plan modifier
		m.DatadogConfig, diag = types.ObjectValueFrom(ctx, eventLogStreamingDatadogConfigModel{}.AttributeTypes(), eventLogStreamingDatadogConfigModel{
			Site:   types.StringValue(string(r.DestinationConfig.Datadog.Site)),
			ApiKey: types.StringValue(""),
		})
		diags.Append(diag...)
		m.GrafanaLokiConfig = types.ObjectNull(eventLogStreamingGrafanaLokiConfigModel{}.AttributeTypes())
	case eventlogstreaming.DestinationTypeGrafanaLoki:
		// For masked responses, we only get non-sensitive data
		// Sensitive fields will be preserved by the plan modifier
		m.GrafanaLokiConfig, diag = types.ObjectValueFrom(ctx, eventLogStreamingGrafanaLokiConfigModel{}.AttributeTypes(), eventLogStreamingGrafanaLokiConfigModel{
			Hostname: types.StringValue(r.DestinationConfig.GrafanaLoki.Hostname),
			Username: types.StringValue(r.DestinationConfig.GrafanaLoki.Username),
			Password: types.StringValue(""), // Will be preserved by plan modifier if state has value
		})
		diags.Append(diag...)
		m.DatadogConfig = types.ObjectNull(eventLogStreamingDatadogConfigModel{}.AttributeTypes())
	default:
		m.DatadogConfig = types.ObjectNull(eventLogStreamingDatadogConfigModel{}.AttributeTypes())
		m.GrafanaLokiConfig = types.ObjectNull(eventLogStreamingGrafanaLokiConfigModel{}.AttributeTypes())
	}

	return diags
}

// Create creates the resource and sets the initial Terraform state.
func (r *eventLogStreamingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eventLogStreamingModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	tflog.Info(ctx, "Creating event log streaming config")

	createRequest := eventlogstreaming.CreateEventLogStreamingRequest{
		ProjectID:       plan.ProjectID.ValueString(),
		DestinationType: eventlogstreaming.DestinationType(plan.DestinationType.ValueString()),
	}

	destinationConfig, diags := plan.toDestinationConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	createRequest.DestinationConfig = destinationConfig

	// Mask sensitive data in logs before making the request
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "api_key", "username", "password")

	createResponse, err := r.client.EventLogStreaming.Create(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create event log streaming config", err.Error())
		return
	}

	tflog.Info(ctx, "Event log streaming config created")

	// Now update the state with the response
	diags = plan.refreshFromEventLogStreaming(ctx, createResponse.EventLogStreamingConfig)
	resp.Diagnostics.Append(diags...)
	plan.ID = types.StringValue(plan.ProjectID.ValueString() + "." + plan.DestinationType.ValueString())
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *eventLogStreamingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the current state.
	var state eventLogStreamingModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	ctx = tflog.SetField(ctx, "destination_type", state.DestinationType.ValueString())
	tflog.Info(ctx, "Reading event log streaming config")

	getResp, err := r.client.EventLogStreaming.Get(ctx, eventlogstreaming.GetEventLogStreamingRequest{
		ProjectID:       state.ProjectID.ValueString(),
		DestinationType: eventlogstreaming.DestinationType(state.DestinationType.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get event log streaming config", err.Error())
		return
	}

	tflog.Info(ctx, "Event log streaming config read")

	// Create a temporary state to hold the masked response data
	var tempState eventLogStreamingModel
	tempState.ID = state.ID
	tempState.ProjectID = state.ProjectID
	tempState.DestinationType = state.DestinationType
	tempState.LastUpdated = state.LastUpdated

	// Populate the temporary state with masked response data
	diags = tempState.refreshFromMaskedEventLogStreaming(ctx, getResp.EventLogStreamingConfig)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Merge sensitive values from existing state into the temporary state
	switch state.DestinationType.ValueString() {
	case string(eventlogstreaming.DestinationTypeDatadog):
		if !state.DatadogConfig.IsNull() && !state.DatadogConfig.IsUnknown() {
			var existingConfig eventLogStreamingDatadogConfigModel
			if diag := state.DatadogConfig.As(ctx, &existingConfig, basetypes.ObjectAsOptions{}); !diag.HasError() {
				// Preserve the existing API key
				var tempDatadogConfig eventLogStreamingDatadogConfigModel
				if diag := tempState.DatadogConfig.As(ctx, &tempDatadogConfig, basetypes.ObjectAsOptions{}); !diag.HasError() {
					tempState.DatadogConfig, diag = types.ObjectValueFrom(ctx, eventLogStreamingDatadogConfigModel{}.AttributeTypes(), eventLogStreamingDatadogConfigModel{
						Site:   tempDatadogConfig.Site,
						ApiKey: existingConfig.ApiKey, // Preserve existing sensitive value
					})
					resp.Diagnostics.Append(diag...)
				}
			}
		}
	case string(eventlogstreaming.DestinationTypeGrafanaLoki):
		if !state.GrafanaLokiConfig.IsNull() && !state.GrafanaLokiConfig.IsUnknown() {
			var existingConfig eventLogStreamingGrafanaLokiConfigModel
			if diag := state.GrafanaLokiConfig.As(ctx, &existingConfig, basetypes.ObjectAsOptions{}); !diag.HasError() {
				// Extract values from tempState safely
				var tempConfig eventLogStreamingGrafanaLokiConfigModel
				if diag := tempState.GrafanaLokiConfig.As(ctx, &tempConfig, basetypes.ObjectAsOptions{}); !diag.HasError() {
					// Preserve the existing password
					tempState.GrafanaLokiConfig, diag = types.ObjectValueFrom(ctx, eventLogStreamingGrafanaLokiConfigModel{}.AttributeTypes(), eventLogStreamingGrafanaLokiConfigModel{
						Hostname: tempConfig.Hostname,
						Username: tempConfig.Username,
						Password: existingConfig.Password, // Preserve existing sensitive value
					})
					resp.Diagnostics.Append(diag...)
				}
			}
		}
	}

	// Set the merged state
	diags = resp.State.Set(ctx, tempState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *eventLogStreamingResource) Update(
	ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	var plan eventLogStreamingModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", plan.ProjectID.ValueString())
	ctx = tflog.SetField(ctx, "destination_type", plan.DestinationType.ValueString())
	tflog.Info(ctx, "Updating event log streaming config")

	updateRequest := eventlogstreaming.UpdateEventLogStreamingRequest{
		ProjectID:       plan.ProjectID.ValueString(),
		DestinationType: eventlogstreaming.DestinationType(plan.DestinationType.ValueString()),
	}

	destinationConfig, diags := plan.toDestinationConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	updateRequest.DestinationConfig = destinationConfig

	// Mask sensitive data in logs before making the request
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "api_key", "username", "password")

	updateResponse, err := r.client.EventLogStreaming.Update(ctx, updateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update event log streaming config", err.Error())
		return
	}

	tflog.Info(ctx, "Event log streaming config updated")

	// Now update the state with the response
	diags = plan.refreshFromEventLogStreaming(ctx, updateResponse.EventLogStreamingConfig)
	resp.Diagnostics.Append(diags...)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state.
func (r *eventLogStreamingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eventLogStreamingModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_id", state.ProjectID.ValueString())
	ctx = tflog.SetField(ctx, "destination_type", state.DestinationType.ValueString())
	tflog.Info(ctx, "Deleting event log streaming config")

	_, err := r.client.EventLogStreaming.Delete(ctx, eventlogstreaming.DeleteEventLogStreamingRequest{
		ProjectID:       state.ProjectID.ValueString(),
		DestinationType: eventlogstreaming.DestinationType(state.DestinationType.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete event log streaming config", err.Error())
		return
	}

	tflog.Info(ctx, "Event log streaming config deleted")
}

func (r *eventLogStreamingResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	parts := strings.Split(req.ID, ".")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "The ID must be in the format <project_id>.<destination_type>")
		return
	}

	ctx = tflog.SetField(ctx, "id", req.ID)
	ctx = tflog.SetField(ctx, "project_id", parts[0])
	ctx = tflog.SetField(ctx, "destination_type", parts[1])

	tflog.Info(ctx, "Importing event log streaming config")
	resp.State.SetAttribute(ctx, path.Root("id"), req.ID)
	resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("destination_type"), parts[1])
}
