# Development

Thanks for contributing to Stytch's Terraform Provider plugin! If you run into trouble, find us in [Slack].

## Setup

### Prerequisites

- [Terraform](https://www.terraform.io/)
- A [Stytch](https://stytch.com) workspace
- A [workspace management key + secret](https://stytch.com/dashboard/settings/management-api) for your Stytch workspace
  - Create a new key and store the key + secret somewhere safe

### Configuring your environment

It is highly recommended that you put your workspace management key + secret in environment variables.

```sh
export STYTCH_WORKSPACE_KEY_ID=my_workspace_key_id_goes_here
export STYTCH_WORKSPACE_KEY_SECRET=my_secret_goes_here
```

The `stytch` provider will attempt to read environment variables for configuring the client.
You can also set these directly in the provider configuration, but it is not recommended.

## Issues and Pull Requests

Please file issues in this repo. We don't have an issue template yet, but for now, say whatever you think is important!

If you have non-trivial changes you'd like us to incorporate, please open an issue first so we can discuss the changes before starting on a pull request. (It's fine to start with the PR for a typo or simple bug.) If we think the changes align with the direction of the project, we'll either ask you to open the PR or assign someone on the Stytch team to make the changes.

When you're ready for someone to look at your issue or PR, assign `@stytchauth/client-libraries` (GitHub should do this automatically). If we don't acknowledge it within one business day, please escalate it by tagging `@stytchauth/engineering` in a comment or letting us know in [Slack].

[Slack]: https://stytch.slack.com/join/shared_invite/zt-2f0fi1ruu-ub~HGouWRmPARM1MTwPESA
