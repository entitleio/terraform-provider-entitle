Applies the same settings to many Entitle roles at once using the asynchronous bulk API (`PATCH /public/v1/roles/bulk`). Use it to set the workflow or requestability of a group of roles — for example every role returned by an `entitle_roles` data source — without managing each one individually. It works the same way as [`entitle_resource_settings`](resource_settings.md) and [`entitle_integration_settings`](integration_settings.md); roles have no owner or maintainers.

## Key Concepts

- **Only configured attributes are managed**: Only the attributes set in the configuration are sent. Everything else on the roles is left untouched. At least one of `requestable` or `workflow_id` must be set.
- **Incremental updates**: Roles added to `role_ids` receive every configured setting. Roles that stay only receive the settings that changed.
- **Removing an attribute stops managing it**: If an attribute is removed from the configuration, nothing is sent and its current value stays as it is.
- **Bulk & asynchronous**: IDs are split into chunks of `bulk.chunk_size` (provider setting). Each chunk is submitted as a bulk job and the provider waits until the job reports `completed`.
- **All-or-nothing per chunk**: If any ID in a chunk does not exist or cannot be edited — or if the workflow cannot be found — the API rejects the whole chunk, nothing in it is applied, and the error lists the offending IDs.
- **One bulk operation per tenant**: Entitle runs bulk operations one at a time per tenant (across roles, resources and integrations). While another one is queued or running, the provider waits and retries automatically. Because of this, `bulk.parallelism` above `1` does not make bulk jobs run concurrently; it does speed up the batched reads described below.

## Removing Roles and Destroy

- **Removing an ID**: Removing an ID from `role_ids` leaves that role's settings as they are.
- **Destroy**: Destroying this resource (or removing the block) only removes it from Terraform state. All settings stay as they are in Entitle, the same as `entitle_role_synced`.

## Drift Detection

- **Every refresh reads every role in batches**: Roles are read with the list endpoint, 100 IDs per request, with up to `bulk.parallelism` requests at a time.
- **Changed settings are re-applied**: A role whose configured settings differ is shown as a change in the next plan, and apply sends every configured setting to that role only.
- **Unmanaged attributes are ignored**: Changes to attributes not set in the configuration are not drift.
- **Deleted roles**: A role that no longer exists is dropped from state. If its ID is still in `role_ids`, the next apply fails with a 404 listing that ID; remove it from the configuration.

## Error Handling

- **Failed create**: Nothing is saved to state, so the next apply creates the resource again. Settings that were already applied are simply applied again.
- **Failed update**: State keeps every role the update may have touched, so the next refresh detects what was applied and the next apply finishes the change.
- **Timeout**: Each bulk operation, including waiting for other bulk operations to finish, times out after 60 minutes.

~> **Note:** Do not manage the same setting on the same role from both `entitle_role_settings` and `entitle_role` / `entitle_role_synced`, or from two `entitle_role_settings` resources — they will overwrite each other.

## Example Usage

### Make Every Role of a Resource Non-Requestable

```terraform
data "entitle_roles" "repo" {
  resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

resource "entitle_role_settings" "frozen" {
  role_ids    = data.entitle_roles.repo.roles[*].id
  requestable = false
}
```

### Set the Workflow of an Integration's Admin Roles

```terraform
data "entitle_roles" "admins" {
  integration_id = data.entitle_integration.aws.id
  filter {
    search = "admin"
  }
}

resource "entitle_role_settings" "admins" {
  role_ids    = data.entitle_roles.admins.roles[*].id
  workflow_id = data.entitle_workflow.two_step_approval.id
}
```

### Explicit List of Roles

```terraform
resource "entitle_role_settings" "selected" {
  role_ids = [
    "7d080bfa-9143-11ee-b9d1-0242ac120002",
    "7d080bfa-9143-11ee-b9d1-0242ac120003",
  ]
  requestable = true
  workflow_id = "7d080bfa-9143-11ee-b9d1-0242ac120004"
}
```

## Notes

- `allowed_durations` is not supported yet, because the bulk API does not accept it
- `entitle_roles` returns a single page of results (at most 100); for more roles, read several pages and `flatten` their IDs
- `role_ids` must contain at least one UUID
- Import is not supported — declare the resource and apply
- Changing any attribute never replaces the resource; changes are applied in place
