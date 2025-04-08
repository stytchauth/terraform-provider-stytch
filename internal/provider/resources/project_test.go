package resources_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/stytch-management-go/v2/pkg/models/projects"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

var (
	liveProjectRegex = regexp.MustCompile(`^project-live-.*$`)
	testProjectRegex = regexp.MustCompile(`^project-test-.*$`)
)

type projectResourceArgs struct {
	name                  string
	vertical              projects.Vertical
	liveImpersonation     bool
	testImpersonation     bool
	liveCrossOrgPasswords bool
	testCrossOrgPasswords bool
}

func projectResource(args projectResourceArgs) string {
	return fmt.Sprintf(`
resource "stytch_project" "test" {
  name                             = "%s"
  vertical                         = "%s"
  live_user_impersonation_enabled  = %t
  test_user_impersonation_enabled  = %t
  live_cross_org_passwords_enabled = %t
  test_cross_org_passwords_enabled = %t
}
`, args.name,
		string(args.vertical),
		args.liveImpersonation,
		args.testImpersonation,
		args.liveCrossOrgPasswords,
		args.testCrossOrgPasswords,
	)
}

func TestAccProjectResource(t *testing.T) {
	for _, vertical := range []projects.Vertical{projects.VerticalConsumer, projects.VerticalB2B} {
		t.Run(string(vertical), func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
							name:     "test",
							vertical: vertical,
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test"),
							resource.TestCheckResourceAttr("stytch_project.test", "vertical", string(vertical)),
							// Verify values were set that should've been configured by the provider
							resource.TestMatchResourceAttr("stytch_project.test", "test_project_id", testProjectRegex),
							resource.TestMatchResourceAttr("stytch_project.test", "live_project_id", liveProjectRegex),
						),
					},
					{
						// Import state testing
						ResourceName:      "stytch_project.test",
						ImportState:       true,
						ImportStateVerify: true,
						// Ignore timestamp fields because rounding differences in various APIs can result in different
						// values being returned
						ImportStateVerifyIgnore: []string{"created_at", "last_updated"},
					},
					{
						// Update and Read testing
						Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
							name:     "test2",
							vertical: vertical,
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test2"),
							resource.TestCheckResourceAttr("stytch_project.test", "vertical", string(vertical)),
							// Verify values were set that should've been configured by the provider
							resource.TestMatchResourceAttr("stytch_project.test", "live_project_id", liveProjectRegex),
							resource.TestMatchResourceAttr("stytch_project.test", "test_project_id", testProjectRegex),
						),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}
	for _, tc := range []struct {
		liveImpersonationBefore bool
		testImpersonationBefore bool
		liveImpersonationAfter  bool
		testImpersonationAfter  bool
	}{
		{true, true, false, false},
		{true, false, false, true},
		{false, true, true, true},
		// false, false is already tested above
	} {
		t.Run(fmt.Sprintf("user impersonation %t-%t", tc.liveImpersonationBefore, tc.testImpersonationBefore), func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
							name:              "test",
							vertical:          projects.VerticalConsumer,
							liveImpersonation: tc.liveImpersonationBefore,
							testImpersonation: tc.testImpersonationBefore,
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test"),
							resource.TestCheckResourceAttr("stytch_project.test", "live_user_impersonation_enabled", strconv.FormatBool(tc.liveImpersonationBefore)),
							resource.TestCheckResourceAttr("stytch_project.test", "test_user_impersonation_enabled", strconv.FormatBool(tc.testImpersonationBefore)),
						),
					},
					{
						// Import state testing
						ResourceName:      "stytch_project.test",
						ImportState:       true,
						ImportStateVerify: true,
						// Ignore timestamp fields because rounding differences in various APIs can result in different
						// values being returned
						ImportStateVerifyIgnore: []string{"created_at", "last_updated"},
					},
					{
						// Update and Read testing
						Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
							name:              "test2",
							vertical:          projects.VerticalConsumer,
							liveImpersonation: tc.liveImpersonationAfter,
							testImpersonation: tc.testImpersonationAfter,
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test2"),
							resource.TestCheckResourceAttr("stytch_project.test", "live_user_impersonation_enabled", strconv.FormatBool(tc.liveImpersonationAfter)),
							resource.TestCheckResourceAttr("stytch_project.test", "test_user_impersonation_enabled", strconv.FormatBool(tc.testImpersonationAfter)),
						),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}

	for _, tc := range []struct {
		liveCrossOrgPasswordsBefore bool
		testCrossOrgPasswordsBefore bool
		liveCrossOrgPasswordsAfter  bool
		testCrossOrgPasswordsAfter  bool
	}{
		{true, true, false, false},
		{true, false, false, true},
		{false, true, true, true},
		// false, false is already tested above
	} {
		t.Run(fmt.Sprintf("cross org passwords %t-%t", tc.liveCrossOrgPasswordsBefore, tc.testCrossOrgPasswordsBefore), func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
							name:                  "test",
							vertical:              projects.VerticalB2B,
							liveCrossOrgPasswords: tc.liveCrossOrgPasswordsBefore,
							testCrossOrgPasswords: tc.testCrossOrgPasswordsBefore,
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test"),
							resource.TestCheckResourceAttr("stytch_project.test", "live_cross_org_passwords_enabled", strconv.FormatBool(tc.liveCrossOrgPasswordsBefore)),
							resource.TestCheckResourceAttr("stytch_project.test", "test_cross_org_passwords_enabled", strconv.FormatBool(tc.testCrossOrgPasswordsBefore)),
						),
					},
					{
						// Import state testing
						ResourceName:      "stytch_project.test",
						ImportState:       true,
						ImportStateVerify: true,
						// Ignore timestamp fields because rounding differences in various APIs can result in different
						// values being returned
						ImportStateVerifyIgnore: []string{"created_at", "last_updated"},
					},
					{
						// Update and Read testing
						Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
							name:                  "test2",
							vertical:              projects.VerticalB2B,
							liveCrossOrgPasswords: tc.liveCrossOrgPasswordsAfter,
							testCrossOrgPasswords: tc.testCrossOrgPasswordsAfter,
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test2"),
							resource.TestCheckResourceAttr("stytch_project.test", "live_cross_org_passwords_enabled", strconv.FormatBool(tc.liveCrossOrgPasswordsAfter)),
							resource.TestCheckResourceAttr("stytch_project.test", "test_cross_org_passwords_enabled", strconv.FormatBool(tc.testCrossOrgPasswordsAfter)),
						),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}
}
