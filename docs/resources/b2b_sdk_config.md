---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "stytch_b2b_sdk_config Resource - stytch"
subcategory: ""
description: |-
  Manages the configuration of your JavaScript, React Native, iOS, or Android SDKs for a B2B project
---

# stytch_b2b_sdk_config (Resource)

Manages the configuration of your JavaScript, React Native, iOS, or Android SDKs for a B2B project



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `config` (Attributes) The B2B project SDK configuration. (see [below for nested schema](#nestedatt--config))
- `project_id` (String) The ID of the B2B project for which to set the SDK config. This can be either a live project ID or test project ID. You may only specify one SDK config per project.

### Read-Only

- `id` (String) A computed ID field used for Terraform resource management.
- `last_updated` (String) Timestamp of the last Terraform update of the order.

<a id="nestedatt--config"></a>
### Nested Schema for `config`

Required:

- `basic` (Attributes) The basic configuration for the B2B project SDK. This includes enabling the SDK. (see [below for nested schema](#nestedatt--config--basic))

Optional:

- `dfppa` (Attributes) The Device Fingerprinting Protected Auth configuration for the B2B project SDK. (see [below for nested schema](#nestedatt--config--dfppa))
- `magic_links` (Attributes) The magic links configuration for the B2B project SDK. (see [below for nested schema](#nestedatt--config--magic_links))
- `oauth` (Attributes) The OAuth configuration for the B2B project SDK. (see [below for nested schema](#nestedatt--config--oauth))
- `otps` (Attributes) The OTPs configuration for the B2B project SDK. (see [below for nested schema](#nestedatt--config--otps))
- `passwords` (Attributes) The passwords configuration for the B2B project SDK. (see [below for nested schema](#nestedatt--config--passwords))
- `sessions` (Attributes) The session configuration for the B2B project SDK. (see [below for nested schema](#nestedatt--config--sessions))
- `sso` (Attributes) The SSO configuration for the B2B project SDK. (see [below for nested schema](#nestedatt--config--sso))
- `totps` (Attributes) The TOTPs configuration for the B2B project SDK. (see [below for nested schema](#nestedatt--config--totps))

<a id="nestedatt--config--basic"></a>
### Nested Schema for `config.basic`

Required:

- `enabled` (Boolean) A boolean indicating whether the B2B project SDK is enabled. This allows the SDK to manage user and session data.

Optional:

- `allow_self_onboarding` (Boolean) A boolean indicating whether self-onboarding is allowed for members in the SDK.
- `bundle_ids` (List of String) A list of bundle IDs authorized for use in the SDK.
- `create_new_members` (Boolean) A boolean indicating whether new members can be created with the SDK.
- `domains` (Attributes List) A list of domains authorized for use in the SDK. (see [below for nested schema](#nestedatt--config--basic--domains))
- `enable_member_permissions` (Boolean) A boolean indicating whether member permissions RBAC are enabled in the SDK.

<a id="nestedatt--config--basic--domains"></a>
### Nested Schema for `config.basic.domains`

Optional:

- `domain` (String) The domain name. Stytch uses the same-origin policy to determine matches.
- `slug_pattern` (String) SlugPattern is the slug pattern which can be used to support authentication flows specific to each organization. An examplevalue here might be 'https://{{slug}}.example.com'. The value **must** include '{{slug}}' as a placeholder for the slug.



<a id="nestedatt--config--dfppa"></a>
### Nested Schema for `config.dfppa`

Optional:

- `enabled` (String) A boolean indicating whether Device Fingerprinting Protected Auth is enabled in the SDK.
- `lookup_timeout_seconds` (Number) How long to wait for a DFPPA lookup to complete before timing out.
- `on_challenge` (String) The action to take when a DFPPA 'challenge' verdict is returned.


<a id="nestedatt--config--magic_links"></a>
### Nested Schema for `config.magic_links`

Optional:

- `enabled` (Boolean) A boolean indicating whether magic links endpoints are enabled in the SDK.
- `pkce_required` (Boolean) PKCERequired is a boolean indicating whether PKCE is required for magic links. PKCE increases security by introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow from the same application on the device. This prevents a malicious app from intercepting a redirect and authenticating with the users token. PKCE is enabled by default for mobile SDKs.


<a id="nestedatt--config--oauth"></a>
### Nested Schema for `config.oauth`

Optional:

- `enabled` (Boolean) A boolean indicating whether OAuth endpoints are enabled in the SDK.
- `pkce_required` (Boolean) PKCERequired is a boolean indicating whether PKCE is required for OAuth. PKCE increases security by introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow from the same application on the device. This prevents a malicious app from intercepting a redirect and authenticating with the users token. PKCE is enabled by default for mobile SDKs.


<a id="nestedatt--config--otps"></a>
### Nested Schema for `config.otps`

Optional:

- `email_enabled` (Boolean) A boolean indicating whether the email OTP endpoints are enabled in the SDK.
- `sms_autofill_metadata` (Attributes List) A list of metadata that can be used for autofill of SMS OTPs. (see [below for nested schema](#nestedatt--config--otps--sms_autofill_metadata))
- `sms_enabled` (Boolean) A boolean indicating whether the SMS OTP endpoints are enabled in the SDK.

<a id="nestedatt--config--otps--sms_autofill_metadata"></a>
### Nested Schema for `config.otps.sms_autofill_metadata`

Optional:

- `bundle_id` (String) The ID of the bundle to use for autofill. This should be the associated bundle ID.
- `metadata_type` (String) The type of metadata to use for autofill. This should be either 'domain' or 'hash'.
- `metadata_value` (String) MetadataValue is the value of the metadata to use for autofill. This should be the associated domain name (for metadata type 'domain') or application hash (for metadata type 'hash').



<a id="nestedatt--config--passwords"></a>
### Nested Schema for `config.passwords`

Optional:

- `enabled` (Boolean) A boolean indicating whether password endpoints are enabled in the SDK.
- `pkce_required_for_password_resets` (Boolean) PKCERequiredForPasswordResets is a boolean indicating whether PKCE is required for password resets. PKCE increases security by introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow from the same application on the device. This prevents a malicious app from intercepting a redirect and authenticating with the users token. PKCE is enabled by default for mobile SDKs.


<a id="nestedatt--config--sessions"></a>
### Nested Schema for `config.sessions`

Optional:

- `max_session_duration_minutes` (Number) The maximum session duration that can be created in minutes.


<a id="nestedatt--config--sso"></a>
### Nested Schema for `config.sso`

Optional:

- `enabled` (Boolean) A boolean indicating whether SSO endpoints are enabled in the SDK.
- `pkce_required` (Boolean) PKCERequired is a boolean indicating whether PKCE is required for SSO. PKCE increases security by introducing a one-time secret for each auth flow to ensure the user starts and completes each auth flow from the same application on the device. This prevents a malicious app from intercepting a redirect and authenticating with the users token. PKCE is enabled by default for mobile SDKs.


<a id="nestedatt--config--totps"></a>
### Nested Schema for `config.totps`

Optional:

- `create_totps` (Boolean) A boolean indicating whether TOTP creation is enabled in the SDK.
- `enabled` (Boolean) A boolean indicating whether TOTP endpoints are enabled in the SDK.