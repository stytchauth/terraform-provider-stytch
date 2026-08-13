# Example: manage two trusted_metadata keys on a long-lived organization,
# leaving every other key (application-written data) untouched
resource "stytch_organization_trusted_metadata" "internal" {
  project_slug     = stytch_project.example.project_slug
  environment_slug = "live"
  project_secret   = stytch_secret.example.secret
  organization_id  = data.stytch_organization.internal.organization_id

  keys = {
    grants = jsonencode({
      version = 1
      feat = {
        digital_twin = { tier = "internal" }
      }
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
