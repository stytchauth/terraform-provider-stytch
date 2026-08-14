# Example: look up an existing organization by slug to wire its organization_id
# into other resources (organization_id and organization_external_id lookups
# are also supported - set exactly one)
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
