# Example: Stream event logs to Datadog
resource "stytch_event_log_streaming" "datadog" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = stytch_environment.production.environment_slug
  destination_type = "DATADOG"
  enabled          = true

  datadog = {
    site    = "US"
    api_key = var.datadog_api_key
  }
}

# Example: Stream event logs to Grafana Loki
resource "stytch_event_log_streaming" "grafana" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = stytch_environment.production.environment_slug
  destination_type = "GRAFANA_LOKI"
  enabled          = true

  grafana_loki = {
    hostname = "logs.example.com"
    username = "stytch-logs"
    password = var.grafana_loki_password
  }
}

# Example: Create a disabled configuration
resource "stytch_event_log_streaming" "datadog_disabled" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = stytch_environment.staging.environment_slug
  destination_type = "DATADOG"
  enabled          = false

  datadog = {
    site    = "EU"
    api_key = var.datadog_api_key
  }
}
