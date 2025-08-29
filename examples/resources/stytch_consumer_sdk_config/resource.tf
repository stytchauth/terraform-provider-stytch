# Manage Consumer SDK configuration
resource "stytch_consumer_sdk_config" "consumer_sdk_config" {
  project_id = stytch_project.consumer_project.test_project_id
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
