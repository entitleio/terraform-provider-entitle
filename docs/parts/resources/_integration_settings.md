Applies the same settings to many Entitle integrations at once using the asynchronous bulk API (`PATCH /public/v1/integrations/bulk`). Use it to set the workflow, owner, maintainers or requestability of a group of integrations — for example every integration returned by an `entitle_integrations` data source — without managing each one individually. It works the same way as [`entitle_resource_settings`](resource_settings.md).

## Key Concepts

- **Only configured attributes are managed**: Only the attributes set in the configuration are sent. Everything else on the integrations is left untouched. At least one of `requestable`, `workflow_id`, `owner`, `maintainers` or `additional_maintainers` must be set.
- **Incremental updates**: Integrations added to `integration_ids` receive every configured setting. Integrations that stay only receive the settings that changed.
- **Removing an attribute stops managing it**: If an attribute is removed from the configuration, nothing is sent and its current value stays as it is.
- **Bulk & asynchronous**: IDs are split into chunks of `bulk.chunk_size` (provider setting). Each chunk is submitted as a bulk job and the provider waits until the job reports `completed`.
- **All-or-nothing per chunk**: If any ID in a chunk does not exist or cannot be edited — or if the workflow, owner or a maintainer cannot be found — the API rejects the whole chunk, nothing in it is applied, and the error lists the offending IDs.
- **One bulk operation per tenant**: Entitle runs bulk operations one at a time per tenant (across roles, resources and integrations). While another one is queued or running, the provider waits and retries automatically. Because of this, `bulk.parallelism` above `1` does not make bulk jobs run concurrently; it does speed up the batched reads described below.

## Maintainers

There are two mutually exclusive ways to manage maintainers:

- **`maintainers` (replace)**: The configured list becomes the exact maintainer list of every integration; existing maintainers not in the list are removed. An empty list (`maintainers = []`) removes all maintainers.
- **`additional_maintainers` (add)**: The configured maintainers are added and existing maintainers are kept. Before adding, the provider reads the affected integrations (in the same batches) and records which maintainers they already had. Taking a maintainer out of the list, or an integration out of `integration_ids`, removes only the maintainers this resource added. The record is kept in the provider's private state, so it does not appear in plans.

Maintainers, like the owner, can be given as a UUID or an email address.

Switching from `additional_maintainers` to `maintainers` replaces the whole list, so the maintainers added earlier are removed unless they are in the new list. Switching from `maintainers` to `additional_maintainers` keeps the current list and adds to it; the maintainers set by the earlier replace count as pre-existing and are never removed.

## Removing Integrations and Destroy

- **Removing an ID**: Removing an ID from `integration_ids` leaves that integration's settings as they are. Only additional maintainers this resource added are removed from it.
- **Destroy**: Destroying this resource (or removing the block) only removes it from Terraform state. All settings and maintainers — including additional maintainers — stay as they are in Entitle, the same as `entitle_role_synced`.

## Drift Detection

- **Every refresh reads every integration in batches**: Integrations are read with the list endpoint, 100 IDs per request, with up to `bulk.parallelism` requests at a time.
- **Changed settings are re-applied**: An integration whose configured settings differ — or that is missing one of the `additional_maintainers` — is shown as a change in the next plan, and apply sends every configured setting to that integration only.
- **Matching**: The owner and maintainers match by UUID or email, ignoring case. With `maintainers`, any extra maintainer is drift; with `additional_maintainers`, extra maintainers are ignored.
- **Unmanaged attributes are ignored**: Changes to attributes not set in the configuration are not drift.
- **Deleted integrations**: An integration that no longer exists is dropped from state. If its ID is still in `integration_ids`, the next apply fails with a 404 listing that ID; remove it from the configuration.

## Error Handling

- **Failed create**: Nothing is saved to state, so the next apply creates the resource again. Additional maintainers added before the failure are removed first. Settings that were already applied are simply applied again. If that clean-up fails, a warning is shown and those maintainers are treated as pre-existing on the next apply.
- **Failed update**: State keeps every integration and additional maintainer the update may have touched, so the next refresh detects what was applied and the next apply finishes the change.
- **Timeout**: Each bulk operation, including waiting for other bulk operations to finish, times out after 60 minutes.

~> **Note:** Do not manage the same setting on the same integration from both `entitle_integration_settings` and `entitle_integration`, or from two `entitle_integration_settings` resources — they will overwrite each other.

-> **Note:** If an integration drifts and is removed from `integration_ids` in the same change, Terraform plans no change for it. Additional maintainers added to it are removed by the next apply that changes this resource.

## Example Usage

### Set Workflow and Owner for All Integrations

```terraform
data "entitle_integrations" "all" {}

resource "entitle_integration_settings" "all" {
  integration_ids = data.entitle_integrations.all.integrations[*].id
  workflow_id     = data.entitle_workflow.manager_approval.id
  owner           = "platform-owner@example.com"
}
```

### Replace Maintainers

```terraform
resource "entitle_integration_settings" "maintainers" {
  integration_ids = [
    "7d080bfa-9143-11ee-b9d1-0242ac120001",
    "7d080bfa-9143-11ee-b9d1-0242ac120002",
  ]

  maintainers = [
    { type = "user", id = "alice@example.com" },
    { type = "group", id = "7d080bfa-9143-11ee-b9d1-0242ac120003" },
  ]
}
```

### Add Maintainers Without Removing Existing Ones

```terraform
resource "entitle_integration_settings" "on_call" {
  integration_ids = data.entitle_integrations.all.integrations[*].id

  additional_maintainers = [
    { type = "group", id = "on-call@example.com" },
  ]
}
```

### Make Every Integration Except One Non-Requestable

```terraform
data "entitle_integrations" "others" {
  filter {
    ids          = ["7d080bfa-9143-11ee-b9d1-0242ac120001"]
    ids_operator = "isNot"
  }
}

resource "entitle_integration_settings" "frozen" {
  integration_ids = data.entitle_integrations.others.integrations[*].id
  requestable     = false
}
```

## Notes

- `entitle_integrations` returns a single page of results (at most 100); for more integrations, read several pages and `flatten` their IDs
- `integration_ids` must contain at least one UUID; `additional_maintainers`, when set, at least one maintainer
- Import is not supported — declare the resource and apply
- Changing any attribute never replaces the resource; changes are applied in place
