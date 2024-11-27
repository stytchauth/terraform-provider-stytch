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
			Config: testutil.ConsumerProjectConfig + `
      resource "stytch_secret" "test" {
        project_id = stytch_project.project.test_project_id
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
						// Create and Read testing
						Config: testutil.ProviderConfig + testCase.Config,
						Check:  resource.ComposeAggregateTestCheckFunc(testCase.Checks...),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}
}
