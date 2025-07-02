# Create a Datadog destination for event log streaming.
resource "stytch_event_log_streaming" "datadog" {
  project_id       = stytch_project.consumer_project.test_project_id
  destination_type = "DATADOG"
  datadog_config {
    api_key = "0123456789abcdef0123456789abcdef"
    site    = "US"
  }
}

# Create a Grafana Loki destination for event log streaming.
resource "stytch_event_log_streaming" "grafana_loki" {
  project_id       = stytch_project.consumer_project.test_project_id
  destination_type = "GRAFANA_LOKI"
  grafana_loki_config {
    username = "stytch-logs"
    password = "password"
    hostname = "prod-01.grafana.net"
  }
}
