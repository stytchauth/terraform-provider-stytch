package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccConsumerSDKConfigResource(t *testing.T) {
	for _, testCase := range []testutil.TestCase{
		{
			Name: "disabled",
			Config: testutil.ConsumerProjectConfig + `
      resource "stytch_consumer_sdk_config" "test" {
        project_id = stytch_project.project.test_project_id
        config = {
          basic = {
            enabled = false
          }
        }
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.enabled", "false"),
			},
		},
		{
			Name: "enabled-simple",
			Config: testutil.ConsumerProjectConfig + `
      resource "stytch_consumer_sdk_config" "test" {
        project_id = stytch_project.project.test_project_id
        config = {
          basic = {
            enabled          = true
            create_new_users = true
            domains          = []
            bundle_ids       = []
          }
        }
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.enabled", "true"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.create_new_users", "true"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.domains.#", "0"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.bundle_ids.#", "0"),
			},
		},
		{
			Name: "enabled-complex",
			Config: testutil.ConsumerProjectConfig + `
      resource "stytch_consumer_sdk_config" "test" {
        project_id = stytch_project.project.test_project_id
        config = {
          basic = {
            enabled          = true
            create_new_users = true
            domains          = []
            bundle_ids       = ["com.stytch.app1", "com.stytch.app2"]
          }
          sessions = {
            max_session_duration_minutes = 60
          }
          magic_links = {
            login_or_create_enabled = true
            send_enabled            = true
            pkce_required           = true
          }
          cookies = {
            http_only = "ENFORCED"
          }
        }
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.enabled", "true"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.create_new_users", "true"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.domains.#", "0"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.bundle_ids.#", "2"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.bundle_ids.0", "com.stytch.app1"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.bundle_ids.1", "com.stytch.app2"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.sessions.max_session_duration_minutes", "60"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.magic_links.login_or_create_enabled", "true"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.magic_links.send_enabled", "true"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.magic_links.pkce_required", "true"),
				resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.cookies.http_only", "ENFORCED"),
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
						ResourceName:      "stytch_consumer_sdk_config.test",
						ImportState:       true,
						ImportStateVerify: true,
						// Ignore timestamp fields because rounding differences in various APIs can result in different
						// values being returned
						ImportStateVerifyIgnore: []string{"last_updated"},
					},
					{
						// Update and Read testing
						Config: testutil.ProviderConfig + testutil.ConsumerProjectConfig + `
              resource "stytch_consumer_sdk_config" "test" {
                project_id = stytch_project.project.test_project_id
                config = {
                  basic = {
                    enabled          = true
                    create_new_users = true
                  }
                  oauth = {
                    enabled = true
                    pkce_required = true
                  }
                }
              }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.enabled", "true"),
							resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.create_new_users", "true"),
							resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.oauth.enabled", "true"),
							resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.oauth.pkce_required", "true"),
						),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}
}

func TestAccConsumerSDKConfigResource_Invalid(t *testing.T) {
	for _, errorCase := range []testutil.ErrorCase{
		{
			Name: "applied to B2B project",
			Config: testutil.B2BProjectConfig + `
      resource "stytch_consumer_sdk_config" "test" {
        project_id = stytch_project.project.test_project_id
        config = {
          basic = {
            enabled = false
          }
        }
      }`,
		},
	} {
		errorCase.AssertAnyError(t)
	}
}
