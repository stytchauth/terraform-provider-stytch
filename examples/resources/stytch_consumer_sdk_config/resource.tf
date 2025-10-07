# Manage Consumer SDK configuration
resource "stytch_consumer_sdk_config" "consumer_sdk_config" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = stytch_environment.test.environment_slug
  config = {
    basic = {
      enabled    = true
      domains    = []
      bundle_ids = []
    }
    sessions = {
      max_session_duration_minutes = 60
    }
    magic_links = {
      login_or_create_enabled = true
      send_enabled            = true
      pkce_required           = true
    }
    cookies = {
      http_only = "DISABLED"
    }
  }
}
