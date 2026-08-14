package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
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
	"github.com/stytchauth/stytch-go/v18/stytch"
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
	Force           types.Bool   `tfsdk:"force"`
	TrustedMetadata types.Map    `tfsdk:"trusted_metadata"`
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
	if !providerClients.ExperimentalEnabled {
		resp.Diagnostics.AddError(experimentalGateError("stytch_organization_trusted_metadata"))
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
			"The entire trusted_metadata object of a B2B organization, managed authoritatively: the object holds exactly " +
			"the top-level keys declared here, out-of-band writes surface as plan diffs, keys removed from the " +
			"configuration (or present remotely but not declared) are deleted on apply by writing an explicit null, and " +
			"destroy deletes every key in state. Trusted metadata must therefore have a single writer - do not combine " +
			"this resource with application code writing to the same organization's trusted_metadata, and never declare " +
			"two of these resources for one organization. Creation refuses an organization that already has " +
			"trusted_metadata unless force is set; import instead to adopt existing content (the first plan after import " +
			"may show a formatting-only diff as the configuration's JSON formatting replaces the imported canonical form " +
			"- one harmless apply, then it never recurs). The organization itself is never created or deleted. " +
			"Authentication uses a project secret for the " +
			"environment - create one with the stytch_secret resource; importing requires that secret in the " +
			"STYTCH_IMPORT_PROJECT_SECRET environment variable, because Terraform provides no configuration values " +
			"during import - and so does the first plan afterwards, which refreshes from a state that does not yet " +
			"carry project_secret. Concurrent applies within one run are serialized per organization (the API offers " +
			"no compare-and-swap).",
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
				Description: "The ID of the organization whose trusted_metadata is managed.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"force": schema.BoolAttribute{
				Optional: true,
				Description: "Allow creation to take ownership of an organization that already has trusted_metadata, " +
					"overwriting it with the configured object (keys not declared here are deleted). Defaults to false, " +
					"which makes creation refuse such organizations - importing is the sanctioned way to adopt existing " +
					"content.",
			},
			"trusted_metadata": schema.MapAttribute{
				Required:    true,
				ElementType: jsontypes.NormalizedType{},
				Description: "The organization's complete trusted_metadata object: one entry per top-level key, each value " +
					"the key's content as a JSON document (use jsonencode). The organization's trusted_metadata is made to " +
					"hold exactly these keys.",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.LengthAtLeast(1)),
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
	if config.TrustedMetadata.IsNull() || config.TrustedMetadata.IsUnknown() {
		return
	}
	// JSON validity is enforced by the element type; a literal null needs its
	// own rejection because the API interprets null as key deletion.
	for key, value := range config.TrustedMetadata.Elements() {
		normalized, ok := value.(jsontypes.Normalized)
		if !ok || normalized.IsUnknown() {
			continue
		}
		if normalized.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("trusted_metadata").AtMapKey(key),
				"Null value",
				fmt.Sprintf("The value for key %q is null. Remove the key from the map instead of setting it to null.", key),
			)
			continue
		}
		if strings.TrimSpace(normalized.ValueString()) == "null" {
			resp.Diagnostics.AddAttributeError(
				path.Root("trusted_metadata").AtMapKey(key),
				"Null JSON value",
				fmt.Sprintf("The value for key %q is JSON null, which the Stytch API interprets as deleting the key. Remove the key from the map instead.", key),
			)
		}
	}
}

// trustedMetadataBody builds the update payload: the configured JSON verbatim
// (json.RawMessage, so numbers survive without a float64 round-trip) for every
// declared key, and an explicit null for each removed key - the API's only way
// to delete a top-level key.
func trustedMetadataBody(set map[string]string, removed []string) (map[string]any, error) {
	body := make(map[string]any, len(set)+len(removed))
	for key, value := range set {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("the value for key %q is empty; every key needs a JSON document (a null map element is not allowed)", key)
		}
		if !json.Valid([]byte(value)) {
			return nil, fmt.Errorf("the value for key %q is not valid JSON", key)
		}
		if strings.TrimSpace(value) == "null" {
			return nil, fmt.Errorf("the value for key %q is JSON null, which would delete the key", key)
		}
		body[key] = json.RawMessage(value)
	}
	for _, key := range removed {
		if _, ok := set[key]; !ok {
			body[key] = nil
		}
	}
	return body, nil
}

