# Create a custom password configuration policy using LUDS validation
resource "stytch_password_config" "example" {
  project_slug                   = "my-project"
  environment_slug               = "production"
  check_breach_on_creation       = true
  check_breach_on_authentication = true
  validate_on_authentication     = true
  validation_policy              = "LUDS"
  luds_min_password_length       = 16
  luds_min_password_complexity   = 4
}
