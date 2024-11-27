# Copyright (c) HashiCorp, Inc.

# Create a custom password configuration policy using LUDS validation
resource "stytch_password_config" "password_config" {
  project_id                     = stytch_project.consumer_project.test_project_id
  check_breach_on_creation       = true
  check_breach_on_authentication = true
  validate_on_authentication     = true
  validation_policy              = "LUDS"
  luds_min_password_length       = 16
  luds_min_password_complexity   = 4
}
