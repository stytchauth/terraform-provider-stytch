# Define a JWT template for a Stytch project
resource "stytch_jwt_template" "session_template" {
  project_id       = stytch_project.consumer_project.test_project_id
  template_type    = "SESSION"
  template_content = <<EOT
  {
    "role": {{ user.trusted_metadata.role }},
    "scope": "openid profile email"
  }
  EOT
  custom_audience  = "my-audience"
}
