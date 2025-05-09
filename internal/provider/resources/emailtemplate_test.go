package resources_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

type emailTemplateTestCase struct {
	testutil.TestCase
	shouldSkip bool
}

func TestAccEmailTemplateResource(t *testing.T) {
	customDomain := os.Getenv("STYTCH_CUSTOM_DOMAIN")
	for _, testCase := range []emailTemplateTestCase{
		{
			TestCase: testutil.TestCase{
				Name: "prebuilt",
				Config: testutil.ConsumerProjectConfig + `
        resource "stytch_email_template" "test" {
          live_project_id = stytch_project.project.live_project_id
          template_id = "tf-test-prebuilt"
          name = "tf-test-prebuilt"
          prebuilt_customization = {
            button_border_radius = 3
            button_color         = "#105ee9"
            button_text_color    = "#ffffff"
            font_family          = "GEORGIA"
            text_alignment       = "CENTER"
          }
        }`,
				Checks: []resource.TestCheckFunc{
					resource.TestCheckResourceAttr("stytch_email_template.test", "template_id", "tf-test-prebuilt"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "name", "tf-test-prebuilt"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_border_radius", "3"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_color", "#105ee9"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_text_color", "#ffffff"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.font_family", "GEORGIA"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.text_alignment", "CENTER"),
				},
			},
			shouldSkip: false,
		},
		{
			TestCase: testutil.TestCase{
				Name: "prebuilt-with-sender",
				Config: testutil.ConsumerProjectConfig + `
      resource "stytch_email_template" "test" {
        live_project_id = stytch_project.project.live_project_id
        template_id = "tf-test-prebuilt2"
        name = "tf-test-prebuilt2"
        sender_information = {
          from_local_part = "noreply"
          from_domain = "` + customDomain + `"
          from_name = "Stytch"
          reply_to_local_part = "support"
          reply_to_name = "Support"
        }
        prebuilt_customization = {
          button_border_radius = 3
          button_color         = "#105ee9"
          button_text_color    = "#ffffff"
          font_family          = "GEORGIA"
          text_alignment       = "CENTER"
        }
      }`,
				Checks: []resource.TestCheckFunc{
					resource.TestCheckResourceAttr("stytch_email_template.test", "template_id", "tf-test-prebuilt2"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "name", "tf-test-prebuilt2"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_local_part", "noreply"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_domain", customDomain),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_name", "Stytch"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.reply_to_local_part", "support"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.reply_to_name", "Support"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_border_radius", "3"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_color", "#105ee9"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_text_color", "#ffffff"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.font_family", "GEORGIA"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.text_alignment", "CENTER"),
				},
			},
			shouldSkip: customDomain == "",
		},
		{
			TestCase: testutil.TestCase{
				Name: "custom login",
				Config: testutil.ConsumerProjectConfig + `
      resource "stytch_email_template" "test" {
        live_project_id = stytch_project.project.live_project_id
        template_id = "tf-test-custom-login"
        name = "tf-test-custom-login"
        sender_information = {
          from_local_part = "noreply"
          from_domain = "` + customDomain + `"
          from_name = "Stytch"
          reply_to_local_part = "support"
          reply_to_name = "Support"
        }
        custom_html_customization = {
          template_type = "LOGIN"
          html_content = "<h1>Login now: {{magic_link_url}}</h1>"
          plaintext_content = "Plaintext login now: {{magic_link_url}}"
          subject = "Login to ` + customDomain + `"
        }
      }`,
				Checks: []resource.TestCheckFunc{
					resource.TestCheckResourceAttr("stytch_email_template.test", "template_id", "tf-test-custom-login"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "name", "tf-test-custom-login"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_local_part", "noreply"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_domain", customDomain),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_name", "Stytch"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.reply_to_local_part", "support"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.reply_to_name", "Support"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.template_type", "LOGIN"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.html_content", "<h1>Login now: {{magic_link_url}}</h1>"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.plaintext_content", "Plaintext login now: {{magic_link_url}}"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.subject", "Login to "+customDomain),
				},
			},
			shouldSkip: customDomain == "",
		},
		{
			TestCase: testutil.TestCase{
				Name: "custom reset password",
				Config: testutil.ConsumerProjectConfig + `
      resource "stytch_email_template" "test" {
        live_project_id = stytch_project.project.live_project_id
        template_id = "tf-test-custom-reset-password"
        name = "tf-test-custom-reset-password"
        sender_information = {
          from_local_part = "noreply"
          from_domain = "` + customDomain + `"
          from_name = "Stytch"
          reply_to_local_part = "support"
          reply_to_name = "Support"
        }
        custom_html_customization = {
          template_type = "RESET_PASSWORD"
          html_content = "<h1>Reset password now: {{reset_password_url}}</h1>"
          plaintext_content = "Plaintext reset password now: {{reset_password_url}}"
          subject = "Reset password for ` + customDomain + `"
        }
      }`,
				Checks: []resource.TestCheckFunc{
					resource.TestCheckResourceAttr("stytch_email_template.test", "template_id", "tf-test-custom-reset-password"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "name", "tf-test-custom-reset-password"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_local_part", "noreply"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_domain", customDomain),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_name", "Stytch"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.reply_to_local_part", "support"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.reply_to_name", "Support"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.template_type", "RESET_PASSWORD"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.html_content", "<h1>Reset password now: {{reset_password_url}}</h1>"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.plaintext_content", "Plaintext reset password now: {{reset_password_url}}"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.subject", "Reset password for "+customDomain),
				},
			},
			shouldSkip: customDomain == "",
		},
		{
			TestCase: testutil.TestCase{
				Name: "custom otp",
				Config: testutil.ConsumerProjectConfig + `
      resource "stytch_email_template" "test" {
        live_project_id = stytch_project.project.live_project_id
        template_id = "tf-test-custom-otp"
        name = "tf-test-custom-otp"
        sender_information = {
          from_local_part = "noreply"
          from_domain = "` + customDomain + `"
          from_name = "Stytch"
          reply_to_local_part = "support"
          reply_to_name = "Support"
        }
        custom_html_customization = {
          template_type = "ONE_TIME_PASSCODE"
          html_content = "<h1>One time passcode: {{otp_code}}</h1>"
          plaintext_content = "Plaintext one time passcode: {{otp_code}}"
          subject = "OTP for ` + customDomain + `"
        }
      }`,
				Checks: []resource.TestCheckFunc{
					resource.TestCheckResourceAttr("stytch_email_template.test", "template_id", "tf-test-custom-otp"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "name", "tf-test-custom-otp"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_local_part", "noreply"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_domain", customDomain),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_name", "Stytch"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.reply_to_local_part", "support"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.reply_to_name", "Support"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.template_type", "ONE_TIME_PASSCODE"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.html_content", "<h1>One time passcode: {{otp_code}}</h1>"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.plaintext_content", "Plaintext one time passcode: {{otp_code}}"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.subject", "OTP for "+customDomain),
				},
			},
			shouldSkip: customDomain == "",
		},
		{
			TestCase: testutil.TestCase{
				Name: "custom-without-plaintext",
				Config: testutil.ConsumerProjectConfig + `
      resource "stytch_email_template" "test" {
        live_project_id = stytch_project.project.live_project_id
        template_id = "tf-test-custom-2"
        name = "tf-test-custom-2"
        sender_information = {
          from_local_part = "noreply"
          from_domain = "` + customDomain + `"
          from_name = "Stytch"
          reply_to_local_part = "support"
          reply_to_name = "Support"
        }
        custom_html_customization = {
          template_type = "LOGIN"
          html_content = "<h1>Login now: {{magic_link_url}}</h1>"
          subject = "Login to ` + customDomain + `"
        }
      }`,
				Checks: []resource.TestCheckFunc{
					resource.TestCheckResourceAttr("stytch_email_template.test", "template_id", "tf-test-custom-2"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "name", "tf-test-custom-2"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_local_part", "noreply"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_domain", customDomain),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.from_name", "Stytch"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.reply_to_local_part", "support"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "sender_information.reply_to_name", "Support"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.template_type", "LOGIN"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.html_content", "<h1>Login now: {{magic_link_url}}</h1>"),
					resource.TestCheckResourceAttr("stytch_email_template.test", "custom_html_customization.subject", "Login to "+customDomain),
				},
			},
			shouldSkip: customDomain == "",
		},
	} {
		t.Run(testCase.Name, func(t *testing.T) {
			if testCase.shouldSkip {
				t.Skip("Skipping test due to missing STYTCH_CUSTOM_DOMAIN environment variable")
			}
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create and Read testing
						Config: testutil.ProviderConfig + testCase.Config,
						Check:  resource.ComposeAggregateTestCheckFunc(testCase.Checks...),
					},
					{
						// Import state testing
						ResourceName:      "stytch_email_template.test",
						ImportState:       true,
						ImportStateVerify: true,
						// Ignore timestamp fields because rounding differences in various APIs can result in different
						// values being returned
						ImportStateVerifyIgnore: []string{"last_updated"},
					},
					{
						// Update and Read testing
						Config: testutil.ProviderConfig + testutil.ConsumerProjectConfig + `
            resource "stytch_email_template" "test" {
              live_project_id = stytch_project.project.live_project_id
              template_id = "tf-test"
              name = "tf-test"
              prebuilt_customization = {
                button_border_radius = 2
                button_color         = "#101010"
                button_text_color    = "#abcdef"
                font_family          = "TAHOMA"
                text_alignment       = "LEFT"
              }
            }`,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttr("stytch_email_template.test", "template_id", "tf-test"),
							resource.TestCheckResourceAttr("stytch_email_template.test", "name", "tf-test"),
							resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_border_radius", "2"),
							resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_color", "#101010"),
							resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_text_color", "#abcdef"),
							resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.font_family", "TAHOMA"),
							resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.text_alignment", "LEFT"),
						),
						// Delete testing automatically occurs in resource.TestCase
					},
				},
			})
		})
	}
}

