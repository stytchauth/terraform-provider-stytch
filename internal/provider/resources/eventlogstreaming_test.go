package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/eventlogstreaming"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

type eventLogStreamingTestConfig struct {
	TestName                 string
	DestinationType          eventlogstreaming.DestinationType
	InitialDatadogConfig     eventlogstreaming.DatadogConfig
	InitialGrafanaLokiConfig eventlogstreaming.GrafanaLokiConfig
	UpdatedDatadogConfig     eventlogstreaming.DatadogConfig
	UpdatedGrafanaLokiConfig eventlogstreaming.GrafanaLokiConfig
}

func destinationConfigString(t *testing.T, tc eventLogStreamingTestConfig, useUpdated bool) string {
	t.Helper()

	var destinationConfig string

	switch tc.DestinationType {
	case eventlogstreaming.DestinationTypeDatadog:
		var config eventlogstreaming.DatadogConfig
		if useUpdated {
			config = tc.UpdatedDatadogConfig
		} else {
			config = tc.InitialDatadogConfig
		}
		destinationConfig = fmt.Sprintf(`
			datadog_config = {
				site = "%s"
				api_key = "%s"
			}
		`, config.Site, config.APIKey)
	case eventlogstreaming.DestinationTypeGrafanaLoki:
		var config eventlogstreaming.GrafanaLokiConfig
		if useUpdated {
			config = tc.UpdatedGrafanaLokiConfig
		} else {
			config = tc.InitialGrafanaLokiConfig
		}
		destinationConfig = fmt.Sprintf(`
			grafana_loki_config = {
				hostname = "%s"
				username = "%s"
				password = "%s"
			}
		`, config.Hostname, config.Username, config.Password)
	default:
		t.Fatalf("unexpected destination type: %s", tc.DestinationType)
	}
	return destinationConfig
}

