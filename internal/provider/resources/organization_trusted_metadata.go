package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stytchauth/stytch-go/v18/stytch/b2b/organizations"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/clients"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/projectapi"
)

var (
	_ resource.Resource                   = &organizationTrustedMetadataResource{}
	_ resource.ResourceWithConfigure      = &organizationTrustedMetadataResource{}
	_ resource.ResourceWithImportState    = &organizationTrustedMetadataResource{}
	_ resource.ResourceWithValidateConfig = &organizationTrustedMetadataResource{}
)

func NewOrganizationTrustedMetadataResource() resource.Resource {
	return &organizationTrustedMetadataResource{}
}

type organizationTrustedMetadataResource struct {
	projectAPI *projectapi.Factory
}

type organizationTrustedMetadataModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectSlug     types.String `tfsdk:"project_slug"`
	EnvironmentSlug types.String `tfsdk:"environment_slug"`
	ProjectSecret   types.String `tfsdk:"project_secret"`
	OrganizationID  types.String `tfsdk:"organization_id"`
	Keys            types.Map    `tfsdk:"keys"`
	LastUpdated     types.String `tfsdk:"last_updated"`
}

func (r *organizationTrustedMetadataResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	providerClients, ok := req.ProviderData.(*clients.Clients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *clients.Clients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.projectAPI = providerClients.ProjectAPI
}

func (r *organizationTrustedMetadataResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_trusted_metadata"
}

func (r *organizationTrustedMetadataResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "**Experimental**: this resource is available only when the STYTCH_PROVIDER_USE_EXPERIMENTAL_RESOURCES " +
			"environment variable is set to 1, and its schema may change in a future release without a major version bump. " +
			"A declared set of top-level trusted_metadata keys on a B2B organization, managed additively: the " +
			"provider writes only the keys listed here and leaves every other key (typically application-written data) " +
			"untouched, matching the metadata API's top-level merge semantics. Removing a key from the map, or destroying " +
			"the resource, deletes that key from the organization by writing an explicit null. The organization itself is " +
			"never created or deleted. Authentication uses a project secret for the environment - create one with the " +
			"stytch_secret resource; importing requires that secret in the STYTCH_IMPORT_PROJECT_SECRET environment " +
			"variable, because Terraform provides no configuration values during import - and so does the first plan " +
			"afterwards, which refreshes from a state that does not yet carry project_secret. Concurrent applies within " +
			"one run are serialized per organization. Avoid concurrent out-of-band writes to the same keys (the API " +
			"offers no compare-and-swap).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "A computed ID field used for Terraform resource management (format: project_slug.environment_slug.organization_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the project to which the organization belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_slug": schema.StringAttribute{
				Required:    true,
				Description: "The slug of the environment to which the organization belongs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_secret": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "A project secret for the environment, used to authenticate against the project-level Stytch API.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"organization_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the organization whose trusted_metadata keys are managed.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"keys": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "The top-level trusted_metadata keys this resource owns. Each value is the key's content as a " +
					"JSON document (use jsonencode). Keys not listed here are never touched. A key name containing a comma " +
					"cannot be listed in an import ID.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the last Terraform update.",
			},
		},
	}
}

func (r *organizationTrustedMetadataResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config organizationTrustedMetadataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.Keys.IsNull() || config.Keys.IsUnknown() {
		return
	}
	for key, value := range config.Keys.Elements() {
		str, ok := value.(types.String)
		if !ok || str.IsNull() || str.IsUnknown() {
			continue
		}
		if !json.Valid([]byte(str.ValueString())) {
			resp.Diagnostics.AddAttributeError(
				path.Root("keys").AtMapKey(key),
				"Invalid JSON value",
				fmt.Sprintf("The value for key %q must be a valid JSON document (use jsonencode).", key),
			)
			return
		}
	}
}

// trustedMetadataBody builds the partial update payload: parsed JSON for every
// owned key, and an explicit null for each removed key - the API's only way to
// delete a top-level key.
func trustedMetadataBody(set map[string]string, removed []string) (map[string]any, error) {
	body := make(map[string]any, len(set)+len(removed))
	for key, value := range set {
		var parsed any
		if err := json.Unmarshal([]byte(value), &parsed); err != nil {
			return nil, fmt.Errorf("the value for key %q is not valid JSON: %w", key, err)
		}
		body[key] = parsed
	}
	for _, key := range removed {
		if _, ok := set[key]; !ok {
			body[key] = nil
		}
	}
	return body, nil
}

// jsonSemanticallyEqual reports whether two strings encode the same JSON
// value, so that formatting and key-order differences between the config and
// the API's re-marshaled response do not produce perpetual diffs.
func jsonSemanticallyEqual(a, b string) bool {
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

func removedKeys(previous, current map[string]string) []string {
	var removed []string
	for key := range previous {
		if _, ok := current[key]; !ok {
			removed = append(removed, key)
		}
	}
	return removed
}

func parseOrganizationTrustedMetadataImportID(id string) (projectSlug, environmentSlug, organizationID string, keys []string, err error) {
	parts := strings.SplitN(id, ".", 4)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", nil, fmt.Errorf("the ID must be in the format <project_slug>.<environment_slug>.<organization_id>[.<key1,key2,...>], got %q", id)
	}
	if len(parts) == 4 && parts[3] != "" {
		for _, key := range strings.Split(parts[3], ",") {
			if key == "" {
				return "", "", "", nil, fmt.Errorf("the keys segment contains an empty key name, got %q", id)
			}
			keys = append(keys, key)
		}
	}
	return parts[0], parts[1], parts[2], keys, nil
}

