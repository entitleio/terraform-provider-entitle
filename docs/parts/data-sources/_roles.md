Retrieve a list of Entitle Roles within a specific resource. Use this data source when you need to discover available roles for a given resource or integration, search for roles by name, or dynamically reference role IDs in bundles, policies, or other resources.

Roles are the most granular access unit in Entitle — they map to real permission levels within integrated applications (e.g., "read-only", "contributor", "admin"). Each role belongs to exactly one resource. [Read more about roles](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).

## When to Use This Data Source

- Discovering all roles available within a resource before referencing them
- Searching for a role by name within a known resource (e.g., finding the "admin" role)
- Dynamically referencing roles in bundles, policies, or prerequisite permissions
- Auditing which roles exist on a resource
- Looking up specific roles by ID (up to 100 per data source), across resources and integrations

## Key Concepts

- **Scope**: Set one of `resource_id` (roles of one resource), `integration_id` (every role of an integration) or `filter.ids`
- **ids**: `filter.ids` returns specific roles by ID, up to 100; with `ids_operator = "isNot"` it returns every role except them
- **after_id**: A cursor for reading a long list page by page without skipping or repeating items while it changes. Cannot be combined with a `page` other than 1 or with `search`
- **search and ids**: `search` cannot be combined with `filter.ids` unless `external_id` is also set
- **Pagination**: At most 100 results per page

## Example Usage

### List All Roles for a Resource

```terraform
data "entitle_roles" "all" {
  resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "all_role_ids" {
  value = data.entitle_roles.all.roles[*].id
}
```

### Search for a Specific Role by Name

```terraform
data "entitle_roles" "admin_role" {
  resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  filter {
    search = "admin"
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
  resource_id = data.entitle_resources.backend_repo.resources[0].id
  filter { search = "read" }
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

### Paginate Through Large Resource Role Lists

```terraform
data "entitle_roles" "page_one" {
  resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  filter {
    page     = 1
    per_page = 50
  }
}
```

### Paginate Through Large Integration Role Lists

```terraform
data "entitle_roles" "page_one" {
  integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  filter {
    page     = 1
    per_page = 50
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

### Read an Integration's Roles with a Cursor

Pass the ID of the last role of the previous page as `after_id`:

```terraform
data "entitle_roles" "next_page" {
  integration_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
  filter {
    after_id = "7d080bfa-9143-11ee-b9d1-0242ac120099"
    per_page = 100
  }
}
```

### Output All Role Details

```terraform
data "entitle_roles" "all_roles" {
  resource_id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "roles" {
  value = data.entitle_roles.all_roles.roles
}
```
