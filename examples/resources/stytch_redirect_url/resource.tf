# Manage Stytch Redirect URLs in a consumer project
resource "stytch_redirect_url" "consumer_redirect_url" {
  project_id = stytch_project.consumer_project.test_project_id
  url        = "http://localhost:3000/consumer"
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
      is_default = false
    }
  ]
}

# Manage a Stytch Redirect URL in a B2B project (includes DISCOVERY type)
resource "stytch_redirect_url" "b2b_redirect_url" {
  project_id = stytch_project.b2b_project.test_project_id
  url        = "http://localhost:3000/b2b"
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
      is_default = false
    },
    {
      type       = "DISCOVERY"
      is_default = true
    }
  ]
}
