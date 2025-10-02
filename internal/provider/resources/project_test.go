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

type projectResourceArgs struct {
	name                         string
	vertical                     projects.Vertical
	projectSlug                  *string
	liveEnvironmentSlug          *string
	liveEnvironmentName          *string
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

func projectResource(args projectResourceArgs) string {
	config := fmt.Sprintf(`
resource "stytch_project" "test" {
  name     = "%s"
  vertical = "%s"`, args.name, string(args.vertical))

	if args.projectSlug != nil {
		config += fmt.Sprintf("\n  project_slug = \"%s\"", *args.projectSlug)
	}

	config += "\n  live_environment = {"
	if args.liveEnvironmentSlug != nil {
		config += fmt.Sprintf("\n    environment_slug = \"%s\"", *args.liveEnvironmentSlug)
	}
	envName := "Production"
	if args.liveEnvironmentName != nil {
		envName = *args.liveEnvironmentName
	}
	config += fmt.Sprintf("\n    name = \"%s\"", envName)

	if args.crossOrgPasswordsEnabled != nil {
		config += fmt.Sprintf("\n    cross_org_passwords_enabled = %t", *args.crossOrgPasswordsEnabled)
	}
	if args.userImpersonationEnabled != nil {
		config += fmt.Sprintf("\n    user_impersonation_enabled = %t", *args.userImpersonationEnabled)
	}
	if args.zeroDowntimeSessionMigration != nil {
		config += fmt.Sprintf("\n    zero_downtime_session_migration_url = \"%s\"", *args.zeroDowntimeSessionMigration)
	}
	if args.userLockSelfServeEnabled != nil {
		config += fmt.Sprintf("\n    user_lock_self_serve_enabled = %t", *args.userLockSelfServeEnabled)
	}
	if args.userLockThreshold != nil {
		config += fmt.Sprintf("\n    user_lock_threshold = %d", *args.userLockThreshold)
	}
	if args.userLockTTL != nil {
		config += fmt.Sprintf("\n    user_lock_ttl = %d", *args.userLockTTL)
	}
	if args.idpAuthorizationURL != nil {
		config += fmt.Sprintf("\n    idp_authorization_url = \"%s\"", *args.idpAuthorizationURL)
	}
	if args.idpDCREnabled != nil {
		config += fmt.Sprintf("\n    idp_dynamic_client_registration_enabled = %t", *args.idpDCREnabled)
	}
	if args.idpDCRTemplate != nil {
		config += fmt.Sprintf("\n    idp_dynamic_client_registration_access_token_template_content = \"%s\"", *args.idpDCRTemplate)
	}

	config += "\n  }"
	config += "\n}"
	return config
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
						Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
							name:                "AccProjectResource",
							projectSlug:         &projectSlug,
							vertical:            vertical,
							liveEnvironmentName: &prodEnv,
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
						Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
							name:                "test2",
							vertical:            vertical,
							liveEnvironmentName: strPtr("Live Environment"),
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
				Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
					name:                         "Environment Config Test",
					vertical:                     projects.VerticalB2B,
					liveEnvironmentName:          strPtr("Production"),
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
				Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
					name:                     "Environment Config Test",
					vertical:                 projects.VerticalB2B,
					liveEnvironmentName:      strPtr("Production Updated"),
					crossOrgPasswordsEnabled: &falseVal,
					userImpersonationEnabled: &falseVal,
					userLockSelfServeEnabled: &falseVal,
					userLockThreshold:        &threshold,
					userLockTTL:              &ttl,
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
				Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
					name:                "Test Without Live Env",
					vertical:            projects.VerticalConsumer,
					liveEnvironmentName: strPtr("Production"),
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
				Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
					name:                "Test Removal",
					vertical:            projects.VerticalConsumer,
					liveEnvironmentName: strPtr("Production"),
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
