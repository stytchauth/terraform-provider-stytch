# Example: Prebuilt Email Template
resource "stytch_email_template" "prebuilt_template" {
  project_slug = "my-project"
  template_id  = "custom-login-template"
  name         = "Custom Login Template"

  prebuilt_customization = {
    button_color         = "#FF5733"
    button_text_color    = "#FFFFFF"
    button_border_radius = 8.0
    font_family          = "HELVETICA"
    text_alignment       = "CENTER"
  }
}

# Example: Custom HTML Email Template for Login
resource "stytch_email_template" "custom_html_login" {
  project_slug = "my-project"
  template_id  = "custom-html-login"
  name         = "Custom HTML Login Template"

  sender_information = {
    from_local_part = "hello"
    from_domain     = "myapp.com"
    from_name       = "MyApp Team"
  }

  custom_html_customization = {
    template_type     = "LOGIN"
    html_content      = <<-EOT
      <html>
        <body>
          <h1>Welcome back!</h1>
          <p>Click the link below to log in to your account:</p>
          <a href="{{magic_link_url}}">Log In</a>
        </body>
      </html>
    EOT
    plaintext_content = <<-EOT
      Welcome back!

      Click the link below to log in to your account:
      {{magic_link_url}}
    EOT
    subject           = "Log in to your account"
  }
}

# Example: Custom HTML Email Template for OTP
resource "stytch_email_template" "custom_html_otp" {
  project_slug = "my-project"
  template_id  = "custom-html-otp"
  name         = "Custom HTML OTP Template"

  sender_information = {
    from_local_part = "noreply"
    from_domain     = "myapp.com"
    from_name       = "MyApp Security"
  }

  custom_html_customization = {
    template_type     = "ONE_TIME_PASSCODE"
    html_content      = <<-EOT
      <html>
        <body>
          <h1>Your verification code</h1>
          <p>Enter this code to verify your identity:</p>
          <h2>{{otp_code}}</h2>
          <p>This code expires in 10 minutes.</p>
        </body>
      </html>
    EOT
    plaintext_content = <<-EOT
      Your verification code

      Enter this code to verify your identity:
      {{otp_code}}

      This code expires in 10 minutes.
    EOT
    subject           = "Your verification code"
  }
}
