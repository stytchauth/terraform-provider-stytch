package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccCountryCodeAllowlistResource(t *testing.T) {
	for _, testCase := range []testutil.TestCase{
		{
			Name: "sms_basic",
			Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + `
      resource "stytch_country_code_allowlist" "test" {
        project_slug     = stytch_project.test.project_slug
        environment_slug = stytch_environment.test.environment_slug
        delivery_method  = "sms"
        country_codes    = ["CA", "GB", "US"]
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttrSet("stytch_country_code_allowlist.test", "id"),
				resource.TestCheckResourceAttr("stytch_country_code_allowlist.test", "delivery_method", "sms"),
				resource.TestCheckResourceAttr("stytch_country_code_allowlist.test", "country_codes.#", "3"),
			},
		},
		{
			Name: "whatsapp_basic",
			Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + `
      resource "stytch_country_code_allowlist" "test" {
        project_slug     = stytch_project.test.project_slug
        environment_slug = stytch_environment.test.environment_slug
        delivery_method  = "whatsapp"
        country_codes    = ["CA", "US", "MX"]
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttrSet("stytch_country_code_allowlist.test", "id"),
				resource.TestCheckResourceAttr("stytch_country_code_allowlist.test", "delivery_method", "whatsapp"),
				resource.TestCheckResourceAttr("stytch_country_code_allowlist.test", "country_codes.#", "3"),
			},
		},
	} {
		t.Run(testCase.Name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Test Create and Read.
						Config: testutil.ProviderConfig + testCase.Config,
						Check:  resource.ComposeAggregateTestCheckFunc(testCase.Checks...),
					},
					{
						// Test ImportState.
						ResourceName:            "stytch_country_code_allowlist.test",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"last_updated"},
					},
					// Delete is automatically tested in resource.TestCase.
				},
			})
		})
	}
}

func TestAccCountryCodeAllowlistResourceUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create with US, CA, FR
				Config: testutil.ProviderConfig + testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Test Environment",
				}) + `
        resource "stytch_country_code_allowlist" "test" {
          project_slug     = stytch_project.test.project_slug
          environment_slug = stytch_environment.test.environment_slug
          delivery_method  = "sms"
          country_codes    = ["CA", "US", "FR"]
        }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_country_code_allowlist.test", "country_codes.#", "3"),
				),
			},
			{
				// Update to add GB
				Config: testutil.ProviderConfig + testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Test Environment",
				}) + `
        resource "stytch_country_code_allowlist" "test" {
          project_slug     = stytch_project.test.project_slug
          environment_slug = stytch_environment.test.environment_slug
          delivery_method  = "sms"
          country_codes    = ["CA", "FR", "US", "GB"]
        }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_country_code_allowlist.test", "country_codes.#", "4"),
					resource.TestCheckResourceAttr("stytch_country_code_allowlist.test", "country_codes.0", "CA"),
					resource.TestCheckResourceAttr("stytch_country_code_allowlist.test", "country_codes.1", "FR"),
					resource.TestCheckResourceAttr("stytch_country_code_allowlist.test", "country_codes.2", "GB"),
					resource.TestCheckResourceAttr("stytch_country_code_allowlist.test", "country_codes.3", "US"),
				),
			},
		},
	})
}

func TestAccCountryCodeAllowlistResourceStateUpgrade(t *testing.T) {
	v1Config := testutil.V1ConsumerProjectConfig + `
resource "stytch_country_code_allowlist" "test" {
  project_id      = stytch_project.test.live_project_id
  delivery_method = "sms"
  country_codes   = ["US", "CA", "GB"]
}
`

	v3Config := testutil.ConsumerProjectConfig + `
resource "stytch_country_code_allowlist" "test" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_project.test.live_environment.environment_slug
  delivery_method  = "sms"
  country_codes    = ["US", "CA", "GB"]
}
`

	resource.Test(t, resource.TestCase{
		Steps: testutil.StateUpgradeTestSteps(v1Config, v3Config),
	})
}