// TestAccEventLogStreamingResource performs acceptance tests for the
// stytch_event_log_streaming resource.
func TestAccEventLogStreamingResource(t *testing.T) {
	const resourceName = "stytch_event_log_streaming.test"

	for _, tc := range []eventLogStreamingTestConfig{
		{
			TestName:        "datadog",
			DestinationType: eventlogstreaming.DestinationTypeDatadog,
			InitialDatadogConfig: eventlogstreaming.DatadogConfig{
				Site:   eventlogstreaming.DatadogSiteUs,
				APIKey: "0123456789abcdef0123456789abcdef",
			},
			InitialGrafanaLokiConfig: eventlogstreaming.GrafanaLokiConfig{},
			UpdatedDatadogConfig: eventlogstreaming.DatadogConfig{
				Site:   eventlogstreaming.DatadogSiteEu,
				APIKey: "ffffffffffffffffffffffffffffffff",
			},
			UpdatedGrafanaLokiConfig: eventlogstreaming.GrafanaLokiConfig{},
		},
		{
			TestName:             "grafana_loki",
			DestinationType:      eventlogstreaming.DestinationTypeGrafanaLoki,
			InitialDatadogConfig: eventlogstreaming.DatadogConfig{},
			InitialGrafanaLokiConfig: eventlogstreaming.GrafanaLokiConfig{
				Hostname: "loki.example.stytch.com",
				Username: "loki",
				Password: "password",
			},
			UpdatedDatadogConfig: eventlogstreaming.DatadogConfig{},
			UpdatedGrafanaLokiConfig: eventlogstreaming.GrafanaLokiConfig{
				Hostname: "loki2.example.stytch.com",
				Username: "loki2",
				Password: "thisisnotaverysecurepassword",
			},
		},
	} {
		t.Run(tc.TestName, func(t *testing.T) {
			// Build initial Terraform configuration.
			projectConfig := testutil.ConsumerProjectConfig

			initialDestinationConfig := destinationConfigString(t, tc, false)

			initialConfig := projectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + fmt.Sprintf(`
				resource "stytch_event_log_streaming" "test" {
					project_slug = stytch_project.test.project_slug
					environment_slug = stytch_environment.test.environment_slug
					destination_type = "%s"
					%s
				}
				`, string(tc.DestinationType), initialDestinationConfig)

			// Check initial configuration.
			initialChecks := []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "destination_type", string(tc.DestinationType)),
				resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
			}
			switch tc.DestinationType {
			case eventlogstreaming.DestinationTypeDatadog:
				initialChecks = append(initialChecks, resource.TestCheckResourceAttr(resourceName, "datadog_config.site", string(tc.InitialDatadogConfig.Site)))
				initialChecks = append(initialChecks, resource.TestCheckResourceAttr(resourceName, "datadog_config.api_key", tc.InitialDatadogConfig.APIKey))
			case eventlogstreaming.DestinationTypeGrafanaLoki:
				initialChecks = append(initialChecks, resource.TestCheckResourceAttr(resourceName, "grafana_loki_config.hostname", tc.InitialGrafanaLokiConfig.Hostname))
				initialChecks = append(initialChecks, resource.TestCheckResourceAttr(resourceName, "grafana_loki_config.username", tc.InitialGrafanaLokiConfig.Username))
				initialChecks = append(initialChecks, resource.TestCheckResourceAttr(resourceName, "grafana_loki_config.password", tc.InitialGrafanaLokiConfig.Password))
			}

			// Disable configuration for testing enabled/disabled status
			disabledDestinationConfig := destinationConfigString(t, tc, false)
			disabledConfig := projectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + fmt.Sprintf(`
				resource "stytch_event_log_streaming" "test" {
					project_slug = stytch_project.test.project_slug
					environment_slug = stytch_environment.test.environment_slug
					destination_type = "%s"
					enabled = false
					%s
				}
				`, string(tc.DestinationType), disabledDestinationConfig)

			// Enable and update terraform configuration
			updatedDestinationConfig := destinationConfigString(t, tc, true)
			updatedConfig := projectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + fmt.Sprintf(`
				resource "stytch_event_log_streaming" "test" {
					project_slug = stytch_project.test.project_slug
					environment_slug = stytch_environment.test.environment_slug
					destination_type = "%s"
					enabled = true
					%s
				}
				`, string(tc.DestinationType), updatedDestinationConfig)

			// Check updated configuration.
			updatedChecks := []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "destination_type", string(tc.DestinationType)),
				resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
			}
			switch tc.DestinationType {
			case eventlogstreaming.DestinationTypeDatadog:
				updatedChecks = append(updatedChecks, resource.TestCheckResourceAttr(resourceName, "datadog_config.site", string(tc.UpdatedDatadogConfig.Site)))
				updatedChecks = append(updatedChecks, resource.TestCheckResourceAttr(resourceName, "datadog_config.api_key", tc.UpdatedDatadogConfig.APIKey))
			case eventlogstreaming.DestinationTypeGrafanaLoki:
				updatedChecks = append(updatedChecks, resource.TestCheckResourceAttr(resourceName, "grafana_loki_config.hostname", tc.UpdatedGrafanaLokiConfig.Hostname))
				updatedChecks = append(updatedChecks, resource.TestCheckResourceAttr(resourceName, "grafana_loki_config.username", tc.UpdatedGrafanaLokiConfig.Username))
				updatedChecks = append(updatedChecks, resource.TestCheckResourceAttr(resourceName, "grafana_loki_config.password", tc.UpdatedGrafanaLokiConfig.Password))
			}

			// Build delete Terraform configuration.
			deleteConfig := projectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			})

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Test Create and Read.
						Config: initialConfig,
						Check:  resource.ComposeAggregateTestCheckFunc(initialChecks...),
					},
					{
						// Test ImportState.
						ResourceName:      resourceName,
						ImportState:       true,
						ImportStateVerify: true,
						// Sensitive values are not imported, so we ignore them in the test
						ImportStateVerifyIgnore: []string{"last_updated", "datadog_config.api_key", "grafana_loki_config.password"},
					},
					{
						// Test Update - disable streaming.
						Config: disabledConfig,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
						),
					},
					{
						// Test Update - re-enable and change config.
						Config: updatedConfig,
						Check:  resource.ComposeAggregateTestCheckFunc(updatedChecks...),
					},
					{
						// Test Delete and Read.
						Config: deleteConfig,
						Check:  testutil.TestCheckResourceDeleted(resourceName),
					},
				},
			})
		})
	}
}

