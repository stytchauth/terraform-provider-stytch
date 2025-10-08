# First, create email templates for the project
resource "stytch_email_template" "login_primary" {
  project_slug = stytch_project.example.project_slug
  template_id  = "login-primary"
  name         = "Primary Login Template"

  sender_information = {
    from_local_part = "noreply"
    from_domain     = "example.com"
    from_name       = "Example App"
  }

  custom_html_customization = {
    template_type     = "LOGIN"
    subject           = "Log in to your account"
    html_content      = "<html><body><h1>Welcome back!</h1><p>Click here to log in: {{magic_link_url}}</p></body></html>"
    plaintext_content = "Welcome back! Click here to log in: {{magic_link_url}}"
  }
}

resource "stytch_email_template" "signup_primary" {
  project_slug = stytch_project.example.project_slug
  template_id  = "signup-primary"
  name         = "Primary Signup Template"

  sender_information = {
    from_local_part = "noreply"
    from_domain     = "example.com"
    from_name       = "Example App"
  }

  custom_html_customization = {
    template_type     = "SIGNUP"
    subject           = "Welcome to Example App!"
    html_content      = "<html><body><h1>Welcome!</h1><p>Click here to complete your signup: {{magic_link_url}}</p></body></html>"
    plaintext_content = "Welcome! Click here to complete your signup: {{magic_link_url}}"
  }
}

# Set the default email template for LOGIN emails
resource "stytch_default_email_template" "login" {
  project_slug        = stytch_project.example.project_slug
  email_template_type = "LOGIN"
  template_id         = stytch_email_template.login_primary.template_id
}

# Set the default email template for SIGNUP emails
resource "stytch_default_email_template" "signup" {
  project_slug        = stytch_project.example.project_slug
  email_template_type = "SIGNUP"
  template_id         = stytch_email_template.signup_primary.template_id
}
