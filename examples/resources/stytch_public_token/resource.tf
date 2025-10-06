# Manage a public token used for frontend SDKs and OAuth configuration
resource "stytch_public_token" "example" {
  project_slug     = "my-project"
  environment_slug = "production"
}
