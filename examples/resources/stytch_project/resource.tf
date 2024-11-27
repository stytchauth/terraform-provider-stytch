# Create a consumer project
resource "stytch_project" "consumer_project" {
  name     = "tf-consumer"
  vertical = "CONSUMER"
}

# Create a B2B project
resource "stytch_project" "b2b_project" {
  name     = "tf-b2b"
  vertical = "B2B"
}
