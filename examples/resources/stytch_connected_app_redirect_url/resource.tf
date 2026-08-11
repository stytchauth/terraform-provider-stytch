# Example: add a single authorization redirect URL to a connected app whose URL
# arrays are not managed on the stytch_connected_app resource itself
resource "stytch_connected_app_redirect_url" "preview_callback" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "live"
  project_secret   = stytch_secret.example.secret
  client_id        = stytch_connected_app.portal.client_id
  url              = "https://preview.example.com/oauth/callback"
}

# Example: add a post-logout redirect URL to the same connected app
resource "stytch_connected_app_redirect_url" "preview_logout" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "live"
  project_secret   = stytch_secret.example.secret
  client_id        = stytch_connected_app.portal.client_id
  url              = "https://preview.example.com/logged-out"
  type             = "post_logout"
}

resource "stytch_secret" "example" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "live"
}
