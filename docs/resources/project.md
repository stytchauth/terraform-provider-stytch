---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stytch_project Resource - stytch"
subcategory: ""
description: |-
  Manages a project within your Stytch workspace.
---

# stytch_project (Resource)

Manages a project within your Stytch workspace.

## Example Usage

```terraform
# Create a consumer project
resource "stytch_project" "consumer_project" {
  name     = "tf-consumer"
  vertical = "CONSUMER"
}

# Create a B2B project
resource "stytch_project" "b2b_project" {
  name     = "tf-b2b"
  vertical = "B2B"
}

# Create a consumer project with user impersonation enabled
resource "stytch_project" "consumer_project_impersonation" {
  name     = "tf-consumer-impersonation"
  vertical = "CONSUMER"

  live_user_impersonation_enabled = true
  test_user_impersonation_enabled = true
}

# Create a B2B project with cross-org passwords enabled in the test environment
resource "stytch_project" "b2b_project_cross_org" {
  name     = "tf-b2b-cross-org"
  vertical = "B2B"

  test_cross_org_passwords_enabled = true
  live_cross_org_passwords_enabled = false
}

# Create a consumer project with user lock self-serve enabled
# and custom lock thresholds
resource "stytch_project" "consumer_project_lock_self_serve" {
  name     = "tf-consumer-lock-self-serve"
  vertical = "CONSUMER"

  test_user_lock_self_serve_enabled = true
  live_user_lock_self_serve_enabled = true

  test_user_lock_threshold = 20
  live_user_lock_threshold = 5
}

# Create a B2B project with user lock self-serve enabled
# and custom lock TTLs
resource "stytch_project" "b2b_project_lock_self_serve" {
  name     = "tf-b2b-lock-self-serve"
  vertical = "B2B"

  test_user_lock_self_serve_enabled = true
  live_user_lock_self_serve_enabled = true

  test_user_lock_ttl = 300  # 5 minutes
  live_user_lock_ttl = 7200 # 2 hours
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The project's name.
- `vertical` (String) The project's vertical. This cannot be changed after creation.

### Optional

- `live_cross_org_passwords_enabled` (Boolean) Whether cross-org passwords are enabled for the live project.
- `live_user_impersonation_enabled` (Boolean) Whether user impersonation is enabled for the live project.
- `live_user_lock_self_serve_enabled` (Boolean) Whether users in the live project who get locked out should automatically get an unlock email magic link.
- `live_user_lock_threshold` (Number) The number of failed authenticate attempts that will cause a user in the live project to be locked.
- `live_user_lock_ttl` (Number) The time in seconds that the user in the live project remains locked once the lock is set.
- `test_cross_org_passwords_enabled` (Boolean) Whether cross-org passwords are enabled for the test project.
- `test_user_impersonation_enabled` (Boolean) Whether user impersonation is enabled for the test project.
- `test_user_lock_self_serve_enabled` (Boolean) Whether users in the test project who get locked out should automatically get an unlock email magic link.
- `test_user_lock_threshold` (Number) The number of failed authenticate attempts that will cause a user in the test project to be locked.
- `test_user_lock_ttl` (Number) The time in seconds that the user in the test project remains locked once the lock is set.

### Read-Only

- `created_at` (String) The ISO-8601 timestamp when the project was created.
- `id` (String) A computed ID field used for Terraform resource management.
- `last_updated` (String) Timestamp of the last Terraform update of the order.
- `live_oauth_callback_id` (String) The callback ID used in OAuth requests for the live project.
- `live_project_id` (String) The unique identifier for the live project.
- `test_oauth_callback_id` (String) The callback ID used in OAuth requests for the test project.
- `test_project_id` (String) The unique identifier for the test project.

## Import

Import is supported using the following syntax:

```shell
# A Stytch project can be imported by specifying the relevant *live* project ID
terraform import stytch_project.example project-live-00000000-0000-0000-0000-000000000000
```
