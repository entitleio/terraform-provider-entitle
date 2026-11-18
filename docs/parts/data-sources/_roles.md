Retrieve every Entitle Role of a resource or integration. Use this data source when you need to discover available roles for a given resource or integration, search for roles by name, or dynamically reference role IDs in bundles, policies, or other resources.

Roles are the most granular access unit in Entitle — they map to real permission levels within integrated applications (e.g., "read-only", "contributor", "admin"). Each role belongs to exactly one resource. [Read more about roles](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## When to Use This Data Source

- Discovering all roles available within a resource before referencing them
- Searching for a role by name within a known resource (e.g., finding the "admin" role)
- Dynamically referencing roles in bundles, policies, or prerequisite permissions
- Auditing which roles exist on a resource
- Looking up specific roles by ID (up to 100 per data source), across resources and integrations

## Key Concepts

- **Scope**: Set one of `filter.resource_id` (roles of one resource), `filter.integration_id` (every role of an integration, in one data source instead of one per resource) or `filter.ids`. If both `resource_id` and `integration_id` are set, `resource_id` takes precedence. The top-level `resource_id` and `integration_id` are deprecated; move them into `filter`
- **All results**: The data source always returns every matching role. The provider pages through the API with a cursor, so no role is skipped or repeated while the list changes. `filter.page` and `filter.per_page` are deprecated and ignored
- **ids**: `filter.ids` returns specific roles by ID, up to 100; with `ids_operator = "isNot"` it returns every role except them
- **Filters**: `requestable`, `workflow_ids`, `resource_types` and `resource_tags` narrow the results; all filters are combined with AND
- **search**: Matches role names case-insensitively and partially, and can be combined with every other filter. Results are ranked: exact matches first, then case-insensitive matches, then partial matches

## Example Usage

### List All Roles for a Resource

```terraform
data "entitle_roles" "all" {
  filter {
    resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }
}

output "all_role_ids" {
  value = data.entitle_roles.all.roles[*].id
}
```

### Search for a Specific Role by Name

```terraform
data "entitle_roles" "admin_role" {
  filter {
    resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
    search      = "admin"
  }
}

output "admin_role_id" {
  value = data.entitle_roles.admin_role.roles[0].id
}
```

### Use Roles in a Bundle

Find and reference roles dynamically when building a bundle:

```terraform
data "entitle_integration" "github" {
  name = "GitHub Production"
}

data "entitle_resources" "backend_repo" {
  integration_id = data.entitle_integration.github.id
  filter { search = "backend-api" }
}

data "entitle_roles" "read_roles" {
  filter {
    resource_id = data.entitle_resources.backend_repo.resources[0].id
    search      = "read"
  }
}

resource "entitle_bundle" "github_read_bundle" {
  name        = "GitHub Backend Read"
  description = "Read access to the backend API repository"

  workflow = {
    id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }

  roles = [{
    id = data.entitle_roles.read_roles.roles[0].id
  }]

  allowed_durations = [28800, 86400]
}
```

### List Every Role of an Integration

```terraform
data "entitle_roles" "aws" {
  filter {
    integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }
}
```

### Filter by Requestability, Workflow and Resource Tags

```terraform
data "entitle_roles" "prod_not_on_workflow" {
  filter {
    integration_id        = "7d080bfa-9143-11ee-b9d1-0242ac120001"
    requestable           = true
    workflow_ids          = ["7d080bfa-9143-11ee-b9d1-0242ac120002"]
    workflow_ids_operator = "isNot"
    resource_tags         = ["env: prod"]
  }
}
```

### Filter by Resource Type

```terraform
data "entitle_roles" "policies" {
  filter {
    integration_id          = "7d080bfa-9143-11ee-b9d1-0242ac120001"
    resource_types          = ["Policy"]
    resource_types_operator = "contains"
  }
}
```

### Look Up Specific Roles by ID

```terraform
data "entitle_roles" "selected" {
  filter {
    ids = [
      "7d080bfa-9143-11ee-b9d1-0242ac120003",
      "7d080bfa-9143-11ee-b9d1-0242ac120004",
    ]
  }
}
```

### Output All Role Details

```terraform
data "entitle_roles" "all_roles" {
  filter {
    resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  }
}

output "roles" {
  value = data.entitle_roles.all_roles.roles
}
```
