# Create a project without a live environment
resource "stytch_project" "minimal" {
  name     = "My Project"
  vertical = "CONSUMER"
}

# Create a B2B project with a live environment
resource "stytch_project" "b2b_with_live" {
  name     = "My B2B Project"
  vertical = "B2B"

  live_environment = {
    name = "Production"
  }
}

# Create a project with a custom project slug and custom live environment slug
resource "stytch_project" "custom_slugs" {
  name         = "Custom Project"
  vertical     = "CONSUMER"
  project_slug = "my-custom-project"

  live_environment = {
    environment_slug = "prod"
    name             = "Production"
  }
}

# Create a B2B project with cross-org passwords enabled
resource "stytch_project" "b2b_cross_org" {
  name     = "B2B with Cross-Org Passwords"
  vertical = "B2B"

  live_environment = {
    name                        = "Production"
    cross_org_passwords_enabled = true
  }
}

# Create a project with user impersonation enabled
resource "stytch_project" "with_impersonation" {
  name     = "Project with Impersonation"
  vertical = "CONSUMER"

  live_environment = {
    name                       = "Production"
    user_impersonation_enabled = true
  }
}

# Create a project with user lock configuration
resource "stytch_project" "with_user_lock" {
  name     = "Project with User Lock"
  vertical = "CONSUMER"

  live_environment = {
    name                         = "Production"
    user_lock_self_serve_enabled = true
    user_lock_threshold          = 5
    user_lock_ttl                = 7200 # 2 hours in seconds
  }
}

# Create a B2B project with a complex environment configuration
resource "stytch_project" "full_config" {
  name     = "Fully Configured B2B Project"
  vertical = "B2B"

  live_environment = {
    name                                    = "Production"
    environment_slug                        = "production"
    cross_org_passwords_enabled             = true
    user_impersonation_enabled              = true
    zero_downtime_session_migration_url     = "https://example.com/userinfo"
    user_lock_self_serve_enabled            = true
    user_lock_threshold                     = 10
    user_lock_ttl                           = 3600
    idp_authorization_url                   = "https://example.com/.well-known/openid-configuration"
    idp_dynamic_client_registration_enabled = true
  }
}
