# Stytch Terraform Provider: v1 to v3 Migration Guide

This guide details the breaking changes and migration steps required when upgrading from v1 to v3 of the Stytch Terraform Provider.

## Table of Contents

- [Stytch Terraform Provider: v1 to v3 Migration Guide](#stytch-terraform-provider-v1-to-v3-migration-guide)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
    - [Key Changes at a Glance](#key-changes-at-a-glance)
  - [Core Architecture Changes](#core-architecture-changes)
    - [v1 Architecture](#v1-architecture)
    - [v3 Architecture](#v3-architecture)
  - [Migration Strategy](#migration-strategy)
    - [Step 1: Audit Your Current Resources and save current state](#step-1-audit-your-current-resources-and-save-current-state)
    - [Step 2: Plan Your Environment Mapping](#step-2-plan-your-environment-mapping)
    - [Step 3: Upgrade the provider and update Terraform Configuration](#step-3-upgrade-the-provider-and-update-terraform-configuration)
    - [Step 4: Import any test environments](#step-4-import-any-test-environments)
    - [Step 4: State Migration](#step-4-state-migration)
    - [Step 5: Manual Fixes (If necessary)](#step-5-manual-fixes-if-necessary)
  - [Resource-Specific Changes](#resource-specific-changes)
    - [stytch\_project](#stytch_project)
      - [v1 Schema](#v1-schema)
      - [v3 Schema](#v3-schema)
      - [Breaking Changes](#breaking-changes)
    - [stytch\_environment (NEW)](#stytch_environment-new)
      - [v3 Schema](#v3-schema-1)
      - [Notes](#notes)
    - [stytch\_rbac\_policy](#stytch_rbac_policy)
      - [v1 Schema (B2B only)](#v1-schema-b2b-only)
      - [v3 Schema (B2B and Consumer)](#v3-schema-b2b-and-consumer)
      - [Breaking Changes](#breaking-changes-1)
      - [Migration Notes](#migration-notes)
    - [stytch\_email\_template](#stytch_email_template)
      - [v1 Schema](#v1-schema-1)
      - [v3 Schema](#v3-schema-2)
      - [Breaking Changes](#breaking-changes-2)
      - [Migration Notes](#migration-notes-1)
    - [stytch\_default\_email\_template (NEW)](#stytch_default_email_template-new)
      - [v3 Schema](#v3-schema-3)
    - [stytch\_password\_config](#stytch_password_config)
      - [v1 Schema](#v1-schema-2)
      - [v3 Schema](#v3-schema-4)
      - [Breaking Changes](#breaking-changes-3)
    - [Other Resources](#other-resources)
  - [Getting Help](#getting-help)

---

## Overview

The v3 release represents a **major breaking change** to the provider's architecture. The key change is the shift from a **project-based model** with a single live project and a single test project to an **environment-based model**, where many test environments can be configured independently. In addition, we use immutable slugs (which can be user-configured) instead of unique IDs.


### Key Changes at a Glance

1. **Identifier Changes**: `project_id` → `project_slug` + `environment_slug`
2. **New Resources**: `stytch_environment`, `stytch_default_email_template`
3. **Project Resource Redesign**: Projects now manage their live environment inline
4. **New features and better developer experience**: Various resources were redesigned to allow for new features, fix bugs, and provide a better developer experience

---

## Core Architecture Changes

### v1 Architecture

In v1, resources were scoped to either a **live** or **test** project using UUID-based identifiers:

```hcl
resource "stytch_secret" "example" {
  project_id = "project-test-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"  # UUID
}
```

Projects returned both `live_project_id` and `test_project_id`, and you had to choose which one to use for each resource. Only a single test project was supported.

### v3 Architecture

In v3, most resources are scoped to a **project** (identified by a human-readable slug) and an **environment** within that project. This enables us to manage multiple environments. For more information on that, you can read about the launch of custom environments [here](https://changelog.stytch.com/announcements/2025-06-13-custom-environments-intelligent-rate-limiting-ga).

```hcl
resource "stytch_secret" "example" {
  project_slug     = "my-project"      # Human-readable slug
  environment_slug = "production"      # Environment identifier
}
```

You can find these slugs in the Dashboard, v3 PWA endpoints, and generate them yourself when you create projects and environments. Note that slugs are immutable and unique. Project slugs are unique across workspaces, and environment slugs are unique across a project.
 
---

## Migration Strategy

### Step 1: Audit Your Current Resources and save current state

Before upgrading, document:
1. All project IDs (live and test) currently in use
2. The project slugs that correspond to these IDs
3. Which resources use which project IDs
4. Whether resources target "live" or "test" environments

Save or backup your terraform state.

### Step 2: Plan Your Environment Mapping

Map your v1 project IDs to v3 project/environment combinations. For example:

| v1 Field | v1 Value | v3 project_slug | v3 environment_slug |
|----------|----------|-----------------|---------------------|
| `live_project_id` | `project-live-xxx` | `my-project` | `production` |
| `test_project_id` | `project-test-xxx` | `my-project` | `development` |

### Step 3: Upgrade the provider and update Terraform Configuration

Upgrade to version 3.x of the stytch terraform provider.

For `stytch_project`:
1. Move what used to be the `live_*` attributes into the `live_environment` attribute block

Declare `stytch_environment`
1. Create new `stytch_environment` resources. Copy the attributes that were `test_*` from the stytch project resource into this new resource

For each other resource:
1. Replace any references to the stytch_project's `project_id` or `live_project_id` with `project_slug`. 
2. Add the `environment_slug` field (except for `stytch_email_template` and `stytch_default_email_template`)
3. Update attributes as needed

💡 Use `terraform validate` to catch any new requirements or validation issues. Iterate until `terraform validate` shows no errors

### Step 4: Import any test environments

Find the project and environment slugs in the Dashboard, or through PWA. For each of those, run a terraform import. 

⚠️ Make sure you are importing the test environments! The live environment is part of the `stytch_project` resource and will be updated automatically.

```hcl
terraform import stytch_environment.<my_environment> <project_slug>.<environment_slug>
```

### Step 4: State Migration

We'll use Terraform's state upgrade capability to automatically update the state and pointers from the v1 "project_id" reference to the v3 slug reference. All you need to do is run a terraform plan. 

```
terraform plan
```

This will communicate with Stytch via PWA V3 and map the old ids to the new slugs. Assuming everything worked fine, your plan should be empty. 

**NOTE: If the plan suggests any resource replacement, do not apply. Check the steps above and make sure the slugs or slug references are setup properly**

### Step 5: Manual Fixes (If necessary)

Most cases will require no fix beyond this. If for some reason you are stuck, you have two options:

1. Restore the backed up state and perform the steps again
2. Delete the resource that is problematic from state and manaully re-import it. Note that sensitive data in stytch_event_log_streaming and the stytch_secret resource cannot be imported.

---

## Resource-Specific Changes

- [stytch_project](#stytch_project)
- [stytch_environment (NEW)](#stytch_environment-new)
- [stytch_rbac_policy](#stytch_rbac_policy)
- [stytch_email_template](#stytch_email_template)
- [stytch_default_email_template (NEW)](#stytch_default_email_template-new)
- [stytch_password_config](#stytch_password_config)
- [All Other Resources](#other-resources)

### stytch_project

The project resource has been significantly redesigned. What used to be the live project settings are now inside the `live_environment` attribute block

#### v1 Schema

```hcl
resource "stytch_project" "example" {
  name     = "My Project"
  vertical = "B2B"

  # Live environment settings
  live_user_impersonation_enabled = true
  live_cross_org_passwords_enabled = false
  live_user_lock_threshold = 10
  live_user_lock_ttl = 3600
  live_user_lock_self_serve_enabled = false

  # Test environment settings
  test_user_impersonation_enabled = true
  test_cross_org_passwords_enabled = false
  test_user_lock_threshold = 5
  test_user_lock_ttl = 1800
  test_user_lock_self_serve_enabled = true
}
```

#### v3 Schema

```hcl
resource "stytch_project" "example" {
  name         = "My Project"
  vertical     = "B2B"
  project_slug = "my-project"  # Optional, generated if not provided

  # Live environment is now a nested object (optional)
  live_environment = {
    environment_slug = "production"  # Optional, defaults to "production"
    name             = "Production"

    # All the same configuration options as before
    user_impersonation_enabled = true
    cross_org_passwords_enabled = false
    user_lock_threshold = 10
    user_lock_ttl = 3600
    user_lock_self_serve_enabled = false

    # Additional IDP fields
    idp_authorization_url = "https://..."
    idp_dynamic_client_registration_enabled = false
    zero_downtime_session_migration_url = "https://..."
  }
}
```

#### Breaking Changes

| v1 Field | v3 Equivalent |
|----------|---------------|
| `live_*` (fields) | `live_environment.*` (nested) |
| `test_*` (fields) | Use separate `stytch_environment` resource |

- **Test environments must now be created separately** using the new `stytch_environment` resource
- The live environment is optional but recommended to declare inline rather than keep outside Terraform
- Once a `live_environment` is created, it cannot be removed (only the entire project can be deleted)

**NOTE: Projects without live environments will not show up on the Dashboard until a live environment is created via the management API or Terraform**

---

### stytch_environment (NEW)

A new resource for managing test environments within a project.

#### v3 Schema

```hcl
resource "stytch_environment" "test" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "development"  # Optional, generated if not provided
  name             = "Development"

  # Same configuration options as project live_environment
  user_impersonation_enabled = true
  cross_org_passwords_enabled = false
  user_lock_threshold = 5
  user_lock_ttl = 1800
  user_lock_self_serve_enabled = true

  # Additional IDP fields
  idp_authorization_url = "https://..."
  idp_dynamic_client_registration_enabled = false
  zero_downtime_session_migration_url = "https://..."
}
```

#### Notes

- This resource **only supports TEST type environments**. Supporting live enviroments might happen if Stytch supports multiple live environments in the future.
- Live environments are managed via the `stytch_project` resource

---

### stytch_rbac_policy

This resource now supports both B2B and Consumer projects. We also simplified the schema for default roles.

#### v1 Schema (B2B only)

```hcl
resource "stytch_rbac_policy" "example" {
  project_id = "project-test-xxxxxxxx"

  stytch_member {
    role_id     = "member"      # Was read-only but required
    description = "Members"     # Was read-only but required
    permissions = [
      {
        resource_id = "documents"
        actions     = ["read"]
      }
    ]
  }

  stytch_admin {
    role_id     = "admin"       # Was read-only but required
    description = "Admins"      # Was read-only but required
    permissions = [
      {
        resource_id = "*"
        actions     = ["*"]
      }
    ]
  }

  custom_roles = [
    {
      role_id     = "editor"
      description = "Editor"
      permissions = [...]
    }
  ]

  custom_resources = [
    {
      resource_id = "documents"
      description = "Documents"
      available_actions = ["create", "read", "update", "delete"]
    }
  ]
}
```

#### v3 Schema (B2B and Consumer)

```hcl
resource "stytch_rbac_policy" "b2b_example" {
  project_slug     = "my-project"
  environment_slug = "production"

  # For B2B projects 
  stytch_member {
    # role_id and description REMOVED - only permissions now
    permissions = [
      {
        resource_id = "documents"
        actions     = ["read"]
      }
    ]
  }

  stytch_admin {
    # role_id and description REMOVED - only permissions now
    permissions = [
      {
        resource_id = "*"
        actions     = ["*"]
      }
    ]
  }

  custom_roles = [
    {
      role_id     = "editor"
      description = "Editor"
      permissions = [...]
    }
  ]

  custom_resources = [
    {
      resource_id = "documents"
      description = "Documents"
      available_actions = ["create", "read", "update", "delete"]
    }
  ]

  # NEW: Custom scopes
  custom_scopes = [
    {
      scope_id    = "read:sensitive"
      description = "Read sensitive data"
    }
  ]
}

# For Consumer projects
resource "stytch_rbac_policy" "consumer_example" {
  project_slug     = "my-consumer-project"
  environment_slug = "production"

  # For Consumer projects - use stytch_user instead
  stytch_user {
    permissions = [
      {
        resource_id = "profile"
        actions     = ["read", "update"]
      }
    ]
  }

  # ... rest similar to B2B
}
```

#### Breaking Changes

- **CHANGED:** `project_id` → `project_slug` and `environment_slug`
- **REMOVED:** `role_id` and `description` fields from default roles (`stytch_member`, `stytch_admin`, `stytch_user`)
  - Default roles now **only accept `permissions`**
- **ADDED:** `stytch_user` field for Consumer projects
- **ADDED:** `custom_scopes` array for defining custom permission scopes

#### Migration Notes

For B2B projects:
- Remove `role_id` and `description` from `stytch_member` and `stytch_admin` blocks

---

### stytch_email_template

**Special note:** Email templates in v3 are **project-wide** and apply to all environments, unlike most other resources which are environment-specific.

#### v1 Schema

```hcl
resource "stytch_email_template" "example" {
  live_project_id = "project-live-xxxxxxxx"  # Could ONLY use live project
  template_id     = "my-template"
  name            = "My Custom Template"

  sender_information {
    from_local_part     = "noreply"
    from_domain         = "example.com"
    reply_to_local_part = "support"
    reply_to_domain     = "example.com"
  }

  # Either prebuilt_customization OR custom_html_customization
  prebuilt_customization {
    button_color      = "#FF0000"
    button_text_color = "#FFFFFF"
    font_family       = "Arial"
  }
}
```

#### v3 Schema

```hcl
resource "stytch_email_template" "example" {
  project_slug = "my-project"  # NO environment_slug!
  template_id  = "my-template"
  name         = "My Custom Template"

  sender_information {
    from_local_part     = "noreply"
    from_domain         = "example.com"
    reply_to_local_part = "support"
    reply_to_domain     = "example.com"
  }

  # Either prebuilt_customization OR custom_html_customization
  prebuilt_customization {
    button_color      = "#FF0000"
    button_text_color = "#FFFFFF"
    font_family       = "Arial"
  }
}
```

#### Breaking Changes

- **CHANGED:** `live_project_id` → `project_slug`
- **SCOPE CLARIFICATION:** Email templates apply to **all environments** in a project.

#### Migration Notes

In v1, you could only create email templates for live projects, and the changes would be propagated to the test environment. In v3, email templates are automatically applied across all environments within a project.

Note that reads are based off of the live environment.

---

### stytch_default_email_template (NEW)

A new resource for customizing Stytch's built-in default email templates. This is also project-wide and applies to all environments equally.

#### v3 Schema

```hcl
resource "stytch_default_email_template" "login_or_signup" {
  project_slug = "my-project"
  template_id  = "login_or_signup_by_email"

  sender_information {
    from_local_part     = "auth"
    from_domain         = "example.com"
    reply_to_local_part = "support"
    reply_to_domain     = "example.com"
  }

  prebuilt_customization {
    button_color      = "#0066CC"
    button_text_color = "#FFFFFF"
    font_family       = "Helvetica"
  }
}
```

### stytch_password_config

#### v1 Schema

```hcl
resource "stytch_password_config" "example" {
  project_id                     = "project-test-xxxxxxxx"
  check_breach_on_creation       = true
  check_breach_on_authentication = true
  validate_on_authentication     = true
  validation_policy              = "ZXCVB"

  luds_min_password_length    = 12  # int32
  luds_min_password_complexity = 3  # int32
}
```

#### v3 Schema

```hcl
resource "stytch_password_config" "example" {
  project_slug                   = "my-project"
  environment_slug               = "production"
  check_breach_on_creation       = true
  check_breach_on_authentication = true
  validate_on_authentication     = true
  validation_policy              = "ZXCVBN"
}
```

#### Breaking Changes

- **CHANGED:** `project_id` → `project_slug` + `environment_slug`
- **CHANGED:** All boolean fields (`check_breach_on_creation`, `check_breach_on_authentication`, `validate_on_authentication`) now default to True. This is in line with the default password configuration for all newly created environments.

---

### Other Resources

The following resources are functionally equivalent in both v1 and v3 and its only change is the way they reference projects/environments (project_id → project_slug + environment_slug):

- `stytch_b2b_sdk_config`
- `stytch_consumer_sdk_config`
- `stytch_country_code_allowlist`
- `stytch_event_log_streaming`
- `stytch_jwt_template`
- `stytch_password_config`
- `stytch_public_token`
- `stytch_redirect_url`
- `stytch_secret`
- `stytch_trusted_token_profiles`

Consult the [Terraform Registry documentation](https://registry.terraform.io/providers/stytchauth/stytch/latest/docs) for detailed schemas.

---

## Getting Help

If you encounter issues during migration:

1. Check the [Terraform Registry documentation](https://registry.terraform.io/providers/stytchauth/stytch/latest/docs)
2. Review the [GitHub repository](https://github.com/stytchauth/terraform-provider-stytch)
3. Contact [Stytch support](mailto:support@stytch.com) or join the [Stytch Slack community](https://stytch.com/docs/resources/support/overview)