func TestAccEmailTemplateResource_Invalid(t *testing.T) {
	for _, errorCase := range []testutil.ErrorCase{
		{
			Name: "both prebuilt and custom",
			Config: testutil.ConsumerProjectConfig + `
      resource "stytch_email_template" "test" {
        live_project_id = stytch_project.project.live_project_id
        template_id = "tf-test-prebuilt"
        name = "tf-test-prebuilt"
        sender_information = {
          from_local_part = "noreply"
          from_domain = "example.com"
          from_name = "Stytch"
          reply_to_local_part = "support"
          reply_to_name = "Support"
        }
        prebuilt_customization = {
          button_border_radius = 3
          button_color         = "#105ee9"
          button_text_color    = "#ffffff"
          font_family          = "GEORGIA"
          text_alignment       = "CENTER"
        }
        custom_html_customization = {
          template_type = "LOGIN"
          html_content = "<h1>Login now: {{magic_link_url}}</h1>"
          plaintext_content = "Login now: {{magic_link_url}}"
          subject = "Login to example.com"
        }
      }`,
		},
		{
			Name: "custom missing sender_info",
			Config: testutil.ConsumerProjectConfig + `
      resource "stytch_email_template" "test" {
        live_project_id = stytch_project.project.live_project_id
        template_id = "tf-test-prebuilt"
        name = "tf-test-prebuilt"
        custom_html_customization = {
          template_type = "LOGIN"
          html_content = "<h1>Login now: {{magic_link_url}}</h1>"
          plaintext_content = "Login now: {{magic_link_url}}"
          subject = "Login to example.com"
        }
      }`,
		},
	} {
		errorCase.AssertAnyError(t)
	}
}
