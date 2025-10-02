package resources_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/projects"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

var (
	projectSlugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)
	envSlugRegex     = regexp.MustCompile(`^[a-z0-9-]+$`)
)

func strPtr(s string) *string {
	return &s
}

func TestAccProjectResource(t *testing.T) {
	for _, vertical := range []projects.Vertical{projects.VerticalConsumer, projects.VerticalB2B} {
		projectSlug := strings.ToLower(fmt.Sprintf("test-acc-project-resource-%s", string(vertical)))
		prodEnv := "Production"
		t.Run(string(vertical), func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + testutil.ProjectResource(testutil.ProjectResourceArgs{
							Name:                "AccProjectResource",
							ProjectSlug:         &projectSlug,
							Vertical:            vertical,
							LiveEnvironmentName: &prodEnv,
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "AccProjectResource"),
							resource.TestCheckResourceAttr("stytch_project.test", "vertical", string(vertical)),
							resource.TestCheckResourceAttr("stytch_project.test", "project_slug", projectSlug),
							resource.TestCheckResourceAttr("stytch_project.test", "id", projectSlug),
							resource.TestCheckResourceAttrSet("stytch_project.test", "created_at"),
							resource.TestCheckResourceAttr("stytch_project.test", "live_environment.environment_slug", "production"),
							resource.TestCheckResourceAttr("stytch_project.test", "live_environment.name", "Production"),
							resource.TestCheckResourceAttrSet("stytch_project.test", "live_environment.oauth_callback_id"),
							resource.TestCheckResourceAttrSet("stytch_project.test", "live_environment.created_at"),
						),
					},
					{
						// Import state testing
						ResourceName:      "stytch_project.test",
						ImportState:       true,
						ImportStateVerify: true,
						// Ignore timestamp fields because rounding differences can result in different values
						ImportStateVerifyIgnore: []string{"created_at", "last_updated", "live_environment.created_at"},
					},
					{
						// Update and Read testing
						Config: testutil.ProviderConfig + testutil.ProjectResource(testutil.ProjectResourceArgs{
							Name:                "test2",
							Vertical:            vertical,
							LiveEnvironmentName: strPtr("Live Environment"),
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test2"),
							resource.TestCheckResourceAttr("stytch_project.test", "vertical", string(vertical)),
							resource.TestCheckResourceAttr("stytch_project.test", "live_environment.name", "Live Environment"),
						),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}
}

func TestAccProjectResourceWithEnvironmentConfig(t *testing.T) {
	trueVal := true
	falseVal := false
	threshold := int32(5)
	ttl := int32(7200)
	zeroDowntime := "https://example.com/userinfo"
	idpAuthURL := "https://example.com/.well-known/openid-configuration"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create with all environment configurations
				Config: testutil.ProviderConfig + testutil.ProjectResource(testutil.ProjectResourceArgs{
					Name:                         "Environment Config Test",
					Vertical:                     projects.VerticalB2B,
					LiveEnvironmentName:          strPtr("Production"),
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
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.cross_org_passwords_enabled", "true"),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.user_impersonation_enabled", "true"),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.zero_downtime_session_migration_url", zeroDowntime),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.user_lock_self_serve_enabled", "true"),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.user_lock_threshold", "5"),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.user_lock_ttl", "7200"),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.idp_authorization_url", idpAuthURL),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.idp_dynamic_client_registration_enabled", "true"),
				),
			},
			{
				// Update environment configurations
				Config: testutil.ProviderConfig + testutil.ProjectResource(testutil.ProjectResourceArgs{
					Name:                     "Environment Config Test",
					Vertical:                 projects.VerticalB2B,
					LiveEnvironmentName:      strPtr("Production Updated"),
					CrossOrgPasswordsEnabled: &falseVal,
					UserImpersonationEnabled: &falseVal,
					UserLockSelfServeEnabled: &falseVal,
					UserLockThreshold:        &threshold,
					UserLockTTL:              &ttl,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.name", "Production Updated"),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.cross_org_passwords_enabled", "false"),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.user_impersonation_enabled", "false"),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.user_lock_self_serve_enabled", "false"),
				),
			},
		},
	})
}

func TestAccProjectResourceWithoutLiveEnvironment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create project without live environment
				Config: testutil.ProviderConfig + `
resource "stytch_project" "test" {
  name     = "Test Without Live Env"
  vertical = "CONSUMER"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_project.test", "name", "Test Without Live Env"),
					resource.TestCheckResourceAttr("stytch_project.test", "vertical", "CONSUMER"),
					resource.TestCheckNoResourceAttr("stytch_project.test", "live_environment.environment_slug"),
				),
			},
			{
				// Add live environment
				Config: testutil.ProviderConfig + testutil.ProjectResource(testutil.ProjectResourceArgs{
					Name:                "Test Without Live Env",
					Vertical:            projects.VerticalConsumer,
					LiveEnvironmentName: strPtr("Production"),
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_project.test", "name", "Test Without Live Env"),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.environment_slug", "production"),
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.name", "Production"),
				),
			},
		},
	})
}

func TestAccProjectResourceCannotRemoveLiveEnvironment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create project with live environment
				Config: testutil.ProviderConfig + testutil.ProjectResource(testutil.ProjectResourceArgs{
					Name:                "Test Removal",
					Vertical:            projects.VerticalConsumer,
					LiveEnvironmentName: strPtr("Production"),
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_project.test", "live_environment.name", "Production"),
				),
			},
			{
				// Try to remove live environment - should fail
				Config: testutil.ProviderConfig + `
resource "stytch_project" "test" {
  name     = "Test Removal"
  vertical = "CONSUMER"
}`,
				ExpectError: regexp.MustCompile("Cannot remove live_environment"),
			},
		},
	})
}
