An Entitle Synced Role allows Terraform to manage the settings of a role that is **synchronized from an external integration** — one whose lifecycle is controlled by the integration, not by Entitle or Terraform. Unlike [`entitle_role`](role.md), this resource does **not** create or delete the underlying role; it only reads and updates its configuration.

On first apply, Terraform performs a lookup by `name` and `resource.id`, validates that the role is a synced entity (not an Entitle-created role), imports it into state, and immediately applies any fields specified in the configuration (e.g. `workflow`, `allowed_durations`, `requestable`, `prerequisite_permissions`). Fields not specified in the configuration are read from the API and stored as-is. [Read more about roles](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **Synced Role**: A role originating from an external integration — its existence is managed by the integration, not by Terraform
- **Name/External ID + Resource lookup**: Terraform finds the role by matching `name`/`external_id` and `resource.id`; the role must already exist and must be a synced entity
- **Synced validation**: If the matched role is Entitle-managed (not synced from an external system), the provider returns an error — use `entitle_role` for those
- **No-op delete**: Destroying this resource removes it from Terraform state only; no DELETE request is sent to Entitle
- **Immediate configuration apply**: On first apply, any fields specified in the configuration are compared against the existing role and updated if different — no need for a second apply
- **Computed fields**: `workflow`, `allowed_durations`, `requestable`, and `prerequisite_permissions` are all optional — if not specified they are read from the existing role and tracked in state

## entitle_role_synced vs entitle_role

| | `entitle_role` | `entitle_role_synced` |
|---|---|---|
| Use for | Manual integrations, virtual apps, Entitle-managed entities | External integration syncs |
| Creates the role | ✅ | ❌ (must already exist) |
| Deletes the role on destroy | ✅ | ❌ (state-only removal) |
| Manages workflow, durations, etc. | ✅ | ✅ |
| Validates entity is synced | ❌ | ✅ (errors if role is Entitle-managed) |

## Example Usage

### Adopt an Existing Role and Set a Workflow

Look up a role that already exists and assign a workflow to it:

```terraform
resource "entitle_role_synced" "db_admin" {
  name = "Admin"

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }
}
```

### Adopt a Role and Override Allowed Durations

Adopt a connector-synced role, keeping all its current settings but restricting access durations:

```terraform
resource "entitle_role_synced" "s3_read" {
  name = "S3 Read Only"

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  allowed_durations = [3600, 10800, 21600]
}
```

### Adopt a Role and Make It Non-Requestable

Prevent a connector-synced role from appearing in the self-service catalog:

```terraform
resource "entitle_role_synced" "internal_service_account" {
  name        = "Service Account"
  requestable = false

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }
}
```

### Full Adoption with All Settings

Adopt an existing role and fully configure its Entitle settings:

```terraform
resource "entitle_role_synced" "prod_write" {
  name        = "Production Write"
  requestable = true

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations = [3600, 10800]

  prerequisite_permissions = [
    {
      default = true
      role = {
        id = "7d080bfa-9143-11ee-b9d1-0242ac120003"  # prod_read role
      }
    }
  ]
}
```

### Adopt Multiple Connector-Synced Roles Using for_each

Manage all synced roles for a resource consistently:

```terraform
locals {
  synced_roles = {
    "Read Only"  = { workflow_id = "7d080bfa-9143-11ee-b9d1-0242ac120010", durations = [3600, 21600] }
    "Read Write" = { workflow_id = "7d080bfa-9143-11ee-b9d1-0242ac120011", durations = [3600, 10800] }
    "Admin"      = { workflow_id = "7d080bfa-9143-11ee-b9d1-0242ac120012", durations = [3600] }
  }
}

resource "entitle_role_synced" "app_roles" {
  for_each = local.synced_roles

  name = each.key

  resource = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  workflow = {
    id = each.value.workflow_id
  }

  allowed_durations = each.value.durations
}
```

## Import

Synced roles can be imported using the role's UUID:

```shell
terraform import entitle_role_synced.example a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

## Notes and Best Practices

### Role Must Exist and Be Synced Before Apply

The role identified by `name` + `resource.id` must already exist in Entitle **and** must be a role synchronized from an external integration. If the role doesn't exist, or if it was created directly in Entitle (not synced), the provider returns an error. Use `entitle_role` if you need Terraform to create the role, or if you're working with manual integrations or virtual applications.

### Destroy Does Not Delete the Role

Running `terraform destroy` (or removing this resource from your configuration) only removes it from Terraform state. The underlying role in Entitle is left untouched. This is intentional — connector-synced roles are managed by the integration, not by Terraform.

### Configuration Is Applied on First Apply

On first apply, the provider looks up the existing role, imports it into state, and immediately applies any fields you specified in the configuration — comparing them against the current API values and updating only where there is a diff. A second apply is not required to push your settings.

Fields not specified in your configuration (`workflow`, `allowed_durations`, `requestable`, `prerequisite_permissions`) are populated from the API as-is and tracked in state. If they change outside Terraform (e.g., someone edits them in the UI), `terraform plan` will show a diff and the next `apply` will restore the Terraform-managed values.

### Name Must Be Unique Within a Resource

The lookup is performed by exact `name`/`external_id` match within the given `resource.id`. Ensure role name is unique within an integration or use external id.
