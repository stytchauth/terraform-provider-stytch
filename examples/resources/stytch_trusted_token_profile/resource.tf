# Example: Trusted Token Profile with JWK public key type
resource "stytch_trusted_token_profile" "example_jwk" {
  project_slug      = "my-project"
  environment_slug  = "production"
  name              = "My JWK Profile"
  audience          = "https://myapp.example.com"
  issuer            = "https://auth.example.com"
  public_key_type   = "JWK"
  jwks_url          = "https://auth.example.com/.well-known/jwks.json"
  can_jit_provision = false

  attribute_mapping_json = jsonencode({
    "email" = "user_email"
    "name"  = "user_name"
  })
}

# Example: Trusted Token Profile with PEM public key type
resource "stytch_trusted_token_profile" "example_pem" {
  project_slug     = "my-project"
  environment_slug = "production"
  name             = "My PEM Profile"
  audience         = "https://myapp.example.com"
  issuer           = "https://auth.example.com"
  public_key_type  = "PEM"

  pem_files = [
    {
      public_key = <<-EOT
        -----BEGIN PUBLIC KEY-----
        MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4f5wg5l2hKsTeNem/V41
        fGnJm6gOdrj8ym3rFkEU+wT6yv5TqnKkiugWELCHlKt7wc7eNHX0kRi3n5dXvN3w
        ...
        -----END PUBLIC KEY-----
      EOT
    }
  ]
}
