terraform {
  required_providers {
    stytch = {
      source = "registry.terraform.io/stytchauth/stytch"
    }
  }
}

provider "stytch" {}

resource "stytch_project" "consumer_project" {
  name     = "tf-consumer"
  vertical = "CONSUMER"
}

resource "stytch_project" "b2b_project" {
  name     = "tf-b2b"
  vertical = "B2B"
}

resource "stytch_redirect_url" "consumer_redirect_url" {
  project_id = stytch_project.consumer_project.test_project_id
  url        = "http://localhost:3000/consumer"
  valid_types = [
    {
      type       = "LOGIN"
      is_default = true
    },
    {
      type       = "SIGNUP"
      is_default = true
    },
    {
      type       = "INVITE"
      is_default = false
    }
  ]
}

resource "stytch_redirect_url" "b2b_redirect_url" {
  project_id = stytch_project.b2b_project.test_project_id
  url        = "http://localhost:3000/b2b"
  valid_types = [
    {
      type       = "LOGIN"
      is_default = true
    },
    {
      type       = "SIGNUP"
      is_default = true
    },
    {
      type       = "INVITE"
      is_default = false
    },
    {
      type       = "DISCOVERY"
      is_default = true
    }
  ]
}

resource "stytch_password_config" "consumer_password_config" {
  project_id                     = stytch_project.consumer_project.test_project_id
  check_breach_on_creation       = true
  check_breach_on_authentication = true
  validate_on_authentication     = true
  validation_policy              = "LUDS"
  luds_min_password_length       = 16
  luds_min_password_complexity   = 4
}

resource "stytch_consumer_sdk_config" "consumer_sdk_config" {
  project_id = stytch_project.consumer_project.test_project_id
  config = {
    basic = {
      enabled          = true
      create_new_users = true
      domains          = []
      bundle_ids       = []
    }
    sessions = {
      max_session_duration_minutes = 60
    }
    magic_links = {
      login_or_create_enabled = true
      send_enabled            = true
      pkce_required           = true
    }
  }
}

resource "stytch_b2b_sdk_config" "b2b_sdk_config" {
  project_id = stytch_project.b2b_project.test_project_id
  config = {
    basic = {
      enabled                   = true
      create_new_members        = true
      allow_self_onboarding     = true
      enable_member_permissions = true
      domains                   = []
      bundle_ids                = ["com.stytch.app", "com.stytch.app2"]
    }
    sessions = {
      max_session_duration_minutes = 60
    }
    totps = {
      enabled      = true
      create_totps = true
    }
    dfppa = {
      enabled                = "ENABLED"
      on_challenge           = "TRIGGER_CAPTCHA"
      lookup_timeout_seconds = 3
    }
  }
}

resource "stytch_secret" "consumer_secret" {
  project_id = stytch_project.consumer_project.live_project_id
}

resource "stytch_public_token" "b2b_public_token" {
  project_id = stytch_project.b2b_project.live_project_id
}

resource "stytch_email_template" "consumer_prebuilt_email_template" {
  live_project_id = stytch_project.consumer_project.live_project_id
  template_id     = "consumer_prebuilt_tf"
  name            = "consumer prebuilt"
  prebuilt_customization = {
    button_border_radius = 3
    button_color         = "#105ee9"
    button_text_color    = "#ffffff"
    font_family          = "GEORGIA"
    text_alignment       = "CENTER"
  }
}

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
