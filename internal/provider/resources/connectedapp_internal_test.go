package resources

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stytchauth/stytch-go/v18/stytch/consumer/connectedapps"
	capclients "github.com/stytchauth/stytch-go/v18/stytch/consumer/connectedapps/clients"
	"github.com/stytchauth/stytch-go/v18/stytch/stytcherror"
)

func TestConnectedAppBodyRoundTripsAllMutableFields(t *testing.T) {
	commonFields := map[string]any{
		"client_name":                   "name",
		"client_description":            "desc",
		"redirect_urls":                 []string{"https://a.example.com/cb"},
		"post_logout_redirect_urls":     []string{},
		"access_token_expiry_minutes":   int32(5),
		"access_token_custom_audience":  "aud",
		"access_token_template_content": `{"k":"v"}`,
		"id_token_template_content":     "",
		"logo_url":                      "",
	}
	firstPartyWant := map[string]any{
		"full_access_allowed":               true,
		"bypass_consent_for_offline_access": true,
	}
	for key, value := range commonFields {
		firstPartyWant[key] = value
	}

	tests := []struct {
		name       string
		clientType string
		want       map[string]any
	}{
		{"first-party clients carry the first-party-only flags", "first_party", firstPartyWant},
		{"third-party clients omit the first-party-only flags", "third_party", commonFields},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := connectedapps.ConnectedApp{
				ClientID:                      "connected-app-test-123",
				ClientType:                    test.clientType,
				ClientName:                    "name",
				ClientDescription:             "desc",
				RedirectURLs:                  []string{"https://a.example.com/cb"},
				PostLogoutRedirectURLs:        nil,
				FullAccessAllowed:             true,
				AccessTokenExpiryMinutes:      5,
				AccessTokenCustomAudience:     "aud",
				AccessTokenTemplateContent:    `{"k":"v"}`,
				IDTokenTemplateContent:        "",
				LogoURL:                       "",
				BypassConsentForOfflineAccess: true,
			}
			body := connectedAppBody(app)
			if !reflect.DeepEqual(body, test.want) {
				t.Fatalf("got %#v\nwant %#v", body, test.want)
			}
			if len(body) != len(test.want) {
				t.Fatalf("got %d keys, want %d", len(body), len(test.want))
			}
			if _, ok := body["client_type"]; ok {
				t.Fatal("client_type is immutable and must not be sent on update")
			}
		})
	}
}

// connectedAppBody enumerates the mutable fields by hand, so a field added to
// the SDK's UpdateParams would silently be reset to its default on every write
// until someone noticed. Fail here instead, on the next SDK bump.
func TestConnectedAppBodyCoversEveryUpdateParam(t *testing.T) {
	body := connectedAppBody(connectedapps.ConnectedApp{ClientType: "first_party"})

	paramsType := reflect.TypeOf(capclients.UpdateParams{})
	for i := range paramsType.NumField() {
		key := strings.Split(paramsType.Field(i).Tag.Get("json"), ",")[0]
		// client_id travels in the path, not the body.
		if key == "" || key == "-" || key == "client_id" {
			continue
		}
		if _, ok := body[key]; !ok {
			t.Errorf("UpdateParams has %q but connectedAppBody never sends it: a write would reset it to the API default", key)
		}
		delete(body, key)
	}
	for key := range body {
		t.Errorf("connectedAppBody sends %q, which is not an UpdateParams field", key)
	}
}

func TestIsFirstPartyClient(t *testing.T) {
	tests := map[string]bool{
		"first_party":        true,
		"first_party_public": true,
		"third_party":        false,
		"third_party_public": false,
	}
	for clientType, want := range tests {
		if got := isFirstPartyClient(clientType); got != want {
			t.Fatalf("isFirstPartyClient(%q) = %v, want %v", clientType, got, want)
		}
	}
}

func TestParseConnectedAppImportID(t *testing.T) {
	projectSlug, environmentSlug, clientID, err := parseConnectedAppImportID("proj.env.connected-app-test-abc-123")
	if err != nil {
		t.Fatal(err)
	}
	if projectSlug != "proj" || environmentSlug != "env" || clientID != "connected-app-test-abc-123" {
		t.Fatalf("got %q %q %q", projectSlug, environmentSlug, clientID)
	}
	if _, _, _, err := parseConnectedAppImportID("only.two"); err == nil {
		t.Fatal("expected error for malformed ID")
	}
	if _, _, _, err := parseConnectedAppImportID(".env.cid"); err == nil {
		t.Fatal("expected error for an empty segment")
	}
}

