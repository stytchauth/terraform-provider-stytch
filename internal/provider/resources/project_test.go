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
	liveImpersonation     *bool
	testImpersonation     *bool
	liveCrossOrgPasswords *bool
	testCrossOrgPasswords *bool
	liveUserLockSelfServe *bool
	testUserLockSelfServe *bool
	liveUserLockThreshold *int32
	testUserLockThreshold *int32
	liveUserLockTTL       *int32
	testUserLockTTL       *int32
}

func projectResource(args projectResourceArgs) string {
	config := fmt.Sprintf(`
resource "stytch_project" "test" {
  name     = "%s"
  vertical = "%s"`, args.name, string(args.vertical))

	if args.liveImpersonation != nil {
		config += fmt.Sprintf("\n  live_user_impersonation_enabled = %t", *args.liveImpersonation)
	}
	if args.testImpersonation != nil {
		config += fmt.Sprintf("\n  test_user_impersonation_enabled = %t", *args.testImpersonation)
	}
	if args.liveCrossOrgPasswords != nil {
		config += fmt.Sprintf("\n  live_cross_org_passwords_enabled = %t", *args.liveCrossOrgPasswords)
	}
	if args.testCrossOrgPasswords != nil {
		config += fmt.Sprintf("\n  test_cross_org_passwords_enabled = %t", *args.testCrossOrgPasswords)
	}
	if args.liveUserLockSelfServe != nil {
		config += fmt.Sprintf("\n  live_user_lock_self_serve_enabled = %t", *args.liveUserLockSelfServe)
	}
	if args.testUserLockSelfServe != nil {
		config += fmt.Sprintf("\n  test_user_lock_self_serve_enabled = %t", *args.testUserLockSelfServe)
	}
	if args.liveUserLockThreshold != nil {
		config += fmt.Sprintf("\n  live_user_lock_threshold = %d", *args.liveUserLockThreshold)
	}
	if args.testUserLockThreshold != nil {
		config += fmt.Sprintf("\n  test_user_lock_threshold = %d", *args.testUserLockThreshold)
	}
	if args.liveUserLockTTL != nil {
		config += fmt.Sprintf("\n  live_user_lock_ttl = %d", *args.liveUserLockTTL)
	}
	if args.testUserLockTTL != nil {
		config += fmt.Sprintf("\n  test_user_lock_ttl = %d", *args.testUserLockTTL)
	}

	config += "\n}"
	return config
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

	// Test user impersonation settings
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
							liveImpersonation: &tc.liveImpersonationBefore,
							testImpersonation: &tc.testImpersonationBefore,
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
							liveImpersonation: &tc.liveImpersonationAfter,
							testImpersonation: &tc.testImpersonationAfter,
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

	// Test cross org passwords settings
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
							liveCrossOrgPasswords: &tc.liveCrossOrgPasswordsBefore,
							testCrossOrgPasswords: &tc.testCrossOrgPasswordsBefore,
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
							liveCrossOrgPasswords: &tc.liveCrossOrgPasswordsAfter,
							testCrossOrgPasswords: &tc.testCrossOrgPasswordsAfter,
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

	// Test user lock self-serve settings
	for _, tc := range []struct {
		liveUserLockSelfServeBefore bool
		testUserLockSelfServeBefore bool
		liveUserLockSelfServeAfter  bool
		testUserLockSelfServeAfter  bool
		liveUserLockThresholdBefore int32
		testUserLockThresholdBefore int32
		liveUserLockThresholdAfter  int32
		testUserLockThresholdAfter  int32
	}{
		{true, true, false, false, 1, 2, 1, 2},
		{true, false, false, true, 1, 2, 1, 2},
		{false, true, true, true, 1, 2, 1, 2},
		// false, false is already tested above
	} {
		t.Run(fmt.Sprintf("user lock self-serve %t-%t", tc.liveUserLockSelfServeBefore, tc.testUserLockSelfServeBefore), func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
							name:                  "test",
							vertical:              projects.VerticalConsumer,
							liveUserLockSelfServe: &tc.liveUserLockSelfServeBefore,
							testUserLockSelfServe: &tc.testUserLockSelfServeBefore,
							liveUserLockThreshold: &tc.liveUserLockThresholdBefore,
							testUserLockThreshold: &tc.testUserLockThresholdBefore,
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test"),
							resource.TestCheckResourceAttr("stytch_project.test", "live_user_lock_self_serve_enabled", strconv.FormatBool(tc.liveUserLockSelfServeBefore)),
							resource.TestCheckResourceAttr("stytch_project.test", "test_user_lock_self_serve_enabled", strconv.FormatBool(tc.testUserLockSelfServeBefore)),
							resource.TestCheckResourceAttr("stytch_project.test", "live_user_lock_threshold", strconv.Itoa(int(tc.liveUserLockThresholdBefore))),
							resource.TestCheckResourceAttr("stytch_project.test", "test_user_lock_threshold", strconv.Itoa(int(tc.testUserLockThresholdBefore))),
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
							vertical:              projects.VerticalConsumer,
							liveUserLockSelfServe: &tc.liveUserLockSelfServeAfter,
							testUserLockSelfServe: &tc.testUserLockSelfServeAfter,
							liveUserLockThreshold: &tc.liveUserLockThresholdAfter,
							testUserLockThreshold: &tc.testUserLockThresholdAfter,
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test2"),
							resource.TestCheckResourceAttr("stytch_project.test", "live_user_lock_self_serve_enabled", strconv.FormatBool(tc.liveUserLockSelfServeAfter)),
							resource.TestCheckResourceAttr("stytch_project.test", "test_user_lock_self_serve_enabled", strconv.FormatBool(tc.testUserLockSelfServeAfter)),
							resource.TestCheckResourceAttr("stytch_project.test", "live_user_lock_threshold", strconv.Itoa(int(tc.liveUserLockThresholdAfter))),
							resource.TestCheckResourceAttr("stytch_project.test", "test_user_lock_threshold", strconv.Itoa(int(tc.testUserLockThresholdAfter))),
						),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}

	// Test user lock TTL settings
	for _, tc := range []struct {
		liveUserLockSelfServeBefore bool
		testUserLockSelfServeBefore bool
		liveUserLockSelfServeAfter  bool
		testUserLockSelfServeAfter  bool
		liveUserLockTTLBefore       int32
		testUserLockTTLBefore       int32
		liveUserLockTTLAfter        int32
		testUserLockTTLAfter        int32
	}{
		{true, true, false, false, 300, 300, 1800, 1800},
		{true, false, false, true, 300, 300, 1800, 1800},
		{false, true, true, true, 300, 300, 1800, 1800},
		// false, false is already tested above
	} {
		t.Run(fmt.Sprintf("user lock TTL %t-%t", tc.liveUserLockSelfServeBefore, tc.testUserLockSelfServeBefore), func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + projectResource(projectResourceArgs{
							name:                  "test",
							vertical:              projects.VerticalConsumer,
							liveUserLockSelfServe: &tc.liveUserLockSelfServeBefore,
							testUserLockSelfServe: &tc.testUserLockSelfServeBefore,
							liveUserLockTTL:       &tc.liveUserLockTTLBefore,
							testUserLockTTL:       &tc.testUserLockTTLBefore,
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test"),
							resource.TestCheckResourceAttr("stytch_project.test", "live_user_lock_self_serve_enabled", strconv.FormatBool(tc.liveUserLockSelfServeBefore)),
							resource.TestCheckResourceAttr("stytch_project.test", "test_user_lock_self_serve_enabled", strconv.FormatBool(tc.testUserLockSelfServeBefore)),
							resource.TestCheckResourceAttr("stytch_project.test", "live_user_lock_ttl", strconv.Itoa(int(tc.liveUserLockTTLBefore))),
							resource.TestCheckResourceAttr("stytch_project.test", "test_user_lock_ttl", strconv.Itoa(int(tc.testUserLockTTLBefore))),
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
							vertical:              projects.VerticalConsumer,
							liveUserLockSelfServe: &tc.liveUserLockSelfServeAfter,
							testUserLockSelfServe: &tc.testUserLockSelfServeAfter,
							liveUserLockTTL:       &tc.liveUserLockTTLAfter,
							testUserLockTTL:       &tc.testUserLockTTLAfter,
						}),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test2"),
							resource.TestCheckResourceAttr("stytch_project.test", "live_user_lock_self_serve_enabled", strconv.FormatBool(tc.liveUserLockSelfServeAfter)),
							resource.TestCheckResourceAttr("stytch_project.test", "test_user_lock_self_serve_enabled", strconv.FormatBool(tc.testUserLockSelfServeAfter)),
							resource.TestCheckResourceAttr("stytch_project.test", "live_user_lock_ttl", strconv.Itoa(int(tc.liveUserLockTTLAfter))),
							resource.TestCheckResourceAttr("stytch_project.test", "test_user_lock_ttl", strconv.Itoa(int(tc.testUserLockTTLAfter))),
						),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}
}
