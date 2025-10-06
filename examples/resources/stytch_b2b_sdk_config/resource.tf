# Manage B2B SDK configuration
resource "stytch_b2b_sdk_config" "b2b_sdk_config" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = stytch_environment.test.environment_slug
  config = {
    basic = {
      enabled                   = true
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
      enabled      = "ENABLED"
      on_challenge = "TRIGGER_CAPTCHA"
    }
    cookies = {
      http_only = "DISABLED"
    }
  }
}
