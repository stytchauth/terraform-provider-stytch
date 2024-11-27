# Manage a Stytch project secret
resource "stytch_secret" "secret" {
  project_id = stytch_project.consumer_project.live_project_id
}
