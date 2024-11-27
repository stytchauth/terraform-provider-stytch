# Copyright (c) HashiCorp, Inc.

# Manage a custom RBAC policy
resource "stytch_rbac_policy" "b2b_rbac_policy" {
  project_id = stytch_project.b2b_project.test_project_id
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
}
