package resources_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stytchauth/stytch-management-go/v2/pkg/models/countrycodeallowlist"
	"github.com/stytchauth/stytch-management-go/v2/pkg/models/projects"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

// testCheckResourceDeleted checks that a resource has been deleted from the state.
func testCheckResourceDeleted(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Look up resource in state.
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			// Resource already removed from state.
			return nil
		}

		// If the resource is still present, return an error.
		if rs.Primary.ID != "" {
			return fmt.Errorf("resource %s still exists with ID: %s", resourceName, rs.Primary.ID)
		}

		return nil
	}
}

// TestAccCountryCodeAllowlistResource performs acceptance tests for the
// stytch_country_code_allowlist resource.
func TestAccCountryCodeAllowlistResource(t *testing.T) {
	const resourceName = "stytch_country_code_allowlist.test_allowlist"

	for _, tc := range []struct {
		TestName            string
		Vertical            projects.Vertical
		DeliveryMethod      countrycodeallowlist.DeliveryMethod
		InitialCountryCodes []string
		UpdateCountryCodes  []string
	}{
		{
			TestName:            "b2c_sms_country_code_allowlist",
			Vertical:            projects.VerticalConsumer,
			DeliveryMethod:      countrycodeallowlist.DeliveryMethodSMS,
			InitialCountryCodes: []string{"CA", "US"},
			UpdateCountryCodes:  []string{"CA", "MX", "US"},
		},
		{
			TestName:            "b2c_whatsapp_country_code_allowlist",
			Vertical:            projects.VerticalConsumer,
			DeliveryMethod:      countrycodeallowlist.DeliveryMethodWhatsApp,
			InitialCountryCodes: []string{"CA", "US"},
			UpdateCountryCodes:  []string{"CA", "MX", "US"},
		},
		{
			TestName:            "b2b_sms_country_code_allowlist",
			Vertical:            projects.VerticalB2B,
			DeliveryMethod:      countrycodeallowlist.DeliveryMethodSMS,
			InitialCountryCodes: []string{"CA", "US"},
			UpdateCountryCodes:  []string{"CA", "MX", "US"},
		},
		{
			TestName:            "b2b_whatsapp_country_code_allowlist",
			Vertical:            projects.VerticalB2B,
			DeliveryMethod:      countrycodeallowlist.DeliveryMethodWhatsApp,
			InitialCountryCodes: []string{"CA", "US"},
			UpdateCountryCodes:  []string{"CA", "MX", "US"},
		},
	} {
		t.Run(tc.TestName, func(t *testing.T) {
			// Build initial Terraform configuration.
			projectConfig := testutil.ConsumerProjectConfig
			if tc.Vertical == projects.VerticalB2B {
				projectConfig = testutil.B2BProjectConfig
			}
			initialConfig := projectConfig + fmt.Sprintf(`
				resource "stytch_country_code_allowlist" "test_allowlist" {
					project_id       = stytch_project.project.test_project_id
					delivery_method  = "%s"
					country_codes    = ["%s"]
				}
				`, string(tc.DeliveryMethod), strings.Join(tc.InitialCountryCodes, `", "`))

			// Check initial configuration.
			initialChecks := []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "delivery_method", string(tc.DeliveryMethod)),
				resource.TestCheckResourceAttr(resourceName, "country_codes.#", "2"),
			}

			// Build update Terraform configuration
			updateConfig := projectConfig + fmt.Sprintf(`
				resource "stytch_country_code_allowlist" "test_allowlist" {
					project_id       = stytch_project.project.test_project_id
					delivery_method  = "%s"
					country_codes    = ["%s"]
				}
				`, string(tc.DeliveryMethod), strings.Join(tc.UpdateCountryCodes, `", "`))

			// Check updated configuration.
			updateChecks := []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "delivery_method", string(tc.DeliveryMethod)),
				resource.TestCheckResourceAttr(resourceName, "country_codes.#", "3"),
			}

			// Build delete Terraform configuration.
			deleteConfig := projectConfig

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Test Create and Read.
						Config: testutil.ProviderConfig + initialConfig,
						Check:  resource.ComposeAggregateTestCheckFunc(initialChecks...),
					},
					{
						// Test ImportState.
						ResourceName:            resourceName,
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"last_updated"},
					},
					{
						// Test Update and Read.
						Config: testutil.ProviderConfig + updateConfig,
						Check:  resource.ComposeAggregateTestCheckFunc(updateChecks...),
					},
					{
						// Test Delete and Read.
						Config: testutil.ProviderConfig + deleteConfig,
						Check:  testCheckResourceDeleted(resourceName),
					},
				},
			})
		})
	}
}

// TestAccCountryCodeAllowlistResource_Invalid tests invalid configurations for
// stytch_country_code_allowlist.
func TestAccCountryCodeAllowlistResource_Invalid(t *testing.T) {
	for _, errorCase := range []testutil.ErrorCase{
		{
			Name: "invalid delivery method",
			Config: testutil.ConsumerProjectConfig + `
				resource "stytch_country_code_allowlist" "test_allowlist" {
					project_id       = stytch_project.project.test_project_id
					delivery_method  = "email"
					country_codes    = ["US", "CA"]
				}
				`,
		},
	} {
		errorCase.AssertAnyError(t)
	}
}
