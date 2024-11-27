---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stytch_rbac_policy Resource - stytch"
subcategory: ""
description: |-
  A role-based access control (RBAC) policy for a B2B project.
---

# stytch_rbac_policy (Resource)

A role-based access control (RBAC) policy for a B2B project.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `project_id` (String) The unique identifier for the project.

### Optional

- `custom_resources` (Attributes Set) Resources that exist within the project beyond those defined within the stytch_resources (see [below for nested schema](#nestedatt--custom_resources))
- `custom_roles` (Attributes Set) Additional roles that exist within the project beyond the stytch_member or stytch_admin roles (see [below for nested schema](#nestedatt--custom_roles))
- `stytch_admin` (Attributes) The role assigned to admins within an organization (see [below for nested schema](#nestedatt--stytch_admin))
- `stytch_member` (Attributes) The default role given to members within the project (see [below for nested schema](#nestedatt--stytch_member))

### Read-Only

- `id` (String) A computed ID field used for Terraform resource management.
- `last_updated` (String) Timestamp of the last Terraform update of the order.
- `stytch_resources` (Attributes Set) StytchResources consists of resources created by Stytch that always exist. This field will be returned in relevant Policy objects but can never be overridden or deleted. (see [below for nested schema](#nestedatt--stytch_resources))

<a id="nestedatt--custom_resources"></a>
### Nested Schema for `custom_resources`

Optional:

- `available_actions` (Set of String) The actions that can be granted for this resource
- `description` (String) A description of the resource
- `resource_id` (String) A human-readable name that is unique within the project


<a id="nestedatt--custom_roles"></a>
### Nested Schema for `custom_roles`

Optional:

- `description` (String) A description of the role
- `permissions` (Attributes Set) (see [below for nested schema](#nestedatt--custom_roles--permissions))
- `role_id` (String) A human-readable name that is unique within the project

<a id="nestedatt--custom_roles--permissions"></a>
### Nested Schema for `custom_roles.permissions`

Optional:

- `actions` (Set of String) An array of actions that the role can perform on the given resource
- `resource_id` (String) The ID of the resource that the role can perform actions on.



<a id="nestedatt--stytch_admin"></a>
### Nested Schema for `stytch_admin`

Optional:

- `description` (String) A description of the role
- `permissions` (Attributes Set) (see [below for nested schema](#nestedatt--stytch_admin--permissions))
- `role_id` (String) A human-readable name that is unique within the project

<a id="nestedatt--stytch_admin--permissions"></a>
### Nested Schema for `stytch_admin.permissions`

Optional:

- `actions` (Set of String) An array of actions that the role can perform on the given resource
- `resource_id` (String) The ID of the resource that the role can perform actions on.



<a id="nestedatt--stytch_member"></a>
### Nested Schema for `stytch_member`

Optional:

- `description` (String) A description of the role
- `permissions` (Attributes Set) (see [below for nested schema](#nestedatt--stytch_member--permissions))
- `role_id` (String) A human-readable name that is unique within the project

<a id="nestedatt--stytch_member--permissions"></a>
### Nested Schema for `stytch_member.permissions`

Optional:

- `actions` (Set of String) An array of actions that the role can perform on the given resource
- `resource_id` (String) The ID of the resource that the role can perform actions on.



<a id="nestedatt--stytch_resources"></a>
### Nested Schema for `stytch_resources`

Optional:

- `available_actions` (Set of String) The actions that can be granted for this resource
- `description` (String) A description of the resource
- `resource_id` (String) A human-readable name that is unique within the project