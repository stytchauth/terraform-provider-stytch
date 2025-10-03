package resources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/projects"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccEnvironmentResource(t *testing.T) {
	envSlug := "test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create a test environment
				Config: testutil.ProviderConfig + testutil.ProjectResource(testutil.ProjectResourceArgs{
					Name:                "Environment Test Project",
					Vertical:            projects.VerticalConsumer,
					LiveEnvironmentName: strPtr("Production"),
				}) + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug:     "stytch_project.test.project_slug",
					Name:            "Test Environment",
					EnvironmentSlug: &envSlug,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_environment.test", "name", "Test Environment"),
					resource.TestCheckResourceAttr("stytch_environment.test", "environment_slug", envSlug),
					resource.TestCheckResourceAttrSet("stytch_environment.test", "oauth_callback_id"),
					resource.TestCheckResourceAttrSet("stytch_environment.test", "created_at"),
					resource.TestMatchResourceAttr("stytch_environment.test", "id", regexp.MustCompile(`^[a-z0-9-]+\.test$`)),
				),
			},
			{
				// Import state testing
				ResourceName:      "stytch_environment.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore timestamp fields
				ImportStateVerifyIgnore: []string{"created_at", "last_updated"},
			},
			{
				// Update the environment
				Config: testutil.ProviderConfig + testutil.ProjectResource(testutil.ProjectResourceArgs{
					Name:                "Environment Test Project",
					Vertical:            projects.VerticalConsumer,
					LiveEnvironmentName: strPtr("Production"),
				}) + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Updated Test Environment",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_environment.test", "name", "Updated Test Environment"),
				),
			},
			// Delete testing automatically occurs in resource.TestCase
		},
	})
}

func TestAccEnvironmentResourceWithConfig(t *testing.T) {
	trueVal := true
	falseVal := false
	threshold := int32(15)
	ttl := int32(1800)
	zeroDowntime := "https://example.com/test-userinfo"
	idpAuthURL := "https://test.example.com/.well-known/openid-configuration"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create project
				Config: testutil.ProviderConfig + testutil.ProjectResource(testutil.ProjectResourceArgs{
					Name:                "Config Test Project",
					Vertical:            projects.VerticalB2B,
					LiveEnvironmentName: strPtr("Production"),
				}),
			},
			{
				// Create environment with all configs
				Config: testutil.ProviderConfig + testutil.ProjectResource(testutil.ProjectResourceArgs{
					Name:                "Config Test Project",
					Vertical:            projects.VerticalB2B,
					LiveEnvironmentName: strPtr("Production"),
				}) + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug:                  "stytch_project.test.project_slug",
					Name:                         "Test with Config",
					CrossOrgPasswordsEnabled:     &trueVal,
					UserImpersonationEnabled:     &trueVal,
					ZeroDowntimeSessionMigration: &zeroDowntime,
					UserLockSelfServeEnabled:     &trueVal,
					UserLockThreshold:            &threshold,
					UserLockTTL:                  &ttl,
					IdpAuthorizationURL:          &idpAuthURL,
					IdpDCREnabled:                &trueVal,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_environment.test", "cross_org_passwords_enabled", "true"),
					resource.TestCheckResourceAttr("stytch_environment.test", "user_impersonation_enabled", "true"),
					resource.TestCheckResourceAttr("stytch_environment.test", "zero_downtime_session_migration_url", zeroDowntime),
					resource.TestCheckResourceAttr("stytch_environment.test", "user_lock_self_serve_enabled", "true"),
					resource.TestCheckResourceAttr("stytch_environment.test", "user_lock_threshold", "15"),
					resource.TestCheckResourceAttr("stytch_environment.test", "user_lock_ttl", "1800"),
					resource.TestCheckResourceAttr("stytch_environment.test", "idp_authorization_url", idpAuthURL),
					resource.TestCheckResourceAttr("stytch_environment.test", "idp_dynamic_client_registration_enabled", "true"),
				),
			},
			{
				// Update environment configs
				Config: testutil.ProviderConfig + testutil.ProjectResource(testutil.ProjectResourceArgs{
					Name:                "Config Test Project",
					Vertical:            projects.VerticalB2B,
					LiveEnvironmentName: strPtr("Production"),
				}) + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug:              "stytch_project.test.project_slug",
					Name:                     "Test with Updated Config",
					CrossOrgPasswordsEnabled: &falseVal,
					UserImpersonationEnabled: &falseVal,
					UserLockSelfServeEnabled: &falseVal,
					UserLockThreshold:        &threshold,
					UserLockTTL:              &ttl,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_environment.test", "name", "Test with Updated Config"),
					resource.TestCheckResourceAttr("stytch_environment.test", "cross_org_passwords_enabled", "false"),
					resource.TestCheckResourceAttr("stytch_environment.test", "user_impersonation_enabled", "false"),
					resource.TestCheckResourceAttr("stytch_environment.test", "user_lock_self_serve_enabled", "false"),
				),
			},
		},
	})
}
