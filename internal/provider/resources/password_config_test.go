package resources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccPasswordConfigResource(t *testing.T) {
	for _, testCase := range []testutil.TestCase{
		{
			Name: "basic_zxcvbn",
			Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + `
      resource "stytch_password_config" "test" {
        project_slug                     = stytch_project.test.project_slug
        environment_slug                 = stytch_environment.test.environment_slug
        check_breach_on_authentication   = false
        validation_policy                = "ZXCVBN"
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttrSet("stytch_password_config.test", "id"),
				// Confirms that default value is set correctly
				resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_creation", "true"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_authentication", "false"),
				// Confirms that default value is set correctly
				resource.TestCheckResourceAttr("stytch_password_config.test", "validate_on_authentication", "true"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "validation_policy", "ZXCVBN"),
			},
		},
		{
			Name: "luds_policy",
			Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + `
      resource "stytch_password_config" "test" {
        project_slug                     = stytch_project.test.project_slug
        environment_slug                 = stytch_environment.test.environment_slug
        check_breach_on_creation         = true
        check_breach_on_authentication   = false
        validate_on_authentication       = true
        validation_policy                = "LUDS"
        luds_min_password_length         = 12
        luds_min_password_complexity     = 3
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttrSet("stytch_password_config.test", "id"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "validation_policy", "LUDS"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "luds_min_password_length", "12"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "luds_min_password_complexity", "3"),
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
						ResourceName:            "stytch_password_config.test",
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

func TestAccPasswordConfigResourceUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create with ZXCVBN
				Config: testutil.ProviderConfig + testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Test Environment",
				}) + `
        resource "stytch_password_config" "test" {
          project_slug                     = stytch_project.test.project_slug
          environment_slug                 = stytch_environment.test.environment_slug
          check_breach_on_creation         = false
          validation_policy                = "ZXCVBN"
        }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_password_config.test", "validation_policy", "ZXCVBN"),
					resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_creation", "false"),
				),
			},
			{
				// Update to LUDS
				Config: testutil.ProviderConfig + testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Test Environment",
				}) + `
        resource "stytch_password_config" "test" {
          project_slug                     = stytch_project.test.project_slug
          environment_slug                 = stytch_environment.test.environment_slug
          check_breach_on_creation         = true
          validation_policy                = "LUDS"
          luds_min_password_length         = 16
          luds_min_password_complexity     = 4
        }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_password_config.test", "validation_policy", "LUDS"),
					resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_creation", "true"),
					resource.TestCheckResourceAttr("stytch_password_config.test", "luds_min_password_length", "16"),
					resource.TestCheckResourceAttr("stytch_password_config.test", "luds_min_password_complexity", "4"),
				),
			},
		},
	})
}

func TestAccPasswordConfigResourceValidation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Test that LUDS fields with ZXCVBN policy produce an error
				Config: testutil.ProviderConfig + testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Test Environment",
				}) + `
        resource "stytch_password_config" "test" {
          project_slug                     = stytch_project.test.project_slug
          environment_slug                 = stytch_environment.test.environment_slug
          validation_policy                = "ZXCVBN"
          luds_min_password_length         = 12
        }`,
				ExpectError: regexp.MustCompile("luds_min_password_length cannot be set when validation_policy is ZXCVBN"),
			},
			{
				// Test that LUDS complexity field with ZXCVBN policy produces an error
				Config: testutil.ProviderConfig + testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Test Environment",
				}) + `
        resource "stytch_password_config" "test" {
          project_slug                     = stytch_project.test.project_slug
          environment_slug                 = stytch_environment.test.environment_slug
          validation_policy                = "ZXCVBN"
          luds_min_password_complexity     = 3
        }`,
				ExpectError: regexp.MustCompile("luds_min_password_complexity cannot be set when validation_policy is ZXCVBN"),
			},
		},
	})
}

func TestAccPasswordConfigResourceStateUpgrade(t *testing.T) {
	v1Config := testutil.V1ConsumerProjectConfig + `
resource "stytch_password_config" "test" {
  project_id                     = stytch_project.test.live_project_id
  validation_policy              = "ZXCVBN"
  check_breach_on_creation       = true
  check_breach_on_authentication = true
  validate_on_authentication     = true
}
`

	v3Config := testutil.ConsumerProjectConfig + `
resource "stytch_password_config" "test" {
  project_slug                   = stytch_project.test.project_slug
  environment_slug               = stytch_project.test.live_environment.environment_slug
  validation_policy              = "ZXCVBN"
  check_breach_on_creation       = true
  check_breach_on_authentication = true
  validate_on_authentication     = true
}
`

	resource.Test(t, resource.TestCase{
		Steps: testutil.StateUpgradeTestSteps(v1Config, v3Config),
	})
}
