Retrieve a list of Entitle Resources within a specific integration. Use this data source to discover available resources, search by name, or dynamically look up resource IDs for use when creating roles, building bundles, or constructing policies.

Resources are the middle tier of the Entitle access model (Integration → Resource → Role) — they represent specific assets or systems within an application, such as individual AWS accounts, GitHub repositories, or database schemas. [Read more about resources](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **integration_id**: The mandatory filter — resources are always scoped to a specific integration
- **search**: Optional text filter to narrow results by resource name
- **Pagination**: Use `page` and `per_page` to navigate large result sets
- **Difference from entitle_resource**: Use `entitle_resources` (plural) to search and list; use `entitle_resource` (singular) to retrieve full details for a specific resource

## When to Use This Data Source

- Discovering all resources available within an integration
- Searching for a specific resource by name within a known integration
- Dynamically finding resource IDs to pass to `entitle_role` or `entitle_roles`
- Auditing what resources exist under an integration

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

### Required

- `integration_id` (String) The unique identifier of the integration to list resources for (UUID format). **This filter is mandatory** — resources are always scoped to a specific integration.

### Optional

- `filter` (Block) Optional filters:
    - `search` (String) Text search to filter resources by name.
    - `page` (Number) Page number to return, starting from 1.
    - `per_page` (Number) Number of results per page.

## Returned Attributes

- `resources` (Attributes List) The list of resources matching the query:
    - `id` (String) The resource's unique identifier.
    - `name` (String) The resource's display name.
    - `integration` (Attributes) The parent integration:
        - `id` (String) The integration's unique identifier.
        - `name` (String) The integration's display name.

## Notes

- Use `entitle_resource` (singular) to retrieve full details for a specific resource — `entitle_resources` (plural) returns only `id`, `name`, and `integration` per resource
- `integration_id` is mandatory — you cannot query resources across all integrations at once
- If `search` matches many results, use pagination to retrieve all matches
