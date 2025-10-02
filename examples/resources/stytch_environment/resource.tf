# Create a basic test environment with default slug
resource "stytch_environment" "test" {
  project_slug = stytch_project.example.project_slug
  name         = "Test"
}

# Create a test environment with a custom slug
resource "stytch_environment" "staging" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "staging"
  name             = "Staging"
}

# Create a test environment with user impersonation enabled
resource "stytch_environment" "test_with_impersonation" {
  project_slug               = stytch_project.example.project_slug
  environment_slug           = "test"
  name                       = "Test"
  user_impersonation_enabled = true
}

# Create a B2B test environment with cross-org passwords enabled
resource "stytch_environment" "b2b_test" {
  project_slug                = stytch_project.b2b_example.project_slug
  environment_slug            = "test"
  name                        = "Test"
  cross_org_passwords_enabled = true
}

# Create a test environment with user lock configuration
resource "stytch_environment" "test_with_lock" {
  project_slug                 = stytch_project.example.project_slug
  environment_slug             = "test"
  name                         = "Test"
  user_lock_self_serve_enabled = true
  user_lock_threshold          = 15
  user_lock_ttl                = 1800 # 30 minutes in seconds
}

