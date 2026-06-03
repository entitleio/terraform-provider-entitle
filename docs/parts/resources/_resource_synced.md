An Entitle Synced Resource allows Terraform to manage the settings of a resource that is **synchronized from an external integration** — one whose lifecycle is controlled by the integration, not by Entitle or Terraform. Unlike [`entitle_resource`](resource.md), this resource does **not** create or delete the underlying resource; it only reads and updates its configuration.

On first apply, Terraform performs a lookup by `name` and `integration.id`, validates that the resource is a synced entity (not an Entitle-created resource), and imports it into state. All other fields (`workflow`, `allowed_durations`, `requestable`, `owner`, `maintainers`, `prerequisite_permissions`, etc.) are read from the API and stored in state. You can then override any of them in subsequent applies. [Read more about resources](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **Synced Resource**: A resource originating from an external integration — its existence is managed by the integration, not by Terraform
- **Name + Integration lookup**: Terraform finds the resource by matching `name` and `integration.id`; the resource must already exist and must be a synced entity
- **Synced validation**: If the matched resource belongs to a manual or virtual integration, the provider returns an error — use `entitle_resource` for those
- **No-op delete**: Destroying this resource removes it from Terraform state only; no DELETE request is sent to Entitle
- **Computed fields**: `workflow`, `allowed_durations`, `requestable`, `owner`, `maintainers`, and `prerequisite_permissions` are all optional — if not specified they are read from the existing resource and tracked in state

## entitle_resource_synced vs entitle_resource

| | `entitle_resource` | `entitle_resource_synced` |
|---|---|---|
| Use for | Manual integrations, virtual apps | External integration syncs (GCP, AWS, GitHub, Okta…) |
| Creates the resource | ✅ | ❌ (must already exist) |
| Deletes the resource on destroy | ✅ | ❌ (state-only removal) |
| Manages workflow, owner, durations, etc. | ✅ | ✅ |
| Validates entity is synced | ❌ | ✅ (errors if resource is Entitle-managed) |

## Example Usage

### Adopt an Existing Resource and Set a Workflow

Look up a resource that already exists and assign a workflow and owner to it:

```terraform
resource "entitle_resource_synced" "gcp_data_platform" {
  name = "GCP - data-platform-prod"

  integration = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  owner = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120003"
  }
}
```

### Adopt a Resource and Override Allowed Durations

Keep all current settings but restrict access durations:

```terraform
resource "entitle_resource_synced" "aws_prod_account" {
  name = "AWS Production Account 123456789"

  integration = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  allowed_durations = [3600, 28800, 86400]
}
```

### Adopt a Resource and Make It Non-Requestable

Prevent a synced resource from appearing in the self-service catalog:

```terraform
resource "entitle_resource_synced" "internal_service" {
  name        = "Internal Service Account"
  requestable = false

  integration = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }
}
```

### Full Adoption with All Settings

Adopt an existing resource and fully configure its Entitle settings:

```terraform
data "entitle_integration" "gcp" {
  name = "GCP Production"
}

data "entitle_user" "platform_lead" {
  email = "platform-lead@example.com"
}

data "entitle_workflow" "manager_approval" {
  name = "Manager Approval"
}

resource "entitle_resource_synced" "gcp_prod" {
  name        = "GCP - production-project"
  requestable = true

  integration = {
    id = data.entitle_integration.gcp.id
  }

  workflow = {
    id = data.entitle_workflow.manager_approval.id
  }

  owner = {
    id = data.entitle_user.platform_lead.id
  }

  allowed_durations = [3600, 28800, 86400]

  user_defined_tags = ["gcp", "production", "critical"]
}
```

### Adopt Multiple Synced Resources Using for_each

Manage all synced resources for an integration consistently:

```terraform
data "entitle_resources" "gcp_all" {
  integration_id = data.entitle_integration.gcp.id
}

resource "entitle_resource_synced" "gcp" {
  for_each = { for r in data.entitle_resources.gcp_all.resources : r.name => r }

  name = each.value.name

  integration = {
    id = data.entitle_integration.gcp.id
  }

  workflow = {
    id = data.entitle_workflow.manager_approval.id
  }

  owner = {
    id = data.entitle_user.platform_lead.id
  }

  allowed_durations = [3600, 28800, 86400]
}
```

## Import

Synced resources can be imported using the resource's UUID:

```shell
terraform import entitle_resource_synced.example a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

## Notes and Best Practices

### Resource Must Exist and Be Synced Before Apply

The resource identified by `name` + `integration.id` must already exist in Entitle **and** must belong to a synced integration (not manual or virtual). If the resource doesn't exist, or belongs to a manual/virtual integration, the provider returns an error. Use `entitle_resource` if you need Terraform to create the resource.

### Destroy Does Not Delete the Resource

Running `terraform destroy` (or removing this resource from your configuration) only removes it from Terraform state. The underlying resource in Entitle is left untouched. This is intentional — connector-synced resources are managed by the integration, not by Terraform.

### Computed Fields Track API State

Fields not specified in your configuration (`workflow`, `allowed_durations`, `requestable`, `owner`, `maintainers`, `prerequisite_permissions`) are populated from the API on first apply and tracked in state. If they change outside Terraform (e.g., someone edits them in the UI), `terraform plan` will show a diff and the next `apply` will restore the Terraform-managed values.

### Name Must Be Unique Within an Integration

The lookup is performed by exact `name` match within the given `integration.id`. If multiple resources share the same name under one integration, the provider returns the first match. Ensure resource names are unique within an integration to avoid ambiguity.
