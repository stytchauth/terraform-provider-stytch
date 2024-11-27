# Stytch Terraform Provider

The Stytch Terraform Provider is the official plugin for managing your Stytch workspace configuration via Terraform.

This provider is currently in a _beta_ state. Please report any bugs or

## Documentation

Coming soon.

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
      version = ">= 0.0.1" # Refer to docs for latest version
    }
  }
}

provider "stytch" {}
```

```shell
$ terraform init
```

## Support

If you've found a bug, [open an issue](https://github.com/stytchauth/stytch-management-go/issues/new)!

If you have questions or want help troubleshooting, join us in [Slack](https://stytch.com/docs/resources/support/overview) or email support@stytch.com.

If you've found a security vulnerability, please follow our [responsible disclosure instructions](https://stytch.com/docs/resources/security-and-trust/security#:~:text=Responsible%20disclosure%20program).

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md)

## Code of Conduct

Everyone interacting in the Stytch project's codebases, issue trackers, chat rooms and mailing lists is expected to follow the [code of conduct](CODE_OF_CONDUCT.md).
