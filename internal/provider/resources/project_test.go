package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/stytch-management-go/pkg/models/projects"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

var (
	liveProjectRegex = regexp.MustCompile(`^project-live-.*$`)
	testProjectRegex = regexp.MustCompile(`^project-test-.*$`)
)

func projectResource(name string, vertical projects.Vertical) string {
	return fmt.Sprintf(`
resource "stytch_project" "test" {
  name     = "%s"
  vertical = "%s"
}
`, name, string(vertical))
}

func TestAccProjectResource(t *testing.T) {
	for _, vertical := range []projects.Vertical{projects.VerticalConsumer, projects.VerticalB2B} {
		t.Run(string(vertical), func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + projectResource("test", vertical),
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_project.test", "name", "test"),
							resource.TestCheckResourceAttr("stytch_project.test", "vertical", string(vertical)),
							// Verify values were set that should've been configured by the provider
							resource.TestMatchResourceAttr("stytch_project.test", "live_project_id", liveProjectRegex),
							resource.TestMatchResourceAttr("stytch_project.test", "test_project_id", testProjectRegex),
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
						Config: testutil.ProviderConfig + projectResource("test2", vertical),
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
}
