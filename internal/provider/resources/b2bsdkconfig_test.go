package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccB2BSDKConfigResource(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		sdkConfig string
		checks    []resource.TestCheckFunc
	}{
		{
			name: "disabled",
			sdkConfig: testutil.B2BProjectConfig + `
      resource "stytch_b2b_sdk_config" "test" {
        project_id = stytch_project.project.test_project_id
        config = {
          basic = {
            enabled = false
          }
        }
      }`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.enabled", "false"),
			},
		},
		{
			name: "enabled-simple",
			sdkConfig: testutil.B2BProjectConfig + `
      resource "stytch_b2b_sdk_config" "test" {
        project_id = stytch_project.project.test_project_id
        config = {
          basic = {
            enabled                   = true
            create_new_members        = false
            allow_self_onboarding     = false
            enable_member_permissions = false
            domains                   = []
            bundle_ids                = []
          }
        }
      }`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.enabled", "true"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.create_new_members", "false"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.allow_self_onboarding", "false"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.enable_member_permissions", "false"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.domains.#", "0"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.bundle_ids.#", "0"),
			},
		},
		{
			name: "enabled-complex",
			sdkConfig: testutil.B2BProjectConfig + `
      resource "stytch_b2b_sdk_config" "test" {
        project_id = stytch_project.project.test_project_id
        config = {
          basic = {
            enabled                   = true
            create_new_members        = true
            allow_self_onboarding     = true
            enable_member_permissions = true
            domains                   = []
            bundle_ids                = ["com.stytch.app", "com.stytch.app2"]
          }
          sessions = {
            max_session_duration_minutes = 60
          }
          totps = {
            enabled      = true
            create_totps = true
          }
          dfppa = {
            enabled                = "ENABLED"
            on_challenge           = "TRIGGER_CAPTCHA"
            lookup_timeout_seconds = 3
          }
        }
      }`,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.enabled", "true"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.create_new_members", "true"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.allow_self_onboarding", "true"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.enable_member_permissions", "true"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.domains.#", "0"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.bundle_ids.#", "2"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.bundle_ids.0", "com.stytch.app"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.basic.bundle_ids.1", "com.stytch.app2"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.sessions.max_session_duration_minutes", "60"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.totps.enabled", "true"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.totps.create_totps", "true"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.dfppa.enabled", "ENABLED"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.dfppa.on_challenge", "TRIGGER_CAPTCHA"),
				resource.TestCheckResourceAttr("stytch_b2b_sdk_config.test", "config.dfppa.lookup_timeout_seconds", "3"),
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + testCase.sdkConfig,
						Check:  resource.ComposeAggregateTestCheckFunc(testCase.checks...),
					},
					{
						// Import state testing
						ResourceName:      "stytch_b2b_sdk_config.test",
						ImportState:       true,
						ImportStateVerify: true,
						// Ignore timestamp fields because rounding differences in various APIs can result in different
						// values being returned
						ImportStateVerifyIgnore: []string{"last_updated"},
					},
					{
						// Update and Read testing
						Config: testutil.ProviderConfig + testutil.B2BProjectConfig + `
              resource "stytch_b2b_sdk_config" "test" {
                project_id = stytch_project.project.test_project_id
                config = {
                  basic = {
                    enabled                   = true
                    create_new_members        = true
                    allow_self_onboarding     = true
                  }
                  oauth = {
                    enabled = true
                    pkce_required = true
                  }
                }
              }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.enabled", "true"),
							resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.create_new_members", "true"),
							resource.TestCheckResourceAttr("stytch_consumer_sdk_config.test", "config.basic.allow_self_onboarding", "true"),
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
