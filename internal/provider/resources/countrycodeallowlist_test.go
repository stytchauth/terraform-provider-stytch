package resources_test

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/countrycodeallowlist"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/projects"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

// TestAccCountryCodeAllowlistResource performs acceptance tests for the
// stytch_country_code_allowlist resource.
func TestAccCountryCodeAllowlistResource(t *testing.T) {
	const resourceName = "stytch_country_code_allowlist.test_allowlist"

	for _, tc := range []struct {
		TestName             string
		Vertical             projects.Vertical
		DeliveryMethod       countrycodeallowlist.DeliveryMethod
		InitialCountryCodes  []string
		UpdateCountryCodes   []string
		ExpectedCountryCodes []string
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
			TestName:            "duplicate country codes",
			Vertical:            projects.VerticalB2B,
			DeliveryMethod:      countrycodeallowlist.DeliveryMethodSMS,
			InitialCountryCodes: []string{"CA", "US"},
			UpdateCountryCodes:  []string{"CA", "MX", "US", "CA", "MX", "US"},
		},
		{
			TestName:            "same country codes",
			Vertical:            projects.VerticalB2B,
			DeliveryMethod:      countrycodeallowlist.DeliveryMethodSMS,
			InitialCountryCodes: []string{"CA", "US"},
			UpdateCountryCodes:  []string{"CA", "us"},
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
				resource.TestCheckResourceAttr(resourceName, "country_codes.#", strconv.Itoa(len(tc.InitialCountryCodes))),
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
				// Updated country codes should match the user input (updateConfig).
				resource.TestCheckResourceAttr(resourceName, "country_codes.#", strconv.Itoa(len(tc.UpdateCountryCodes))),
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
						Check:  testutil.TestCheckResourceDeleted(resourceName),
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
			Error: regexp.MustCompile(`.*delivery_method value must be one of.*`),
		},
		{
			Name: "invalid country codes",
			Config: testutil.ConsumerProjectConfig + `
				resource "stytch_country_code_allowlist" "test_allowlist" {
					project_id       = stytch_project.project.test_project_id
					delivery_method  = "sms"
					country_codes    = ["XX", "YY", "ZZ"]
				}
				`,
			Error: regexp.MustCompile(`.*country_code_allowlist_invalid_country_codes*`),
		},
		{
			Name: "empty country codes",
			Config: testutil.ConsumerProjectConfig + `
				resource "stytch_country_code_allowlist" "test_allowlist" {
					project_id       = stytch_project.project.test_project_id
					delivery_method  = "sms"
					country_codes    = []
				}
				`,
			Error: regexp.MustCompile(`.*country_code_allowlist_empty*`),
		},
		{
			Name: "B2B WhatsApp not supported",
			Config: testutil.B2BProjectConfig + `
				resource "stytch_country_code_allowlist" "test_allowlist" {
					project_id       = stytch_project.project.test_project_id
					delivery_method  = "whatsapp"
					country_codes    = []
				}
				`,
			Error: regexp.MustCompile(`.*country_code_allowlist_b2b_whatsapp_not_supported.*`),
		},
	} {
		if errorCase.Error == nil {
			errorCase.AssertAnyError(t)
		} else {
			errorCase.AssertErrorWith(t, errorCase.Error)
		}
	}
}

// TestAccCountryCodeAllowlistResource_PlanEmpty tests that the same configuration does not result
// in planned updates.
func TestAccCountryCodeAllowlistResource_PlanEmpty(t *testing.T) {
	const resourceName = "stytch_country_code_allowlist.test_allowlist"

	for _, tc := range []struct {
		TestName       string
		Vertical       projects.Vertical
		DeliveryMethod countrycodeallowlist.DeliveryMethod
		CountryCodes   []string
	}{
		{
			TestName:       "standardized country codes",
			Vertical:       projects.VerticalConsumer,
			DeliveryMethod: countrycodeallowlist.DeliveryMethodSMS,
			CountryCodes:   []string{"CA", "US"},
		},
		{
			TestName:       "unsorted country codes",
			Vertical:       projects.VerticalConsumer,
			DeliveryMethod: countrycodeallowlist.DeliveryMethodSMS,
			CountryCodes:   []string{"US", "CA"},
		},
		{
			TestName:       "not normalized country codes",
			Vertical:       projects.VerticalConsumer,
			DeliveryMethod: countrycodeallowlist.DeliveryMethodSMS,
			CountryCodes:   []string{"cA", "us"},
		},
		{
			TestName:       "duplicate country codes",
			Vertical:       projects.VerticalConsumer,
			DeliveryMethod: countrycodeallowlist.DeliveryMethodSMS,
			CountryCodes:   []string{"CA", "CA", "US"},
		},
	} {
		t.Run(tc.TestName, func(t *testing.T) {
			// Build initial Terraform configuration.
			projectConfig := testutil.ConsumerProjectConfig
			if tc.Vertical == projects.VerticalB2B {
				projectConfig = testutil.B2BProjectConfig
			}
			config := projectConfig + fmt.Sprintf(`
				resource "stytch_country_code_allowlist" "test_allowlist" {
					project_id       = stytch_project.project.test_project_id
					delivery_method  = "%s"
					country_codes    = ["%s"]
				}
				`, string(tc.DeliveryMethod), strings.Join(tc.CountryCodes, `", "`))

			// Check configuration.
			checks := []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "delivery_method", string(tc.DeliveryMethod)),
				resource.TestCheckResourceAttr(resourceName, "country_codes.#", strconv.Itoa(len(tc.CountryCodes))),
			}

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Test Create and Read.
						Config: testutil.ProviderConfig + config,
						Check:  resource.ComposeAggregateTestCheckFunc(checks...),
					},
					{
						// Re-apply the same configuration and check that no changes are planned.
						Config:             config,
						PlanOnly:           true,
						ExpectNonEmptyPlan: false,
					},
				},
			})
		})
	}
}
