package resources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccConnectedAppDataSource(t *testing.T) {
	base := testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
		ProjectSlug: "stytch_project.test.project_slug",
		Name:        "Test Environment",
	}) + projectSecretResource + `
resource "stytch_connected_app" "test" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  client_type      = "first_party"
  client_name      = "tf-acc-ds-lookup"
  redirect_urls    = ["http://localhost:3000/callback"]

  access_token_custom_audience = "tf-acc-ds-audience"
}
`
	withDataSource := base + `
data "stytch_connected_app" "by_name" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  client_name      = "tf-acc-ds-lookup"

  depends_on = [stytch_connected_app.test]
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + base,
			},
			{
				Config: testutil.ProviderConfig + withDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.stytch_connected_app.by_name", "client_id", "stytch_connected_app.test", "client_id"),
					resource.TestCheckResourceAttrPair("data.stytch_connected_app.by_name", "client_type", "stytch_connected_app.test", "client_type"),
					resource.TestCheckResourceAttrPair("data.stytch_connected_app.by_name", "access_token_custom_audience", "stytch_connected_app.test", "access_token_custom_audience"),
					resource.TestCheckResourceAttrPair("data.stytch_connected_app.by_name", "client_secret_last_four", "stytch_connected_app.test", "client_secret_last_four"),
					resource.TestCheckResourceAttr("data.stytch_connected_app.by_name", "redirect_urls.#", "1"),
					resource.TestCheckNoResourceAttr("data.stytch_connected_app.by_name", "client_secret"),
				),
			},
		},
	})
}

func TestAccConnectedAppDataSourceNotFound(t *testing.T) {
	config := testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
		ProjectSlug: "stytch_project.test.project_slug",
		Name:        "Test Environment",
	}) + projectSecretResource + `
data "stytch_connected_app" "missing" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  client_name      = "tf-acc-ds-does-not-exist"
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testutil.ProviderConfig + config,
				ExpectError: regexp.MustCompile("Connected app not found"),
			},
		},
	})
}

func TestAccConnectedAppDataSourceAmbiguous(t *testing.T) {
	config := testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
		ProjectSlug: "stytch_project.test.project_slug",
		Name:        "Test Environment",
	}) + projectSecretResource + `
resource "stytch_connected_app" "first" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  client_type      = "first_party"
  client_name      = "tf-acc-ds-duplicate"
}

resource "stytch_connected_app" "second" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  client_type      = "first_party_public"
  client_name      = "tf-acc-ds-duplicate"
}

data "stytch_connected_app" "dup" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  client_name      = "tf-acc-ds-duplicate"

  depends_on = [stytch_connected_app.first, stytch_connected_app.second]
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testutil.ProviderConfig + config,
				ExpectError: regexp.MustCompile("Connected app name is ambiguous"),
			},
		},
	})
}
