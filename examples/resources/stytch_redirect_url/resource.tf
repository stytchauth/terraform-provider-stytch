# Example: Redirect URL for login only
resource "stytch_redirect_url" "login_redirect" {
  project_slug     = "my-project"
  environment_slug = "production"
  url              = "https://myapp.example.com/auth/callback"

  valid_types = [
    {
      type       = "LOGIN"
      is_default = true
    }
  ]
}

# Example: Redirect URL for multiple authentication types
resource "stytch_redirect_url" "multi_auth_redirect" {
  project_slug     = "my-project"
  environment_slug = "production"
  url              = "https://myapp.example.com/authenticate"

  valid_types = [
    {
      type       = "LOGIN"
      is_default = true
    },
    {
      type       = "SIGNUP"
      is_default = true
    },
    {
      type       = "RESET_PASSWORD"
      is_default = false
    }
  ]
}

# Example: Development environment redirect URL
resource "stytch_redirect_url" "dev_redirect" {
  project_slug     = "my-project"
  environment_slug = "development"
  url              = "http://localhost:3000/auth/callback"

  valid_types = [
    {
      type       = "LOGIN"
      is_default = true
    },
    {
      type       = "SIGNUP"
      is_default = true
    },
    {
      type       = "INVITE"
      is_default = true
    }
  ]
}
