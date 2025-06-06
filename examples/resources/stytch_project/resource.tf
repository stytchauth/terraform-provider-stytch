# Create a consumer project
resource "stytch_project" "consumer_project" {
  name     = "tf-consumer"
  vertical = "CONSUMER"
}

# Create a B2B project
resource "stytch_project" "b2b_project" {
  name     = "tf-b2b"
  vertical = "B2B"
}

# Create a consumer project with user impersonation enabled
resource "stytch_project" "consumer_project_impersonation" {
  name     = "tf-consumer-impersonation"
  vertical = "CONSUMER"

  live_user_impersonation_enabled = true
  test_user_impersonation_enabled = true
}

# Create a B2B project with cross-org passwords enabled in the test environment
resource "stytch_project" "b2b_project_cross_org" {
  name     = "tf-b2b-cross-org"
  vertical = "B2B"

  test_cross_org_passwords_enabled = true
  live_cross_org_passwords_enabled = false
}

# Create a consumer project with user lock self-serve enabled
# and custom lock thresholds
resource "stytch_project" "consumer_project_lock_self_serve" {
  name     = "tf-consumer-lock-self-serve"
  vertical = "CONSUMER"

  test_user_lock_self_serve_enabled = true
  live_user_lock_self_serve_enabled = true

  test_user_lock_threshold = 20
  live_user_lock_threshold = 5
}

# Create a B2B project with user lock self-serve enabled
# and custom lock TTLs
resource "stytch_project" "b2b_project_lock_self_serve" {
  name     = "tf-b2b-lock-self-serve"
  vertical = "B2B"

  test_user_lock_self_serve_enabled = true
  live_user_lock_self_serve_enabled = true

  test_user_lock_ttl = 300  # 5 minutes
  live_user_lock_ttl = 7200 # 2 hours
}