// canonicalJSON re-encodes a raw API value deterministically: decoding with
// UseNumber keeps every number literal verbatim (a float64 round-trip would
// corrupt 1.0, 1e2, and integers above 2^53), and HTML escaping is disabled
// so values containing <, >, or & match the configuration text.
func canonicalJSON(raw json.RawMessage) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// rawOrganization carries trusted_metadata as raw JSON. The typed SDK decodes
// the object into map[string]any, turning every number into a float64 and
// corrupting literals the float64 round-trip cannot represent - before
// Terraform ever sees them.
type rawOrganization struct {
	OrganizationID         string                     `json:"organization_id"`
	OrganizationName       string                     `json:"organization_name"`
	OrganizationSlug       string                     `json:"organization_slug"`
	OrganizationExternalID string                     `json:"organization_external_id"`
	TrustedMetadata        map[string]json.RawMessage `json:"trusted_metadata"`
}

// The identifier may be an organization ID, slug, or external ID - the API
// accepts all three in the path. Errors from the management API (project-ID
// resolution) are intentionally distinct types from stytch-go's, so isNotFound
// matches only a missing organization, never a missing environment.
func getOrganizationRaw(ctx context.Context, c stytch.Client, identifier string) (*rawOrganization, error) {
	var resp struct {
		Organization rawOrganization `json:"organization"`
	}
	err := c.NewRequest(ctx, stytch.RequestParams{
		Method:  "GET",
		Path:    fmt.Sprintf("/v1/b2b/organizations/%s", url.PathEscape(identifier)),
		V:       &resp,
		Headers: map[string][]string{},
	})
	if err != nil {
		return nil, err
	}
	return &resp.Organization, nil
}

func removedKeys(previous map[string]string, current map[string]string) []string {
	var removed []string
	for key := range previous {
		if _, ok := current[key]; !ok {
			removed = append(removed, key)
		}
	}
	sort.Strings(removed)
	return removed
}

func parseOrganizationTrustedMetadataImportID(id string) (projectSlug, environmentSlug, organizationID string, err error) {
	parts := strings.SplitN(id, ".", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("the ID must be in the format <project_slug>.<environment_slug>.<organization_id>, got %q", id)
	}
	return parts[0], parts[1], parts[2], nil
}

func (m *organizationTrustedMetadataModel) setID() {
	m.ID = types.StringValue(fmt.Sprintf("%s.%s.%s",
		m.ProjectSlug.ValueString(), m.EnvironmentSlug.ValueString(), m.OrganizationID.ValueString()))
}

func (m *organizationTrustedMetadataModel) metadataValues() (map[string]string, error) {
	values := map[string]string{}
	if m.TrustedMetadata.IsNull() || m.TrustedMetadata.IsUnknown() {
		return values, nil
	}
	for key, value := range m.TrustedMetadata.Elements() {
		normalized, ok := value.(jsontypes.Normalized)
		if !ok {
			return nil, fmt.Errorf("unexpected element type %T for key %q", value, key)
		}
		values[key] = normalized.ValueString()
	}
	return values, nil
}

func metadataMapValue(ctx context.Context, values map[string]string) (types.Map, error) {
	elements := make(map[string]jsontypes.Normalized, len(values))
	for key, value := range values {
		elements[key] = jsontypes.NewNormalizedValue(value)
	}
	mapValue, diags := types.MapValueFrom(ctx, jsontypes.NormalizedType{}, elements)
	if diags.HasError() {
		return types.MapNull(jsontypes.NormalizedType{}), fmt.Errorf("building trusted_metadata map: %v", diags.Errors())
	}
	return mapValue, nil
}

func (r *organizationTrustedMetadataResource) getOrganization(ctx context.Context, m organizationTrustedMetadataModel) (*rawOrganization, error) {
	client, err := r.projectAPI.ForB2BEnvironment(ctx, m.ProjectSlug.ValueString(), m.EnvironmentSlug.ValueString(), m.ProjectSecret.ValueString())
	if err != nil {
		return nil, err
	}
	return getOrganizationRaw(ctx, client.Organizations.C, m.OrganizationID.ValueString())
}

