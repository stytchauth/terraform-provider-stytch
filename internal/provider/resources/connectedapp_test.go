package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/projectapi"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

const projectSecretResource = `
resource "stytch_secret" "test" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
}
`

func connectedAppConfig(fields string) string {
	return testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
		ProjectSlug: "stytch_project.test.project_slug",
		Name:        "Test Environment",
	}) + projectSecretResource + fmt.Sprintf(`
resource "stytch_connected_app" "test" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
%s
}
`, fields)
}

func TestAccConnectedAppResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + connectedAppConfig(`
  client_type   = "first_party"
  client_name   = "tf-acc-full"
  redirect_urls = ["http://localhost:3000/callback"]

  full_access_allowed               = true
  bypass_consent_for_offline_access = true
  access_token_expiry_minutes       = 5
  access_token_custom_audience      = "tf-acc-audience"
  access_token_template_content     = "{\"custom_claim\": \"static_value\"}"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_connected_app.test", "client_name", "tf-acc-full"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "client_type", "first_party"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "access_token_expiry_minutes", "5"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "access_token_custom_audience", "tf-acc-audience"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "access_token_template_content", `{"custom_claim": "static_value"}`),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "id_token_template_content", "{}"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "full_access_allowed", "true"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "redirect_urls.#", "1"),
					resource.TestCheckResourceAttrSet("stytch_connected_app.test", "client_id"),
					resource.TestCheckResourceAttrSet("stytch_connected_app.test", "client_secret"),
					resource.TestCheckResourceAttrSet("stytch_connected_app.test", "client_secret_last_four"),
					resource.TestCheckResourceAttrSet("stytch_connected_app.test", "id"),
				),
			},
			{
				Config: testutil.ProviderConfig + connectedAppConfig(`
  client_type   = "first_party"
  client_name   = "tf-acc-full-renamed"
  redirect_urls = ["http://localhost:3000/callback", "http://localhost:3000/callback2"]

  client_description                = "updated description"
  full_access_allowed               = false
  bypass_consent_for_offline_access = true
  access_token_expiry_minutes       = 10
  access_token_custom_audience      = "tf-acc-audience"
  access_token_template_content     = "{\"custom_claim\": \"static_value\"}"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_connected_app.test", "client_name", "tf-acc-full-renamed"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "client_description", "updated description"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "full_access_allowed", "false"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "access_token_expiry_minutes", "10"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "redirect_urls.#", "2"),
					resource.TestCheckResourceAttrSet("stytch_connected_app.test", "client_secret"),
				),
			},
			{
				Config: testutil.ProviderConfig + connectedAppConfig(`
  client_type   = "first_party"
  client_name   = "tf-acc-full-renamed"
  redirect_urls = ["http://localhost:3000/callback", "http://localhost:3000/callback2"]

  full_access_allowed               = false
  bypass_consent_for_offline_access = true
  access_token_expiry_minutes       = 10
  access_token_custom_audience      = "tf-acc-audience"
  access_token_template_content     = "{\"custom_claim\": \"static_value\"}"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("stytch_connected_app.test", "client_description"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "client_name", "tf-acc-full-renamed"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "access_token_custom_audience", "tf-acc-audience"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "access_token_template_content", `{"custom_claim": "static_value"}`),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "access_token_expiry_minutes", "10"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "redirect_urls.#", "2"),
				),
			},
			{
				ResourceName: "stytch_connected_app.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["stytch_connected_app.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					secret, ok := s.RootModule().Resources["stytch_secret.test"]
					if !ok {
						return "", fmt.Errorf("secret not found in state")
					}
					// The env var is the provider's only route to the project secret during
					// an import, and this func is the last hook that runs before the read.
					t.Setenv(projectapi.ImportSecretEnvVar, secret.Primary.Attributes["secret"])
					return rs.Primary.ID, nil
				},
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"client_secret", "last_updated", "project_secret"},
			},
		},
	})
}

func TestAccConnectedAppResourcePublicClient(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + connectedAppConfig(`
  client_type = "first_party_public"
  client_name = "tf-acc-public"
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_connected_app.test", "client_type", "first_party_public"),
					resource.TestCheckResourceAttrSet("stytch_connected_app.test", "client_id"),
					resource.TestCheckNoResourceAttr("stytch_connected_app.test", "client_secret"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "access_token_template_content", "{}"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "id_token_template_content", "{}"),
				),
			},
			{
				Config: testutil.ProviderConfig + connectedAppConfig(`
  client_type         = "first_party_public"
  client_name         = "tf-acc-public-renamed"
  full_access_allowed = true
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_connected_app.test", "client_name", "tf-acc-public-renamed"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "full_access_allowed", "true"),
				),
			},
		},
	})
}

func TestAccConnectedAppResourceThirdParty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + connectedAppConfig(`
  client_type   = "third_party"
  client_name   = "tf-acc-third-party"
  redirect_urls = ["http://localhost:3000/callback"]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_connected_app.test", "client_type", "third_party"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "full_access_allowed", "false"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "bypass_consent_for_offline_access", "false"),
					resource.TestCheckResourceAttrSet("stytch_connected_app.test", "client_id"),
					resource.TestCheckResourceAttrSet("stytch_connected_app.test", "client_secret"),
				),
			},
			{
				Config: testutil.ProviderConfig + connectedAppConfig(`
  client_type   = "third_party"
  client_name   = "tf-acc-third-party-renamed"
  redirect_urls = ["http://localhost:3000/callback", "http://localhost:3000/callback2"]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_connected_app.test", "client_name", "tf-acc-third-party-renamed"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "redirect_urls.#", "2"),
					resource.TestCheckResourceAttr("stytch_connected_app.test", "full_access_allowed", "false"),
				),
			},
		},
	})
}

func TestAccConnectedAppResourceB2B(t *testing.T) {
	config := testutil.B2BProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
		ProjectSlug: "stytch_project.test.project_slug",
		Name:        "Test Environment",
	}) + projectSecretResource + `
resource "stytch_connected_app" "test" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  client_type      = "first_party"
  client_name      = "tf-acc-b2b"
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_connected_app.test", "client_name", "tf-acc-b2b"),
					resource.TestCheckResourceAttrSet("stytch_connected_app.test", "client_id"),
				),
			},
		},
	})
}

func TestAccConnectedAppResourceInvalidClientType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + connectedAppConfig(`
  client_type = "nonsense"
  client_name = "tf-acc-invalid"
`),
				ExpectError: regexp.MustCompile(`client_type`),
			},
		},
	})
}

func TestAccConnectedAppResourceZeroAccessTokenExpiry(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + connectedAppConfig(`
  client_type                 = "first_party"
  client_name                 = "tf-acc-zero-expiry"
  access_token_expiry_minutes = 0
`),
				ExpectError: regexp.MustCompile(`must be at least 1`),
			},
		},
	})
}

func TestAccConnectedAppResourceFirstPartyOnlyFlags(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + connectedAppConfig(`
  client_type         = "third_party"
  client_name         = "tf-acc-first-party-flag"
  full_access_allowed = true
`),
				ExpectError: regexp.MustCompile(`full_access_allowed is not valid`),
			},
		},
	})
}
