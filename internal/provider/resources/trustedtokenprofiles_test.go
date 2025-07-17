package resources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func pemFileConfigString(t *testing.T, pemFiles []string) string {
	t.Helper()

	var pemFileConfig string

	if len(pemFiles) == 0 {
		return ""
	}

	pemFileConfig = `					pem_files = [
	`
	for _, pemFile := range pemFiles {
		// Note that we need to escape the newline characters in the PEM file
		// We use this with the heredoc syntax to ensure that the PEM file is properly formatted
		pemFileConfig += fmt.Sprintf(`					{ public_key = <<EOT
		"%s"
		EOT
		}
		`, pemFile)
	}

	pemFileConfig += `
		]
	`
	return pemFileConfig
}

// TestAccTrustedTokenProfileResource performs acceptance tests for the
// stytch_trusted_token_profiles resource.
func TestAccTrustedTokenProfilesResource(t *testing.T) {
	const resourceName = "stytch_trusted_token_profiles.test_profile"

	type testConfig struct {
		Name             string
		Audience         string
		Issuer           string
		JwksUrl          string
		PemFiles         []string
		AttributeMapping map[string]string
		PublicKeyType    string
	}

	for _, tc := range []struct {
		TestName string
		Initial  testConfig
		Update   testConfig
	}{
		{
			TestName: "trusted_token_profile_jwk",
			Initial: testConfig{
				Name:          "Test Profile JWK",
				Audience:      "test-profile-jwk",
				Issuer:        "https://test-profile-jwk-issuer.com",
				PublicKeyType: "jwk",
				JwksUrl:       "https://test-profile-jwk-issuer.com/.well-known/jwks.json",
			},
			Update: testConfig{
				Name:     "test-profile-jwk-updated",
				Audience: "test-profile-jwk-updated",
				Issuer:   "https://test-profile-jwk-issuer-updated.com",
				JwksUrl:  "https://test-profile-jwk-issuer-updated.com/.well-known/jwks.json",
			},
		},
		// {
		// 	TestName: "trusted_token_profile_pem",
		// 	Initial: testConfig{
		// 		Name:          "Test Profile PEM",
		// 		Audience:      "test-profile-pem",
		// 		Issuer:        "https://test-profile-pem-issuer.com",
		// 		PublicKeyType: "pem",
		// 		PemFiles: []string{
		// 			"-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkhhkiG9w0BAQEEOCAQ8AMIIBCgKCAQEA4f5wg5l2hKsTeNem/V41\nfGnJm6gOdrj8ym3rFkEjWT2btYK36hY+c2QKfPU5O7w=\n-----END PUBLIC KEY-----",
		// 		},
		// 	},
		// 	Update: testConfig{
		// 		Name:     "Test Profile PEM Updated",
		// 		Audience: "test-profile-pem-updated",
		// 		Issuer:   "https://test-profile-pem-issuer-updated.com",
		// 		PemFiles: []string{
		// 			"-----BEGIN PUBLIC KEY-----\nMMMBIjANBgkhhkiG9w0BAQEEOCAQ8AMIIBCgKCAQEA4f5wg5l2hKsTeNem/V41\nfGnJm6gOdrj8ym3rFkEjWT2btYK36hY+c2QKfPU5O7w=\n-----END PUBLIC KEY-----",
		// 		},
		// 	},
		// },
		// {
		// 	TestName: "trusted_token_profile_with_attributes",
		// 	Initial: testConfig{
		// 		Name:             "Test Profile Attributes",
		// 		Audience:         "test-profile-attributes",
		// 		Issuer:           "https://test-profile-attributes-issuer.com",
		// 		PublicKeyType:    "jwk",
		// 		JwksUrl:          "https://test-profile-attributes-issuer.com/.well-known/jwks.json",
		// 		AttributeMapping: map[string]string{"email": "example@example.com", "name": "example"},
		// 	},
		// 	Update: testConfig{
		// 		Name:             "Test Profile Attributes Updated",
		// 		Audience:         "test-profile-attributes-updated",
		// 		Issuer:           "https://test-profile-attributes-issuer-updated.com",
		// 		AttributeMapping: map[string]string{"email": "example-updated@example.com", "name": "example"},
		// 	},
		// },
	} {
		t.Run(tc.TestName, func(t *testing.T) {
			// Build initial Terraform configuration.
			projectConfig := testutil.ConsumerProjectConfig

			initialResourceConfig := fmt.Sprintf(`
				resource "stytch_trusted_token_profiles" "test_profile" {
					project_id = stytch_project.project.test_project_id
					name       = "%s"
					audience   = "%s"
					issuer     = "%s"
					public_key_type = "%s"
			`, tc.Initial.Name, tc.Initial.Audience, tc.Initial.Issuer, tc.Initial.PublicKeyType)

			// Add JWKS URL if provided
			if tc.Initial.JwksUrl != "" {
				initialResourceConfig += fmt.Sprintf(`
					jwks_url   = "%s"
				`, tc.Initial.JwksUrl)
			}

			// Add attribute mapping if provided
			if len(tc.Initial.AttributeMapping) > 0 {
				initialResourceConfig += `
					attribute_mapping = {
				`
				for k, v := range tc.Initial.AttributeMapping {
					initialResourceConfig += fmt.Sprintf(`						"%s" = "%s"
					`, k, v)
				}
				initialResourceConfig += `
					}
				`
			}

			// Add PEM files if provided
			if len(tc.Initial.PemFiles) > 0 {
				initialResourceConfig += pemFileConfigString(t, tc.Initial.PemFiles)
			}

			initialResourceConfig += `
				}
			`

			initialConfig := projectConfig + initialResourceConfig

			fmt.Println("initialConfig: ", initialConfig)

			// Check initial configuration.
			initialChecks := []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(resourceName, "name", tc.Initial.Name),
				resource.TestCheckResourceAttr(resourceName, "audience", tc.Initial.Audience),
				resource.TestCheckResourceAttr(resourceName, "issuer", tc.Initial.Issuer),
				resource.TestCheckResourceAttrSet(resourceName, "profile_id"),
				resource.TestCheckResourceAttrSet(resourceName, "id"),
				resource.TestCheckResourceAttrSet(resourceName, "last_updated"),
				resource.TestCheckResourceAttr(resourceName, "public_key_type", tc.Initial.PublicKeyType),
			}

			if tc.Initial.JwksUrl != "" {
				initialChecks = append(initialChecks, resource.TestCheckResourceAttr(resourceName, "jwks_url", tc.Initial.JwksUrl))
			}

			if len(tc.Initial.AttributeMapping) > 0 {
				for k, v := range tc.Initial.AttributeMapping {
					initialChecks = append(initialChecks, resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("attribute_mapping.%s", k), v))
				}
			}

			if len(tc.Initial.PemFiles) > 0 {
				initialChecks = append(initialChecks, resource.TestCheckResourceAttr(resourceName, "pem_files.#", fmt.Sprintf("%d", len(tc.Initial.PemFiles))))
			}

			// // Build update Terraform configuration
			// updateResourceConfig := fmt.Sprintf(`
			// 	resource "stytch_trusted_token_profiles" "test_profile" {
			// 		project_id = stytch_project.project.test_project_id
			// 		name       = "%s"
			// 		audience   = "%s"
			// 		issuer     = "%s"
			// 		public_key_type = "%s"
			// `, tc.Update.Name, tc.Update.Audience, tc.Update.Issuer, tc.Update.PublicKeyType)

			// // Add JWKS URL if provided
			// if tc.Update.JwksUrl != "" {
			// 	updateResourceConfig += fmt.Sprintf(`
			// 		jwks_url   = "%s"
			// 	`, tc.Update.JwksUrl)
			// }

			// // Add attribute mapping if provided
			// if len(tc.Update.AttributeMapping) > 0 {
			// 	updateResourceConfig += `
			// 		attribute_mapping = {
			// 	`
			// 	for k, v := range tc.Update.AttributeMapping {
			// 		updateResourceConfig += fmt.Sprintf(`						"%s" = "%s"
			// 		`, k, v)
			// 	}
			// 	updateResourceConfig += `
			// 		}
			// 	`
			// }

			// // Add PEM files if provided
			// if len(tc.Update.PemFiles) > 0 {
			// 	updateResourceConfig += pemFileConfigString(t, tc.Update.PemFiles)
			// }

			// updateResourceConfig += `
			// 	}
			// `

			// updateConfig := projectConfig + updateResourceConfig

			// // Check updated configuration.
			// updateChecks := []resource.TestCheckFunc{
			// 	resource.TestCheckResourceAttr(resourceName, "name", tc.Update.Name),
			// 	resource.TestCheckResourceAttr(resourceName, "audience", tc.Update.Audience),
			// 	resource.TestCheckResourceAttr(resourceName, "issuer", tc.Update.Issuer),
			// }

			// if tc.Update.JwksUrl != "" {
			// 	updateChecks = append(updateChecks, resource.TestCheckResourceAttr(resourceName, "jwks_url", tc.Update.JwksUrl))
			// }

			// if len(tc.Update.AttributeMapping) > 0 {
			// 	for k, v := range tc.Update.AttributeMapping {
			// 		updateChecks = append(updateChecks, resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("attribute_mapping.%s", k), v))
			// 	}
			// }

			// if len(tc.Update.PemFiles) > 0 {
			// 	updateChecks = append(updateChecks, resource.TestCheckResourceAttr(resourceName, "pem_files.#", fmt.Sprintf("%d", len(tc.Update.PemFiles))))
			// }

			// // Build delete Terraform configuration.
			// deleteConfig := projectConfig

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Test Create and Read.
						Config: testutil.ProviderConfig + initialConfig,
						Check:  resource.ComposeAggregateTestCheckFunc(initialChecks...),
					},
					// {
					// 	// Test ImportState.
					// 	ResourceName:            resourceName,
					// 	ImportState:             true,
					// 	ImportStateVerify:       true,
					// 	ImportStateVerifyIgnore: []string{"last_updated"},
					// },
					// {
					// 	// Test Update and Read.
					// 	Config: testutil.ProviderConfig + updateConfig,
					// 	Check:  resource.ComposeAggregateTestCheckFunc(updateChecks...),
					// },
					// {
					// 	// Test Delete and Read.
					// 	Config: testutil.ProviderConfig + deleteConfig,
					// 	Check:  testutil.TestCheckResourceDeleted(resourceName),
					// },
				},
			})
		})
	}
}

