# Manage a Stytch project secret
resource "stytch_secret" "secret" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = stytch_environment.test.environment_slug
}
