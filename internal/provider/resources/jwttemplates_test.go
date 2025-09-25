package resources_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/jwttemplates"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

// TestAccJWTTemplateResource performs acceptance tests for the stytch_jwt_template resource.
func TestAccJWTTemplateResource(t *testing.T) {
	const resourceName = "stytch_jwt_template.test"

	for _, tc := range []struct {
		Name               string
		TemplateType       jwttemplates.TemplateType
		InitialContent     string
		InitialHasAudience bool
		InitialAudience    string
		UpdateContent      string
		UpdateAudience     string
	}{
		{
			Name:               "session",
			TemplateType:       jwttemplates.TemplateTypeSession,
			InitialContent:     `{ "role": {{ user.trusted_metadata.role }} }`,
			InitialHasAudience: false,
			UpdateContent:      `{ "role": {{ user.trusted_metadata.role }}, "scope": "openid profile email" }`,
			UpdateAudience:     "test-aud",
		},
		{
			Name:               "m2m",
			TemplateType:       jwttemplates.TemplateTypeM2M,
			InitialContent:     `{ "role": {{ client.trusted_metadata.role }} }`,
			InitialHasAudience: true,
			InitialAudience:    "my-audience",
			UpdateContent:      `{ "tier": {{ client.trusted_metadata.subscription_tier }} }`,
			UpdateAudience:     "other-aud",
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			// Build initial Terraform configuration
			initialConfig := testutil.ConsumerProjectConfig + fmt.Sprintf(`
resource "stytch_jwt_template" "test" {
  project_id       = stytch_project.project.test_project_id
  template_type    = "%s"
  template_content = "%s"
`, string(tc.TemplateType), strings.ReplaceAll(tc.InitialContent, `"`, `\"`))
			if tc.InitialHasAudience {
				initialConfig += fmt.Sprintf("  custom_audience = \"%s\"\n", tc.InitialAudience)
			}
			initialConfig += "}\n"

			// Initial checks
			initialChecks := []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "template_type", string(tc.TemplateType)),
				resource.TestCheckResourceAttr(resourceName, "template_content", tc.InitialContent),
			}
			if tc.InitialHasAudience {
				initialChecks = append(initialChecks,
					resource.TestCheckResourceAttr(resourceName, "custom_audience", tc.InitialAudience),
				)
			}

			// Build update Terraform configuration
			updateConfig := testutil.ConsumerProjectConfig + fmt.Sprintf(`
resource "stytch_jwt_template" "test" {
  project_id       = stytch_project.project.test_project_id
  template_type    = "%s"
  template_content = "%s"
  custom_audience  = "%s"
}
`, string(tc.TemplateType), strings.ReplaceAll(tc.UpdateContent, `"`, `\"`), tc.UpdateAudience)

			// Update checks
			updateChecks := []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "template_content", tc.UpdateContent),
				resource.TestCheckResourceAttr(resourceName, "custom_audience", tc.UpdateAudience),
			}

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + initialConfig,
						Check:  resource.ComposeAggregateTestCheckFunc(initialChecks...),
					},
					{
						// Import state testing
						ResourceName:            resourceName,
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"last_updated"},
					},
					{
						// Update and Read testing
						Config: testutil.ProviderConfig + updateConfig,
						Check:  resource.ComposeAggregateTestCheckFunc(updateChecks...),
					},
				},
			})
		})
	}
}

// TestAccJWTTemplateResource_Invalid tests invalid configurations for stytch_jwt_template.
func TestAccJWTTemplateResource_Invalid(t *testing.T) {
	for _, errorCase := range []testutil.ErrorCase{
		{
			Name: "invalid template type",
			Config: testutil.ConsumerProjectConfig + `
resource "stytch_jwt_template" "test" {
  project_id       = stytch_project.project.test_project_id
  template_type    = "UNKNOWN"
  template_content = "{}"
}
`,
		},
	} {
		errorCase.AssertAnyError(t)
	}
}
