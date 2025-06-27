# Define a country code allowlist for a Stytch project.
resource "stytch_country_code_allowlist" "sms_country_code_allowlist" {
  project_id      = stytch_project.consumer_project.test_project_id
  delivery_method = "sms"
  country_codes   = ["US", "CA"]
}
