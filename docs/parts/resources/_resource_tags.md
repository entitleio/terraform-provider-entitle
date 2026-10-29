Adds a set of tags to many Entitle resources at once using the asynchronous bulk API (`PATCH /public/v1/resources/bulk`). Use it to tag large groups of resources — for example every resource returned by an `entitle_resources` data source — without managing each one individually.

## Key Concepts

- **Additive**: Only the tags listed in `tags` are managed. Other tags on the resources are left untouched.
- **Bulk & asynchronous**: IDs are split into chunks of `bulk.chunk_size` (provider setting). Each chunk is submitted as a bulk job and the provider waits until the job reports `completed`.
- **All-or-nothing per chunk**: If any ID in a chunk does not exist, belongs to an internal Entitle integration, or cannot be edited, the API rejects the whole chunk, nothing in it is applied, and the error lists the offending IDs.
- **One bulk operation per tenant**: Entitle runs bulk operations one at a time per tenant (across roles, resources and integrations). While another one is queued or running, the provider waits and retries automatically. Because of this, `bulk.parallelism` above `1` does not make bulk jobs run concurrently; it does speed up the per-resource reads described below.
- **Incremental updates**: Changing `resource_ids` or `tags` only sends the difference.

## Tag Ownership

The provider only removes tags it added itself.

- **Pre-existing tags are never removed**: Before adding tags — on create, and when IDs or tags are added later — the provider reads each affected resource and records which of the tags it already had. Those tags are never removed by this resource. The record is kept in the provider's private state, so it does not appear in plans.
- **Removing an ID**: Removing an ID from `resource_ids` removes the tags Terraform added to that resource only.
- **Removing a tag**: Removing a tag from `tags` removes it from the resources where Terraform added it, and keeps it where it was pre-existing.
- **Destroy**: Destroying this resource (or removing the block) removes the tags Terraform added from all resources. Resources that no longer exist are skipped.
- **Re-added tags become managed**: If a pre-existing tag is removed outside Terraform and Terraform re-adds it, Terraform manages it from then on.
- **Stop managing without removing**: To keep the tags in Entitle, use `terraform state rm` or a `removed` block with `lifecycle { destroy = false }` (Terraform 1.7+).

## Drift Detection

- **Every refresh reads every resource**: One API request per resource, with up to `bulk.parallelism` requests at a time. With thousands of resources this makes `terraform plan` noticeably slower.
- **Tag matching**: A tag counts as present if it is among the resource's user-defined or integration tags. Matching ignores case and surrounding whitespace.
- **Missing tags are re-added**: A resource missing any of the tags is shown as a change in the next plan, and apply re-adds the tags to that resource only.
- **Extra tags are ignored**: Tags added outside Terraform do not cause drift.
- **Deleted resources**: A resource that no longer exists is dropped from state. If its ID is still in `resource_ids`, the next apply fails with a 404 listing that ID; remove it from the configuration.

## Error Handling

- **Failed create**: The resource is marked as tainted. The next apply removes the tags it added and creates it again.
- **Failed update**: State keeps every resource and tag the update may have touched, so the next refresh detects what was applied and the next apply finishes the change.
- **Timeout**: Each bulk operation, including waiting for other bulk operations to finish, times out after 60 minutes.

~> **Note:** Do not manage the same tag on the same resource from both `entitle_resource_tags` and the `user_defined_tags` attribute of `entitle_resource` / `entitle_resource_synced` — they will overwrite each other. Two `entitle_resource_tags` resources may share a tag: the one created later treats it as pre-existing. If the one that added it is removed, the tag is removed too, and the other one re-adds it on its next apply.

-> **Note:** If a resource loses a tag outside Terraform and is removed from `resource_ids` in the same change, Terraform plans no change for it. Its tags are cleaned up by the next apply that changes this resource, or on destroy.

## Example Usage

### Tag All Resources of an Integration

```terraform
data "entitle_resources" "aws_policies" {
  integration_id = data.entitle_integration.aws.id
}

resource "entitle_resource_tags" "pci" {
  resource_ids = data.entitle_resources.aws_policies.resources[*].id
  tags         = ["pci", "sox"]
}
```

### Tag More Than One Page of Resources

`entitle_resources` returns a single page of results (at most 100). Read several pages and flatten their IDs:

```terraform
data "entitle_resources" "aws_policies" {
  count          = 5 # covers up to 500 resources
  integration_id = data.entitle_integration.aws.id

  filter {
    page     = count.index + 1
    per_page = 100
  }
}

resource "entitle_resource_tags" "pci" {
  resource_ids = flatten([for page in data.entitle_resources.aws_policies : page.resources[*].id])
  tags         = ["pci", "sox"]
}
```

### Tag an Explicit List of Resources

```terraform
resource "entitle_resource_tags" "payments" {
  resource_ids = [
    "7d080bfa-9143-11ee-b9d1-0242ac120001",
    "7d080bfa-9143-11ee-b9d1-0242ac120002",
  ]
  tags = ["payments"]
}
```

### Combine Several Sources

```terraform
data "entitle_resources" "prod" {
  integration_id = data.entitle_integration.aws.id
  filter { search = "prod" }
}

data "entitle_resources" "billing" {
  integration_id = data.entitle_integration.gcp.id
  filter { search = "billing" }
}

resource "entitle_resource_tags" "critical" {
  resource_ids = concat(data.entitle_resources.prod.resources[*].id, data.entitle_resources.billing.resources[*].id)
  tags         = ["critical"]
}
```

### Stop Managing Tags Without Removing Them

```terraform
removed {
  from = entitle_resource_tags.pci

  lifecycle {
    destroy = false
  }
}
```

### Tune Bulk Limits

Chunk size and concurrency are configured once for the whole provider:

```terraform
provider "entitle" {
  bulk {
    chunk_size  = 500
    parallelism = 2
  }
}
```

## Notes

- Both `resource_ids` and `tags` must contain at least one element; every ID must be a UUID
- Import is not supported — declare the resource and apply; tags that already exist are recorded as pre-existing
- Changing `resource_ids` or `tags` never replaces the resource; changes are applied in place
