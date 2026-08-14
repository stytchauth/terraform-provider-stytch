package resources

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/clients"
)

func TestExperimentalGateBlocksConfigure(t *testing.T) {
	r := &organizationTrustedMetadataResource{}
	var resourceResp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: &clients.Clients{}}, &resourceResp)
	if !resourceResp.Diagnostics.HasError() {
		t.Fatal("expected the resource to refuse configuration without the experimental flag")
	}
	if detail := resourceResp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, ExperimentalEnvVar) {
		t.Fatalf("the diagnostic must name the environment variable, got %q", detail)
	}

	d := &organizationDataSource{}
	var dataSourceResp datasource.ConfigureResponse
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: &clients.Clients{}}, &dataSourceResp)
	if !dataSourceResp.Diagnostics.HasError() {
		t.Fatal("expected the data source to refuse configuration without the experimental flag")
	}
	if detail := dataSourceResp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, ExperimentalEnvVar) {
		t.Fatalf("the diagnostic must name the environment variable, got %q", detail)
	}
}

func TestExperimentalGateAllowsConfigureWhenEnabled(t *testing.T) {
	r := &organizationTrustedMetadataResource{}
	var resourceResp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: &clients.Clients{ExperimentalEnabled: true}}, &resourceResp)
	if resourceResp.Diagnostics.HasError() {
		t.Fatalf("expected configuration to succeed, got %v", resourceResp.Diagnostics.Errors())
	}

	d := &organizationDataSource{}
	var dataSourceResp datasource.ConfigureResponse
	d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: &clients.Clients{ExperimentalEnabled: true}}, &dataSourceResp)
	if dataSourceResp.Diagnostics.HasError() {
		t.Fatalf("expected configuration to succeed, got %v", dataSourceResp.Diagnostics.Errors())
	}
}

func TestTrustedMetadataBody(t *testing.T) {
	body, err := trustedMetadataBody(
		map[string]string{
			"grants": `{"plan":"enterprise","limits":{"seats":50}}`,
			"note":   `"hello"`,
		},
		[]string{"legacy", "grants"},
	)
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["note"] != "hello" {
		t.Fatalf("note round-tripped to %#v", decoded["note"])
	}
	if decoded["legacy"] != nil {
		t.Fatalf("removed key must be null, got %#v", decoded["legacy"])
	}
	// A key both set and removed must be written, not nulled.
	if decoded["grants"] == nil {
		t.Fatal("a key present in the set must never be nulled")
	}
}

func TestTrustedMetadataBodyPreservesLargeNumbers(t *testing.T) {
	body, err := trustedMetadataBody(map[string]string{"external_id": "12345678901234567890"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "12345678901234567890") {
		t.Fatalf("large integer must survive verbatim, got %s", encoded)
	}
}

func TestTrustedMetadataBodyRejectsInvalidNullAndEmptyJSON(t *testing.T) {
	if _, err := trustedMetadataBody(map[string]string{"grants": "{not json"}, nil); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
	if _, err := trustedMetadataBody(map[string]string{"grants": " null "}, nil); err == nil {
		t.Fatal("expected an error for a JSON null value")
	}
	if _, err := trustedMetadataBody(map[string]string{"grants": ""}, nil); err == nil {
		t.Fatal("expected an error for an empty value")
	}
}

func TestCanonicalJSONDoesNotEscapeHTML(t *testing.T) {
	canonical, err := canonicalJSON(json.RawMessage(`"<a&b>"`))
	if err != nil {
		t.Fatal(err)
	}
	if canonical != `"<a&b>"` {
		t.Fatalf("got %q", canonical)
	}
}

// The API returns trusted_metadata verbatim as written; canonicalJSON must
// stay semantically equal to the original literal for every value the
// configuration could contain - a float64 round-trip would corrupt several of
// these and produce a permanent diff loop.
func TestCanonicalJSONPreservesSemanticEquality(t *testing.T) {
	literals := []string{
		`1.0`,
		`1e2`,
		`0.1`,
		`12345678901234567890`,
		`{"version": 1.0, "big": 12345678901234567890, "nested": {"exp": 2.5e3}}`,
		`"<a&b>"`,
		`{"b": 1, "a": [1.0, 2]}`,
	}
	for _, literal := range literals {
		canonical, err := canonicalJSON(json.RawMessage(literal))
		if err != nil {
			t.Fatalf("%s: %v", literal, err)
		}
		equal, diags := jsontypes.NewNormalizedValue(literal).StringSemanticEquals(
			context.Background(), jsontypes.NewNormalizedValue(canonical))
		if diags.HasError() {
			t.Fatalf("%s: %v", literal, diags.Errors())
		}
		if !equal {
			t.Fatalf("canonicalJSON(%s) = %s is not semantically equal to its input", literal, canonical)
		}
	}
}

func TestMetadataJSON(t *testing.T) {
	rendered, err := metadataJSON(nil)
	if err != nil {
		t.Fatal(err)
	}
	if rendered != "{}" {
		t.Fatalf("nil metadata must render as an empty object, got %q", rendered)
	}

	rendered, err = metadataJSON(map[string]json.RawMessage{
		"big":  json.RawMessage(`12345678901234567890`),
		"html": json.RawMessage(`"<a&b>"`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, "12345678901234567890") || !strings.Contains(rendered, `"<a&b>"`) {
		t.Fatalf("values must pass through verbatim, got %s", rendered)
	}
}

func TestRemovedKeys(t *testing.T) {
	removed := removedKeys(
		map[string]string{"legacy": "{}", "grants": "{}", "old": "{}"},
		map[string]string{"grants": "{}"},
	)
	if !slices.Equal(removed, []string{"legacy", "old"}) {
		t.Fatalf("got %v, want sorted [legacy old]", removed)
	}
}

func TestParseOrganizationTrustedMetadataImportID(t *testing.T) {
	projectSlug, environmentSlug, organizationID, err := parseOrganizationTrustedMetadataImportID(
		"proj.live.organization-live-1234")
	if err != nil {
		t.Fatal(err)
	}
	if projectSlug != "proj" || environmentSlug != "live" || organizationID != "organization-live-1234" {
		t.Fatalf("got %s/%s/%s", projectSlug, environmentSlug, organizationID)
	}

	for _, invalid := range []string{"proj.live", "proj..org", "..", ""} {
		if _, _, _, err := parseOrganizationTrustedMetadataImportID(invalid); err == nil {
			t.Fatalf("expected an error for %q", invalid)
		}
	}
}

func TestMetadataMapValueRoundTrip(t *testing.T) {
	mapValue, err := metadataMapValue(context.Background(), map[string]string{"grants": `{"a":1}`})
	if err != nil {
		t.Fatal(err)
	}
	model := organizationTrustedMetadataModel{TrustedMetadata: mapValue}
	values, err := model.metadataValues()
	if err != nil {
		t.Fatal(err)
	}
	if values["grants"] != `{"a":1}` {
		t.Fatalf("round trip lost the value, got %#v", values)
	}
}
