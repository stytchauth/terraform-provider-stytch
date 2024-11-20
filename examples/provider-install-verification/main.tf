terraform {
  required_providers {
    stytch = {
      source = "registry.terraform.io/stytchauth/stytch"
    }
  }
}

provider "stytch" {}

data "stytch_password_config_policy" "example" {}
