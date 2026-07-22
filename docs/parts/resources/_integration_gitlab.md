Manages a GitLab integration in Entitle.

GitLab is a web-based DevOps platform that allows software development teams to collaborate, manage source code repositories, track issues, and automate the software delivery pipeline. Entitle can manage **groups** (all GitLab versions) and **projects** (on-premises version only).

For more information on setting up GitLab with Entitle, see the [GitLab integration guide](https://docs.beyondtrust.com/entitle/docs/entitle-integration-gitlab).

## Prerequisites

Before creating this resource you will need:

1. **GitLab domain** — your GitLab login URL. For GitLab SaaS use `https://gitlab.com`; for self-hosted instances use your own domain.
2. **Personal Access Token** — created in GitLab under *Edit Profile → Access Tokens* with the `api` scope selected. Copy the generated token — you will not be able to view it again.

> **Note:** Entitle can fetch user emails from GitLab only when a user has populated the **Email** or **Public email** fields on their GitLab profile.

## Connection Data

The `connection_data` block configures the credentials and SSL settings Entitle uses to connect to GitLab:

| Attribute | Required | Default | Description |
|---|---|---|---|
| `private_token` | Yes | — | GitLab Personal Access Token with `api` scope |
| `domain` | No | `https://gitlab.com` | GitLab instance URL |
| `ssl_verify` | No | `true` | Whether to verify the SSL certificate. Set to `false` only when using self-signed certificates without providing `ssl_ca_cert` |
| `ssl_ca_cert` | No | — | Path to a custom CA certificate file (PEM format). Required for self-signed certificates |

### SSL / Certificate notes

- If `ssl_verify = true` and no `ssl_ca_cert` is provided, standard public certificate verification is used (suitable for `https://gitlab.com` and most hosted instances).
- If `ssl_verify = false`, SSL verification is disabled entirely — use only as a last resort.
- For self-hosted GitLab with a self-signed certificate, set `ssl_ca_cert` to the path of your CA file (e.g. `/etc/ssl/certs/custom_ca.pem`). The Entitle agent must have read access to that path.

To convert a `.crt` file to the single-line PEM format expected by the API you can use:

```bash
awk '{printf "%s\\n", $0}' /path/to/ca.crt
```

## Example Usage

### GitLab SaaS (gitlab.com)

```terraform
resource "entitle_integration_gitlab" "gitlab_saas" {
  name = "GitLab - Engineering Org"

  connection_data = {
    domain        = "https://gitlab.com"
    private_token = var.gitlab_token
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations = [3600, 21600, 86400]
  requestable             = true
}
```

### Self-Hosted GitLab with SSL

```terraform
resource "entitle_integration_gitlab" "gitlab_selfhosted" {
  name = "GitLab - Self-Hosted"

  connection_data = {
    domain        = "https://gitlab.example.com"
    private_token = var.gitlab_token
    ssl_verify    = true
    ssl_ca_cert   = "/etc/ssl/certs/gitlab_ca.pem"
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  agent_token = {
    name = entitle_agent_token.internal_agent.name
  }


  allowed_durations       = [3600, 21600]
  requestable             = true
}
```

### Self-Hosted GitLab — SSL Verification Disabled

```terraform
resource "entitle_integration_gitlab" "gitlab_no_ssl" {
  name = "GitLab - Internal (no SSL verify)"

  connection_data = {
    domain        = "https://gitlab.internal.example.com"
    private_token = var.gitlab_token
    ssl_verify    = false
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }


  allowed_durations       = [-1]
  requestable             = true
}
```

## Import

Existing GitLab integrations can be imported using the integration UUID:

```shell
terraform import entitle_integration_gitlab.example a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

Use the `entitle_integration` data source to look it up by name:

```terraform
data "entitle_integration" "gitlab" {
  name = "My GitLab Integration"
}

output "gitlab_integration_id" {
  value = data.entitle_integration.gitlab.id
}
```

Alternatively, navigate to the **Integrations** page in the Entitle UI and copy the ID from the browser URL:

```
https://app.entitle.io/integrations/a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

## Notes and Best Practices

- **`allow_creating_accounts` must be `false`** — GitLab user accounts are managed externally and cannot be created by Entitle. Setting this to `true` will produce a validation error.
- **`allow_changing_account_permissions` must be `true`** — Entitle manages group and project memberships, which requires permission to change account permissions.
- Store `private_token` in a secrets manager and reference it via a sensitive Terraform variable rather than hardcoding it in configuration files.
- For on-premises or VPC-internal GitLab instances, pair this resource with an `entitle_agent_token` so that the Entitle agent handles outbound connectivity to your GitLab server.
- Entitle manages GitLab **groups** on all versions, and **projects** on self-hosted (on-premises) versions only.
