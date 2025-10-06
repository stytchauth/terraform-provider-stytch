# Example: JWT Template for SESSION tokens
resource "stytch_jwt_template" "session_template" {
  project_slug     = "my-project"
  environment_slug = "production"
  template_type    = "SESSION"
  template_content = <<-EOT
    {
      "role": {{ user.trusted_metadata.role }},
      "permissions": {{ user.trusted_metadata.permissions }}
    }
  EOT
  custom_audience  = "https://myapp.example.com"
}

# Example: JWT Template for M2M tokens
resource "stytch_jwt_template" "m2m_template" {
  project_slug     = "my-project"
  environment_slug = "production"
  template_type    = "M2M"
  template_content = <<-EOT
    {
      "tier": {{ client.trusted_metadata.subscription_tier }},
      "org_id": {{ client.trusted_metadata.organization_id }}
    }
  EOT
}