func (r *organizationTrustedMetadataResource) write(ctx context.Context, m organizationTrustedMetadataModel, removed []string) error {
	set, err := m.metadataValues()
	if err != nil {
		return err
	}
	body, err := trustedMetadataBody(set, removed)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}

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
	tflog.Info(ctx, "Writing organization trusted_metadata")

	unlock := r.projectAPI.LockClient(plan.OrganizationID.ValueString())
	defer unlock()

	org, err := r.getOrganization(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get organization", err.Error())
		return
	}

	// Creation means introducing trusted_metadata, not adopting it: existing
	// content is refused so a first apply can never silently destroy data
	// written by another party. Import adopts; force overwrites.
	var existing []string
	for key := range org.TrustedMetadata {
		existing = append(existing, key)
	}
	sort.Strings(existing)
	if len(existing) > 0 && !plan.Force.ValueBool() {
		resp.Diagnostics.AddError(
			"Organization already has trusted_metadata",
			fmt.Sprintf("Organization %s already has trusted_metadata keys (%s). Import the resource to adopt the "+
				"existing content, or set force = true to overwrite it (keys not in the configuration are deleted).",
				plan.OrganizationID.ValueString(), strings.Join(existing, ", ")),
		)
		return
	}

	planned, err := plan.metadataValues()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read trusted_metadata from plan", err.Error())
		return
	}
	remote := make(map[string]string, len(org.TrustedMetadata))
	for key := range org.TrustedMetadata {
		remote[key] = ""
	}

	if err := r.write(ctx, plan, removedKeys(remote, planned)); err != nil {
		resp.Diagnostics.AddError("Failed to write organization trusted_metadata", err.Error())
		return
	}

	tflog.Info(ctx, "Wrote organization trusted_metadata")

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

	org, err := r.getOrganization(ctx, state)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to get organization", err.Error())
		return
	}

	// The resource owns the whole object, so every remote top-level key is
	// reflected into state: out-of-band additions and changes become plan
	// diffs. The Normalized element type suppresses formatting-only drift.
	refreshed := make(map[string]string, len(org.TrustedMetadata))
	for key, remote := range org.TrustedMetadata {
		canonical, err := canonicalJSON(remote)
		if err != nil {
			resp.Diagnostics.AddError("Failed to re-encode remote trusted_metadata value", err.Error())
			return
		}
		refreshed[key] = canonical
	}
	state.TrustedMetadata, err = metadataMapValue(ctx, refreshed)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build trusted_metadata state", err.Error())
		return
	}

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

	planned, err := plan.metadataValues()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read trusted_metadata from plan", err.Error())
		return
	}
	previous, err := state.metadataValues()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read trusted_metadata from state", err.Error())
		return
	}

	ctx = tflog.SetField(ctx, "organization_id", plan.OrganizationID.ValueString())
	tflog.Info(ctx, "Updating organization trusted_metadata")

	unlock := r.projectAPI.LockClient(plan.OrganizationID.ValueString())
	defer unlock()

	if err := r.write(ctx, plan, removedKeys(previous, planned)); err != nil {
		resp.Diagnostics.AddError("Failed to update organization trusted_metadata", err.Error())
		return
	}

	tflog.Info(ctx, "Updated organization trusted_metadata")

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

	owned, err := state.metadataValues()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read trusted_metadata from state", err.Error())
		return
	}
	if len(owned) == 0 {
		return
	}
	body := make(map[string]any, len(owned))
	for key := range owned {
		body[key] = nil
	}

	ctx = tflog.SetField(ctx, "organization_id", state.OrganizationID.ValueString())
	tflog.Info(ctx, "Deleting organization trusted_metadata")

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
		resp.Diagnostics.AddError("Failed to delete organization trusted_metadata", err.Error())
		return
	}

	tflog.Info(ctx, "Deleted organization trusted_metadata")
}

func (r *organizationTrustedMetadataResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectSlug, environmentSlug, organizationID, err := parseOrganizationTrustedMetadataImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_slug"), projectSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_slug"), environmentSlug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), organizationID)...)
	// The Read that follows import populates trusted_metadata with the
	// organization's current content.
	seed, err := metadataMapValue(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build trusted_metadata state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("trusted_metadata"), seed)...)
}
