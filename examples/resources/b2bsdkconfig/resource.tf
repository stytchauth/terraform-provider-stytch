# Copyright (c) HashiCorp, Inc.

# Manage B2B SDK configuration
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
