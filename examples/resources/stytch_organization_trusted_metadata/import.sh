# A Stytch organization trusted_metadata resource can be imported by specifying the project slug, environment slug,
# organization ID, and optionally the comma-separated top-level keys it should own (omitting the keys segment imports
# ownership of no keys; the next apply adopts the keys declared in configuration)
# Format: project_slug.environment_slug.organization_id[.key1,key2] - a key name containing a comma cannot be imported
# Terraform passes no configuration values during import, so the project secret must come from the environment
STYTCH_IMPORT_PROJECT_SECRET=<project secret> terraform import stytch_organization_trusted_metadata.example my-project.live.organization-live-11111111-1111-1111-1111-111111111111.grants,support_tier
