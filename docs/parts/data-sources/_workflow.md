A workflow in Entitle defines the Just-In-Time (JIT) access approval process — specifying who must approve access requests, in what order, and under what conditions. Workflows are defined once and can be assigned to integrations, resources, roles, and bundles to control how access requests are handled across your organization.

Use this data source to look up an existing workflow by ID or name — for example, to reference it when creating integrations, resources, roles, or bundles without managing the workflow itself in Terraform. [Read more about workflows](https://docs.beyondtrust.com/entitle/docs/approval-workflows).

## Key Concepts

- **Workflow**: A reusable approval definition with one or more conditional rules
- **Rules**: Each rule applies when its conditions match (duration, group, schedule) and defines the approval flow
- **Approval Steps**: Sequential stages of approval within a rule
- **Approval Entities**: Who can approve at each step (users, groups, managers, automatic)
- **ID vs Name lookup**: Use `id` for performance-critical automation; use `name` for human-friendly lookups

## When to Use This Data Source

- Referencing an existing workflow when creating integrations, resources, roles, or bundles
- Looking up workflow IDs for use in other resources without managing the workflow via Terraform
- Inspecting a workflow's rules and approval configuration

## Example Usage

### Look Up a Workflow by ID

```terraform
data "entitle_workflow" "manager_approval" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "workflow_name" {
  value = data.entitle_workflow.manager_approval.name
}
```

### Look Up a Workflow by Name

```terraform
data "entitle_workflow" "auto_approve" {
  name = "Auto-Approve Development"
}

output "auto_approve_id" {
  value = data.entitle_workflow.auto_approve.id
}
```

### Reference a Workflow in an Integration

```terraform
data "entitle_workflow" "standard_approval" {
  name = "Manager Approval"
}

resource "entitle_integration" "github" {
  name            = "GitHub Production"
  connection_json = jsonencode({ token = var.github_token, organization = var.github_org })

  application = { name = "github" }
  owner       = { id = "7d080bfa-9143-11ee-b9d1-0242ac120001" }

  workflow = {
    id = data.entitle_workflow.standard_approval.id
  }

  allowed_durations       = [3600, 28800]
  allow_creating_accounts = true
}
```

### Reference a Workflow in a Role

```terraform
data "entitle_workflow" "security_approval" {
  name = "Security Team Approval"
}

resource "entitle_role" "prod_admin" {
  name        = "Production Admin"
  requestable = true

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = data.entitle_workflow.security_approval.id
  }

  allowed_durations = [3600, 7200]
}
```

### Reference a Workflow in a Bundle

```terraform
data "entitle_workflow" "auto_approve" {
  name = "Auto-Approve"
}

resource "entitle_bundle" "dev_tools" {
  name        = "Developer Tools"
  description = "Standard developer access package"

  workflow = {
    id = data.entitle_workflow.auto_approve.id
  }

  roles = [
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120001" },
    { id = "7d080bfa-9143-11ee-b9d1-0242ac120002" }
  ]

  allowed_durations = [28800, 86400]
}
```

### Inspect Workflow Rules

```terraform
data "entitle_workflow" "complex_workflow" {
  name = "Duration-Based Approval"
}

output "workflow_rules" {
  value = data.entitle_workflow.complex_workflow.rules
}

output "first_rule_steps" {
  value = data.entitle_workflow.complex_workflow.rules[0].approval_flow.steps
}
```

## Query Parameters

### Optional (at least one should be provided)

- `id` (String) The unique identifier of the workflow (UUID format). Preferred for performance-critical queries.
- `name` (String) The workflow's display name. When querying by name, the provider paginates through all workflows until a match is found — this may be slower in organizations with many workflows. For performance-critical automation, prefer `id`.

## Returned Attributes

- `id` (String) The workflow's unique identifier (UUID format).
- `name` (String) The workflow's display name.
- `rules` (Attributes List) The list of conditional approval rules. Rules are evaluated in order (by `sort_order`); the first matching rule determines the approval process:
    - `sort_order` (Number) The evaluation order (lower numbers evaluated first).
    - `under_duration` (Number) Maximum access duration in seconds for which this rule applies.
    - `any_schedule` (Boolean) Whether this rule applies at any time (`true`) or only during specific schedules (`false`).
    - `in_groups` (Attributes List) Groups for which this rule applies:
        - `id` (String) The group's unique identifier.
        - `name` (String) The group's display name.
    - `in_schedules` (Attributes List) Schedules during which this rule applies:
        - `id` (String) The schedule's unique identifier.
        - `name` (String) The schedule's display name.
    - `approval_flow` (Attributes) The approval process for requests matching this rule:
        - `steps` (Attributes List) Ordered approval steps:
            - `sort_order` (Number) The step execution order.
            - `operator` (String) `"or"` (any one approver) or `"and"` (all approvers must approve).
            - `approval_entities` (Attributes List) Entities who can approve:
                - `type` (String) The approver type (`"Automatic"`, `"Manager"`, `"ResourceOwner"`, `"Group"`, `"User"`, etc.)
                - `user` (Attributes) User details (when `type` is `"User"`).
                - `group` (Attributes) Group details (when `type` is `"Group"`).
                - `schedule` (Attributes) Schedule details.
                - `channel` (Attributes) Slack/Teams channel details.
                - `webhook` (Attributes) Webhook details.
            - `notified_entities` (Attributes List) Entities notified without needing to approve (same structure as `approval_entities`).

## Notes

- Prefer `id` over `name` for programmatic or CI/CD use cases
- If neither `id` nor `name` is provided, the query may fail or return unexpected results
- For creating and managing workflows, use the `entitle_workflow` resource — see its documentation for full examples, patterns, and best practices
