# Copyright (c) HashiCorp, Inc.

# Manage a public token used for frontend SDKs and OAuth configuration
resource "stytch_public_token" "public_token" {
  project_id = stytch_project.b2b_project.live_project_id
}
