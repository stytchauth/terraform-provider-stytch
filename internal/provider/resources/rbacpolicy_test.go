package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccRBACPolicyResource(t *testing.T) {
	for _, testCase := range []testutil.TestCase{
		{
			Name: "basic",
			Config: testutil.B2BProjectConfig + `
      resource "stytch_rbac_policy" "test" {
        project_id = stytch_project.project.test_project_id
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
			Name: "admin-resource",
			Config: testutil.B2BProjectConfig + `
			resource "stytch_rbac_policy" "test" {
				project_id = stytch_project.project.test_project_id
				stytch_admin = {
					permissions = [
						{
							resource_id = "custom_resource_1"
							actions     = ["read"]
						}
					]
				}

				custom_resources = [
					{
						resource_id       = "custom_resource_1"
						description       = "A custom resource for testing."
						available_actions = ["read", "write"]
					}
				]
			}
			`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_resources.#", "1"),
				resource.TestCheckTypeSetElemNestedAttrs("stytch_rbac_policy.test", "stytch_admin.*", map[string]string{
					"role_id":       "my-custom-admin",
					"description":   "My custom admin role",
					"permissions.#": "2",
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
						ResourceName:      "stytch_rbac_policy.test",
						ImportState:       true,
						ImportStateVerify: true,
						// Ignore timestamp fields because rounding differences in various APIs can result in different
						// values being returned
						ImportStateVerifyIgnore: []string{"last_updated"},
					},
					{
						// Update and Read testing
						Config: testutil.ProviderConfig + testutil.B2BProjectConfig + `
            resource "stytch_rbac_policy" "test" {
              project_id = stytch_project.project.test_project_id
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
							resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_roles.0.role_id", "new-admin"),
							resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_roles.0.description", "My custom admin role"),
							resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_roles.0.permissions.#", "1"),
							resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_resources.#", "1"),
							resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_resources.0.resource_id", "my-only-resource"),
							resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_resources.0.description", "My only resource"),
							resource.TestCheckResourceAttr("stytch_rbac_policy.test", "custom_resources.0.available_actions.#", "4"),
						),
					},
					// Delete testing automatically occurs in resource.TestCase
				},
			})
		})
	}
}
