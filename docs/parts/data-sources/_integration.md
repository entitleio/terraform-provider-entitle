An Entitle Integration represents a configured connection to an external application or system. It is the top-level container in the Entitle access model — each integration contains resources, which contain roles (Integration → Resource → Role). Integrations include all the configuration Entitle needs to read permissions, manage access, and handle approval requests for a connected application.

Use this data source to look up an existing integration by ID or name, for example to find its ID when creating resources or roles, or to inspect its current configuration. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **Integration**: A named connection to a specific instance of an application (e.g., a specific AWS account, GitHub organization)
- **Application**: The type of system connected (e.g., `"aws"`, `"github"`, `"slack"`)
- **Workflow**: The default approval process for all resources and roles under this integration
- **Owner**: The primary responsible user for this integration
- **Maintainers**: Secondary owners who assist with administrative tasks
- **ID vs Name lookup**: Use `id` for performance-critical queries; use `name` for human-friendly lookups

## When to Use This Data Source

- Retrieving an integration's ID to reference when creating resources (`entitle_resource`)
- Looking up integration details for use in outputs or other resource configurations
- Finding available integrations before creating associated resources or roles

## Example Usage

### Look Up an Integration by ID

```terraform
data "entitle_integration" "aws_prod" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "integration_name" {
  value = data.entitle_integration.aws_prod.name
}
```

### Look Up an Integration by Name

```terraform
data "entitle_integration" "github" {
  name = "GitHub Production"
}

output "github_workflow_id" {
  value = data.entitle_integration.github.workflow.id
}
```

### Use Integration ID When Creating Resources

```terraform
data "entitle_integration" "slack" {
  name = "Slack - Engineering Workspace"
}

resource "entitle_resource" "slack_channel" {
  name        = "Slack - #engineering"
  requestable = true

  integration = {
    id = data.entitle_integration.slack.id
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  }

  allowed_durations = [28800, 86400]
}
```

### Use Integration ID to List Its Resources

```terraform
data "entitle_integration" "postgres" {
  name = "Internal PostgreSQL"
}

data "entitle_resources" "db_resources" {
  integration_id = data.entitle_integration.postgres.id
}

output "available_databases" {
  value = data.entitle_resources.db_resources.resources[*].name
}
```

### Inspect Integration Configuration

```terraform
data "entitle_integration" "existing" {
  name = "AWS Development"
}

output "integration_details" {
  value = {
    id                       = data.entitle_integration.existing.id
    application              = data.entitle_integration.existing.application.name
    requestable              = data.entitle_integration.existing.requestable
    allow_creating_accounts  = data.entitle_integration.existing.allow_creating_accounts
    readonly                 = data.entitle_integration.existing.readonly
    allowed_durations        = data.entitle_integration.existing.allowed_durations
  }
}
```

## Query Parameters

### Optional (at least one should be provided)

- `id` (String) The unique identifier of the integration (UUID format). Preferred for performance-critical queries.
- `name` (String) The integration's display name. When querying by name, the provider paginates through all integrations until a match is found — this may be slower in large organizations.

## Returned Attributes

- `id` (String) The integration's unique identifier (UUID format).
- `application` (Attributes) The connected application:
    - `name` (String) The application's type name (e.g., `"aws"`, `"github"`, `"slack"`).
- `workflow` (Attributes) The default approval workflow:
    - `id` (String) The workflow's unique identifier.
    - `name` (String) The workflow's display name.
- `allowed_durations` (Set of Number) Available access duration options in seconds.
- `allow_changing_account_permissions` (Boolean) Whether Entitle can modify permissions in the application.
- `allow_creating_accounts` (Boolean) Whether Entitle can create new accounts in the application.
- `auto_assign_recommended_maintainers` (Boolean) Whether recommended maintainers are auto-assigned.
- `auto_assign_recommended_owners` (Boolean) Whether recommended owners are auto-assigned.
- `notify_about_external_permission_changes` (Boolean) Whether external permission changes trigger notifications.
- `readonly` (Boolean) Whether the integration operates in read-only/ticket mode.
- `requestable` (Boolean) Whether JIT access requests are enabled.
- `requestable_by_default` (Boolean) Whether resources are requestable by default.
- `maintainers` (Attributes List) Secondary owners:
    - `type` (String) `"user"` or `"group"`.
    - `entity` (Attributes):
        - `id` (String) The maintainer's unique identifier.
        - `email` (String) The maintainer's email address.
