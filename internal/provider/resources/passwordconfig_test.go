// Copyright (c) HashiCorp, Inc.

package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccPasswordConfigResource(t *testing.T) {
	for _, testCase := range []testutil.TestCase{
		{
			Name: "luds",
			Config: testutil.ConsumerProjectConfig + `
      resource "stytch_password_config" "test" {
        project_id = stytch_project.project.test_project_id
        validation_policy = "LUDS"
        check_breach_on_creation = true
        check_breach_on_authentication = true
        validate_on_authentication = true
        luds_min_password_length = 8
        luds_min_password_complexity = 1
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_password_config.test", "validation_policy", "LUDS"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_creation", "true"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_authentication", "true"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "validate_on_authentication", "true"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "validation_policy", "LUDS"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "luds_min_password_length", "8"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "luds_min_password_complexity", "1"),
			},
		},
		{
			Name: "zxcvbn",
			Config: testutil.ConsumerProjectConfig + `
      resource "stytch_password_config" "test" {
        project_id = stytch_project.project.test_project_id
        check_breach_on_creation = true
        check_breach_on_authentication = true
        validate_on_authentication = true
        validation_policy = "ZXCVBN"
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_password_config.test", "validation_policy", "ZXCVBN"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_creation", "true"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_authentication", "true"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "validate_on_authentication", "true"),
				resource.TestCheckResourceAttr("stytch_password_config.test", "validation_policy", "ZXCVBN"),
			},
		},
	} {
		t.Run(testCase.Name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + testCase.Config,
						Check:  resource.ComposeAggregateTestCheckFunc(testCase.Checks...),
					},
					{
						// Import state testing
						ResourceName:      "stytch_password_config.test",
						ImportState:       true,
						ImportStateVerify: true,
						// Ignore timestamp fields because rounding differences in various APIs can result in different
						// values being returned
						ImportStateVerifyIgnore: []string{"last_updated"},
					},
					{
						// Update and Read testing - switch to LUDS
						Config: testutil.ProviderConfig + testutil.ConsumerProjectConfig + `
              resource "stytch_password_config" "test" {
                project_id = stytch_project.project.test_project_id
                check_breach_on_creation = false
                check_breach_on_authentication = false
                validate_on_authentication = false
                validation_policy = "LUDS"
                luds_min_password_length = 12
                luds_min_password_complexity = 2
              }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_creation", "false"),
							resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_authentication", "false"),
							resource.TestCheckResourceAttr("stytch_password_config.test", "validate_on_authentication", "false"),
							resource.TestCheckResourceAttr("stytch_password_config.test", "validation_policy", "LUDS"),
							resource.TestCheckResourceAttr("stytch_password_config.test", "luds_min_password_length", "12"),
							resource.TestCheckResourceAttr("stytch_password_config.test", "luds_min_password_complexity", "2"),
						),
					},
					{
						// Update and Read testing - switch to ZXCVBN
						Config: testutil.ProviderConfig + testutil.ConsumerProjectConfig + `
              resource "stytch_password_config" "test" {
                project_id = stytch_project.project.test_project_id
                check_breach_on_creation = false
                check_breach_on_authentication = false
                validate_on_authentication = false
                validation_policy = "ZXCVBN"
              }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_creation", "false"),
							resource.TestCheckResourceAttr("stytch_password_config.test", "check_breach_on_authentication", "false"),
							resource.TestCheckResourceAttr("stytch_password_config.test", "validate_on_authentication", "false"),
							resource.TestCheckResourceAttr("stytch_password_config.test", "validation_policy", "ZXCVBN"),
						),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}
}
