# B2B Project RBAC Policy Example
# This example shows how to configure RBAC for a B2B project with custom roles and resources

resource "stytch_rbac_policy" "b2b_example" {
  project_slug     = stytch_project.b2b_project.project_slug
  environment_slug = stytch_environment.test.environment_slug

  # Define custom roles
  custom_roles = [
    {
      role_id     = "document-editor"
      description = "Can create and edit documents"
      permissions = [
        {
          resource_id = "custom-documents"
          actions     = ["read", "create", "update"]
        }
      ]
    },
    {
      role_id     = "document-viewer"
      description = "Can only view documents"
      permissions = [
        {
          resource_id = "custom-documents"
          actions     = ["read"]
        }
      ]
    }
  ]

  # Define custom resources
  custom_resources = [
    {
      resource_id       = "custom-documents"
      description       = "Organization documents"
      available_actions = ["create", "read", "update", "delete"]
    }
  ]
}

# B2B Project with Modified Default Roles
# This example shows how to add custom resource permissions to the stytch_admin default role
# Note: You must include all required Stytch resource permissions when modifying default roles

resource "stytch_rbac_policy" "b2b_with_admin_perms" {
  project_slug     = stytch_project.b2b_project.project_slug
  environment_slug = stytch_environment.test.environment_slug

  # Configure the stytch_admin default role with additional permissions
  # IMPORTANT: Must include all default Stytch permissions (stytch.member, stytch.organization, etc.)
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
        resource_id = "custom-documents"
        actions     = ["*"]
      }
    ]
  }

  # Configure the stytch_member default role with limited permissions
  stytch_member = {
    permissions = [
      {
        resource_id = "stytch.member"
        actions     = ["read"]
      },
      {
        resource_id = "stytch.organization"
        actions     = ["read"]
      },
      {
        resource_id = "custom-documents"
        actions     = ["read"]
      }
    ]
  }

  custom_resources = [
    {
      resource_id       = "custom-documents"
      description       = "Organization documents"
      available_actions = ["create", "read", "update", "delete"]
    }
  ]
}

# Consumer Project RBAC Policy Example
# This example shows how to configure RBAC for a Consumer project with custom scopes

resource "stytch_rbac_policy" "consumer_example" {
  project_slug     = stytch_project.consumer_project.project_slug
  environment_slug = stytch_environment.test.environment_slug

  # Configure the stytch_user default role (optional)
  stytch_user = {
    permissions = [
      {
        resource_id = "stytch.user"
        actions     = ["read", "update"]
      },
      {
        resource_id = "user-profile"
        actions     = ["read", "update"]
      }
    ]
  }

  # Define custom roles
  custom_roles = [
    {
      role_id     = "premium-user"
      description = "Premium subscription user"
      permissions = [
        {
          resource_id = "premium-content"
          actions     = ["read", "download"]
        }
      ]
    },
    {
      role_id     = "content-creator"
      description = "Can create and manage content"
      permissions = [
        {
          resource_id = "user-content"
          actions     = ["create", "read", "update", "delete"]
        }
      ]
    }
  ]

  # Define custom resources
  custom_resources = [
    {
      resource_id       = "premium-content"
      description       = "Premium subscription content"
      available_actions = ["read", "download"]
    },
    {
      resource_id       = "user-content"
      description       = "User-generated content"
      available_actions = ["create", "read", "update", "delete"]
    },
    {
      resource_id       = "user-profile"
      description       = "User profile information"
      available_actions = ["read", "update"]
    }
  ]

  # Define custom scopes (Consumer projects only)
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
    },
    {
      scope       = "content:create"
      description = "Ability to create content"
      permissions = [
        {
          resource_id = "user-content"
          actions     = ["create"]
        }
      ]
    }
  ]
}
