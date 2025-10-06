package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccPublicTokenResource(t *testing.T) {
	for _, testCase := range []testutil.TestCase{
		{
			Name: "basic",
			Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + `
      resource "stytch_public_token" "test" {
        project_slug = stytch_project.test.project_slug
				environment_slug = stytch_environment.test.environment_slug
      }

      # Create a second public token to ensure we can delete the first one
      # (API prevents deleting the last public token for an environment)
      resource "stytch_public_token" "test2" {
        project_slug = stytch_project.test.project_slug
				environment_slug = stytch_environment.test.environment_slug
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttrSet("stytch_public_token.test", "public_token"),
				resource.TestCheckResourceAttrSet("stytch_public_token.test", "created_at"),
				resource.TestCheckResourceAttrSet("stytch_public_token.test", "project_slug"),
				resource.TestCheckResourceAttrSet("stytch_public_token.test", "environment_slug"),
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
					{
						// Test ImportState.
						ResourceName:      "stytch_public_token.test",
						ImportState:       true,
						ImportStateVerify: true,
						// Ignore timestamp field
						ImportStateVerifyIgnore: []string{"created_at"},
					},
					// Delete is automatically tested in resource.TestCase.
				},
			})
		})
	}
}
