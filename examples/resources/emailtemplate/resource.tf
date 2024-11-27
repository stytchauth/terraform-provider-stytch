# Copyright (c) HashiCorp, Inc.

# Create a new prebuilt email template
resource "stytch_email_template" "prebuilt_email_template" {
  live_project_id = stytch_project.consumer_project.live_project_id
  template_id     = "prebuilt_tf"
  name            = "prebuilt"
  prebuilt_customization = {
    button_border_radius = 3
    button_color         = "#105ee9"
    button_text_color    = "#ffffff"
    font_family          = "GEORGIA"
    text_alignment       = "CENTER"
  }
}

# Create a new custom HTML email template
resource "stytch_email_template" "custom_html_email_template" {
  live_project_id = stytch_project.consumer_project.live_project_id
  template_id     = "custom_html_tf"
  name            = "custom_html"
  sender_information = {
    from_local_part     = "noreply"
    from_domain         = "example.com"
    from_name           = "Stytch"
    reply_to_local_part = "support"
    reply_to_name       = "Support"
  }
  custom_html_customization = {
    template_type     = "LOGIN"
    html_content      = "<h1>Login now: {{magic_link_url}}</h1>"
    plaintext_content = "Login now: {{magic_link_url}}"
    subject           = "Login to ` + customDomain + `"
  }
}
