---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stytch_public_token Resource - stytch"
subcategory: ""
description: |-
  A public token used for SDK authentication and OAuth integrations.
---

# stytch_public_token (Resource)

A public token used for SDK authentication and OAuth integrations.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `project_id` (String) The unique identifier for the project.

### Read-Only

- `created_at` (String) The ISO-8601 timestamp when the public token was created.
- `public_token` (String) The public token value. This is a unique ID which is also the identifier for the token.