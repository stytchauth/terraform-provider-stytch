# Example: look up an existing connected app by name to wire its client_id into other resources
data "stytch_connected_app" "portal" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "live"
  project_secret   = stytch_secret.example.secret
  client_name      = "Portal"
}

resource "stytch_secret" "example" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "live"
}
