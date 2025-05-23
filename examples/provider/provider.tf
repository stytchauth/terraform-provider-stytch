terraform {
  required_providers {
    stytch = {
      source  = "registry.terraform.io/stytchauth/stytch"
      version = "TODO" # Find the latest version at https://registry.terraform.io/providers/stytchauth/stytch/latest
    }
  }
}

# Configuration-based authentication
provider "stytch" {
  workspace_key_id     = "workspace-key-prod-00000000-0000-0000-0000-000000000000"
  workspace_key_secret = "***"
}

# The provider can also be configured via environment variables
# Use the STYTCH_WORKSPACE_KEY_ID and STYTCH_WORKSPACE_KEY_SECRET environment variables
# This is the recommended way to configure the provider.
provider "stytch" {}
