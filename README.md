# Stytch Terraform Provider

The Stytch Terraform Provider is the official plugin for managing your Stytch workspace configuration via Terraform.

> [!IMPORTANT]
> This is the v3 version of the terraform provider, which has various breaking changes from v1, as well as new functionality. It uses the new v3 version of the Stytch Management API and its respective Go SDK. If you are currently using v1, please read the [migration guide](./migrating_v1_to_v3.md).

## Documentation

The latest documentation is available at the [Terraform registry stytch provider](https://registry.terraform.io/providers/stytchauth/stytch/latest/docs).

## Getting started

### Requirements

- [Terraform](https://www.terraform.io/downloads)
- A [Stytch](https://stytch.com) workspace
  - Workspace management key to authenticate with the Stytch Management API

### Installation

Terraform uses the [Terraform Registry](https://registry.terraform.io/) to download and install providers. To install
this provider, copy and paste the following code into your Terraform configuration. Then, run `terraform init`.

```terraform
terraform {
  required_providers {
    stytch = {
      source  = "stytchauth/stytch"
      version = ">= 3.0.0" # Refer to docs for latest version
    }
  }
}

provider "stytch" {}
```

```shell
$ terraform init
```

## Previous version support

- The v1 version of the provider is still available in the Terraform registry and in the v1 branch of this repository.
- As of November 17, 2025, the v1 version of the provider will not get any new features. Maintenance will be limited to major bug fixes and security upgrades. 
- V1 will reach end of life **not before** June 30, 2026. A more detailed deprecation timeline will be published in our official docs.

## Support

If you've found a bug, [open an issue](https://github.com/stytchauth/stytch-management-go/issues/new)!

If you have questions or want help troubleshooting, join us in [Slack](https://stytch.com/docs/resources/support/overview) or email support@stytch.com.

If you've found a security vulnerability, please follow our [responsible disclosure instructions](https://stytch.com/docs/resources/security-and-trust/security#:~:text=Responsible%20disclosure%20program).

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md)

## Code of Conduct

Everyone interacting in the Stytch project's codebases, issue trackers, chat rooms and mailing lists is expected to follow the [code of conduct](CODE_OF_CONDUCT.md).
