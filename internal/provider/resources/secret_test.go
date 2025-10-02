package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccSecretResource(t *testing.T) {
	for _, testCase := range []testutil.TestCase{
		{
			Name: "basic",
			Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + `
      resource "stytch_secret" "test" {
        project_slug = stytch_project.test.project_slug
				environment_slug = stytch_environment.test.environment_slug
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttrSet("stytch_secret.test", "secret_id"),
				resource.TestCheckResourceAttrSet("stytch_secret.test", "secret"),
			},
		},
	} {
		t.Run(testCase.Name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Test Create and Read.
						Config: testutil.ProviderConfig + testCase.Config,
						Check:  resource.ComposeAggregateTestCheckFunc(testCase.Checks...),
					},
					// Delete is automatically tested in resource.TestCase.
				},
			})
		})
	}
}
