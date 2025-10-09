# Event log streaming configurations can be imported by specifying the project slug, environment slug, and destination type
# Note that sensitive values (API keys, passwords) are not imported and will need to be set manually.
terraform import stytch_event_log_streaming.example my-project-slug.my-environment-slug.DATADOG
