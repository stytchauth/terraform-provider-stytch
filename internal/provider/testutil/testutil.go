package testutil

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider"
)

const (
	// providerConfig is a shared configuration to combine with the actual test configuration so the Stytch client is
	// properly configured. The tester should set the STYTCH_ environment variables for the workspace key and secret to
	// allow the tests to run properly.
	ProviderConfig = `provider "stytch" {}`
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"stytch": providerserver.NewProtocol6WithError(provider.New("test")()),
}

var ConsumerProjectConfig = `
resource "stytch_project" "project" {
  name     = "test"
  vertical = "CONSUMER"
}`

var B2BProjectConfig = `
resource "stytch_project" "project" {
  name     = "test-b2b"
  vertical = "B2B"
}`

type TestCase struct {
	Name   string
	Config string
	Checks []resource.TestCheckFunc
}

type ErrorCase struct {
	Name   string
	Config string
	Error  *regexp.Regexp
}

func (e *ErrorCase) AssertAnyError(t *testing.T) {
	t.Helper()
	t.Run(e.Name, func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      ProviderConfig + e.Config,
					ExpectError: regexp.MustCompile(`.*`),
				},
			},
		})
	})
}