func TestOverlayString(t *testing.T) {
	const serverValue = "value-from-server"
	tests := []struct {
		name  string
		plan  types.String
		state types.String
		want  any
	}{
		{"plan set overwrites the server value", types.StringValue("new"), types.StringValue("old"), "new"},
		{"plan set with no prior state still overwrites", types.StringValue("new"), types.StringNull(), "new"},
		{"plan cleared with prior state sends an empty string", types.StringNull(), types.StringValue("old"), ""},
		{"plan and state both null leaves the server value alone", types.StringNull(), types.StringNull(), serverValue},
		{"unknown plan leaves the server value alone", types.StringUnknown(), types.StringValue("old"), serverValue},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := map[string]any{"client_description": serverValue}
			overlayString(body, "client_description", test.plan, test.state)
			if body["client_description"] != test.want {
				t.Fatalf("got %#v, want %#v", body["client_description"], test.want)
			}
			if len(body) != 1 {
				t.Fatalf("overlayString must not add keys, got %#v", body)
			}
		})
	}
}

func TestSetFromStrings(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   []string
	}{
		{"nil becomes a null set", nil, nil},
		{"empty slice becomes a null set", []string{}, nil},
		{"single value is preserved", []string{"https://a.example.com/cb"}, []string{"https://a.example.com/cb"}},
		{
			"multiple values are preserved",
			[]string{"https://b.example.com/cb", "https://a.example.com/cb"},
			[]string{"https://a.example.com/cb", "https://b.example.com/cb"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := setFromStrings(context.Background(), test.values)
			if test.want == nil {
				if !set.IsNull() {
					t.Fatalf("expected a null set, got %#v", set)
				}
				return
			}
			if set.IsNull() {
				t.Fatal("expected a non-null set")
			}
			var got []string
			if diags := set.ElementsAs(context.Background(), &got, false); diags.HasError() {
				t.Fatalf("failed to read elements: %v", diags)
			}
			slices.Sort(got)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestURLListHelpers(t *testing.T) {
	urls := []string{"https://a.example.com", "https://b.example.com"}
	if got := appendURL(urls, "https://a.example.com"); len(got) != 2 {
		t.Fatalf("append existing should be a no-op, got %v", got)
	}
	if got := appendURL(urls, "https://c.example.com"); len(got) != 3 {
		t.Fatalf("expected 3 urls, got %v", got)
	}
	if got := removeURL(urls, "https://a.example.com"); !reflect.DeepEqual(got, []string{"https://b.example.com"}) {
		t.Fatalf("remove failed: %v", got)
	}
	if got := removeURL(urls, "https://missing.example.com"); len(got) != 2 {
		t.Fatalf("remove missing should be a no-op, got %v", got)
	}
	if removeURL([]string{"https://a.example.com"}, "https://a.example.com") == nil {
		t.Fatal("remove to empty must return an empty non-nil slice")
	}
}

func TestParseConnectedAppRedirectURLImportID(t *testing.T) {
	projectSlug, environmentSlug, clientID, urlType, u, err := parseConnectedAppRedirectURLImportID(
		"proj.env.connected-app-test-abc.authorization.https://app.example.com/cb?x=1")
	if err != nil {
		t.Fatal(err)
	}
	if projectSlug != "proj" || environmentSlug != "env" || clientID != "connected-app-test-abc" ||
		urlType != "authorization" || u != "https://app.example.com/cb?x=1" {
		t.Fatalf("got %q %q %q %q %q", projectSlug, environmentSlug, clientID, urlType, u)
	}
	if _, _, _, _, _, err := parseConnectedAppRedirectURLImportID("proj.env.client.bogus.https://x"); err == nil {
		t.Fatal("expected error for invalid type segment")
	}
	if _, _, _, _, _, err := parseConnectedAppRedirectURLImportID("a.b.c.authorization"); err == nil {
		t.Fatal("expected error for missing url segment")
	}
}

func TestIsNotFound(t *testing.T) {
	if !isNotFound(stytcherror.Error{StatusCode: 404}) {
		t.Fatal("404 should be not-found")
	}
	if isNotFound(stytcherror.Error{StatusCode: 500}) {
		t.Fatal("500 is not not-found")
	}
	if isNotFound(errors.New("plain")) {
		t.Fatal("plain error is not not-found")
	}
}

func TestMatchConnectedAppsByName(t *testing.T) {
	apps := []connectedapps.ConnectedApp{
		{ClientID: "connected-app-test-1", ClientName: "Portal"},
		{ClientID: "connected-app-test-2", ClientName: "portal"},
		{ClientID: "connected-app-test-3", ClientName: "Portal"},
		{ClientID: "connected-app-test-4", ClientName: "Other"},
		{ClientID: "connected-app-test-5", ClientName: "Portal Admin"},
	}
	matches := matchConnectedAppsByName(apps, "Portal")
	if len(matches) != 2 || matches[0].ClientID != "connected-app-test-1" || matches[1].ClientID != "connected-app-test-3" {
		t.Fatalf("expected exact-case matches 1 and 3, got %+v", matches)
	}
	if got := matchConnectedAppsByName(apps, "Porta"); len(got) != 0 {
		t.Fatalf("expected no prefix matches, got %+v", got)
	}
	if got := matchConnectedAppsByName(apps, "Missing"); len(got) != 0 {
		t.Fatalf("expected no matches, got %+v", got)
	}
	if got := matchConnectedAppsByName(nil, "Portal"); len(got) != 0 {
		t.Fatalf("expected no matches on nil input, got %+v", got)
	}
}
