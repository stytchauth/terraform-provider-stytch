# A Stytch organization trusted_metadata resource can be imported by specifying the project slug, environment slug,
# and organization ID; the organization's current trusted_metadata content is adopted into state on the first refresh.
# Importing is the sanctioned way to take over an organization that already has trusted_metadata (creation refuses
# such organizations unless force = true).
# Format: project_slug.environment_slug.organization_id
# Terraform passes no configuration values during import, so the project secret must come from the environment
STYTCH_IMPORT_PROJECT_SECRET=<project secret> terraform import stytch_organization_trusted_metadata.example my-project.live.organization-live-11111111-1111-1111-1111-111111111111
