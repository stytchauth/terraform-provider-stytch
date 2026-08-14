package resources_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stytchauth/stytch-go/v18/stytch/b2b/b2bstytchapi"
	"github.com/stytchauth/stytch-go/v18/stytch/b2b/organizations"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/environments"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/projectapi"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/resources"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

// Organizations cannot be created by Terraform configuration (the resource
// deliberately has no org lifecycle), so the fixture organization is created
// through the API in a step's PreConfig - which only runs under TF_ACC - with
// a known slug that the configuration resolves via the stytch_organization
// data source. The project created by the configuration owns the organization,
// so the harness's destroy cleans everything up.
type orgFixture struct {
	projectSlug     string
	environmentSlug string
	secret          string
}

func captureOrgFixture(fixture *orgFixture) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, target := range map[string]*string{
			"stytch_project.test":     &fixture.projectSlug,
			"stytch_environment.test": &fixture.environmentSlug,
			"stytch_secret.test":      &fixture.secret,
		} {
			rs, ok := s.RootModule().Resources[name]
			if !ok {
				return fmt.Errorf("%s not found in state", name)
			}
			switch name {
			case "stytch_project.test":
				*target = rs.Primary.Attributes["project_slug"]
			case "stytch_environment.test":
				*target = rs.Primary.Attributes["environment_slug"]
			case "stytch_secret.test":
				*target = rs.Primary.Attributes["secret"]
			}
			if *target == "" {
				return fmt.Errorf("%s captured an empty value", name)
			}
		}
		return nil
	}
}

func createFixtureOrganization(t *testing.T, fixture *orgFixture, slug string, trustedMetadata map[string]any) {
	t.Helper()
	ctx := context.Background()

	var mgmtOpts []api.APIOption
	if baseURI := os.Getenv("STYTCH_MANAGEMENT_BASE_URI"); baseURI != "" {
		mgmtOpts = append(mgmtOpts, api.WithBaseURI(baseURI))
	}
	mgmt := api.NewClient(os.Getenv("STYTCH_WORKSPACE_KEY_ID"), os.Getenv("STYTCH_WORKSPACE_KEY_SECRET"), mgmtOpts...)
	envResp, err := mgmt.Environments.Get(ctx, environments.GetRequest{
		ProjectSlug:     fixture.projectSlug,
		EnvironmentSlug: fixture.environmentSlug,
	})
	if err != nil {
		t.Fatalf("resolving fixture project ID: %v", err)
	}

	opts := []b2bstytchapi.Option{b2bstytchapi.WithSkipJWKSInitialization()}
	if baseURI := os.Getenv("STYTCH_PROJECT_API_BASE_URI"); baseURI != "" {
		opts = append(opts, b2bstytchapi.WithBaseURI(baseURI))
	}
	client, err := b2bstytchapi.NewClient(envResp.Environment.ProjectID, fixture.secret, opts...)
	if err != nil {
		t.Fatalf("building fixture project API client: %v", err)
	}

	_, err = client.Organizations.Create(ctx, &organizations.CreateParams{
		OrganizationName: "tf-acc " + slug,
		OrganizationSlug: slug,
		TrustedMetadata:  trustedMetadata,
	})
	if err != nil {
		t.Fatalf("creating fixture organization %q: %v", slug, err)
	}
}

func orgTrustedMetadataBaseConfig() string {
	return testutil.B2BProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
		ProjectSlug: "stytch_project.test.project_slug",
		Name:        "Test Environment",
	}) + projectSecretResource
}

func orgTrustedMetadataConfig(orgSlug string, fields string) string {
	return orgTrustedMetadataBaseConfig() + fmt.Sprintf(`
data "stytch_organization" "test" {
  project_slug      = stytch_project.test.project_slug
  environment_slug  = stytch_environment.test.environment_slug
  project_secret    = stytch_secret.test.secret
  organization_slug = "%s"
}

resource "stytch_organization_trusted_metadata" "test" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  organization_id  = data.stytch_organization.test.organization_id
%s
}
`, orgSlug, fields)
}

