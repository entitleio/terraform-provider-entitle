Manages a Bitbucket integration in Entitle.

Bitbucket is a Git-based source code repository hosting service owned by Atlassian. Entitle can manage workspace and repository permissions in Bitbucket.

For more information on setting up Bitbucket with Entitle, see the [Bitbucket integration guide](https://docs.beyondtrust.com/entitle/docs/entitle-integration-bitbucket).

## Prerequisites

Before creating this resource you will need an Atlassian API token with Bitbucket scopes:

1. Go to the [Atlassian API Tokens](https://id.atlassian.com/manage-profile/security/api-tokens) page and click **Create API token with scopes**.
2. Set a name and expiration for the token, then click **Next**.
3. Choose **Bitbucket** as the API token app, then click **Next**.
4. Grant the token the following scopes, then click **Next**:
   ```
   admin:workspace:bitbucket
   admin:repository:bitbucket
   read:workspace:bitbucket
   read:repository:bitbucket
   read:permission:bitbucket
   write:permission:bitbucket
   delete:permission:bitbucket
   ```
5. Review the token details and click **Create token**.
6. Copy the generated token — you will not be able to view it again.
7. Copy the account email from the [Atlassian Email](https://id.atlassian.com/manage-profile/email) page.

> **Note:** Bitbucket's API does not expose user email addresses. To let Entitle automatically match Bitbucket users to identities by email, configure the optional `jira_credentials` block using a Jira API token — Entitle uses the Jira API to retrieve user emails.

## Connection Data

The `connection_data` block configures the credentials Entitle uses to connect to Bitbucket:

| Attribute | Required | Description |
|---|---|---|
| `email` | Yes | Atlassian account email address used to authenticate with Bitbucket |
| `app_token` | Yes | Atlassian API token with Bitbucket scopes (see Prerequisites) |
| `jira_credentials` | No | Optional Jira credentials used to look up user email addresses for email-based matching |

The `jira_credentials` block, when provided, requires:

| Attribute | Required | Description |
|---|---|---|
| `url` | Yes | Jira instance URL (e.g. `https://your-domain.atlassian.net`) |
| `key` | Yes | Jira API token used to authenticate with the Jira API |
| `user` | Yes | Jira account email address associated with the API token |

## Example Usage

### Basic

```terraform
resource "entitle_integration_bitbucket" "bitbucket" {
  name = "Bitbucket - Engineering Org"

  connection_data = {
    email     = var.bitbucket_email
    app_token = var.bitbucket_app_token
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations = [3600, 21600, 86400]
  requestable       = true
}
```

### With Jira credentials for email matching

```terraform
resource "entitle_integration_bitbucket" "bitbucket" {
  name = "Bitbucket - Engineering Org"

  connection_data = {
    email     = var.bitbucket_email
    app_token = var.bitbucket_app_token

    jira_credentials = {
      url  = "https://your-domain.atlassian.net"
      key  = var.jira_api_token
      user = var.jira_user_email
    }
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations = [3600, 21600]
  requestable       = true
}
```

## Import

Existing Bitbucket integrations can be imported using the integration UUID:

```shell
terraform import entitle_integration_bitbucket.example a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

Use the `entitle_integration` data source to look it up by name:

```terraform
data "entitle_integration" "bitbucket" {
  name = "My Bitbucket Integration"
}

output "bitbucket_integration_id" {
  value = data.entitle_integration.bitbucket.id
}
```

Alternatively, navigate to the **Integrations** page in the Entitle UI and copy the ID from the browser URL:

```
https://app.entitle.io/integrations/a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

## Notes and Best Practices

- **`allow_creating_accounts` must be `false`** — Bitbucket user accounts are managed externally and cannot be created by Entitle. Setting this to `true` will produce a validation error.
- **`allow_changing_account_permissions` must be `true`** — Entitle manages workspace and repository permissions, which requires permission to change account permissions.
- `email`, `app_token`, and `jira_credentials` are write-only attributes; store their values in a secrets manager and reference them via sensitive Terraform variables rather than hardcoding them in configuration files.
- Since `connection_data` fields are write-only, changing them will not be detected as drift — update the values explicitly and apply again to rotate credentials.
- Configure `jira_credentials` if you want Entitle to match Bitbucket users to identities by email; without it, Bitbucket users can only be matched by other identifiers.
