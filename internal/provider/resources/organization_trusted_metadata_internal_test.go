package resources

import (
	"context"
	"reflect"
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

	want := map[string]any{
		"grants": map[string]any{
			"version": float64(1),
			"feat":    map[string]any{"digital_twin": map[string]any{"tier": "internal"}},
		},
		"note":   "hello",
		"legacy": nil,
	}
	if !reflect.DeepEqual(body, want) {
		t.Fatalf("got %#v, want %#v", body, want)
	}
	// A key both set and removed must be written, not nulled.
	if body["grants"] == nil {
		t.Fatal("a key present in the set must never be nulled")
	}
}

func TestTrustedMetadataBodyRejectsInvalidJSON(t *testing.T) {
	_, err := trustedMetadataBody(map[string]string{"grants": "{not json"}, nil)
	if err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

func TestJSONSemanticallyEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"key order", `{"a":1,"b":2}`, `{"b":2,"a":1}`, true},
		{"whitespace", `{"a": 1}`, `{"a":1}`, true},
		{"different values", `{"a":1}`, `{"a":2}`, false},
		{"invalid left", ``, `{"a":1}`, false},
		{"array order matters", `[1,2]`, `[2,1]`, false},
	}
	for _, tc := range cases {
		if got := jsonSemanticallyEqual(tc.a, tc.b); got != tc.want {
			t.Errorf("%s: jsonSemanticallyEqual(%q, %q) = %v, want %v", tc.name, tc.a, tc.b, got, tc.want)
		}
	}
}

func TestRemovedKeys(t *testing.T) {
	removed := removedKeys(
		map[string]string{"grants": "{}", "legacy": "{}"},
		map[string]string{"grants": "{}"},
	)
	if !slices.Equal(removed, []string{"legacy"}) {
		t.Fatalf("got %v, want [legacy]", removed)
	}
}

func TestParseOrganizationTrustedMetadataImportID(t *testing.T) {
	projectSlug, environmentSlug, organizationID, keys, err := parseOrganizationTrustedMetadataImportID(
		"proj.live.organization-live-1234.grants,flags")
	if err != nil {
		t.Fatal(err)
	}
	if projectSlug != "proj" || environmentSlug != "live" || organizationID != "organization-live-1234" {
		t.Fatalf("got %s/%s/%s", projectSlug, environmentSlug, organizationID)
	}
	if !slices.Equal(keys, []string{"grants", "flags"}) {
		t.Fatalf("got keys %v", keys)
	}

	_, _, _, keys, err = parseOrganizationTrustedMetadataImportID("proj.live.organization-live-1234")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected no keys, got %v", keys)
	}

	for _, invalid := range []string{"proj.live", "proj..org", "proj.live.org.a,,b"} {
		if _, _, _, _, err := parseOrganizationTrustedMetadataImportID(invalid); err == nil {
			t.Fatalf("expected an error for %q", invalid)
		}
	}
}
