Use this data source to look up an existing Entitle Workflow Webhook, either by its identifier or by name. This is useful when a webhook is managed outside Terraform (created in the Entitle UI or by another team) but you still need its `id` to reference it from an approval workflow step.

Exactly one of `id` or `name` must be provided. Lookups by name are case sensitive and fail if more than one webhook shares the same name — use `id` in that case.

## Example Usage

### Look Up by Name

```terraform
data "entitle_workflow_webhook" "approvals" {
  name = "approvals-bot"
}

output "approvals_webhook_url" {
  value = data.entitle_workflow_webhook.approvals.url
}
```

### Look Up by ID

```terraform
data "entitle_workflow_webhook" "approvals" {
  id = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

### Reference a Webhook From an Approval Workflow

```terraform
data "entitle_workflow_webhook" "approvals" {
  name = "approvals-bot"
}

resource "entitle_workflow" "sensitive_access" {
  name = "Sensitive Access"

  rules = [{
    sort_order = 1

    approval_flow = {
      steps = [{
        sort_order = 1
        operator   = "or"

        approval_entities = [{
          type = "Manager"
        }]

        notified_entities = [{
          type = "Webhook"
          webhook = {
            id = data.entitle_workflow_webhook.approvals.id
          }
        }]
      }]
    }

    in_groups    = []
    in_schedules = []
  }]
}
```

## Notes

- `headers` is marked sensitive and is redacted from plan and apply output, but it is written to Terraform state — protect your state backend accordingly
- Looking up by name lists all workflow webhooks in the organization and matches exactly one by name; prefer `id` in large organizations or when names are not unique
