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

# An example of adding a permission to the Stytch Admin role
resource "stytch_rbac_policy" "b2b_rbac_policy_with_admin_perms" {
  project_id = stytch_project.another_b2b_project.test_project_id
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

# INVALID! An example of trying to add to admin permissions without including the default Stytch permissions
# This will fail during the tf apply stage of resource creation.
# resource "stytch_rbac_policy" "I_AM_INVALID" {
#   project_id = stytch_project.third_b2b_project.test_project_id
#   stytch_admin = {
#     permissions = [
#       MISSING DEFAULT PERMISSIONS HERE!
#       {
#         resource_id = "my-only-resource"
#         actions     = ["read"]
#       }
#     ]
#   }
# 
#   custom_resources = [
#     {
#       resource_id       = "my-only-resource"
#       description       = "My only resource"
#       available_actions = ["read", "write"]
#     }
#   ]
# }
