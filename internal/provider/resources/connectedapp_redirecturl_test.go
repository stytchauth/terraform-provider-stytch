package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

// The parent must never set redirect_urls or post_logout_redirect_urls here: the
// children own those arrays, and the refresh steps rely on the parent's copies
// staying null.
func connectedAppWithChildren(childBlocks string) string {
	return testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
		ProjectSlug: "stytch_project.test.project_slug",
		Name:        "Test Environment",
	}) + projectSecretResource + fmt.Sprintf(`
resource "stytch_connected_app" "test" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  client_type      = "first_party"
  client_name      = "tf-acc-child-urls"

  access_token_expiry_minutes  = 7
  access_token_custom_audience = "tf-acc-probe-audience"
}
%s
`, childBlocks)
}

func childURL(name, u, urlType string) string {
	return fmt.Sprintf(`
resource "stytch_connected_app_redirect_url" %q {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  client_id        = stytch_connected_app.test.client_id
  url              = %q
  type             = %q
}
`, name, u, urlType)
}

func parentSurvivedChildWrites() resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr("stytch_connected_app.test", "client_name", "tf-acc-child-urls"),
		resource.TestCheckResourceAttr("stytch_connected_app.test", "client_type", "first_party"),
		resource.TestCheckResourceAttr("stytch_connected_app.test", "access_token_expiry_minutes", "7"),
		resource.TestCheckResourceAttr("stytch_connected_app.test", "access_token_custom_audience", "tf-acc-probe-audience"),
		resource.TestCheckResourceAttr("stytch_connected_app.test", "access_token_template_content", "{}"),
		resource.TestCheckResourceAttrSet("stytch_connected_app.test", "client_secret_last_four"),
	)
}

func TestAccConnectedAppRedirectURLResource(t *testing.T) {
	three := childURL("a", "http://localhost:3000/cb-a", "authorization") +
		childURL("b", "http://localhost:3000/cb-b", "authorization") +
		childURL("c", "http://localhost:3000/logout", "post_logout")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + connectedAppWithChildren(three),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stytch_connected_app_redirect_url.a", "id"),
					resource.TestCheckResourceAttr("stytch_connected_app_redirect_url.a", "type", "authorization"),
					resource.TestCheckResourceAttr("stytch_connected_app_redirect_url.b", "type", "authorization"),
					resource.TestCheckResourceAttr("stytch_connected_app_redirect_url.c", "type", "post_logout"),
				),
			},
			{
				RefreshState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stytch_connected_app_redirect_url.a", "id"),
					resource.TestCheckResourceAttrSet("stytch_connected_app_redirect_url.b", "id"),
					resource.TestCheckResourceAttrSet("stytch_connected_app_redirect_url.c", "id"),
					parentSurvivedChildWrites(),
				),
			},
			{
				Config: testutil.ProviderConfig + connectedAppWithChildren(
					childURL("a", "http://localhost:3000/cb-a", "authorization")+
						childURL("c", "http://localhost:3000/logout", "post_logout")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stytch_connected_app_redirect_url.a", "id"),
					resource.TestCheckResourceAttrSet("stytch_connected_app_redirect_url.c", "id"),
					testutil.TestCheckResourceDeleted("stytch_connected_app_redirect_url.b"),
				),
			},
			{
				RefreshState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_connected_app_redirect_url.a", "url", "http://localhost:3000/cb-a"),
					resource.TestCheckResourceAttr("stytch_connected_app_redirect_url.c", "url", "http://localhost:3000/logout"),
					parentSurvivedChildWrites(),
				),
			},
		},
	})
}
