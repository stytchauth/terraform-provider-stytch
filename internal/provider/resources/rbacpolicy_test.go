package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

// TestAccRBACPolicyResource_B2B tests RBAC policy for B2B projects
func TestAccRBACPolicyResource_B2B(t *testing.T) {
	for _, testCase := range []testutil.TestCase{
		{
			Name: "basic",
			Config: testutil.B2BProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + `
      resource "stytch_rbac_policy" "test" {
        project_slug     = stytch_project.test.project_slug
        environment_slug = stytch_environment.test.environment_slug

        custom_roles = [
          {
            role_id     = "my-custom-admin"
            description = "My custom admin role"
            permissions = [
              {
                resource_id = "my-resource"
                actions     = ["create", "read", "update", "delete"]
              },
              {
                resource_id = "my-other-resource"
                actions     = ["read"]
              }
            ]
          },
          {
            role_id     = "my-custom-user"
            description = "My custom user role"
            permissions = [
              {
                resource_id = "my-resource"
                actions     = ["read"]
              },
              {
                resource_id = "my-other-resource"
                actions     = ["read"]
              }
            ]
          }
        ]
        custom_resources = [
          {
            resource_id       = "my-resource"
            description       = "My custom resource"
            available_actions = ["create", "read", "update", "delete"]
          },
          {
            resource_id       = "my-other-resource"
            description       = "My other custom resource"
            available_actions = ["read"]
          }
        ]
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_roles.#", "2"),
				resource.TestCheckTypeSetElemNestedAttrs("stytch_rbac_policy.test", "custom_roles.*", map[string]string{
					"role_id":       "my-custom-admin",
					"description":   "My custom admin role",
					"permissions.#": "2",
				}),
				resource.TestCheckTypeSetElemNestedAttrs("stytch_rbac_policy.test", "custom_roles.*", map[string]string{
					"role_id":       "my-custom-user",
					"description":   "My custom user role",
					"permissions.#": "2",
				}),
				resource.TestCheckTypeSetElemNestedAttrs("stytch_rbac_policy.test", "custom_resources.*", map[string]string{
					"resource_id":         "my-resource",
					"description":         "My custom resource",
					"available_actions.#": "4",
				}),
				resource.TestCheckTypeSetElemNestedAttrs("stytch_rbac_policy.test", "custom_resources.*", map[string]string{
					"resource_id":         "my-other-resource",
					"description":         "My other custom resource",
					"available_actions.#": "1",
				}),
			},
		},
		{
			Name: "admin-resource-with-default",
			Config: testutil.B2BProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + `
			resource "stytch_rbac_policy" "test" {
				project_slug     = stytch_project.test.project_slug
				environment_slug = stytch_environment.test.environment_slug

				stytch_admin = {
					permissions = [
						{
							resource_id = "stytch.member"
							actions     = ["*"]
						},
						{
							resource_id = "stytch.organization"
							actions     = ["*"]
						},
						{
							resource_id = "stytch.sso"
							actions     = ["*"]
						},
						{
							resource_id = "stytch.scim"
							actions     = ["*"]
						},
						{
							resource_id = "my-only-resource"
							actions     = ["read"]
						}
					]
				}

				custom_resources = [
					{
						resource_id       = "my-only-resource"
						description       = "My only resource"
						available_actions = ["read", "write"]
					}
				]
			}
			`,
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
					{
						// Import state testing
						ResourceName:            "stytch_rbac_policy.test",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"last_updated"},
					},
					{
						// Update and Read testing
						Config: testutil.ProviderConfig + testutil.B2BProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
							ProjectSlug: "stytch_project.test.project_slug",
							Name:        "Test Environment",
						}) + `
            resource "stytch_rbac_policy" "test" {
              project_slug     = stytch_project.test.project_slug
              environment_slug = stytch_environment.test.environment_slug

              custom_roles = [
                {
                  role_id     = "new-admin"
                  description = "My custom admin role"
                  permissions = [
                    {
                      resource_id = "my-only-resource"
                      actions     = ["create", "read", "update", "delete"]
                    }
                  ]
                }
              ]
              custom_resources = [
                {
                  resource_id       = "my-only-resource"
                  description       = "My only resource"
                  available_actions = ["create", "read", "update", "delete"]
                }
              ]
            }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_roles.#", "1"),
							resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_resources.#", "1"),
						),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}
}

// TestAccRBACPolicyResource_Consumer tests RBAC policy for Consumer projects
func TestAccRBACPolicyResource_Consumer(t *testing.T) {
	for _, testCase := range []testutil.TestCase{
		{
			Name: "basic",
			Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + `
      resource "stytch_rbac_policy" "test" {
        project_slug     = stytch_project.test.project_slug
        environment_slug = stytch_environment.test.environment_slug

        custom_roles = [
          {
            role_id     = "premium-user"
            description = "Premium user role"
            permissions = [
              {
                resource_id = "premium-content"
                actions     = ["read", "download"]
              }
            ]
          }
        ]
        custom_resources = [
          {
            resource_id       = "premium-content"
            description       = "Premium content resource"
            available_actions = ["read", "download"]
          }
        ]
        custom_scopes = [
          {
            scope       = "premium:access"
            description = "Access to premium features"
            permissions = [
              {
                resource_id = "premium-content"
                actions     = ["read"]
              }
            ]
          }
        ]
      }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_roles.#", "1"),
				resource.TestCheckTypeSetElemNestedAttrs("stytch_rbac_policy.test", "custom_roles.*", map[string]string{
					"role_id":       "premium-user",
					"description":   "Premium user role",
					"permissions.#": "1",
				}),
				resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_resources.#", "1"),
				resource.TestCheckTypeSetElemNestedAttrs("stytch_rbac_policy.test", "custom_resources.*", map[string]string{
					"resource_id":         "premium-content",
					"description":         "Premium content resource",
					"available_actions.#": "2",
				}),
				resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_scopes.#", "1"),
				resource.TestCheckTypeSetElemNestedAttrs("stytch_rbac_policy.test", "custom_scopes.*", map[string]string{
					"scope":         "premium:access",
					"description":   "Access to premium features",
					"permissions.#": "1",
				}),
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
					{
						// Import state testing
						ResourceName:            "stytch_rbac_policy.test",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"last_updated"},
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}
}

// TestAccRBACPolicyResource_Invalid tests validation errors
func TestAccRBACPolicyResource_Invalid(t *testing.T) {
	for _, errorCase := range []testutil.ErrorCase{
		{
			Name: "consumer-project-with-b2b-fields",
			Config: testutil.ConsumerProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + `
			resource "stytch_rbac_policy" "test" {
				project_slug     = stytch_project.test.project_slug
				environment_slug = stytch_environment.test.environment_slug

				stytch_member = {
					permissions = []
				}
			}
			`,
		},
		{
			Name: "b2b-project-with-consumer-fields",
			Config: testutil.B2BProjectConfig + testutil.EnvironmentResource(testutil.EnvironmentResourceArgs{
				ProjectSlug: "stytch_project.test.project_slug",
				Name:        "Test Environment",
			}) + `
			resource "stytch_rbac_policy" "test" {
				project_slug     = stytch_project.test.project_slug
				environment_slug = stytch_environment.test.environment_slug

				stytch_user = {
					permissions = []
				}
			}
			`,
		},
	} {
		errorCase.AssertAnyError(t)
	}
}
