terraform {
  required_providers {
    stytch = {
      source = "registry.terraform.io/stytchauth/stytch"
    }
  }
}

provider "stytch" {}

resource "stytch_project" "consumer_project" {
  name     = "tf-consumer"
  vertical = "CONSUMER"
}

resource "stytch_project" "b2b_project" {
  name     = "tf-b2b"
  vertical = "B2B"
}

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
