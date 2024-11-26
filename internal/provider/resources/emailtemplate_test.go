package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/testutil"
)

func TestAccEmailTemplateResource(t *testing.T) {
	for _, testCase := range []testutil.TestCase{
		{
			Name: "prebuilt",
			Config: testutil.ConsumerProjectConfig + `
        resource "stytch_email_template" "test" {
          live_project_id = stytch_project.project.live_project_id
          template_id = "tf-test"
          name = "tf-test"
          prebuilt_customization = {
            button_border_radius = 3
            button_color         = "#105ee9"
            button_text_color    = "#ffffff"
            font_family          = "GEORGIA"
            text_alignment       = "CENTER"
          }
        }`,
			Checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr("stytch_email_template.test", "template_id", "tf-test"),
				resource.TestCheckResourceAttr("stytch_email_template.test", "name", "tf-test"),
				resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_border_radius", "3"),
				resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_color", "#105ee9"),
				resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.button_text_color", "#ffffff"),
				resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.font_family", "GEORGIA"),
				resource.TestCheckResourceAttr("stytch_email_template.test", "prebuilt_customization.text_alignment", "CENTER"),
			},
		},
	} {
		t.Run(testCase.Name, func(t *testing.T) {
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
