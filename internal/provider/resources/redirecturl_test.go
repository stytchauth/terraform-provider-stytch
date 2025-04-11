package resources_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/stytch-management-go/v2/pkg/models/redirecturls"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func redirectType(typ redirecturls.RedirectType, isDefault bool) string {
	return fmt.Sprintf(`{type = "%s", is_default = %t}`, typ, isDefault)
}

func redirectURLResource(validTypes ...string) string {
	types := strings.Join(validTypes, ", ")
	return fmt.Sprintf(`
resource "stytch_project" "test_project" {
  name     = "test"
  vertical = "CONSUMER"
}

resource "stytch_redirect_url" "test" {
  project_id = stytch_project.test_project.test_project_id
  url        = "http://localhost:3000/consumer"
  valid_types = [%s]
}
`, types)
}

func TestAccRedirectURLResource(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		redirectTypes []redirecturls.URLRedirectType
	}{
		{name: "login-only", redirectTypes: []redirecturls.URLRedirectType{
			{Type: redirecturls.RedirectTypeLogin, IsDefault: true},
		}},
		{name: "multiple", redirectTypes: []redirecturls.URLRedirectType{
			{Type: redirecturls.RedirectTypeLogin, IsDefault: true},
			{Type: redirecturls.RedirectTypeSignup, IsDefault: false},
			{Type: redirecturls.RedirectTypeResetPassword, IsDefault: false},
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			checks := []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_redirect_url.test", "url", "http://localhost:3000/consumer"),
			}
			var typeStrings []string
			for _, typ := range testCase.redirectTypes {
				checks = append(checks, resource.TestCheckTypeSetElemNestedAttrs(
					"stytch_redirect_url.test", "valid_types.*", map[string]string{
						"type":       string(typ.Type),
						"is_default": strconv.FormatBool(typ.IsDefault),
					}))
				typeStrings = append(typeStrings, redirectType(typ.Type, typ.IsDefault))
			}
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + redirectURLResource(typeStrings...),
						Check:  resource.ComposeAggregateTestCheckFunc(checks...),
					},
					{
						// Update and Read testing - change to *only* signup redirect
						Config: testutil.ProviderConfig + redirectURLResource(redirectType(redirecturls.RedirectTypeSignup, true)),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_redirect_url.test", "url", "http://localhost:3000/consumer"),
							resource.TestCheckResourceAttr("stytch_redirect_url.test", "valid_types.0.type", string(redirecturls.RedirectTypeSignup)),
							resource.TestCheckResourceAttr("stytch_redirect_url.test", "valid_types.0.is_default", "true"),
						),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}
}
