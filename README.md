[![Terraform Registry](https://img.shields.io/badge/Terraform-Registry-5C4EE5?logo=terraform)](https://registry.terraform.io/providers/entitleio/entitle/latest)
[![OpenTofu Registry](https://img.shields.io/badge/OpenTofu-Registry-FFDA18?logo=opentofu)](https://search.opentofu.org/provider/entitleio/entitle/latest)
[![Release Version](https://img.shields.io/github/v/release/entitleio/terraform-provider-entitle?logo=github)](https://github.com/entitleio/terraform-provider-entitle/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/entitleio/terraform-provider-entitle)](https://goreportcard.com/report/github.com/entitleio/terraform-provider-entitle)
![Code Scanning](https://img.shields.io/badge/CodeQL-Enabled-brightgreen)
[![Scorecard](https://api.scorecard.dev/projects/github.com/entitleio/terraform-provider-entitle/badge)](https://scorecard.dev/viewer/?uri=github.com/entitleio/terraform-provider-entitle)
[![Last Commit](https://img.shields.io/github/last-commit/entitleio/terraform-provider-entitle)](https://github.com/entitleio/terraform-provider-entitle/commits)

<div align="center">
<img
    width=70%
    height=70%
    src=".github/logo-with-terraform.png"
    alt="header"
/>

<h1>Entitle Terraform Provider</h1>
</div>

The Terraform Provider for Entitle allows you to manage resources and data sources related to Entitle, a platform that provides a seamless way to grant employees granular and just-in-time access within cloud infrastructures and SaaS applications.

### Supported Resources
* **Workflow** (`entitle_workflow`) — A Just-In-Time approval process: who approves, in what order, for how long. Assignable to integrations, resources, roles, and bundles.
* **Integration** (`entitle_integration`) — A configured connection to a specific instance of an application (e.g. a particular AWS account, GitHub org, or Slack workspace), including credentials and access settings.
* **Resource** (`entitle_resource`) — An entity within an integration that users can gain access to via a role (e.g. a database, repository, or user group). Use this for manual/virtual integrations where Entitle manages the resource lifecycle.
* **Resource Synced** (`entitle_resource_synced`) — Adopts an existing resource that is automatically synchronized from an external integration (GCP, AWS, GitHub, Okta, etc.). Terraform manages its Entitle settings (owner, workflow, durations) without creating or deleting the underlying resource.
* **Role** (`entitle_role`) — The atomic permission unit within a resource (e.g. `readonly`, `admin`). Roles can carry their own workflow, allowed durations, and prerequisite permissions. Use this for manual/virtual integrations where Entitle manages the role lifecycle.
* **Role Synced** (`entitle_role_synced`) — Adopts an existing role that is automatically synchronized from an external integration. Terraform manages its Entitle settings without creating or deleting the underlying role.
* **Bundle** (`entitle_bundle`) — A cross-application package of roles that can be requested or revoked as a single action — effectively a "super role" spanning multiple integrations.
* **Policy** (`entitle_policy`) — A rule that automatically grants birthright permissions to users in a group, and revokes them on group leave.
* **Agent Token** (`entitle_agent_token`) — Credential used by the on-prem Entitle Agent to authenticate with the platform when connecting private/internal systems.
* **Permission** (`entitle_permission`) — **Import-only.** Represents an active granted entitlement; created by Entitle through the request/approval flow. Use this to bring existing permissions under Terraform management for tracking or bulk revocation.
* **Access Request Forward** (`entitle_access_request_forward`) — Delegates a user's pending access request responsibilities to a colleague (vacation, leave, role change).
* **Access Review Forward** (`entitle_access_review_forward`) — Delegates a user's access review responsibilities during periodic review campaigns.

### Supported Data Sources

The provider also exposes 17 data sources for looking up existing Entitle objects (use these instead of hardcoding UUIDs in your configuration):

| Singular (lookup one)              | Plural (list / filter)      |
|------------------------------------|-----------------------------|
| `entitle_user`                     | `entitle_users`             |
| `entitle_role`                     | `entitle_roles`             |
| `entitle_resource`                 | `entitle_resources`         |
| `entitle_workflow`                 | —                           |
| `entitle_bundle`                   | —                           |
| `entitle_policy`                   | —                           |
| `entitle_integration`              | —                           |
| `entitle_agent_token`              | —                           |
| `entitle_access_request_forward`   | —                           |
| `entitle_access_review_forward`    | —                           |
| —                                  | `entitle_accounts`          |
| —                                  | `entitle_applications`      |
| —                                  | `entitle_directory_groups`  |
| —                                  | `entitle_permissions`       |

See the [Terraform Registry documentation](https://registry.terraform.io/providers/entitleio/entitle/latest/docs) for full schemas.

## Prerequisites

Before using the Entitle Terraform Provider, ensure you have the following:

### Provider Configuration
Configure the Entitle Terraform Provider with your API key and endpoint:

* `api_key` (Required): The API key for authenticating with the Entitle API.
* `endpoint` (Optional): The URL endpoint for the Entitle API. default: https://api.entitle.io.

To use the Entitle Terraform Provider, you must configure it with the necessary authentication details.

```hcl
provider "entitle" {
  api_key = "your_api_key"
  endpoint = "https://api.entitle.io"
}
```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine 
(see [Requirements](#requirements) below).

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24.3

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:


To compile the provider, run `go install`. This will build the provider and put the provider binary in 
the `$GOPATH/bin` directory.

before starting you need to overide the configuration for the terraform provider to be search first of all localy in
`$GOPATH/bin`

```hcl
provider_installation {

  dev_overrides {
      "entitleio/entitle" = "/absolute/path/to/your/GOPATH/bin"  # e.g. "/Users/you/go/bin" — Terraform does not expand $GOPATH; the path must be literal
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
