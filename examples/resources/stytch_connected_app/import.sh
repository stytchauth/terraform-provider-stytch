# A Stytch connected app can be imported by specifying the project slug, environment slug, and client ID
# Format: project_slug.environment_slug.client_id
# Terraform passes no configuration values during import, so the project secret must come from the environment
STYTCH_IMPORT_PROJECT_SECRET=<project secret> terraform import stytch_connected_app.example my-project.live.connected-app-test-11111111-1111-1111-1111-111111111111
