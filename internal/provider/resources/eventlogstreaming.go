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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/eventlogstreaming"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/utils"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                 = &eventLogStreamingResource{}
	_ resource.ResourceWithConfigure    = &eventLogStreamingResource{}
	_ resource.ResourceWithImportState  = &eventLogStreamingResource{}
	_ resource.ResourceWithUpgradeState = &eventLogStreamingResource{}
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

// NewEventLogStreamingResource is a helper function to simplify the provider implementation.
func NewEventLogStreamingResource() resource.Resource {
	return &eventLogStreamingResource{}
}

type eventLogStreamingResource struct {
	client *api.API
}

type eventLogStreamingModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectSlug       types.String `tfsdk:"project_slug"`
	EnvironmentSlug   types.String `tfsdk:"environment_slug"`
	LastUpdated       types.String `tfsdk:"last_updated"`
	DestinationType   types.String `tfsdk:"destination_type"`
	DatadogConfig     types.Object `tfsdk:"datadog_config"`
	GrafanaLokiConfig types.Object `tfsdk:"grafana_loki_config"`
	Enabled           types.Bool   `tfsdk:"enabled"`
}

type eventLogStreamingResourceModelV0 struct {
	ProjectID       types.String `tfsdk:"project_id"`
	DestinationType types.String `tfsdk:"destination_type"`
	// We need to fetch these from the previous version to preserve sensitive data
	// in the StateUpgrade
	DatadogConfig     types.Object `tfsdk:"datadog_config"`
	GrafanaLokiConfig types.Object `tfsdk:"grafana_loki_config"`
}

var eventLogStreamingResourceLegacySchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"project_id": schema.StringAttribute{
			Required: true,
		},
		"destination_type": schema.StringAttribute{
			Required: true,
		},
		"datadog_config": schema.SingleNestedAttribute{
			Optional: true,
			Attributes: map[string]schema.Attribute{
				"site": schema.StringAttribute{
					Optional: true,
				},
				"api_key": schema.StringAttribute{
					Optional:  true,
					Sensitive: true,
				},
			},
		},
		"grafana_loki_config": schema.SingleNestedAttribute{
			Optional: true,
			Attributes: map[string]schema.Attribute{
				"hostname": schema.StringAttribute{
					Optional: true,
				},
				"username": schema.StringAttribute{
					Optional: true,
				},
				"password": schema.StringAttribute{
					Optional:  true,
					Sensitive: true,
				},
			},
		},
	},
}

type datadogConfigModel struct {
	Site   types.String `tfsdk:"site"`
	APIKey types.String `tfsdk:"api_key"`
}

