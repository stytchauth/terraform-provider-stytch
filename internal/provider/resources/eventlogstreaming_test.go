package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/eventlogstreaming"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

type testConfig struct {
	TestName                 string
	DestinationType          eventlogstreaming.DestinationType
	InitialDatadogConfig     eventlogstreaming.DatadogConfig
	InitialGrafanaLokiConfig eventlogstreaming.GrafanaLokiConfig
	UpdatedDatadogConfig     eventlogstreaming.DatadogConfig
	UpdatedGrafanaLokiConfig eventlogstreaming.GrafanaLokiConfig
}

func destinationConfigString(t *testing.T, tc testConfig, useUpdated bool) string {
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

	for _, tc := range []testConfig{
		{
			TestName:        "datadog",
			DestinationType: eventlogstreaming.DestinationTypeDatadog,
			InitialDatadogConfig: eventlogstreaming.DatadogConfig{
				Site:   eventlogstreaming.DatadogSiteUS,
				APIKey: "0123456789abcdef0123456789abcdef",
			},
			InitialGrafanaLokiConfig: eventlogstreaming.GrafanaLokiConfig{},
			UpdatedDatadogConfig: eventlogstreaming.DatadogConfig{
				Site:   eventlogstreaming.DatadogSiteEU,
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
				Password: "drowssap",
			},
		},
	} {
		t.Run(tc.TestName, func(t *testing.T) {
			// Build initial Terraform configuration.
			projectConfig := testutil.ConsumerProjectConfig

			initialDestinationConfig := destinationConfigString(t, tc, false)

			initialConfig := projectConfig + fmt.Sprintf(`
				resource "stytch_event_log_streaming" "test" {
					project_id = stytch_project.project.test_project_id
					destination_type = "%s"
					%s
				}
				`, string(tc.DestinationType), initialDestinationConfig)

			// Check initial configuration.
			initialChecks := []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "destination_type", string(tc.DestinationType)),
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

			// Update terraform configuration
			updatedDestinationConfig := destinationConfigString(t, tc, true)
			updatedConfig := projectConfig + fmt.Sprintf(`
				resource "stytch_event_log_streaming" "test" {
					project_id = stytch_project.project.test_project_id
					destination_type = "%s"
					%s
				}
				`, string(tc.DestinationType), updatedDestinationConfig)

			// Check updated configuration.
			updatedChecks := []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "destination_type", string(tc.DestinationType)),
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
			deleteConfig := projectConfig

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Test Create and Read.
						Config: testutil.ProviderConfig + initialConfig,
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
						// Test Update and Read.
						Config: testutil.ProviderConfig + updatedConfig,
						Check:  resource.ComposeAggregateTestCheckFunc(updatedChecks...),
					},
					{
						// Test Delete and Read.
						Config: testutil.ProviderConfig + deleteConfig,
						Check:  testutil.TestCheckResourceDeleted(resourceName),
					},
				},
			})
		})
	}
}
