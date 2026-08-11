# Example: a first-party OAuth client with its redirect URLs managed inline
resource "stytch_connected_app" "web_app" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "live"
  project_secret   = stytch_secret.example.secret
  client_type      = "first_party"
  client_name      = "Example Web App"
  redirect_urls    = ["https://app.example.com/oauth/callback"]

  access_token_expiry_minutes = 30
}

resource "stytch_secret" "example" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "live"
}
