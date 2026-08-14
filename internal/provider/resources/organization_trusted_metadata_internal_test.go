package resources

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

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
			"grants": `{"version":1,"feat":{"digital_twin":{"tier":"internal"}}}`,
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

func TestTrustedMetadataBodyRejectsInvalidAndNullJSON(t *testing.T) {
	if _, err := trustedMetadataBody(map[string]string{"grants": "{not json"}, nil); err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
	if _, err := trustedMetadataBody(map[string]string{"grants": " null "}, nil); err == nil {
		t.Fatal("expected an error for a JSON null value")
	}
}

func TestCanonicalJSONDoesNotEscapeHTML(t *testing.T) {
	canonical, err := canonicalJSON("<a&b>")
	if err != nil {
		t.Fatal(err)
	}
	if canonical != `"<a&b>"` {
		t.Fatalf("got %q", canonical)
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
