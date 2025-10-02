package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/projects"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

type environmentResourceArgs struct {
	projectSlug                  string
	environmentSlug              *string
	name                         string
	crossOrgPasswordsEnabled     *bool
	userImpersonationEnabled     *bool
	zeroDowntimeSessionMigration *string
	userLockSelfServeEnabled     *bool
	userLockThreshold            *int32
	userLockTTL                  *int32
	idpAuthorizationURL          *string
	idpDCREnabled                *bool
	idpDCRTemplate               *string
}

func environmentResource(args environmentResourceArgs) string {
	config := fmt.Sprintf(`
resource "stytch_environment" "test" {
  project_slug = %s
  name         = "%s"`, args.projectSlug, args.name)

	if args.environmentSlug != nil {
		config += fmt.Sprintf("\n  environment_slug = \"%s\"", *args.environmentSlug)
	}
	if args.crossOrgPasswordsEnabled != nil {
		config += fmt.Sprintf("\n  cross_org_passwords_enabled = %t", *args.crossOrgPasswordsEnabled)
	}
	if args.userImpersonationEnabled != nil {
		config += fmt.Sprintf("\n  user_impersonation_enabled = %t", *args.userImpersonationEnabled)
	}
	if args.zeroDowntimeSessionMigration != nil {
		config += fmt.Sprintf("\n  zero_downtime_session_migration_url = \"%s\"", *args.zeroDowntimeSessionMigration)
	}
	if args.userLockSelfServeEnabled != nil {
		config += fmt.Sprintf("\n  user_lock_self_serve_enabled = %t", *args.userLockSelfServeEnabled)
	}
	if args.userLockThreshold != nil {
		config += fmt.Sprintf("\n  user_lock_threshold = %d", *args.userLockThreshold)
	}
	if args.userLockTTL != nil {
		config += fmt.Sprintf("\n  user_lock_ttl = %d", *args.userLockTTL)
	}
	if args.idpAuthorizationURL != nil {
		config += fmt.Sprintf("\n  idp_authorization_url = \"%s\"", *args.idpAuthorizationURL)
	}
	if args.idpDCREnabled != nil {
		config += fmt.Sprintf("\n  idp_dynamic_client_registration_enabled = %t", *args.idpDCREnabled)
	}
	if args.idpDCRTemplate != nil {
		config += fmt.Sprintf("\n  idp_dynamic_client_registration_access_token_template_content = \"%s\"", *args.idpDCRTemplate)
	}

	config += "\n}"
	return config
}

func TestAccEnvironmentResource(t *testing.T) {
	envSlug := "test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create a test environment
				Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
					name:                "Environment Test Project",
					vertical:            projects.VerticalConsumer,
					liveEnvironmentName: "Production",
				}) + environmentResource(environmentResourceArgs{
					projectSlug:     "stytch_project.test.project_slug",
					name:            "Test Environment",
					environmentSlug: &envSlug,
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
				Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
					name:                "Environment Test Project",
					vertical:            projects.VerticalConsumer,
					liveEnvironmentName: "Production",
				}) + environmentResource(environmentResourceArgs{
					projectSlug: "stytch_project.test.project_slug",
					name:        "Updated Test Environment",
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
				Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
					name:                "Config Test Project",
					vertical:            projects.VerticalB2B,
					liveEnvironmentName: "Production",
				}),
			},
			{
				// Create environment with all configs
				Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
					name:                "Config Test Project",
					vertical:            projects.VerticalB2B,
					liveEnvironmentName: "Production",
				}) + environmentResource(environmentResourceArgs{
					projectSlug:                  "stytch_project.test.project_slug",
					name:                         "Test with Config",
					crossOrgPasswordsEnabled:     &trueVal,
					userImpersonationEnabled:     &trueVal,
					zeroDowntimeSessionMigration: &zeroDowntime,
					userLockSelfServeEnabled:     &trueVal,
					userLockThreshold:            &threshold,
					userLockTTL:                  &ttl,
					idpAuthorizationURL:          &idpAuthURL,
					idpDCREnabled:                &trueVal,
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
				Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
					name:                "Config Test Project",
					vertical:            projects.VerticalB2B,
					liveEnvironmentName: "Production",
				}) + environmentResource(environmentResourceArgs{
					projectSlug:              "stytch_project.test.project_slug",
					name:                     "Test with Updated Config",
					crossOrgPasswordsEnabled: &falseVal,
					userImpersonationEnabled: &falseVal,
					userLockSelfServeEnabled: &falseVal,
					userLockThreshold:        &threshold,
					userLockTTL:              &ttl,
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
