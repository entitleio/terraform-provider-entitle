Retrieve a list of Entitle Resources within a specific integration. Use this data source to discover available resources, search by name, or dynamically look up resource IDs for use when creating roles, building bundles, or constructing policies.

Resources are the middle tier of the Entitle access model (Integration → Resource → Role) — they represent specific assets or systems within an application, such as individual AWS accounts, GitHub repositories, or database schemas. [Read more about resources](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **integration_id**: Lists the resources of one integration. Required unless `filter.ids` is set
- **ids**: `filter.ids` returns specific resources by ID (up to 100), across integrations; with `ids_operator = "isNot"` it returns every resource except them
- **after_id**: A cursor for reading a long list page by page without skipping or repeating items while it changes
- **search**: Optional text filter to narrow results by resource name
- **Pagination**: Use `page` and `per_page` to navigate large result sets (at most 100 results per page)
- **Difference from entitle_resource**: Use `entitle_resources` (plural) to search and list; use `entitle_resource` (singular) to retrieve full details for a specific resource

## When to Use This Data Source

- Discovering all resources available within an integration
- Searching for a specific resource by name within a known integration
- Dynamically finding resource IDs to pass to `entitle_role` or `entitle_roles`
- Auditing what resources exist under an integration
- Selecting a group of resources to tag or configure in bulk with `entitle_resource_tags` or `entitle_resource_settings`

## Example Usage

### List All Resources for an Integration

```terraform
data "entitle_resources" "all" {
  integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "resource_names" {
  value = data.entitle_resources.all.resources[*].name
}
```

### Search for a Resource by Name

```terraform
data "entitle_resources" "prod_resources" {
  integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  filter {
    search = "production"
  }
}

output "prod_resource_id" {
  value = data.entitle_resources.prod_resources.resources[0].id
}
```

### Use Resource IDs When Creating Roles

```terraform
data "entitle_integration" "github" {
  name = "GitHub Production"
}

data "entitle_resources" "repos" {
  integration_id = data.entitle_integration.github.id
  filter { search = "backend-api" }
}

resource "entitle_role" "backend_read" {
  name        = "Backend API Read"
  requestable = true

  resource = {
    id = data.entitle_resources.repos.resources[0].id
  }

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120002"
  }

  allowed_durations = [28800, 86400]
}
```

### Tag Every Matching Resource

```terraform
data "entitle_resources" "prod" {
  integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  filter {
    search = "production"
  }
}

resource "entitle_resource_tags" "prod" {
  resource_ids = data.entitle_resources.prod.resources[*].id
  tags         = ["production"]
}
```

### Look Up Specific Resources by ID

`filter.ids` returns up to 100 resources by ID, from any integration:

```terraform
data "entitle_resources" "selected" {
  filter {
    ids = [
      "7d080bfa-9143-11ee-b9d1-0242ac120001",
      "7d080bfa-9143-11ee-b9d1-0242ac120002",
    ]
  }
}
```

### List All Resources and Find Roles for Each

Discover resources and then list roles within each:

```terraform
data "entitle_integration" "aws" {
  name = "AWS Production"
}

data "entitle_resources" "aws_resources" {
  integration_id = data.entitle_integration.aws.id
}

# Reference the first resource to list its roles
data "entitle_roles" "first_resource_roles" {
  resource_id = data.entitle_resources.aws_resources.resources[0].id
}

output "roles_in_first_resource" {
  value = data.entitle_roles.first_resource_roles.roles
}
```

### Paginate Through Large Resource Lists

```terraform
data "entitle_resources" "page_one" {
  integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  filter {
    page     = 1
    per_page = 50
  }
}
```

## Query Parameters

### Optional

- `integration_id` (String) The unique identifier of the integration to list resources for (UUID format). Required unless `filter.ids` is set.
- `filter` (Block) Optional filters:
    - `ids` (List of String) Resource IDs to return, up to 100. Does not need `integration_id`.
    - `ids_operator` (String) `is` (default) returns the given resources; `isNot` returns every resource except them. Requires `ids`.
    - `after_id` (String) Cursor: return the resources after this ID (results are sorted by ID). Cannot be combined with a `page` other than 1 or with `search`.
    - `search` (String) Text search to filter resources by name. Cannot be combined with `ids` unless `external_id` is set.
    - `external_id` (String) Filter by the resource's external ID.
    - `page` (Number) Page number to return, starting from 1.
    - `per_page` (Number) Number of results per page (at most 100).

## Returned Attributes

- `resources` (Attributes List) The list of resources matching the query:
    - `id` (String) The resource's unique identifier.
    - `name` (String) The resource's display name.
    - `integration` (Attributes) The parent integration:
        - `id` (String) The integration's unique identifier.
        - `name` (String) The integration's display name.

## Notes

- Use `entitle_resource` (singular) to retrieve full details for a specific resource — `entitle_resources` (plural) returns only `id`, `name`, and `integration` per resource
- Without `filter.ids`, `integration_id` is required — resources cannot be listed across all integrations at once
- If `search` matches many results, use pagination to retrieve all matches
- `resources` contains only the requested page (at most 100 results); to collect more, read several pages (e.g. with `count`, or by passing the last ID as `after_id`) and `flatten` their `resources[*].id`
