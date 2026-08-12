# A Stytch connected app redirect URL can be imported by specifying the project slug, environment slug, client ID, type, and URL
# Format: project_slug.environment_slug.client_id.type.url - a client_id containing a dot cannot be imported
# Terraform passes no configuration values during import, so the project secret must come from the environment
STYTCH_IMPORT_PROJECT_SECRET=<project secret> terraform import stytch_connected_app_redirect_url.example my-project.live.connected-app-test-11111111-1111-1111-1111-111111111111.authorization.https://preview.example.com/oauth/callback
