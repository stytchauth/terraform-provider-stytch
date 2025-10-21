package testutil

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stytchauth/stytch-management-go/v3/pkg/models/projects"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider"
)

const (
	// ProviderConfig is a shared configuration to combine with the actual test configuration so the
	// Stytch client is properly configured. The tester should set the STYTCH_ environment variables
	// for the workspace key and secret to allow the tests to run properly.
	ProviderConfig = `provider "stytch" {}`
)

// TestAccProtoV6ProviderFactories are used to instantiate a provider during acceptance testing.
// The factory function will be invoked for every Terraform CLI command executed to create a
// provider server to which the CLI can reattach.
var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"stytch": providerserver.NewProtocol6WithError(provider.New("test")()),
}

var ConsumerProjectConfig = ProjectResource(
	ProjectResourceArgs{
		Name:     "test-consumer",
		Vertical: projects.VerticalConsumer,
	})

var B2BProjectConfig = ProjectResource(ProjectResourceArgs{
	Name:     "test-b2b",
	Vertical: projects.VerticalB2B,
})

// V1 provider project configs for state upgrade tests (uses live_project_id)
const V1ConsumerProjectConfig = `
resource "stytch_project" "test" {
  name     = "test-consumer"
  vertical = "CONSUMER"
}
`

const V1B2BProjectConfig = `
resource "stytch_project" "test" {
  name     = "test-b2b"
  vertical = "B2B"
}
`

type ProjectResourceArgs struct {
	Name                         string
	Vertical                     projects.Vertical
	ProjectSlug                  *string
	LiveEnvironmentSlug          *string
	LiveEnvironmentName          *string
	CrossOrgPasswordsEnabled     *bool
	UserImpersonationEnabled     *bool
	ZeroDowntimeSessionMigration *string
	UserLockSelfServeEnabled     *bool
	UserLockThreshold            *int32
	UserLockTTL                  *int32
	IdpAuthorizationURL          *string
	IdpDCREnabled                *bool
	IdpDCRTemplate               *string
}

type EnvironmentResourceArgs struct {
	ProjectSlug                  string
	EnvironmentSlug              *string
	Name                         string
	CrossOrgPasswordsEnabled     *bool
	UserImpersonationEnabled     *bool
	ZeroDowntimeSessionMigration *string
	UserLockSelfServeEnabled     *bool
	UserLockThreshold            *int32
	UserLockTTL                  *int32
	IdpAuthorizationURL          *string
	IdpDCREnabled                *bool
	IdpDCRTemplate               *string
}

func ProjectResource(args ProjectResourceArgs) string {
	config := fmt.Sprintf(`
resource "stytch_project" "test" {
  name     = "%s"
  vertical = "%s"`, args.Name, string(args.Vertical))

	if args.ProjectSlug != nil {
		config += fmt.Sprintf("\n  project_slug = \"%s\"", *args.ProjectSlug)
	}

	config += "\n  live_environment = {"
	if args.LiveEnvironmentSlug != nil {
		config += fmt.Sprintf("\n    environment_slug = \"%s\"", *args.LiveEnvironmentSlug)
	}
	envName := "Production"
	if args.LiveEnvironmentName != nil {
		envName = *args.LiveEnvironmentName
	}
	config += fmt.Sprintf("\n    name = \"%s\"", envName)

	if args.CrossOrgPasswordsEnabled != nil {
		config += fmt.Sprintf("\n    cross_org_passwords_enabled = %t", *args.CrossOrgPasswordsEnabled)
	}
	if args.UserImpersonationEnabled != nil {
		config += fmt.Sprintf("\n    user_impersonation_enabled = %t", *args.UserImpersonationEnabled)
	}
	if args.ZeroDowntimeSessionMigration != nil {
		config += fmt.Sprintf("\n    zero_downtime_session_migration_url = \"%s\"", *args.ZeroDowntimeSessionMigration)
	}
	if args.UserLockSelfServeEnabled != nil {
		config += fmt.Sprintf("\n    user_lock_self_serve_enabled = %t", *args.UserLockSelfServeEnabled)
	}
	if args.UserLockThreshold != nil {
		config += fmt.Sprintf("\n    user_lock_threshold = %d", *args.UserLockThreshold)
	}
	if args.UserLockTTL != nil {
		config += fmt.Sprintf("\n    user_lock_ttl = %d", *args.UserLockTTL)
	}
	if args.IdpAuthorizationURL != nil {
		config += fmt.Sprintf("\n    idp_authorization_url = \"%s\"", *args.IdpAuthorizationURL)
	}
	if args.IdpDCREnabled != nil {
		config += fmt.Sprintf("\n    idp_dynamic_client_registration_enabled = %t", *args.IdpDCREnabled)
	}
	if args.IdpDCRTemplate != nil {
		config += fmt.Sprintf("\n    idp_dynamic_client_registration_access_token_template_content = \"%s\"", *args.IdpDCRTemplate)
	}

	config += "\n  }"
	config += "\n}"
	return config
}

