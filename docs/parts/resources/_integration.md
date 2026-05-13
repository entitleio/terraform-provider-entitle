An Entitle Integration is a configured connection to an external application or system. It represents a specific instance of a supported application (e.g., a particular AWS account, a GitHub organization, or a Slack workspace) and contains all the configuration Entitle needs to read permissions, manage access, and respond to access requests for that system.

Integrations are the top-level container in the Entitle access model. Each integration contains resources, which contain roles — forming a three-level hierarchy: Integration → Resource → Role. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **Integration**: A named, configured connection to a specific instance of an application
- **Application**: The type of system being connected (e.g., `"aws"`, `"github"`, `"slack"`) — chosen from Entitle's supported application catalog
- **connection_json**: The application-specific credentials and configuration (API tokens, account IDs, etc.)
- **Owner**: The user responsible for this integration — used in approval workflows and administrative notifications
- **Workflow**: The default approval process for JIT access requests to any resource under this integration (can be overridden at the resource or role level)
- **Agent Token**: Required for integrations that connect to private/internal systems not reachable from the internet
- **Maintainers**: Secondary owners who assist with administrative responsibilities

## When to Use Integrations

- Connecting a new application to Entitle for the first time
- Managing existing integration settings (owner, workflow, access policies) via IaC
- Enabling or disabling account creation, permission modification, or requestability for an entire application
- Setting up agent-based connectivity for on-premise or private cloud applications

## Example Usage

### Basic Integration (Cloud Application)

Connect a Slack workspace with manager approval for all access requests:

```terraform
resource "entitle_integration" "slack_workspace" {
  name = "Slack - Engineering Workspace"
  connection_json = jsonencode({
    token = var.slack_token
    options = {
      plan = "pro"
    }
  })

  application = {
    name = "slack"
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations       = [3600, 28800, 86400]
  allow_creating_accounts = true
}
```

### AWS Integration with Restricted Settings

Connect an AWS account with account creation disabled and readonly mode for manual review:

```terraform
resource "entitle_integration" "aws_production" {
  name            = "AWS Production Account"
  connection_json = jsonencode({
    account_id  = var.aws_account_id
    role_arn    = var.aws_role_arn
    external_id = var.aws_external_id
  })

  application = {
    name = "aws"
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations                    = [3600, 7200]
  allow_creating_accounts              = false   # Users must already exist in AWS
  allow_changing_account_permissions   = true
  readonly                             = false
  notify_about_external_permission_changes = true
}
```

### Integration with Maintainers

Add secondary owners (maintainers) who help manage the integration:

```terraform
resource "entitle_integration" "github_org" {
  name            = "GitHub Organization"
  connection_json = jsonencode({
    token        = var.github_token
    organization = var.github_org
  })

  application = {
    name = "github"
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  maintainers = [
    {
      type = "user"
      entity = {
        id = "7d080bfa-9143-11ee-b9d1-0242ac120003"  # Security lead
      }
    },
    {
      type = "group"
      entity = {
        id = "7d080bfa-9143-11ee-b9d1-0242ac120004"  # DevOps team group
      }
    }
  ]

  allowed_durations = [3600, 28800]
  allow_creating_accounts = true
}
```

### Agent-Based Integration (Private/Internal System)

Connect an on-premise or VPC-internal system using the Entitle Agent:

```terraform
resource "entitle_agent_token" "internal_db_agent" {
  name = "internal-db-agent"
}

resource "entitle_integration" "internal_postgres" {
  name            = "Internal PostgreSQL (Production)"
  connection_json = jsonencode({
    host     = "postgres.internal.example.com"
    port     = 5432
    database = "production"
    username = var.db_username
    password = var.db_password
  })

  application = {
    name = "postgresql"
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  agent_token = {
    name = entitle_agent_token.internal_db_agent.name
  }

  allowed_durations       = [3600, 28800]
  allow_creating_accounts = false
}
```

### Integration with Prerequisite Permissions

Define permissions that are automatically granted alongside any role from this integration:

```terraform
resource "entitle_integration" "okta_admin" {
  name            = "Okta Admin"
  connection_json = jsonencode({
    domain    = var.okta_domain
    api_token = var.okta_api_token
  })

  application = {
    name = "okta"
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  prerequisite_permissions = [
    {
      default = true
      role = {
        id = "7d080bfa-9143-11ee-b9d1-0242ac120003"  # Okta read-only access
      }
    }
  ]

  allowed_durations = [3600, 7200]
  allow_creating_accounts = false
}
```

### Read-Only Integration

Set up an integration in read-only mode — requests create tickets for manual resolution rather than auto-granting:

```terraform
resource "entitle_integration" "legacy_erp" {
  name            = "Legacy ERP System"
  connection_json = jsonencode({
    api_endpoint = "https://erp.internal.example.com/api"
    api_key      = var.erp_api_key
  })

  application = {
    name = "custom"
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  readonly          = true   # Requests create manual tickets
  requestable       = true
  allowed_durations = [-1]
  allow_creating_accounts = false
}
```

## Import

Existing integrations can be imported using their UUID:

```shell
terraform import entitle_integration.example a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

### Finding the Integration ID

To find the UUID of an existing integration:

1. Log in to the Entitle UI
2. Navigate to the **Integrations** section
3. Click on the integration you want to import
4. The integration ID (UUID) will be visible in the browser URL
    - Example: `https://app.entitle.io/integrations/a1b2c3d4-e5f6-7890-abcd-ef1234567890`

Alternatively, use the `entitle_integration` data source:

```terraform
data "entitle_integration" "existing" {
  name = "My Existing Integration"
}

output "integration_id" {
  value = data.entitle_integration.existing.id
}
```

## Notes and Best Practices

### connection_json Security

- The `connection_json` value typically contains sensitive credentials (API tokens, secrets, passwords)
- Mark the `connection_json` variable as `sensitive = true` in your variable definitions
- Use a secrets manager (AWS Secrets Manager, HashiCorp Vault) to inject credentials at apply time rather than hardcoding them
- Use `jsonencode()` to construct the JSON safely from variables

### Workflow Hierarchy

- The integration-level workflow is the default for all resources and roles under it
- Resource-level workflows override the integration workflow for a specific resource
- Role-level workflows override both the resource and integration workflows for a specific role
- Assign the integration workflow to your most common approval pattern, and override at lower levels only when needed

### allow_creating_accounts

- Set `allow_creating_accounts = false` for systems where user accounts are managed externally (e.g., SSO-provisioned apps) to prevent Entitle from creating duplicate accounts
- Keep `allow_creating_accounts = true` for applications where Entitle should fully manage account lifecycle

### readonly Mode

- Use `readonly = true` for legacy or sensitive systems where automatic permission grants are not safe
- In readonly mode, access requests still go through the approval workflow — but instead of automatically provisioning access, Entitle creates a manual ticket for your IT/ops team to fulfill
