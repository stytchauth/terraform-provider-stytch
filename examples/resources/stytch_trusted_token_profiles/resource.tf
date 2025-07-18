# Manage a JWK based trusted token profile with attribute mapping.
resource "stytch_trusted_token_profiles" "simple_example" {
  project_id      = stytch_project.consumer_project.test_project_id
  name            = "Simple Attribute Mapping"
  audience        = "https://example.com"
  issuer          = "https://auth.example.com"
  public_key_type = "jwk"
  jwks_url        = "https://auth.example.com/.well-known/jwks.json"
  
  attribute_mapping_json = jsonencode({
    "user_id" = "sub"
    "email"   = "email"
    "name"    = "display_name"
  })
}

# Manage a JWK based trusted token profile with complex attribute mapping.
resource "stytch_trusted_token_profiles" "complex_example" {
  project_id      = stytch_project.consumer_project.test_project_id
  name            = "Complex Attribute Mapping"
  audience        = "https://example.com"
  issuer          = "https://auth.example.com"
  public_key_type = "jwk"
  jwks_url        = "https://auth.example.com/.well-known/jwks.json"
  
  attribute_mapping_json = jsonencode({
    "user_id" = "sub"                    # string
    "age"     = 25                       # number
    "active"  = true                     # boolean
    "roles"   = ["admin", "user"]        # array
    "profile" = {                        # nested object
      "name" = "display_name"
      "type" = "user_type"
    }
    "metadata" = {                       # complex nested structure
      "preferences" = {
        "theme" = "dark"
        "notifications" = true
      }
      "tags" = ["verified", "premium"]
    }
  })
}

# Manage a PEM based trusted token profile using herodoc for string formatting
resource "stytch_trusted_token_profiles" "pem_example" {
  project_id      = stytch_project.consumer_project.test_project_id
  name            = "PEM-based Profile"
  audience        = "https://example.com"
  issuer          = "https://auth.example.com"
  public_key_type = "pem"
  
  pem_files = [
    {
      public_key = <<EOT
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4f5wg5l2hKsTeNem/V41
fGnJm6gOdrj8ym3rFkEjWT2btYK36hY+c2QKfPU5O7w=
-----END PUBLIC KEY-----
EOT
    }
  ]
} 

# Manage a PEM based trusted token profile using file for string formatting
resource "stytch_trusted_token_profiles" "pem_example_file" {
  project_id      = stytch_project.consumer_project.test_project_id
  name            = "PEM-based Profile"
  audience        = "https://example.com"
  issuer          = "https://auth.example.com"
  public_key_type = "pem"
  
  pem_files = [
    {
      public_key = file("${path.module}/public.pem")
    },
    {
      public_key = file("${path.module}/public2.pem")
    }
  ]
}