func EnvironmentResource(args EnvironmentResourceArgs) string {
	config := fmt.Sprintf(`
resource "stytch_environment" "test" {
  project_slug = %s
  name         = "%s"`, args.ProjectSlug, args.Name)

	if args.EnvironmentSlug != nil {
		config += fmt.Sprintf("\n  environment_slug = \"%s\"", *args.EnvironmentSlug)
	}
	if args.CrossOrgPasswordsEnabled != nil {
		config += fmt.Sprintf("\n  cross_org_passwords_enabled = %t", *args.CrossOrgPasswordsEnabled)
	}
	if args.UserImpersonationEnabled != nil {
		config += fmt.Sprintf("\n  user_impersonation_enabled = %t", *args.UserImpersonationEnabled)
	}
	if args.ZeroDowntimeSessionMigration != nil {
		config += fmt.Sprintf("\n  zero_downtime_session_migration_url = \"%s\"", *args.ZeroDowntimeSessionMigration)
	}
	if args.UserLockSelfServeEnabled != nil {
		config += fmt.Sprintf("\n  user_lock_self_serve_enabled = %t", *args.UserLockSelfServeEnabled)
	}
	if args.UserLockThreshold != nil {
		config += fmt.Sprintf("\n  user_lock_threshold = %d", *args.UserLockThreshold)
	}
	if args.UserLockTTL != nil {
		config += fmt.Sprintf("\n  user_lock_ttl = %d", *args.UserLockTTL)
	}
	if args.IdpAuthorizationURL != nil {
		config += fmt.Sprintf("\n  idp_authorization_url = \"%s\"", *args.IdpAuthorizationURL)
	}
	if args.IdpDCREnabled != nil {
		config += fmt.Sprintf("\n  idp_dynamic_client_registration_enabled = %t", *args.IdpDCREnabled)
	}
	if args.IdpDCRTemplate != nil {
		config += fmt.Sprintf("\n  idp_dynamic_client_registration_access_token_template_content = \"%s\"", *args.IdpDCRTemplate)
	}

	config += "\n}"
	return config
}

type TestCase struct {
	Name   string
	Config string
	Checks []resource.TestCheckFunc
}

type ErrorCase struct {
	Name   string
	Config string
	Error  *regexp.Regexp
}

func (e *ErrorCase) AssertAnyError(t *testing.T) {
	t.Helper()
	t.Run(e.Name, func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      ProviderConfig + e.Config,
					ExpectError: regexp.MustCompile(`.*`),
				},
			},
		})
	})
}

func (e *ErrorCase) AssertErrorWith(t *testing.T, errRegex *regexp.Regexp) {
	t.Helper()
	t.Run(e.Name, func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      ProviderConfig + e.Config,
					ExpectError: errRegex,
				},
			},
		})
	})
}

// TestCheckResourceDeleted checks that a resource has been deleted from the state.
func TestCheckResourceDeleted(resourceName string) resource.TestCheckFunc {
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

// StateUpgradeTestSteps returns test steps for v0 to v1 state upgrade testing.
// v1Config: Terraform config using the v0 provider schema version (from provider v1)
// v3Config: Terraform config using the current provider schema version (from provider v3)
func StateUpgradeTestSteps(v1Config string, v3Config string) []resource.TestStep {
	return []resource.TestStep{
		{
			// Step 1: Create with v1 provider
			ExternalProviders: map[string]resource.ExternalProvider{
				"stytch": {
					VersionConstraint: "1.6.2",
					Source:            "stytchauth/stytch",
				},
			},
			Config: `provider "stytch" {}` + "\n" + v1Config,
		},
		{
			// Step 2: Verify v3 provider can read v1 state without changes
			ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
			Config:                   ProviderConfig + v3Config,
			PlanOnly:                 true,
			ExpectNonEmptyPlan:       false,
		},
	}
}
