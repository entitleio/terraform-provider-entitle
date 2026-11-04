Retrieve every Entitle Resource matching a filter, within one integration or across all of them. Use this data source to discover available resources, search by name, or dynamically look up resource IDs for use when creating roles, building bundles, or constructing policies.

Resources are the middle tier of the Entitle access model (Integration → Resource → Role) — they represent specific assets or systems within an application, such as individual AWS accounts, GitHub repositories, or database schemas. [Read more about resources](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## Key Concepts

- **Scope**: `filter.integration_id` lists the resources of one integration; without it, resources of every integration are returned. The top-level `integration_id` is deprecated; move it into `filter`
- **All results**: The data source always returns every matching resource. The provider pages through the API with a cursor, so no resource is skipped or repeated while the list changes. `filter.page` and `filter.per_page` are deprecated and ignored
- **ids**: `filter.ids` returns specific resources by ID (up to 100); with `ids_operator = "isNot"` it returns every resource except them
- **Filters**: `requestable`, `workflow_ids`, `owner_ids`, `maintainer_ids`, `types` and `tags` narrow the results; all filters are combined with AND
- **search**: Matches resource names case-insensitively and partially, and can be combined with every other filter. Results are ranked: exact matches first, then case-insensitive matches, then partial matches
- **Difference from entitle_resource**: Use `entitle_resources` (plural) to search and list; use `entitle_resource` (singular) to retrieve full details for a specific resource

## When to Use This Data Source

- Discovering all resources available within an integration
- Searching for a specific resource by name within a known integration
- Finding resources by owner, maintainer, workflow, type or tag
- Dynamically finding resource IDs to pass to `entitle_role` or `entitle_roles`
- Selecting a group of resources to tag or configure in bulk with `entitle_resource_tags` or `entitle_resource_settings`

## Example Usage

### List All Resources for an Integration

```terraform
data "entitle_resources" "all" {
  filter {
    integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }
}

output "resource_names" {
  value = data.entitle_resources.all.resources[*].name
}
```

### Search for a Resource by Name

```terraform
data "entitle_resources" "prod_resources" {
  filter {
    integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
    search         = "production"
  }
}

output "prod_resource_id" {
  value = data.entitle_resources.prod_resources.resources[0].id
}
```

### Filter by Owner, Type and Tags

```terraform
data "entitle_resources" "owned_prod_policies" {
  filter {
    owner_ids      = ["7d080bfa-9143-11ee-b9d1-0242ac120005"]
    types          = ["Policy"]
    types_operator = "contains"
    tags           = ["env: prod", "pci"]
    tags_relation  = "and"
  }
}
```

### Resources Not Maintained by a Group

```terraform
data "entitle_resources" "not_platform" {
  filter {
    integration_id          = "7d080bfa-9143-11ee-b9d1-0242ac120001"
    maintainer_ids          = ["7d080bfa-9143-11ee-b9d1-0242ac120006"]
    maintainer_ids_operator = "notContains"
  }
}
```

### Use Resource IDs When Creating Roles

```terraform
data "entitle_integration" "github" {
  name = "GitHub Production"
}

data "entitle_resources" "repos" {
  filter {
    integration_id = data.entitle_integration.github.id
    search         = "backend-api"
  }
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
  filter {
    integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
    search         = "production"
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

```terraform
data "entitle_integration" "aws" {
  name = "AWS Production"
}

data "entitle_resources" "aws_resources" {
  filter {
    integration_id = data.entitle_integration.aws.id
  }
}

# Reference the first resource to list its roles
data "entitle_roles" "first_resource_roles" {
  filter {
    resource_id = data.entitle_resources.aws_resources.resources[0].id
  }
}

output "roles_in_first_resource" {
  value = data.entitle_roles.first_resource_roles.roles
}
```

## Notes

- Use `entitle_resource` (singular) to retrieve full details for a specific resource — `entitle_resources` (plural) returns only `id`, `external_id`, `name`, and `integration` per resource
- Without `filter.integration_id` (and without other filters), every resource you can read is fetched, which can take a while on large tenants
- `filter.search` is applied by the provider after the other filters, so it never limits which filters can be combined