// TestAccTrustedTokenProfilesResource_Invalid tests invalid configurations for
// stytch_trusted_token_profiles.
func TestAccTrustedTokenProfileResource_Invalid(t *testing.T) {
	for _, errorCase := range []testutil.ErrorCase{
		{
			Name: "missing required fields",
			Config: testutil.ConsumerProjectConfig + `
				resource "stytch_trusted_token_profiles" "test_profile" {
					project_id = stytch_project.project.test_project_id
				}
				`,
			Error: regexp.MustCompile(`.*The argument "name" is required.*`),
		},
		{
			Name: "missing audience",
			Config: testutil.ConsumerProjectConfig + `
				resource "stytch_trusted_token_profiles" "test_profile" {
					project_id = stytch_project.project.test_project_id
					name       = "test-profile"
					issuer     = "https://auth.example.com"
				}
				`,
			Error: regexp.MustCompile(`.*The argument "audience" is required.*`),
		},
		{
			Name: "missing issuer",
			Config: testutil.ConsumerProjectConfig + `
				resource "stytch_trusted_token_profiles" "test_profile" {
					project_id = stytch_project.project.test_project_id
					name       = "test-profile"
					audience   = "https://example.com"
				}
				`,
			Error: regexp.MustCompile(`.*The argument "issuer" is required.*`),
		},
	} {
		if errorCase.Error == nil {
			errorCase.AssertAnyError(t)
		} else {
			errorCase.AssertErrorWith(t, errorCase.Error)
		}
	}
}

