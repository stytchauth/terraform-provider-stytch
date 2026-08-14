package resources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/resources"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccOrganizationDataSource(t *testing.T) {
	t.Setenv(resources.ExperimentalEnvVar, "1")

	var fixture orgFixture
	const orgSlug = "tf-acc-org-datasource"
	const emptyOrgSlug = "tf-acc-org-datasource-empty"

	lookupConfig := orgTrustedMetadataBaseConfig() + orgDataSourceConfig("by_slug", orgSlug) + `
data "stytch_organization" "by_id" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  organization_id  = data.stytch_organization.by_slug.organization_id
}
` + orgDataSourceConfig("empty", emptyOrgSlug)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + orgTrustedMetadataBaseConfig(),
				Check:  captureOrgFixture(&fixture),
			},
			{
				PreConfig: func() {
					createFixtureOrganization(t, &fixture, orgSlug, map[string]any{
						"grants": map[string]any{"tier": "internal"},
					})
					createFixtureOrganization(t, &fixture, emptyOrgSlug, nil)
				},
				Config: testutil.ProviderConfig + lookupConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.stytch_organization.by_slug", "organization_slug", orgSlug),
					resource.TestCheckResourceAttr("data.stytch_organization.by_slug", "organization_name", "tf-acc "+orgSlug),
					resource.TestCheckResourceAttrSet("data.stytch_organization.by_slug", "organization_id"),
					resource.TestCheckResourceAttr("data.stytch_organization.by_slug", "trusted_metadata", `{"grants":{"tier":"internal"}}`),
					resource.TestCheckResourceAttr("data.stytch_organization.by_id", "organization_slug", orgSlug),
					// An organization without trusted_metadata reads as an empty
					// object, never null.
					resource.TestCheckResourceAttr("data.stytch_organization.empty", "trusted_metadata", "{}"),
				),
			},
			{
				Config:      testutil.ProviderConfig + orgTrustedMetadataBaseConfig() + orgDataSourceConfig("missing", "tf-acc-does-not-exist"),
				ExpectError: regexp.MustCompile(`Failed to get organization`),
			},
			{
				// Exactly one lookup attribute must be set.
				Config: testutil.ProviderConfig + orgTrustedMetadataBaseConfig() + `
data "stytch_organization" "ambiguous" {
  project_slug      = stytch_project.test.project_slug
  environment_slug  = stytch_environment.test.environment_slug
  project_secret    = stytch_secret.test.secret
  organization_slug = "one"
  organization_id   = "organization-test-00000000-0000-0000-0000-000000000000"
}
`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}
