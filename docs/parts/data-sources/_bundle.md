An Entitle Bundle is a cross-application package of permissions that can be requested, approved, or revoked in a single action. Bundles group multiple roles from different applications and resources into a logical unit — a "super role" that grants everything a user needs for a specific function or project.

Use this data source to look up an existing bundle by ID or name, for example to reference it in a policy for automatic provisioning, or to inspect its configuration. [Read more about bundles](https://docs.beyondtrust.com/entitle/docs/bundles).

## Key Concepts

- **Bundle**: A named collection of roles across applications, requestable as a single unit
- **Roles**: The individual permissions in the bundle (each linked to a specific resource and integration)
- **Workflow**: The approval process triggered when a user requests the bundle
- **Category**: An organizational label for the bundle (e.g., "Engineering", "Finance")
- **Allowed Durations**: Available access time windows for the bundle
- **ID vs Name lookup**: Query by `id` for performance-critical automation; use `name` for human-friendly lookups

## When to Use This Data Source

- Referencing an existing bundle in a policy (`entitle_policy`) for birthright access assignment
- Looking up bundle IDs for use in outputs, scripts, or other resources
- Inspecting the roles and workflow configuration of a bundle

## Example Usage

### Look Up a Bundle by ID

```terraform
data "entitle_bundle" "engineering_bundle" {
  id = "7d080bfa-9143-11ee-b9d1-0242ac120001"
}

output "bundle_name" {
  value = data.entitle_bundle.engineering_bundle.name
}
```

### Look Up a Bundle by Name

```terraform
data "entitle_bundle" "dev_tools" {
  name = "Developer Tools"
}

output "bundle_roles" {
  value = data.entitle_bundle.dev_tools.roles
}
```

### Reference a Bundle in a Policy

```terraform
data "entitle_bundle" "standard_access" {
  name = "Standard Employee Access"
}

resource "entitle_policy" "all_employees" {
  in_groups = [{
    id   = "7d080bfa-9143-11ee-b9d1-0242ac120001"
    type = "group"
  }]

  bundles = [{
    id = data.entitle_bundle.standard_access.id
  }]
}
```

### Inspect Bundle Contents

```terraform
data "entitle_bundle" "finance_bundle" {
  name = "Finance Team Tools"
}

output "bundle_details" {
  value = {
    id               = data.entitle_bundle.finance_bundle.id
    description      = data.entitle_bundle.finance_bundle.description
    category         = data.entitle_bundle.finance_bundle.category
    tags             = data.entitle_bundle.finance_bundle.tags
    allowed_durations = data.entitle_bundle.finance_bundle.allowed_durations
    role_count       = length(data.entitle_bundle.finance_bundle.roles)
  }
}
```

## Query Parameters

### Optional (at least one should be provided)

- `id` (String) The unique identifier of the bundle (UUID format). Preferred for performance-critical queries.
- `name` (String) The bundle's display name. When querying by name, the provider paginates through all bundles until a match is found — in large organizations this may be slower than ID-based lookups.

## Returned Attributes

- `id` (String) The bundle's unique identifier (UUID format).
- `name` (String) The bundle's display name.
- `description` (String) The bundle's extended description.
- `category` (String) The organizational category (e.g., "Engineering", "Finance").
- `tags` (Set of String) Searchable metadata tags.
- `allowed_durations` (Set of Number) Available access duration options in seconds (`-1` = permanent).
- `workflow` (Attributes) The approval workflow for bundle requests:
    - `id` (String) The workflow's unique identifier.
    - `name` (String) The workflow's display name.
- `roles` (Attributes List) The roles included in this bundle:
    - `id` (String) The role's unique identifier.
    - `name` (String) The role's display name.
    - `resource` (Attributes) The resource associated with the role:
        - `id` (String) The resource's unique identifier.
        - `name` (String) The resource's display name.
        - `integration` (Attributes) The integration associated with the resource:
            - `id` (String) The integration's unique identifier.
            - `name` (String) The integration's display name.
            - `application` (Attributes):
                - `name` (String) The application's name.

## Notes

- Prefer `id` over `name` for programmatic or CI/CD use cases to avoid performance issues in large organizations
- If neither `id` nor `name` is provided, the query will fail
