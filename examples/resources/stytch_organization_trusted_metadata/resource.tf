# Example: authoritatively manage an organization's trusted_metadata - the
# object holds exactly these keys, and out-of-band writes surface as plan diffs
resource "stytch_organization_trusted_metadata" "internal" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "live"
  project_secret   = stytch_secret.example.secret
  organization_id  = data.stytch_organization.internal.organization_id

  trusted_metadata = {
    billing = jsonencode({
      plan  = "enterprise"
      seats = 50
    })
    support_tier = jsonencode("gold")
  }
}

data "stytch_organization" "internal" {
  project_slug      = stytch_project.example.project_slug
  environment_slug  = "live"
  project_secret    = stytch_secret.example.secret
  organization_slug = "internal"
}

resource "stytch_secret" "example" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "live"
}