// TestAccTrustedTokenProfileResource_PlanEmpty tests that the same configuration does not result
// in planned updates.
// func TestAccTrustedTokenProfileResource_PlanEmpty(t *testing.T) {
// 	const resourceName = "stytch_trusted_token_profiles.test_profile"

// 	type testConfig struct {
// 		Name             string
// 		Audience         string
// 		Issuer           string
// 		JwksUrl          string
// 		PemFiles         []string
// 		AttributeMapping map[string]string
// 		PublicKeyType    string
// 	}

// 	for _, tc := range []struct {
// 		TestName string
// 		Config   testConfig
// 	}{
// 		{
// 			TestName: "profile_with_jwks",
// 			Config: testConfig{
// 				Name:             "Test Profile JWK",
// 				Audience:         "test-profile-jwk",
// 				Issuer:           "https://test-profile-jwk-issuer.com",
// 				PublicKeyType:    "jwk",
// 				JwksUrl:          "https://test-profile-jwk-issuer.com/.well-known/jwks.json",
// 				AttributeMapping: map[string]string{"email": "example@example.com", "name": "example"},
// 			},
// 		},
// 		{
// 			TestName: "profile_with_pem",
// 			Config: testConfig{
// 				Name:          "Test Profile PEM",
// 				Audience:      "test-profile-pem",
// 				Issuer:        "https://test-profile-pem-issuer.com",
// 				PublicKeyType: "pem",
// 				PemFiles: []string{
// 					"-----BEGIN PUBLIC KEY-----\nMIIBIjBNBgkhhkiG9w0BAQEEOCAQ8AMIIBCgKCAQEA4f5wg5l2hKsTeNem/V41\nfGnJm6gOdrj8ym3rFkEjWT2btYK36hY+c2QKfPU5O7w=\n-----END PUBLIC KEY-----",
// 				},
// 			},
// 		},
// 		{
// 			TestName: "profile_without_attribute_mapping",
// 			Config: testConfig{
// 				Name:          "Test Profile No Attributes",
// 				Audience:      "test-profile-no-attributes",
// 				Issuer:        "https://test-profile-no-attributes-issuer.com",
// 				PublicKeyType: "jwk",
// 				JwksUrl:       "https://test-profile-no-attributes-issuer.com/.well-known/jwks.json",
// 			},
// 		},
// 	} {
// 		t.Run(tc.TestName, func(t *testing.T) {
// 			// Build initial Terraform configuration.
// 			projectConfig := testutil.ConsumerProjectConfig