type grafanaLokiConfigModel struct {
	Hostname types.String `tfsdk:"hostname"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

var datadogConfigModelAttrTypes = map[string]attr.Type{
	"site":    types.StringType,
	"api_key": types.StringType,
}

var grafanaLokiConfigModelAttrTypes = map[string]attr.Type{
	"hostname": types.StringType,
	"username": types.StringType,
	"password": types.StringType,
}

// refreshFromEventLogStreaming updates the model from a full API response (used in Create/Update)
// Note: This does NOT update the Enabled field - that should be managed separately since
// Enable/Disable are separate API calls that don't return the config.
func (m *eventLogStreamingModel) refreshFromEventLogStreaming(ctx context.Context, r eventlogstreaming.EventLogStreaming) diag.Diagnostics {
	var diags diag.Diagnostics

	// Don't update Enabled - it's managed separately via Enable/Disable calls

	switch r.DestinationType {
	case eventlogstreaming.DestinationTypeDatadog:
		if r.DestinationConfig.Datadog != nil {
			config := datadogConfigModel{
				Site:   types.StringValue(string(r.DestinationConfig.Datadog.Site)),
				APIKey: types.StringValue(r.DestinationConfig.Datadog.APIKey),
			}

			obj, d := types.ObjectValueFrom(ctx, datadogConfigModelAttrTypes, config)
			diags.Append(d...)
			m.DatadogConfig = obj
		}

	case eventlogstreaming.DestinationTypeGrafanaLoki:
		if r.DestinationConfig.GrafanaLoki != nil {
			config := grafanaLokiConfigModel{
				Hostname: types.StringValue(r.DestinationConfig.GrafanaLoki.Hostname),
				Username: types.StringValue(r.DestinationConfig.GrafanaLoki.Username),
				Password: types.StringValue(r.DestinationConfig.GrafanaLoki.Password),
			}

			obj, d := types.ObjectValueFrom(ctx, grafanaLokiConfigModelAttrTypes, config)
			diags.Append(d...)
			m.GrafanaLokiConfig = obj
		}
	}

	return diags
}

// refreshFromMaskedEventLogStreaming updates the model from a masked API response (used in Read/Import).
func (m *eventLogStreamingModel) refreshFromMaskedEventLogStreaming(ctx context.Context, r eventlogstreaming.EventLogStreamingMasked) diag.Diagnostics {
	var diags diag.Diagnostics

	m.Enabled = types.BoolValue(r.StreamingStatus == eventlogstreaming.StreamingStatusActive)

	switch r.DestinationType {
	case eventlogstreaming.DestinationTypeDatadog:
		var currentConfig datadogConfigModel

		// If we have existing state, preserve sensitive fields
		if !m.DatadogConfig.IsNull() {
			d := m.DatadogConfig.As(ctx, &currentConfig, basetypes.ObjectAsOptions{})
			diags.Append(d...)
			if diags.HasError() {
				return diags
			}
		}

		// Update non-sensitive fields from API
		if r.DestinationConfig.Datadog != nil {
			currentConfig.Site = types.StringValue(string(r.DestinationConfig.Datadog.Site))
		}
		// Keep api_key from state as it's masked in API response (or empty on import)

		obj, d := types.ObjectValueFrom(ctx, datadogConfigModelAttrTypes, currentConfig)
		diags.Append(d...)
		m.DatadogConfig = obj

	case eventlogstreaming.DestinationTypeGrafanaLoki:
		var currentConfig grafanaLokiConfigModel

		// If we have existing state, preserve sensitive fields
		if !m.GrafanaLokiConfig.IsNull() {
			d := m.GrafanaLokiConfig.As(ctx, &currentConfig, basetypes.ObjectAsOptions{})
			diags.Append(d...)
			if diags.HasError() {
				return diags
			}
		}

		// Update non-sensitive fields from API
		if r.DestinationConfig.GrafanaLoki != nil {
			currentConfig.Hostname = types.StringValue(r.DestinationConfig.GrafanaLoki.Hostname)
			currentConfig.Username = types.StringValue(r.DestinationConfig.GrafanaLoki.Username)
		}
		// Keep password from state as it's masked in API response (or empty on import)

		obj, d := types.ObjectValueFrom(ctx, grafanaLokiConfigModelAttrTypes, currentConfig)
		diags.Append(d...)
		m.GrafanaLokiConfig = obj
	}

	return diags
}

func (r *eventLogStreamingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_event_log_streaming"
}

func (r *eventLogStreamingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.API)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *api.API, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *eventLogStreamingResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &eventLogStreamingResourceLegacySchema,
			StateUpgrader: r.upgradeEventLogStreamingStateV0ToV1,
		},
	}
}

func (r *eventLogStreamingResource) upgradeEventLogStreamingStateV0ToV1(
	ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse,
) {
	if req.State == nil {
		resp.Diagnostics.AddError(
			"Missing prior state",
			"Legacy event log streaming state upgrade requires existing state data, but none was provided.",
		)
		return
	}

	var prior eventLogStreamingResourceModelV0
	diags := req.State.Get(ctx, &prior)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectSlug, environmentSlug, diags := utils.ResolveLegacyProjectAndEnvironment(
		ctx, r.client, prior.ProjectID.ValueString(),
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	destinationType := strings.ToUpper(prior.DestinationType.ValueString())
	destinationTypeEnum := eventlogstreaming.DestinationType(destinationType)

	getResp, err := r.client.EventLogStreaming.Get(ctx, eventlogstreaming.GetRequest{
		ProjectSlug:     projectSlug,
		EnvironmentSlug: environmentSlug,
		DestinationType: destinationTypeEnum,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to retrieve event log streaming configuration", err.Error())
		return
	}

	newState := eventLogStreamingModel{
		ID:                types.StringValue(fmt.Sprintf("%s.%s.%s", projectSlug, environmentSlug, destinationType)),
		ProjectSlug:       types.StringValue(projectSlug),
		EnvironmentSlug:   types.StringValue(environmentSlug),
		DestinationType:   types.StringValue(destinationType),
		LastUpdated:       types.StringValue(time.Now().Format(time.RFC850)),
		DatadogConfig:     types.ObjectNull(datadogConfigModelAttrTypes),
		GrafanaLokiConfig: types.ObjectNull(grafanaLokiConfigModelAttrTypes),
		Enabled:           types.BoolValue(getResp.EventLogStreamingConfig.StreamingStatus == eventlogstreaming.StreamingStatusActive),
	}

	switch destinationTypeEnum {
	case eventlogstreaming.DestinationTypeDatadog:
		if !prior.DatadogConfig.IsNull() && !prior.DatadogConfig.IsUnknown() {
			newState.DatadogConfig = prior.DatadogConfig
		}
	case eventlogstreaming.DestinationTypeGrafanaLoki:
		if !prior.GrafanaLokiConfig.IsNull() && !prior.GrafanaLokiConfig.IsUnknown() {
			newState.GrafanaLokiConfig = prior.GrafanaLokiConfig
		}
	}
	diags = resp.State.Set(ctx, newState)
	resp.Diagnostics.Append(diags...)
}

func (r *eventLogStreamingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages event log streaming configuration for an environment. Configure streaming to send Stytch event logs to external destinations like Datadog or Grafana Loki.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the event log streaming configuration in the format `project_slug.environment_slug.destination_type`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project for which to configure event log streaming",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment for which to configure event log streaming",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the last Terraform update of the event log streaming configuration.",
			},
			"destination_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of destination to which to send events. Valid values: DATADOG, GRAFANA_LOKI",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(toStrings(eventlogstreaming.DestinationTypes())...),
				},
			},
			"datadog_config": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Datadog configuration. Required when destination_type is DATADOG.",
				Attributes: map[string]schema.Attribute{
					"site": schema.StringAttribute{
						Required:    true,
						Description: "The Datadog site to which to send events. Valid values: US, US3, US5, EU, AP1",
						Validators: []validator.String{
							stringvalidator.OneOf(toStrings(eventlogstreaming.DatadogSites())...),
						},
					},
					"api_key": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "The Datadog API key for submitting logs (must be 32 hex characters)",
						Validators: []validator.String{
							stringvalidator.LengthBetween(32, 32),
							stringvalidator.RegexMatches(regexp.MustCompile(`^[a-f0-9]+$`), "must be a hex string"),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							preserveSensitiveValuePlanModifier{},
						},
					},
				},
			},
			"grafana_loki_config": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Grafana Loki configuration. Required when destination_type is GRAFANA_LOKI.",
				Attributes: map[string]schema.Attribute{
					"hostname": schema.StringAttribute{
						Required:    true,
						Description: "The hostname of the Grafana Loki instance to which to send events",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"username": schema.StringAttribute{
						Required:    true,
						Description: "The username for authenticating the request to a Grafana Loki instance",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"password": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "The password for authenticating the request to a Grafana Loki instance",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							preserveSensitiveValuePlanModifier{},
						},
					},
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether event log streaming is enabled. Defaults to `false` (disabled).",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ValidateConfig validates the configuration for the resource.
func (r *eventLogStreamingResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data eventLogStreamingModel
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", data.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", data.EnvironmentSlug.ValueString())
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
		resp.Diagnostics.AddError("Invalid destination type", fmt.Sprintf("Unsupported destination type: %s", data.DestinationType.ValueString()))
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *eventLogStreamingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eventLogStreamingModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "destination_type", plan.DestinationType.ValueString())
	tflog.Info(ctx, "Creating event log streaming")

	// Build destination config
	destConfig, diags := r.buildDestinationConfig(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the streaming config
	createResp, err := r.client.EventLogStreaming.Create(ctx, eventlogstreaming.CreateRequest{
		ProjectSlug:       plan.ProjectSlug.ValueString(),
		EnvironmentSlug:   plan.EnvironmentSlug.ValueString(),
		DestinationType:   eventlogstreaming.DestinationType(plan.DestinationType.ValueString()),
		DestinationConfig: destConfig,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create event log streaming", err.Error())
		return
	}

	tflog.Info(ctx, "Created event log streaming")

	// Handle enabled/disabled status - newly created configs are disabled by default
	if !plan.Enabled.IsUnknown() && !plan.Enabled.IsNull() && plan.Enabled.ValueBool() {
		_, err := r.client.EventLogStreaming.Enable(ctx, eventlogstreaming.EnableRequest{
			ProjectSlug:     plan.ProjectSlug.ValueString(),
			EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
			DestinationType: eventlogstreaming.DestinationType(plan.DestinationType.ValueString()),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to enable event log streaming", err.Error())
			return
		}
	}

	// Refresh state from create response (does not overwrite enabled status)
	diags = plan.refreshFromEventLogStreaming(ctx, createResp.EventLogStreamingConfig)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s.%s.%s", plan.ProjectSlug.ValueString(), plan.EnvironmentSlug.ValueString(), plan.DestinationType.ValueString()))
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Read refreshes the Terraform state with the latest data.
func (r *eventLogStreamingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state eventLogStreamingModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "destination_type", state.DestinationType.ValueString())
	tflog.Info(ctx, "Reading event log streaming")

	getResp, err := r.client.EventLogStreaming.Get(ctx, eventlogstreaming.GetRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		DestinationType: eventlogstreaming.DestinationType(state.DestinationType.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to get event log streaming", err.Error())
		return
	}

	tflog.Info(ctx, "Read event log streaming")

	// Use helper to update state from masked API response
	diags = state.refreshFromMaskedEventLogStreaming(ctx, getResp.EventLogStreamingConfig)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eventLogStreamingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan eventLogStreamingModel
	var state eventLogStreamingModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", plan.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", plan.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "destination_type", plan.DestinationType.ValueString())
	tflog.Info(ctx, "Updating event log streaming")

	// Build destination config
	destConfig, diags := r.buildDestinationConfig(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the streaming config
	updateResp, err := r.client.EventLogStreaming.Update(ctx, eventlogstreaming.UpdateRequest{
		ProjectSlug:       plan.ProjectSlug.ValueString(),
		EnvironmentSlug:   plan.EnvironmentSlug.ValueString(),
		DestinationType:   eventlogstreaming.DestinationType(plan.DestinationType.ValueString()),
		DestinationConfig: destConfig,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update event log streaming", err.Error())
		return
	}

	tflog.Info(ctx, "Updated event log streaming")

	// Handle enabled/disabled status changes
	if !plan.Enabled.IsUnknown() && !plan.Enabled.IsNull() && plan.Enabled.ValueBool() != state.Enabled.ValueBool() {
		if plan.Enabled.ValueBool() {
			_, err := r.client.EventLogStreaming.Enable(ctx, eventlogstreaming.EnableRequest{
				ProjectSlug:     plan.ProjectSlug.ValueString(),
				EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
				DestinationType: eventlogstreaming.DestinationType(plan.DestinationType.ValueString()),
			})
			if err != nil {
				resp.Diagnostics.AddError("Failed to enable event log streaming", err.Error())
				return
			}
		} else {
			_, err := r.client.EventLogStreaming.Disable(ctx, eventlogstreaming.DisableRequest{
				ProjectSlug:     plan.ProjectSlug.ValueString(),
				EnvironmentSlug: plan.EnvironmentSlug.ValueString(),
				DestinationType: eventlogstreaming.DestinationType(plan.DestinationType.ValueString()),
			})
			if err != nil {
				resp.Diagnostics.AddError("Failed to disable event log streaming", err.Error())
				return
			}
		}
	}

	// Refresh state from update response (does not overwrite enabled status)
	diags = plan.refreshFromEventLogStreaming(ctx, updateResp.EventLogStreamingConfig)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eventLogStreamingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eventLogStreamingModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "project_slug", state.ProjectSlug.ValueString())
	ctx = tflog.SetField(ctx, "environment_slug", state.EnvironmentSlug.ValueString())
	ctx = tflog.SetField(ctx, "destination_type", state.DestinationType.ValueString())
	tflog.Info(ctx, "Deleting event log streaming")

	_, err := r.client.EventLogStreaming.Delete(ctx, eventlogstreaming.DeleteRequest{
		ProjectSlug:     state.ProjectSlug.ValueString(),
		EnvironmentSlug: state.EnvironmentSlug.ValueString(),
		DestinationType: eventlogstreaming.DestinationType(state.DestinationType.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete event log streaming", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted event log streaming")
}

func (r *eventLogStreamingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: project_slug.environment_slug.destination_type
	parts := strings.Split(req.ID, ".")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'project_slug.environment_slug.destination_type', got: %s", req.ID),
		)
		return
	}

	projectSlug := parts[0]
	environmentSlug := parts[1]
	destinationType := parts[2]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), projectSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_slug"), environmentSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("destination_type"), destinationType)...)
}

func (r *eventLogStreamingResource) buildDestinationConfig(ctx context.Context, model *eventLogStreamingModel) (*eventlogstreaming.DestinationConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	var destConfig eventlogstreaming.DestinationConfig

	destType := model.DestinationType.ValueString()

	switch destType {
	case "DATADOG":
		if model.DatadogConfig.IsNull() || model.DatadogConfig.IsUnknown() {
			diags.AddError("Missing Datadog Configuration", "datadog_config configuration is required when destination_type is DATADOG")
			return &destConfig, diags
		}

		var datadog datadogConfigModel
		diags.Append(model.DatadogConfig.As(ctx, &datadog, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return &destConfig, diags
		}

		destConfig.Datadog = &eventlogstreaming.DatadogConfig{
			Site:   eventlogstreaming.DatadogSite(datadog.Site.ValueString()),
			APIKey: datadog.APIKey.ValueString(),
		}

	case "GRAFANA_LOKI":
		if model.GrafanaLokiConfig.IsNull() || model.GrafanaLokiConfig.IsUnknown() {
			diags.AddError("Missing Grafana Loki Configuration", "grafana_loki_config configuration is required when destination_type is GRAFANA_LOKI")
			return &destConfig, diags
		}

		var grafana grafanaLokiConfigModel
		diags.Append(model.GrafanaLokiConfig.As(ctx, &grafana, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return &destConfig, diags
		}

		destConfig.GrafanaLoki = &eventlogstreaming.GrafanaLokiConfig{
			Hostname: grafana.Hostname.ValueString(),
			Username: grafana.Username.ValueString(),
			Password: grafana.Password.ValueString(),
		}

	default:
		diags.AddError("Invalid Destination Type", fmt.Sprintf("Unsupported destination_type: %s", destType))
	}

	return &destConfig, diags
}