func (m *organizationTrustedMetadataModel) setID() {
	m.ID = types.StringValue(fmt.Sprintf("%s.%s.%s",
		m.ProjectSlug.ValueString(), m.EnvironmentSlug.ValueString(), m.OrganizationID.ValueString()))
}

func (m *organizationTrustedMetadataModel) keyValues(ctx context.Context) (map[string]string, error) {
	keys := map[string]string{}
	diags := m.Keys.ElementsAs(ctx, &keys, false)
	if diags.HasError() {
		return nil, fmt.Errorf("reading keys map: %v", diags.Errors())
	}
	return keys, nil
}

func (r *organizationTrustedMetadataResource) write(ctx context.Context, m organizationTrustedMetadataModel, removed []string) error {
	set, err := m.keyValues(ctx)
	if err != nil {
		return err
	}
	body, err := trustedMetadataBody(set, removed)
	if err != nil {
		return err
	}

	unlock := r.projectAPI.LockClient(m.OrganizationID.ValueString())
	defer unlock()

	client, err := r.projectAPI.ForB2BEnvironment(ctx, m.ProjectSlug.ValueString(), m.EnvironmentSlug.ValueString(), m.ProjectSecret.ValueString())
	if err != nil {
		return err
	}

	_, err = client.Organizations.Update(ctx, &organizations.UpdateParams{
		OrganizationID:  m.OrganizationID.ValueString(),
		TrustedMetadata: body,
	})
	return err
}

func (r *organizationTrustedMetadataResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationTrustedMetadataModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "organization_id", plan.OrganizationID.ValueString())
	tflog.Info(ctx, "Writing organization trusted_metadata keys")

	if err := r.write(ctx, plan, nil); err != nil {
		resp.Diagnostics.AddError("Failed to write organization trusted_metadata keys", err.Error())
		return
	}

	tflog.Info(ctx, "Wrote organization trusted_metadata keys")

	plan.setID()
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *organizationTrustedMetadataResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationTrustedMetadataModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.projectAPI.ForB2BEnvironment(ctx, state.ProjectSlug.ValueString(), state.EnvironmentSlug.ValueString(), state.ProjectSecret.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to build project API client", err.Error())
		return
	}

	getResp, err := client.Organizations.Get(ctx, &organizations.GetParams{OrganizationID: state.OrganizationID.ValueString()})
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to get organization", err.Error())
		return
	}

	owned, err := state.keyValues(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read keys from state", err.Error())
		return
	}

	refreshed := map[string]string{}
	for key, stateValue := range owned {
		remote, ok := getResp.Organization.TrustedMetadata[key]
		if !ok {
			continue
		}
		canonical, err := json.Marshal(remote)
		if err != nil {
			resp.Diagnostics.AddError("Failed to re-encode remote trusted_metadata value", err.Error())
			return
		}
		// The state's own formatting is kept whenever it encodes the same value,
		// so config written with jsonencode does not diff against the API's
		// re-marshaled representation.
		if jsonSemanticallyEqual(stateValue, string(canonical)) {
			refreshed[key] = stateValue
		} else {
			refreshed[key] = string(canonical)
		}
	}

	keysValue, diags := types.MapValueFrom(ctx, types.StringType, refreshed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Keys = keysValue

	state.setID()
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *organizationTrustedMetadataResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationTrustedMetadataModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state organizationTrustedMetadataModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planned, err := plan.keyValues(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read keys from plan", err.Error())
		return
	}
	previous, err := state.keyValues(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read keys from state", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "organization_id", plan.OrganizationID.ValueString())
	tflog.Info(ctx, "Updating organization trusted_metadata keys")

	if err := r.write(ctx, plan, removedKeys(previous, planned)); err != nil {
		resp.Diagnostics.AddError("Failed to update organization trusted_metadata keys", err.Error())
		return
	}

	tflog.Info(ctx, "Updated organization trusted_metadata keys")

	plan.setID()
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *organizationTrustedMetadataResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationTrustedMetadataModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owned, err := state.keyValues(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read keys from state", err.Error())
		return
	}
	body := make(map[string]any, len(owned))
	for key := range owned {
		body[key] = nil
	}

	ctx = tflog.SetField(ctx, "organization_id", state.OrganizationID.ValueString())
	tflog.Info(ctx, "Removing organization trusted_metadata keys")

	unlock := r.projectAPI.LockClient(state.OrganizationID.ValueString())
	defer unlock()

	client, err := r.projectAPI.ForB2BEnvironment(ctx, state.ProjectSlug.ValueString(), state.EnvironmentSlug.ValueString(), state.ProjectSecret.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to build project API client", err.Error())
		return
	}

	_, err = client.Organizations.Update(ctx, &organizations.UpdateParams{
		OrganizationID:  state.OrganizationID.ValueString(),
		TrustedMetadata: body,
	})
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Failed to remove organization trusted_metadata keys", err.Error())
		return
	}

	tflog.Info(ctx, "Removed organization trusted_metadata keys")
}

func (r *organizationTrustedMetadataResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectSlug, environmentSlug, organizationID, keys, err := parseOrganizationTrustedMetadataImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	// Imported keys are seeded with empty values; the Read that follows import
	// replaces each with the organization's current content for that key.
	seeded := make(map[string]string, len(keys))
	for _, key := range keys {
		seeded[key] = ""
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), projectSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_slug"), environmentSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), organizationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("keys"), seeded)...)
}