func TestAccEventLogStreamingResource_Invalid(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Test missing datadog_config when destination_type is DATADOG
				Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Test Environment",
				}) + `
        resource "stytch_event_log_streaming" "test" {
          project_slug     = stytch_project.test.project_slug
          environment_slug = stytch_environment.test.environment_slug
          destination_type = "DATADOG"
        }`,
				ExpectError: regexp.MustCompile("datadog_config block is required"),
			},
			{
				// Test missing grafana_loki_config when destination_type is GRAFANA_LOKI
				Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Test Environment",
				}) + `
        resource "stytch_event_log_streaming" "test" {
          project_slug     = stytch_project.test.project_slug
          environment_slug = stytch_environment.test.environment_slug
          destination_type = "GRAFANA_LOKI"
        }`,
				ExpectError: regexp.MustCompile("grafana_loki_config block is required"),
			},
			{
				// Test mixing datadog_config with GRAFANA_LOKI destination
				Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Test Environment",
				}) + `
        resource "stytch_event_log_streaming" "test" {
          project_slug     = stytch_project.test.project_slug
          environment_slug = stytch_environment.test.environment_slug
          destination_type = "GRAFANA_LOKI"

          grafana_loki_config = {
            hostname = "logs.example.com"
            username = "test-user"
            password = "test-password"
          }

          datadog_config = {
            site    = "US"
            api_key = "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"
          }
        }`,
				ExpectError: regexp.MustCompile("datadog_config block is not allowed"),
			},
			{
				// Test invalid API key length
				Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Test Environment",
				}) + `
        resource "stytch_event_log_streaming" "test" {
          project_slug     = stytch_project.test.project_slug
          environment_slug = stytch_environment.test.environment_slug
          destination_type = "DATADOG"

          datadog_config = {
            site    = "US"
            api_key = "tooshort"
          }
        }`,
				ExpectError: regexp.MustCompile("string length must be between 32 and 32"),
			},
			{
				// Test invalid API key format (not hex)
				Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
					ProjectSlug: "stytch_project.test.project_slug",
					Name:        "Test Environment",
				}) + `
        resource "stytch_event_log_streaming" "test" {
          project_slug     = stytch_project.test.project_slug
          environment_slug = stytch_environment.test.environment_slug
          destination_type = "DATADOG"

          datadog_config = {
            site    = "US"
            api_key = "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ"
          }
        }`,
				ExpectError: regexp.MustCompile("must be a hex string"),
			},
		},
	})
}

func TestAccEventLogStreamingResourceStateUpgrade(t *testing.T) {
	v1Config := testutil.V1ConsumerProjectConfig + `
resource "stytch_event_log_streaming" "test" {
  project_id       = stytch_project.test.live_project_id
  destination_type = "DATADOG"
  datadog_config = {
    api_key = "0123456789abcdef0123456789abcdef"
    site    = "US"
  }
}
`

	v3Config := testutil.ConsumerProjectConfig + `
resource "stytch_event_log_streaming" "test" {
  project_slug     = stytch_project.test.project_slug
  environment_slug = stytch_project.test.live_environment.environment_slug
  destination_type = "DATADOG"
  datadog_config = {
    api_key = "0123456789abcdef0123456789abcdef"
    site    = "US"
  }
}
`

	resource.Test(t, resource.TestCase{
		Steps: testutil.StateUpgradeTestSteps(v1Config, v3Config),
	})
}
