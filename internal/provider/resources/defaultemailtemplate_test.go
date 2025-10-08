package resources_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccDefaultEmailTemplateResource(t *testing.T) {
	customDomain := os.Getenv("STYTCH_CUSTOM_DOMAIN")
	if customDomain == "" {
		t.Skip("STYTCH_CUSTOM_DOMAIN environment variable must be set to run this test")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create email templates first, then set one as default
				Config: testutil.ConsumerProjectConfig + `
        resource "stytch_email_template" "login1" {
          project_slug = stytch_project.test.project_slug
          template_id = "tf-test-login-1"
          name = "Login Template 1"
          sender_information = {
            from_local_part = "noreply"
            from_domain = "` + customDomain + `"
            from_name = "Stytch Test"
          }
          custom_html_customization = {
            template_type = "LOGIN"
            subject = "Log in to your account"
            html_content = "<html><body><h1>Welcome back!</h1><p>Click here: {{magic_link_url}}</p></body></html>"
            plaintext_content = "Welcome back! Click here: {{magic_link_url}}"
          }
        }

        resource "stytch_email_template" "login2" {
          project_slug = stytch_project.test.project_slug
          template_id = "tf-test-login-2"
          name = "Login Template 2"
          sender_information = {
            from_local_part = "noreply"
            from_domain = "` + customDomain + `"
            from_name = "Stytch Test"
          }
          custom_html_customization = {
            template_type = "LOGIN"
            subject = "Sign in now"
            html_content = "<html><body><h1>Sign in</h1><p>Click here: {{magic_link_url}}</p></body></html>"
            plaintext_content = "Sign in. Click here: {{magic_link_url}}"
          }
        }

        resource "stytch_default_email_template" "login" {
          project_slug = stytch_project.test.project_slug
          email_template_type = "LOGIN"
          template_id = stytch_email_template.login1.template_id
        }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("stytch_default_email_template.login", "project_slug"),
					resource.TestCheckResourceAttr("stytch_default_email_template.login", "email_template_type", "LOGIN"),
					resource.TestCheckResourceAttr("stytch_default_email_template.login", "template_id", "tf-test-login-1"),
					resource.TestCheckResourceAttrSet("stytch_default_email_template.login", "id"),
				),
			},
			{
				// Import state testing
				ResourceName:            "stytch_default_email_template.login",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			{
				// Update to use the second template
				Config: testutil.ConsumerProjectConfig + `
        resource "stytch_email_template" "login1" {
          project_slug = stytch_project.test.project_slug
          template_id = "tf-test-login-1"
          name = "Login Template 1"
          sender_information = {
            from_local_part = "noreply"
            from_domain = "` + customDomain + `"
            from_name = "Stytch Test"
          }
          custom_html_customization = {
            template_type = "LOGIN"
            subject = "Log in to your account"
            html_content = "<html><body><h1>Welcome back!</h1><p>Click here: {{magic_link_url}}</p></body></html>"
            plaintext_content = "Welcome back! Click here: {{magic_link_url}}"
          }
        }

        resource "stytch_email_template" "login2" {
          project_slug = stytch_project.test.project_slug
          template_id = "tf-test-login-2"
          name = "Login Template 2"
          sender_information = {
            from_local_part = "noreply"
            from_domain = "` + customDomain + `"
            from_name = "Stytch Test"
          }
          custom_html_customization = {
            template_type = "LOGIN"
            subject = "Sign in now"
            html_content = "<html><body><h1>Sign in</h1><p>Click here: {{magic_link_url}}</p></body></html>"
            plaintext_content = "Sign in. Click here: {{magic_link_url}}"
          }
        }

        resource "stytch_default_email_template" "login" {
          project_slug = stytch_project.test.project_slug
          email_template_type = "LOGIN"
          template_id = stytch_email_template.login2.template_id
        }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("stytch_default_email_template.login", "template_id", "tf-test-login-2"),
				),
			},
		},
	})
}
