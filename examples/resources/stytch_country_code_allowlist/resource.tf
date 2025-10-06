# Define a country code allowlist for SMS delivery
resource "stytch_country_code_allowlist" "example" {
  project_slug     = "my-project"
  environment_slug = "production"
  delivery_method  = "sms"
  country_codes    = ["US", "CA", "GB"]
}