// 			// Build resource configuration
// 			resourceConfig := fmt.Sprintf(`
// 				resource "stytch_trusted_token_profiles" "test_profile" {
// 					project_id = stytch_project.project.test_project_id
// 					name       = "%s"
// 					audience   = "%s"
// 					issuer     = "%s"
// 					public_key_type = "%s"
// 			`, tc.Config.Name, tc.Config.Audience, tc.Config.Issuer, tc.Config.PublicKeyType)

// 			// Add JWKS URL if provided
// 			if tc.Config.JwksUrl != "" {
// 				resourceConfig += fmt.Sprintf(`
// 					jwks_url   = "%s"
// 				`, tc.Config.JwksUrl)
// 			}

// 			// Add attribute mapping if provided
// 			if len(tc.Config.AttributeMapping) > 0 {
// 				resourceConfig += `
// 					attribute_mapping = {
// 				`
// 				for k, v := range tc.Config.AttributeMapping {
// 					resourceConfig += fmt.Sprintf(`						"%s" = "%s"
// 					`, k, v)
// 				}
// 				resourceConfig += `
// 					}
// 				`
// 			}

// 			resourceConfig += `
// 				}
// 			`

// 			config := projectConfig + resourceConfig

// 			// Check configuration.
// 			checks := []resource.TestCheckFunc{
// 				resource.TestCheckResourceAttr(resourceName, "name", tc.Config.Name),
// 				resource.TestCheckResourceAttr(resourceName, "audience", tc.Config.Audience),
// 				resource.TestCheckResourceAttr(resourceName, "issuer", tc.Config.Issuer),
// 			}

// 			if tc.Config.JwksUrl != "" {
// 				checks = append(checks, resource.TestCheckResourceAttr(resourceName, "jwks_url", tc.Config.JwksUrl))
// 			}

// 			if len(tc.Config.AttributeMapping) > 0 {
// 				for k, v := range tc.Config.AttributeMapping {
// 					checks = append(checks, resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("attribute_mapping.%s", k), v))
// 				}
// 			}

// 			resource.Test(t, resource.TestCase{
// 				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
// 				Steps: []resource.TestStep{
// 					{
// 						// Test Create and Read.
// 						Config: testutil.ProviderConfig + config,
// 						Check:  resource.ComposeAggregateTestCheckFunc(checks...),
// 					},
// 				},
// 			})
// 		})
// 	}
// }