func TestAccOrganizationTrustedMetadataResource(t *testing.T) {
	t.Setenv(resources.ExperimentalEnvVar, "1")

	var fixture orgFixture
	const orgSlug = "tf-acc-trusted-metadata"

	// jsonencode renders object keys alphabetically and compactly, matching the
	// canonical form Read stores, so ImportStateVerify can compare strings.
	initialConfig := orgTrustedMetadataConfig(orgSlug, `
  trusted_metadata = {
    grants = jsonencode({ tier = "internal", version = 1 })
    note   = jsonencode("managed-by-terraform")
  }
`)
	updatedConfig := orgTrustedMetadataConfig(orgSlug, `
  trusted_metadata = {
    grants  = jsonencode({ tier = "standard", version = 1 })
    support = jsonencode("gold")
  }
`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + orgTrustedMetadataBaseConfig(),
				Check:  captureOrgFixture(&fixture),
			},
			{
				PreConfig: func() { createFixtureOrganization(t, &fixture, orgSlug, nil) },
				Config:    testutil.ProviderConfig + initialConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_organization_trusted_metadata.test", "trusted_metadata.%", "2"),
					resource.TestCheckResourceAttr("stytch_organization_trusted_metadata.test", "trusted_metadata.grants", `{"tier":"internal","version":1}`),
					resource.TestCheckResourceAttr("stytch_organization_trusted_metadata.test", "trusted_metadata.note", `"managed-by-terraform"`),
					resource.TestCheckResourceAttrSet("stytch_organization_trusted_metadata.test", "id"),
					resource.TestCheckResourceAttr("data.stytch_organization.test", "organization_slug", orgSlug),
					resource.TestCheckResourceAttr("data.stytch_organization.test", "organization_name", "tf-acc "+orgSlug),
					resource.TestCheckResourceAttrSet("data.stytch_organization.test", "organization_id"),
				),
			},
			{
				// Convergence: refresh must retain the applied values exactly.
				Config:             testutil.ProviderConfig + initialConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// Update changes one value, adds a key, and removes a key.
				Config: testutil.ProviderConfig + updatedConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_organization_trusted_metadata.test", "trusted_metadata.%", "2"),
					resource.TestCheckResourceAttr("stytch_organization_trusted_metadata.test", "trusted_metadata.grants", `{"tier":"standard","version":1}`),
					resource.TestCheckResourceAttr("stytch_organization_trusted_metadata.test", "trusted_metadata.support", `"gold"`),
					resource.TestCheckNoResourceAttr("stytch_organization_trusted_metadata.test", "trusted_metadata.note"),
				),
			},
			{
				// The removed key must be gone remotely, not just from state.
				Config:             testutil.ProviderConfig + updatedConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      "stytch_organization_trusted_metadata.test",
				ImportState:       true,
				ImportStateVerify: true,
				// project_secret and last_updated are never returned by the API;
				// force is configuration-only.
				ImportStateVerifyIgnore: []string{"project_secret", "last_updated", "force"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					t.Setenv(projectapi.ImportSecretEnvVar, fixture.secret)
					rs, ok := s.RootModule().Resources["stytch_organization_trusted_metadata.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return rs.Primary.ID, nil
				},
			},
		},
	})
}

func TestAccOrganizationTrustedMetadataForce(t *testing.T) {
	t.Setenv(resources.ExperimentalEnvVar, "1")

	var fixture orgFixture
	const orgSlug = "tf-acc-trusted-metadata-force"

	config := func(fields string) string {
		return testutil.ProviderConfig + orgTrustedMetadataConfig(orgSlug, fields)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + orgTrustedMetadataBaseConfig(),
				Check:  captureOrgFixture(&fixture),
			},
			{
				// The fixture organization already has app-written metadata, so
				// creation without force must refuse.
				PreConfig: func() {
					createFixtureOrganization(t, &fixture, orgSlug, map[string]any{
						"app_owned": map[string]any{"source": "application"},
					})
				},
				Config: config(`
  trusted_metadata = {
    grants = jsonencode({ tier = "internal" })
  }
`),
				ExpectError: regexp.MustCompile(`already has trusted_metadata`),
			},
			{
				// force takes ownership: the configured object replaces everything,
				// including the app-written key.
				Config: config(`
  force = true

  trusted_metadata = {
    grants = jsonencode({ tier = "internal" })
  }
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_organization_trusted_metadata.test", "trusted_metadata.%", "1"),
					resource.TestCheckResourceAttr("stytch_organization_trusted_metadata.test", "trusted_metadata.grants", `{"tier":"internal"}`),
				),
			},
			{
				// app_owned must be deleted remotely: if the null-punch failed, the
				// refresh here would adopt it into state and produce a diff.
				Config: config(`
  force = true

  trusted_metadata = {
    grants = jsonencode({ tier = "internal" })
  }
`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccOrganizationTrustedMetadataExperimentalGate(t *testing.T) {
	t.Setenv(resources.ExperimentalEnvVar, "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutil.ProviderConfig + orgTrustedMetadataBaseConfig() + `
resource "stytch_organization_trusted_metadata" "test" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_environment.test.environment_slug
  project_secret   = stytch_secret.test.secret
  organization_id  = "organization-test-00000000-0000-0000-0000-000000000000"

  trusted_metadata = {
    grants = jsonencode({})
  }
}
`,
				ExpectError: regexp.MustCompile(`Experimental feature not enabled`),
			},
		},
	})
}
